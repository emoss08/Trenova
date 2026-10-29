//go:build integration

package realtimebroker

import (
	"testing"
	"time"

	"github.com/bytedance/sonic"
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

func recordChanged(
	t *testing.T,
	tn tenant,
	recordID string,
	audience pulid.ID,
) *realtimeports.RealtimeEnvelope {
	t.Helper()
	payload, err := sonic.Marshal(realtimeports.ResourceInvalidationEvent{
		Resource:    "audited:shipment",
		Action:      "updated",
		RecordID:    recordID,
		Fields:      []string{"status"},
		ActorType:   "session_user",
		ActorUserID: "usr_someone",
		OccurredAt:  time.Now(),
	})
	require.NoError(t, err)

	return &realtimeports.RealtimeEnvelope{
		OrganizationID: tn.org,
		BusinessUnitID: tn.bu,
		AudienceUserID: audience,
		Event:          realtimeports.RealtimeEventInvalidation,
		Payload:        payload,
	}
}

func TestChangeFeed_ReadsWatchedChangesSinceTheCursor(t *testing.T) {
	client := testutil.SetupTestRedis(t)
	cfg := &config.Config{Realtime: realtimeconfig.Config{ShardCount: 4}}
	lc := fxtest.NewLifecycle(t)
	pub := NewPublisher(
		PublisherParams{
			Client:         client,
			RealtimeConfig: &cfg.Realtime,
			Logger:         zap.NewNop(),
			LC:             lc,
		},
	)
	lc.RequireStart()
	t.Cleanup(lc.RequireStop)
	feed := NewChangeFeed(ChangeFeedParams{Client: client, RealtimeConfig: &cfg.Realtime})

	tn, stranger := newTenant(), newTenant()
	before := recordChanged(t, tn, feedRecord, pulid.Nil)
	require.True(t, pub.Enqueue(before))
	require.Eventually(t, func() bool {
		head, err := feed.Head(t.Context(), tn.org, tn.bu)
		return err == nil && head != formatCursor(shardFor(tenantKey(tn.org, tn.bu), 4), "0-0")
	}, waitFor, 20*time.Millisecond)

	cursor, err := feed.Head(t.Context(), tn.org, tn.bu)
	require.NoError(t, err)

	require.True(t, pub.Enqueue(recordChanged(t, tn, "shp_01J9UNWATCHEDUNWATCHED000", pulid.Nil)))
	require.True(t, pub.Enqueue(recordChanged(t, stranger, feedRecord, pulid.Nil)))
	require.True(t, pub.Enqueue(recordChanged(t, tn, feedRecord, pulid.MustNew("usr_"))))
	require.True(t, pub.Enqueue(recordChanged(t, tn, feedRecord, pulid.Nil)))

	var found *realtimeports.RecordChanges
	require.Eventually(t, func() bool {
		found, err = feed.Since(t.Context(), &realtimeports.RecordChangesRequest{
			OrganizationID: tn.org,
			BusinessUnitID: tn.bu,
			Cursor:         cursor,
			Records:        []string{feedRecord},
		})
		return err == nil && len(found.Changes) == 1
	}, waitFor, 20*time.Millisecond)

	assert.Equal(t, feedRecord, found.Changes[0].RecordID)
	assert.Equal(t, []string{"status"}, found.Changes[0].Fields)
	assert.NotEqual(t, cursor, found.Cursor)

	again, err := feed.Since(t.Context(), &realtimeports.RecordChangesRequest{
		OrganizationID: tn.org,
		BusinessUnitID: tn.bu,
		Cursor:         found.Cursor,
		Records:        []string{feedRecord},
	})
	require.NoError(t, err)
	assert.Empty(t, again.Changes, "a change is read once")
}

func TestChangeFeed_ACursorFromAnotherLayoutStartsAtTheHead(t *testing.T) {
	client := testutil.SetupTestRedis(t)
	cfg := &config.Config{Realtime: realtimeconfig.Config{ShardCount: 4}}
	feed := NewChangeFeed(ChangeFeedParams{Client: client, RealtimeConfig: &cfg.Realtime})
	tn := newTenant()

	found, err := feed.Since(t.Context(), &realtimeports.RecordChangesRequest{
		OrganizationID: tn.org,
		BusinessUnitID: tn.bu,
		Cursor:         "not-a-cursor",
		Records:        []string{feedRecord},
	})
	require.NoError(t, err)
	assert.Empty(t, found.Changes)
	head, err := feed.Head(t.Context(), tn.org, tn.bu)
	require.NoError(t, err)
	assert.Equal(t, head, found.Cursor)
}
