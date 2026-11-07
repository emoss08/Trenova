//go:build integration

package realtimebroker

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/config/realtimeconfig"
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx/fxtest"
	"go.uber.org/zap"
)

type heard struct {
	mu      sync.Mutex
	changes []realtimeports.TenantRecordChange
}

func (h *heard) handle(_ context.Context, changes []realtimeports.TenantRecordChange) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.changes = append(h.changes, changes...)
}

func (h *heard) snapshot() []realtimeports.TenantRecordChange {
	h.mu.Lock()
	defer h.mu.Unlock()

	return append([]realtimeports.TenantRecordChange(nil), h.changes...)
}

func TestRecordWatchFeed_HandsOnOnlyChangesToWatchedRecords(t *testing.T) {
	client := testutil.SetupTestRedis(t)
	cfg := &config.Config{Realtime: realtimeconfig.Config{ShardCount: 4}}
	lc := fxtest.NewLifecycle(t)
	pub := NewPublisher(PublisherParams{
		Client:         client,
		RealtimeConfig: &cfg.Realtime,
		Logger:         zap.NewNop(),
		LC:             lc,
		Metrics:        disabledMetrics(t),
	})
	lc.RequireStart()
	t.Cleanup(lc.RequireStop)
	feed := NewRecordWatchFeed(RecordWatchFeedParams{
		Client:         client,
		RealtimeConfig: &cfg.Realtime,
		Logger:         zap.NewNop(),
	})

	tn, stranger := newTenant(), newTenant()
	watched, err := pulid.Parse(feedRecord)
	require.NoError(t, err)
	require.NoError(t, feed.Watch(t.Context(), &realtimeports.WatchRecordRequest{
		OrganizationID: tn.org,
		BusinessUnitID: tn.bu,
		RecordID:       watched,
		Until:          time.Now().Add(time.Hour).Unix(),
	}))

	ctx, cancel := context.WithCancel(t.Context())
	got := &heard{}
	done := make(chan struct{})
	go func() {
		defer close(done)
		assert.NoError(t, feed.Follow(ctx, "agent-waits-test", got.handle))
	}()
	t.Cleanup(func() {
		cancel()
		<-done
	})
	require.Eventually(t, func() bool {
		groups, err := client.XInfoGroups(t.Context(), streamKey(0)).Result()
		return err == nil && len(groups) == 1
	}, waitFor, 20*time.Millisecond, "the follower joined the stream")

	require.True(t, pub.Enqueue(recordChanged(t, tn, "shp_01J9UNWATCHEDUNWATCHED000", pulid.Nil)))
	require.True(t, pub.Enqueue(recordChanged(t, stranger, feedRecord, pulid.Nil)))
	require.True(t, pub.Enqueue(recordChanged(t, tn, feedRecord, pulid.MustNew("usr_"))))
	require.True(t, pub.Enqueue(recordChanged(t, tn, feedRecord, pulid.Nil)))

	require.Eventually(t, func() bool { return len(got.snapshot()) > 0 }, waitFor,
		20*time.Millisecond)
	time.Sleep(200 * time.Millisecond)
	changes := got.snapshot()
	require.Len(t, changes, 1,
		"another record, another tenant's record and a change addressed to one person are not "+
			"handed on")
	assert.Equal(t, tn.org, changes[0].OrganizationID)
	assert.Equal(t, tn.bu, changes[0].BusinessUnitID)
	assert.Equal(t, feedRecord, changes[0].Change.RecordID)
	assert.Equal(t, []string{"status"}, changes[0].Change.Fields)

	require.NoError(t, feed.Unwatch(t.Context(), &realtimeports.WatchRecordRequest{
		OrganizationID: tn.org,
		BusinessUnitID: tn.bu,
		RecordID:       watched,
		Until:          time.Now().Add(time.Hour).Unix(),
	}))
	require.True(t, pub.Enqueue(recordChanged(t, tn, feedRecord, pulid.Nil)))
	time.Sleep(500 * time.Millisecond)
	assert.Len(t, got.snapshot(), 1, "an unwatched record is no longer handed on")
}

func TestRecordWatchFeed_AnUnwatchLeavesALaterWatchInPlace(t *testing.T) {
	client := testutil.SetupTestRedis(t)
	cfg := &config.Config{Realtime: realtimeconfig.Config{ShardCount: 4}}
	feed := NewRecordWatchFeed(RecordWatchFeedParams{
		Client:         client,
		RealtimeConfig: &cfg.Realtime,
		Logger:         zap.NewNop(),
	})
	tn := newTenant()
	record := pulid.MustNew("shp_")
	key := watchKey(tenantKey(tn.org, tn.bu))
	early := time.Now().Add(time.Hour).Unix()
	late := time.Now().Add(3 * time.Hour).Unix()

	for _, until := range []int64{late, early} {
		require.NoError(t, feed.Watch(t.Context(), &realtimeports.WatchRecordRequest{
			OrganizationID: tn.org, BusinessUnitID: tn.bu, RecordID: record, Until: until,
		}))
	}
	score, err := client.ZScore(t.Context(), key, record.String()).Result()
	require.NoError(t, err)
	assert.Equal(t, float64(late), score, "the later expiry is kept")
	ttl, err := client.TTL(t.Context(), key).Result()
	require.NoError(t, err)
	assert.Greater(t, ttl, 3*time.Hour, "the register outlives its last watch")

	require.NoError(t, feed.Unwatch(t.Context(), &realtimeports.WatchRecordRequest{
		OrganizationID: tn.org, BusinessUnitID: tn.bu, RecordID: record, Until: early,
	}))
	_, err = client.ZScore(t.Context(), key, record.String()).Result()
	require.NoError(t, err, "a wait that closes early leaves a later one's watch")

	require.NoError(t, feed.Unwatch(t.Context(), &realtimeports.WatchRecordRequest{
		OrganizationID: tn.org, BusinessUnitID: tn.bu, RecordID: record, Until: late,
	}))
	n, err := client.ZCard(t.Context(), key).Result()
	require.NoError(t, err)
	assert.Zero(t, n)
}
