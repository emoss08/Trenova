package realtimebroker

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/cenkalti/backoff/v5"
	"github.com/emoss08/trenova/internal/infrastructure/config/realtimeconfig"
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	watchFeedRead  = 512
	watchFeedBlock = 5 * time.Second
	// watchKeyGrace keeps a tenant's register a while past its last watch,
	// so a register is never dropped under a wait still open.
	watchKeyGrace = time.Hour
	// staleConsumerIdle is how long a follower may go unseen before its
	// entry in the group is dropped. Followers read without acknowledging,
	// so a dropped one leaves nothing pending behind.
	staleConsumerIdle = 24 * time.Hour
)

func watchKey(tenant string) string {
	return keyPrefix + "watch:" + tenant
}

// unwatchScript removes a record from a tenant's register only when it is
// not watched past the caller's expiry: a later wait on the same record
// registered it again with its own, later expiry, and keeps it.
var unwatchScript = redis.NewScript(`
local score = redis.call('ZSCORE', KEYS[1], ARGV[1])
if score and tonumber(score) <= tonumber(ARGV[2]) then
  return redis.call('ZREM', KEYS[1], ARGV[1])
end
return 0
`)

type RecordWatchFeedParams struct {
	fx.In

	Client         *redis.Client
	RealtimeConfig *realtimeconfig.Config
	Logger         *zap.Logger
}

// RecordWatchFeed reads the realtime stream for the records something waits
// on. Each tenant's watched records are a sorted set scored by when the
// watch ends, so an expired watch costs nothing and is trimmed as new ones
// arrive. A follower reads every shard through a consumer group without
// acknowledging (at most once: a change lost to a crash is what the wait's
// own expiry is for) and checks each batch against the registers in one
// pipelined round trip.
type RecordWatchFeed struct {
	client *redis.Client
	shards int
	l      *zap.Logger
	now    func() int64
}

func NewRecordWatchFeed(p RecordWatchFeedParams) realtimeports.RecordWatchFeed {
	return &RecordWatchFeed{
		client: p.Client,
		shards: p.RealtimeConfig.GetShardCount(),
		l:      p.Logger.Named("realtime.watch-feed"),
		now:    timeutils.NowUnix,
	}
}

func (f *RecordWatchFeed) Watch(ctx context.Context, req *realtimeports.WatchRecordRequest) error {
	if req.RecordID.IsNil() {
		return errors.New("a watched record needs an id")
	}
	key := watchKey(tenantKey(req.OrganizationID, req.BusinessUnitID))
	now := f.now()
	ttl := time.Duration(max(req.Until-now, 0))*time.Second + watchKeyGrace

	pipe := f.client.TxPipeline()
	pipe.ZAddGT(ctx, key, redis.Z{Score: float64(req.Until), Member: req.RecordID.String()})
	pipe.ZRemRangeByScore(ctx, key, "-inf", fmt.Sprintf("(%d", now))
	pipe.ExpireNX(ctx, key, ttl)
	pipe.ExpireGT(ctx, key, ttl)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("register the watched record: %w", err)
	}

	return nil
}

func (f *RecordWatchFeed) Unwatch(
	ctx context.Context,
	req *realtimeports.WatchRecordRequest,
) error {
	key := watchKey(tenantKey(req.OrganizationID, req.BusinessUnitID))
	if err := unwatchScript.Run(ctx, f.client, []string{key},
		req.RecordID.String(), req.Until).Err(); err != nil && !errors.Is(err, redis.Nil) {
		return fmt.Errorf("unregister the watched record: %w", err)
	}

	return nil
}

func (f *RecordWatchFeed) Follow(
	ctx context.Context,
	group string,
	handle func(ctx context.Context, changes []realtimeports.TenantRecordChange),
) error {
	consumer := consumerName()
	keys := make([]string, 0, f.shards)
	shardOf := make(map[string]int, f.shards)
	for shard := range f.shards {
		key := streamKey(shard)
		keys = append(keys, key)
		shardOf[key] = shard
	}
	if err := f.join(ctx, keys, group); err != nil {
		return err
	}
	f.dropStaleConsumers(ctx, keys, group, consumer)

	streams := make([]string, 0, len(keys)*2)
	streams = append(streams, keys...)
	for range keys {
		streams = append(streams, ">")
	}

	retry := backoff.NewExponentialBackOff()
	retry.MaxInterval = 30 * time.Second
	for ctx.Err() == nil {
		read, err := f.client.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    group,
			Consumer: consumer,
			Streams:  streams,
			Count:    watchFeedRead,
			Block:    watchFeedBlock,
			NoAck:    true,
		}).Result()
		switch {
		case err == nil:
			retry.Reset()
		case errors.Is(err, redis.Nil) || ctx.Err() != nil:
			continue
		case strings.Contains(err.Error(), "NOGROUP"):
			if jErr := f.join(ctx, keys, group); jErr != nil {
				f.l.Warn("could not join the realtime stream again", zap.Error(jErr))
			}
			f.pause(ctx, retry.NextBackOff())
			continue
		default:
			f.l.Warn("could not read the realtime stream for watched records", zap.Error(err))
			f.pause(ctx, retry.NextBackOff())
			continue
		}

		if changes := f.watched(ctx, read, shardOf); len(changes) > 0 {
			handle(ctx, changes)
		}
	}

	return nil
}

func (f *RecordWatchFeed) pause(ctx context.Context, wait time.Duration) {
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case <-ctx.Done():
	case <-timer.C:
	}
}

// join creates the group on every shard's stream, starting at its end: a
// follower hears what changes from now on.
func (f *RecordWatchFeed) join(ctx context.Context, keys []string, group string) error {
	for _, key := range keys {
		err := f.client.XGroupCreateMkStream(ctx, key, group, "$").Err()
		if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
			return fmt.Errorf("join the realtime stream %s: %w", key, err)
		}
	}

	return nil
}

// dropStaleConsumers removes the group's followers that have not read in a
// day: processes long gone. Each holds nothing pending, since nothing is
// acknowledged.
func (f *RecordWatchFeed) dropStaleConsumers(
	ctx context.Context,
	keys []string,
	group, self string,
) {
	for _, key := range keys {
		consumers, err := f.client.XInfoConsumers(ctx, key, group).Result()
		if err != nil {
			continue
		}
		for _, c := range consumers {
			if c.Name == self || c.Idle < staleConsumerIdle {
				continue
			}
			if dErr := f.client.XGroupDelConsumer(ctx, key, group, c.Name).Err(); dErr != nil {
				f.l.Debug("could not drop a stale follower", zap.String("consumer", c.Name),
					zap.Error(dErr))
			}
		}
	}
}

type candidate struct {
	org, bu pulid.ID
	change  realtimeports.WatchedRecordChange
}

// watched decodes the batch's tenant-wide invalidations and keeps the ones
// whose record a tenant's register holds, read in one pipelined round trip.
func (f *RecordWatchFeed) watched(
	ctx context.Context,
	read []redis.XStream,
	shardOf map[string]int,
) []realtimeports.TenantRecordChange {
	byTenant := make(map[string][]candidate)
	order := make([]string, 0, 4)
	for _, stream := range read {
		shard := shardOf[stream.Stream]
		for i := range stream.Messages {
			c, tenant, ok := decodeCandidate(shard, &stream.Messages[i])
			if !ok {
				continue
			}
			if _, seen := byTenant[tenant]; !seen {
				order = append(order, tenant)
			}
			byTenant[tenant] = append(byTenant[tenant], c)
		}
	}
	if len(order) == 0 {
		return nil
	}

	pipe := f.client.Pipeline()
	scores := make(map[string]*redis.FloatSliceCmd, len(order))
	for _, tenant := range order {
		members := make([]string, 0, len(byTenant[tenant]))
		for idx := range byTenant[tenant] {
			members = append(members, byTenant[tenant][idx].change.RecordID)
		}
		scores[tenant] = pipe.ZMScore(ctx, watchKey(tenant), members...)
	}
	if _, err := pipe.Exec(ctx); err != nil && !errors.Is(err, redis.Nil) {
		f.l.Warn("could not check changes against the watched records", zap.Error(err))
		return nil
	}

	now := float64(f.now())
	out := make([]realtimeports.TenantRecordChange, 0, 4)
	for _, tenant := range order {
		found := scores[tenant].Val()
		for idx := range byTenant[tenant] {
			if idx >= len(found) || found[idx] < now {
				continue
			}
			c := &byTenant[tenant][idx]
			out = append(out, realtimeports.TenantRecordChange{
				OrganizationID: c.org,
				BusinessUnitID: c.bu,
				Change:         c.change,
			})
		}
	}

	return out
}

// decodeCandidate reads a tenant-wide invalidation of a record: what every
// reader of the tenant may see, never one addressed to a person or a scope.
func decodeCandidate(shard int, msg *redis.XMessage) (candidate, string, bool) {
	e, ok := decodeEntry(shard, msg)
	if !ok || e.event != realtimeports.RealtimeEventInvalidation || e.ephemeral ||
		e.audience != "" || e.scope != "" {
		return candidate{}, "", false
	}
	orgText, buText, found := strings.Cut(e.tenant, ":")
	if !found {
		return candidate{}, "", false
	}
	org, err := pulid.Parse(orgText)
	if err != nil {
		return candidate{}, "", false
	}
	bu, err := pulid.Parse(buText)
	if err != nil {
		return candidate{}, "", false
	}
	var change changedRecord
	if sonic.Unmarshal(e.payload, &change) != nil || change.id() == "" {
		return candidate{}, "", false
	}

	return candidate{
		org: org,
		bu:  bu,
		change: realtimeports.WatchedRecordChange{
			RecordID:    change.id(),
			Resource:    change.Resource,
			Action:      change.Action,
			Fields:      change.Fields,
			ActorType:   change.ActorType,
			ActorUserID: change.ActorUserID,
			At:          change.OccurredAt.Unix(),
		},
	}, e.tenant, true
}

// consumerName names this process in the group: its host, which a container
// platform makes unique per instance. Two processes on one host may share
// it; reading without acknowledging, they still each get their own entries.
func consumerName() string {
	host, err := os.Hostname()
	if err != nil || host == "" {
		return "trenova-worker"
	}

	return host
}
