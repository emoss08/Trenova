package realtimebroker

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var ErrPublisherStopped = errors.New("realtime publisher is stopped")

type PublisherParams struct {
	fx.In

	Client  *redis.Client
	Config  *config.Config
	Logger  *zap.Logger
	Metrics *metrics.Registry `optional:"true"`
	LC      fx.Lifecycle
}

// Publisher writes events to the sharded bus.
//
// A domain write that changes something a browser shows asks for a publish on
// its way out, often several times per record. Those publishes must never cost
// the write its latency or its success, so Enqueue only places the event on a
// bounded queue and a single writer drains it in pipelined batches. One writer
// also keeps the order the process published in, which is the order readers
// see.
type Publisher struct {
	client   *redis.Client
	cfg      config.RealtimeConfig
	l        *zap.Logger
	metrics  *metrics.Realtime
	queue    chan *services.RealtimeEnvelope
	stopOnce sync.Once
	stop     chan struct{}
	done     chan struct{}
	mu       sync.RWMutex
	stopped  bool
}

func NewPublisher(p PublisherParams) *Publisher {
	cfg := *p.Config.GetRealtimeConfig()
	pub := &Publisher{
		client: p.Client,
		cfg:    cfg,
		l:      p.Logger.Named("realtime.publisher"),
		queue:  make(chan *services.RealtimeEnvelope, cfg.GetPublishQueueSize()),
		stop:   make(chan struct{}),
		done:   make(chan struct{}),
	}
	if p.Metrics != nil {
		pub.metrics = p.Metrics.Realtime
	}

	p.LC.Append(fx.Hook{
		OnStart: func(context.Context) error {
			go pub.run()
			return nil
		},
		OnStop: pub.Stop,
	})

	return pub
}

func (p *Publisher) Enqueue(env *services.RealtimeEnvelope) bool {
	if env == nil {
		return false
	}

	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.stopped {
		p.metrics.RecordPublishDropped("stopped")
		return false
	}

	select {
	case p.queue <- env:
		return true
	default:
		p.metrics.RecordPublishDropped("queue_full")
		p.l.Warn("realtime publish queue is full; event dropped",
			zap.String("event", env.Event),
			zap.Int("capacity", cap(p.queue)),
		)
		return false
	}
}

func (p *Publisher) Publish(ctx context.Context, env *services.RealtimeEnvelope) error {
	if env == nil {
		return nil
	}

	shards := p.cfg.GetShardCount()
	writeCtx, cancel := context.WithTimeout(ctx, p.cfg.GetPublishTimeout())
	defer cancel()

	err := p.client.XAdd(writeCtx, p.addArgs(env, shards)).Err()
	if err != nil {
		p.metrics.RecordPublished(env.Event, "error", 1)
		return fmt.Errorf("publish realtime event: %w", err)
	}

	p.metrics.RecordPublished(env.Event, "ok", 1)
	return nil
}

// Stop flushes what is already queued and refuses anything new. It waits for
// the writer no longer than the stop context allows.
func (p *Publisher) Stop(ctx context.Context) error {
	p.stopOnce.Do(func() {
		p.mu.Lock()
		p.stopped = true
		p.mu.Unlock()
		close(p.stop)
	})

	select {
	case <-p.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (p *Publisher) run() {
	defer close(p.done)

	batchSize := p.cfg.GetPublishBatchSize()
	batch := make([]*services.RealtimeEnvelope, 0, batchSize)

	for {
		select {
		case env := <-p.queue:
			batch = append(batch, env)
			batch = p.fill(batch, batchSize)
			p.flush(batch)
			clear(batch)
			batch = batch[:0]
		case <-p.stop:
			p.drain(batch, batchSize)
			return
		}
	}
}

func (p *Publisher) fill(
	batch []*services.RealtimeEnvelope,
	batchSize int,
) []*services.RealtimeEnvelope {
	for len(batch) < batchSize {
		select {
		case env := <-p.queue:
			batch = append(batch, env)
		default:
			return batch
		}
	}
	return batch
}

func (p *Publisher) drain(batch []*services.RealtimeEnvelope, batchSize int) {
	for {
		batch = p.fill(batch[:0], batchSize)
		if len(batch) == 0 {
			return
		}
		p.flush(batch)
		clear(batch)
	}
}

func (p *Publisher) flush(batch []*services.RealtimeEnvelope) {
	if len(batch) == 0 {
		return
	}

	started := time.Now()
	shards := p.cfg.GetShardCount()
	ctx, cancel := context.WithTimeout(context.Background(), p.cfg.GetPublishTimeout())
	defer cancel()

	pipe := p.client.Pipeline()
	cmds := make([]*redis.StringCmd, len(batch))
	for i, env := range batch {
		cmds[i] = pipe.XAdd(ctx, p.addArgs(env, shards))
	}
	_, err := pipe.Exec(ctx)

	p.metrics.RecordFlush(len(batch), time.Since(started))

	if err == nil {
		for _, env := range batch {
			p.metrics.RecordPublished(env.Event, "ok", 1)
		}
		return
	}

	failed := 0
	for i, cmd := range cmds {
		if cmd.Err() != nil {
			failed++
			p.metrics.RecordPublished(batch[i].Event, "error", 1)
			continue
		}
		p.metrics.RecordPublished(batch[i].Event, "ok", 1)
	}
	p.l.Warn("realtime events were not written to the bus",
		zap.Int("failed", failed),
		zap.Int("batch", len(batch)),
		zap.Error(err),
	)
}

func (p *Publisher) addArgs(env *services.RealtimeEnvelope, shards int) *redis.XAddArgs {
	tenant := tenantKey(env.OrganizationID, env.BusinessUnitID)
	values := make([]any, 0, 18)
	values = append(values,
		fieldTenant, tenant,
		fieldEvent, env.Event,
		fieldPayload, env.Payload,
	)
	if env.AudienceUserID.IsNotNil() {
		values = append(values, fieldAudience, env.AudienceUserID.String())
	}
	if env.Scope != "" {
		values = append(values, fieldScope, env.Scope)
	}
	if env.PortalPayload != nil {
		values = append(values, fieldPortal, env.PortalPayload)
	}
	if env.Ephemeral {
		values = append(values, fieldEphemeral, "1")
	}
	if env.PresenceConnectionID != "" {
		values = append(values,
			fieldConnection, env.PresenceConnectionID,
			fieldAction, env.PresenceAction,
		)
	}

	return &redis.XAddArgs{
		Stream: streamKey(shardFor(tenant, shards)),
		MaxLen: p.cfg.GetStreamMaxLen(),
		Approx: true,
		ID:     "*",
		Values: values,
	}
}
