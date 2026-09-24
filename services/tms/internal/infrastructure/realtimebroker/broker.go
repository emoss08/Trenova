package realtimebroker

import (
	"context"
	"errors"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	fanoutBlock        = 5 * time.Second
	fanoutBatch        = 512
	fanoutRetryMin     = 250 * time.Millisecond
	fanoutRetryMax     = 10 * time.Second
	refreshChunk       = 500
	sweepBatch         = 500
	maintenanceTimeout = 10 * time.Second
	cleanupQueueSize   = 4096
)

var (
	ErrBrokerClosed = errors.New("realtime broker is closed")

	errTooManyConnections = errortypes.NewRateLimitError(
		"realtime",
		"Too many live connections are open for this account. Close a tab and try again.",
	)
	errInstanceFull = errortypes.NewRateLimitError(
		"realtime",
		"Live updates are at capacity on this server. Retrying shortly.",
	)
)

type BrokerParams struct {
	fx.In

	Publisher *Publisher
	Client    *redis.Client
	Config    *config.Config
	Logger    *zap.Logger
	Metrics   *metrics.Registry `optional:"true"`
	LC        fx.Lifecycle
}

// Broker fans the bus out to the streams open on this instance and keeps the
// presence register.
//
// Every instance runs one reader over every shard, from where it started; a
// tenant's events all land on one shard, so one tenant's order is the shard's
// order. A reader that reconnects gives back the last cursor it applied and is
// replayed from its shard, as far back as the stream still holds.
type Broker struct {
	*Publisher

	client  *redis.Client
	cfg     config.RealtimeConfig
	l       *zap.Logger
	metrics *metrics.Realtime

	mu      sync.RWMutex
	tenants map[string]map[*subscriber]struct{}
	perUser map[string]int
	total   int
	closed  bool

	cleanups chan cleanupJob

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func NewBroker(p BrokerParams) *Broker {
	ctx, cancel := context.WithCancel(context.Background())
	b := &Broker{
		Publisher: p.Publisher,
		client:    p.Client,
		cfg:       *p.Config.GetRealtimeConfig(),
		l:         p.Logger.Named("realtime.broker"),
		tenants:   make(map[string]map[*subscriber]struct{}),
		perUser:   make(map[string]int),
		cleanups:  make(chan cleanupJob, cleanupQueueSize),
		ctx:       ctx,
		cancel:    cancel,
	}
	if p.Metrics != nil {
		b.metrics = p.Metrics.Realtime
	}

	p.LC.Append(fx.Hook{
		OnStart: b.start,
		OnStop:  b.stop,
	})

	return b
}

func (b *Broker) start(ctx context.Context) error {
	cursors, err := b.headCursors(ctx)
	if err != nil {
		return err
	}

	// The reader is not waited for on stop. A blocking read does not return
	// on cancellation until its block expires, it holds nothing that needs
	// releasing, and it exits on its next wake-up.
	go b.fanout(cursors)
	b.wg.Add(2)
	go b.maintain()
	go b.cleanup()

	b.l.Info("realtime broker started", zap.Int("shards", b.cfg.GetShardCount()))
	return nil
}

func (b *Broker) stop(ctx context.Context) error {
	b.Drain()
	b.cancel()

	done := make(chan struct{})
	go func() {
		b.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Drain refuses new streams and closes every open one with a shutdown notice.
func (b *Broker) Drain() {
	b.mu.Lock()
	b.closed = true
	subs := make([]*subscriber, 0, b.total)
	for _, set := range b.tenants {
		for sub := range set {
			subs = append(subs, sub)
		}
	}
	b.mu.Unlock()

	for _, sub := range subs {
		sub.shutdown(services.RealtimeCloseShutdown)
	}
}

// headCursors is where this instance starts reading: the newest entry of each
// shard. Reading from "$" instead would lose whatever was written between one
// blocking read returning and the next starting.
func (b *Broker) headCursors(ctx context.Context) ([]string, error) {
	shards := b.cfg.GetShardCount()
	pipe := b.client.Pipeline()
	cmds := make([]*redis.XMessageSliceCmd, shards)
	for shard := range shards {
		cmds[shard] = pipe.XRevRangeN(ctx, streamKey(shard), "+", "-", 1)
	}
	if _, err := pipe.Exec(ctx); err != nil && !errors.Is(err, redis.Nil) {
		return nil, err
	}

	cursors := make([]string, shards)
	for shard, cmd := range cmds {
		cursors[shard] = "0-0"
		if msgs, err := cmd.Result(); err == nil && len(msgs) > 0 {
			cursors[shard] = msgs[0].ID
		}
	}
	return cursors, nil
}

func (b *Broker) fanout(cursors []string) {
	shards := len(cursors)
	streams := make([]string, shards*2)
	shardOf := make(map[string]int, shards)
	for shard := range shards {
		streams[shard] = streamKey(shard)
		shardOf[streams[shard]] = shard
	}

	backoff := fanoutRetryMin
	for {
		if b.ctx.Err() != nil {
			return
		}

		copy(streams[shards:], cursors)
		results, err := b.client.XRead(b.ctx, &redis.XReadArgs{
			Streams: streams,
			Count:   fanoutBatch,
			Block:   fanoutBlock,
		}).Result()
		if err != nil {
			if errors.Is(err, redis.Nil) {
				backoff = fanoutRetryMin
				continue
			}
			if b.ctx.Err() != nil {
				return
			}
			b.metrics.RecordFanoutError()
			b.l.Warn("realtime fan-out read failed; retrying", zap.Error(err))
			if !b.sleep(jitter(backoff)) {
				return
			}
			backoff = min(backoff*2, fanoutRetryMax)
			continue
		}
		backoff = fanoutRetryMin

		for _, stream := range results {
			shard, known := shardOf[stream.Stream]
			if !known {
				continue
			}
			for i := range stream.Messages {
				msg := &stream.Messages[i]
				cursors[shard] = msg.ID
				if e, ok := decodeEntry(shard, msg); ok {
					b.dispatch(e)
				}
			}
		}
	}
}

func (b *Broker) dispatch(e *entry) {
	b.mu.RLock()
	set := b.tenants[e.tenant]
	targets := make([]*subscriber, 0, len(set))
	for sub := range set {
		targets = append(targets, sub)
	}
	b.mu.RUnlock()

	for _, sub := range targets {
		if sub.deliver(e) {
			b.metrics.RecordDelivered(e.event)
		}
	}
}

func (b *Broker) Subscribe(
	ctx context.Context,
	req *services.RealtimeSubscribeRequest,
) (services.RealtimeStream, error) {
	conn := req.Connection
	tenant := tenantKey(conn.OrganizationID, conn.BusinessUnitID)
	shard := shardFor(tenant, b.cfg.GetShardCount())
	sub := newSubscriber(conn, tenant, shard, b.cfg.GetSubscriberBuffer(), b.unregister)

	if err := b.register(sub); err != nil {
		return nil, err
	}

	prelude, floor, err := b.setup(ctx, sub, req)
	if err != nil {
		sub.shutdown(services.RealtimeCloseShutdown)
		return nil, err
	}

	sub.release(prelude, floor)
	return sub, nil
}

func (b *Broker) register(sub *subscriber) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		b.metrics.RecordConnectionOpened("closed")
		return errInstanceFull
	}
	if b.total >= b.cfg.GetMaxConnections() {
		b.metrics.RecordConnectionOpened("instance_full")
		return errInstanceFull
	}
	if b.perUser[sub.userID] >= b.cfg.GetMaxConnectionsPerUser() {
		b.metrics.RecordConnectionOpened("user_limit")
		return errTooManyConnections
	}

	set := b.tenants[sub.tenant]
	if set == nil {
		set = make(map[*subscriber]struct{})
		b.tenants[sub.tenant] = set
	}
	set[sub] = struct{}{}
	b.perUser[sub.userID]++
	b.total++
	b.metrics.RecordConnectionOpened("accepted")

	return nil
}

func (b *Broker) unregister(sub *subscriber) {
	b.mu.Lock()
	b.forget(sub)
	b.mu.Unlock()

	b.metrics.RecordConnectionClosed(sub.reason)

	job := cleanupJob{conn: sub.conn, tenant: sub.tenant, scopes: sub.joinedScopes()}
	select {
	case b.cleanups <- job:
	default:
		b.l.Warn("realtime cleanup queue is full; presence will expire on its own",
			zap.String("connectionId", sub.conn.ConnectionID),
		)
	}
}

// forget drops sub from the hub's indexes. The caller holds b.mu.
func (b *Broker) forget(sub *subscriber) {
	set := b.tenants[sub.tenant]
	if _, ok := set[sub]; !ok {
		return
	}

	delete(set, sub)
	if len(set) == 0 {
		delete(b.tenants, sub.tenant)
	}
	b.perUser[sub.userID]--
	if b.perUser[sub.userID] <= 0 {
		delete(b.perUser, sub.userID)
	}
	b.total--
}

// cleanupJob is a closed connection's presence to withdraw. The work leaves the
// goroutine that closed the stream, since that is often the fan-out itself
// shedding a slow reader, and it must not wait on Redis.
type cleanupJob struct {
	conn   services.RealtimeConnection
	tenant string
	scopes []string
}

func (b *Broker) cleanup() {
	defer b.wg.Done()

	run := func(job cleanupJob) {
		ctx, cancel := context.WithTimeout(context.Background(), maintenanceTimeout)
		defer cancel()
		b.dropConnection(ctx, &job.conn, job.tenant, job.scopes)
	}

	for {
		select {
		case job := <-b.cleanups:
			run(job)
		case <-b.ctx.Done():
			for {
				select {
				case job := <-b.cleanups:
					run(job)
				default:
					return
				}
			}
		}
	}
}

// setup registers the connection, joins the tenant's user presence when
// asked, and builds what the reader is sent before anything live.
func (b *Broker) setup(
	ctx context.Context,
	sub *subscriber,
	req *services.RealtimeSubscribeRequest,
) ([]services.RealtimeFrame, streamID, error) {
	if err := b.registerConnection(ctx, &sub.conn); err != nil {
		return nil, streamID{}, err
	}

	replay, floor, reset := b.replay(ctx, sub, req.LastEventID)
	resumed := req.LastEventID != "" && !reset

	var head string
	if !resumed {
		head = b.headCursor(ctx, sub.shard)
	}

	ready, err := encodeFrame(services.RealtimeEventReady, services.RealtimeReadyEvent{
		ConnectionID:        sub.conn.ConnectionID,
		HeartbeatIntervalMs: b.cfg.GetHeartbeatInterval().Milliseconds(),
		Resumed:             resumed,
		Cursor:              head,
	})
	if err != nil {
		return nil, streamID{}, err
	}

	prelude := make([]services.RealtimeFrame, 0, len(replay)+3)
	prelude = append(prelude, ready)
	if reset {
		prelude = append(prelude, services.RealtimeFrame{
			Event: services.RealtimeEventReset,
			Data:  []byte("{}"),
		})
	}

	if req.JoinUsers {
		sub.join(services.RealtimeScopeUsers)
		snapshot, joinErr := b.JoinPresence(ctx, &sub.conn, services.RealtimeScopeUsers)
		if joinErr != nil {
			return nil, streamID{}, joinErr
		}
		frame, encErr := encodeFrame(services.RealtimeEventPresenceState, snapshot)
		if encErr != nil {
			return nil, streamID{}, encErr
		}
		prelude = append(prelude, frame)
	}

	return append(prelude, replay...), floor, nil
}

// headCursor is the newest position of shard, read after the stream is
// registered. Anything written after it reaches the reader live, so resuming
// from it later misses nothing; anything at or before it that was already
// buffered arrives too and at worst repeats on a resume.
func (b *Broker) headCursor(ctx context.Context, shard int) string {
	msgs, err := b.client.XRevRangeN(ctx, streamKey(shard), "+", "-", 1).Result()
	if err != nil || len(msgs) == 0 {
		return formatCursor(shard, "0-0")
	}
	return formatCursor(shard, msgs[0].ID)
}

// replay reads what the reader missed. It reports reset when that cannot be
// known exactly — the cursor is malformed, from another shard layout, or older
// than the stream still holds — and the reader must refetch instead.
func (b *Broker) replay(
	ctx context.Context,
	sub *subscriber,
	lastEventID string,
) ([]services.RealtimeFrame, streamID, bool) {
	if lastEventID == "" {
		return nil, streamID{}, false
	}

	pos, ok := parseCursor(lastEventID)
	if !ok || pos.shard != sub.shard {
		b.metrics.RecordReplay("reset_cursor")
		return nil, streamID{}, true
	}

	key := streamKey(sub.shard)
	oldest, err := b.client.XRangeN(ctx, key, "-", "+", 1).Result()
	if err != nil || len(oldest) == 0 {
		b.metrics.RecordReplay("reset_empty")
		return nil, streamID{}, true
	}
	if first, parsed := parseStreamID(oldest[0].ID); !parsed || pos.id.Less(first) {
		b.metrics.RecordReplay("reset_trimmed")
		return nil, streamID{}, true
	}

	limit := b.cfg.GetReplayScanLimit()
	msgs, err := b.client.XRangeN(ctx, key, "("+pos.id.String(), "+", limit+1).Result()
	if err != nil {
		b.metrics.RecordReplay("reset_error")
		return nil, streamID{}, true
	}
	if int64(len(msgs)) > limit {
		b.metrics.RecordReplay("reset_behind")
		return nil, streamID{}, true
	}

	floor := pos.id
	frames := make([]services.RealtimeFrame, 0, len(msgs))
	for i := range msgs {
		e, decoded := decodeEntry(sub.shard, &msgs[i])
		if !decoded {
			continue
		}
		floor = e.id
		if e.tenant != sub.tenant || e.ephemeral {
			continue
		}
		payload, accepted := sub.accepts(e)
		if !accepted {
			continue
		}
		frames = append(frames, services.RealtimeFrame{ID: e.cursor, Event: e.event, Data: payload})
	}

	b.metrics.RecordReplay("resumed")
	return frames, floor, false
}

func (b *Broker) maintain() {
	defer b.wg.Done()

	ticker := time.NewTicker(b.cfg.GetHeartbeatInterval())
	defer ticker.Stop()

	heartbeat := services.RealtimeFrame{Event: services.RealtimeEventHeartbeat, Data: []byte("{}")}
	for {
		select {
		case <-b.ctx.Done():
			return
		case <-ticker.C:
			subs := b.snapshot()
			for _, sub := range subs {
				sub.send(heartbeat)
			}
			b.refresh(subs)
			b.sweep()
		}
	}
}

func (b *Broker) snapshot() []*subscriber {
	b.mu.RLock()
	defer b.mu.RUnlock()
	subs := make([]*subscriber, 0, b.total)
	for _, set := range b.tenants {
		for sub := range set {
			subs = append(subs, sub)
		}
	}
	return subs
}

func (b *Broker) sleep(d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-b.ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func jitter(d time.Duration) time.Duration {
	half := int64(d / 2)
	if half <= 0 {
		return d
	}
	return time.Duration(half + rand.Int64N(half)) //nolint:gosec // backoff jitter, not security
}

func decodeEntry(shard int, msg *redis.XMessage) (*entry, bool) {
	id, ok := parseStreamID(msg.ID)
	if !ok {
		return nil, false
	}

	e := &entry{id: id, cursor: formatCursor(shard, msg.ID)}
	e.tenant = stringField(msg.Values, fieldTenant)
	e.event = stringField(msg.Values, fieldEvent)
	if e.tenant == "" || e.event == "" {
		return nil, false
	}
	e.audience = stringField(msg.Values, fieldAudience)
	e.scope = stringField(msg.Values, fieldScope)
	e.payload = bytesField(msg.Values, fieldPayload)
	e.portal = bytesField(msg.Values, fieldPortal)
	e.ephemeral = stringField(msg.Values, fieldEphemeral) == "1"
	e.connection = stringField(msg.Values, fieldConnection)
	e.action = stringField(msg.Values, fieldAction)

	return e, true
}

func stringField(values map[string]any, key string) string {
	if v, ok := values[key].(string); ok {
		return v
	}
	return ""
}

func bytesField(values map[string]any, key string) []byte {
	v, ok := values[key].(string)
	if !ok {
		return nil
	}
	return []byte(v)
}

func encodeFrame(event string, data any) (services.RealtimeFrame, error) {
	encoded, err := sonic.Marshal(data)
	if err != nil {
		return services.RealtimeFrame{}, err
	}
	return services.RealtimeFrame{Event: event, Data: encoded}, nil
}
