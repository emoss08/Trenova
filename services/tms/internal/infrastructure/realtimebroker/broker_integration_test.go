//go:build integration

package realtimebroker

import (
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/testutil"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx/fxtest"
	"go.uber.org/zap"
)

const waitFor = 3 * time.Second

func newTestBroker(t *testing.T, mutate func(*config.RealtimeConfig)) (*Broker, *redis.Client) {
	t.Helper()

	client := testutil.SetupTestRedis(t)
	cfg := &config.Config{Realtime: config.RealtimeConfig{
		ShardCount:        4,
		HeartbeatInterval: time.Hour,
	}}
	if mutate != nil {
		mutate(&cfg.Realtime)
	}

	lc := fxtest.NewLifecycle(t)
	pub := NewPublisher(PublisherParams{Client: client, Config: cfg, Logger: zap.NewNop(), LC: lc})
	broker := NewBroker(BrokerParams{
		Publisher: pub,
		Client:    client,
		Config:    cfg,
		Logger:    zap.NewNop(),
		LC:        lc,
	})
	lc.RequireStart()
	t.Cleanup(lc.RequireStop)

	return broker, client
}

type tenant struct {
	org pulid.ID
	bu  pulid.ID
}

func newTenant() tenant {
	return tenant{org: pulid.MustNew("org_"), bu: pulid.MustNew("bu_")}
}

func (tn tenant) conn(userID pulid.ID, portal bool) *services.RealtimeConnection {
	return &services.RealtimeConnection{
		ConnectionID:   pulid.MustNew("rtc_").String(),
		OrganizationID: tn.org,
		BusinessUnitID: tn.bu,
		UserID:         userID,
		Name:           "User " + userID.String()[len(userID.String())-4:],
		Portal:         portal,
	}
}

func subscribe(
	t *testing.T,
	b *Broker,
	conn *services.RealtimeConnection,
	joinUsers bool,
	lastEventID string,
) services.RealtimeStream {
	t.Helper()
	stream, err := b.Subscribe(t.Context(), &services.RealtimeSubscribeRequest{
		Connection:  conn,
		JoinUsers:   joinUsers,
		LastEventID: lastEventID,
	})
	require.NoError(t, err)
	t.Cleanup(stream.Close)
	return stream
}

func invalidation(tn tenant, resource string, audience pulid.ID) *services.RealtimeEnvelope {
	return &services.RealtimeEnvelope{
		OrganizationID: tn.org,
		BusinessUnitID: tn.bu,
		AudienceUserID: audience,
		Event:          services.RealtimeEventInvalidation,
		Payload:        []byte(`{"resource":"` + resource + `","entity":{"secret":true}}`),
		PortalPayload:  []byte(`{"resource":"` + resource + `"}`),
	}
}

func next(t *testing.T, stream services.RealtimeStream) services.RealtimeFrame {
	t.Helper()
	select {
	case frame, ok := <-stream.Frames():
		require.True(t, ok, "stream closed: %s", stream.Reason())
		return frame
	case <-time.After(waitFor):
		t.Fatal("timed out waiting for a frame")
		return services.RealtimeFrame{}
	}
}

func nextOf(t *testing.T, stream services.RealtimeStream, event string) services.RealtimeFrame {
	t.Helper()
	deadline := time.After(waitFor)
	for {
		select {
		case frame, ok := <-stream.Frames():
			require.True(t, ok, "stream closed: %s", stream.Reason())
			if frame.Event == event {
				return frame
			}
		case <-deadline:
			t.Fatalf("timed out waiting for a %s frame", event)
		}
	}
}

func silent(t *testing.T, stream services.RealtimeStream, event string) {
	t.Helper()
	deadline := time.After(300 * time.Millisecond)
	for {
		select {
		case frame, ok := <-stream.Frames():
			if !ok {
				return
			}
			require.NotEqualf(t, event, frame.Event, "unexpected %s: %s", event, frame.Data)
		case <-deadline:
			return
		}
	}
}

func TestBroker_DeliversOnlyWithinTheTenant(t *testing.T) {
	b, _ := newTestBroker(t, nil)
	a, other := newTenant(), newTenant()

	mine := subscribe(t, b, a.conn(pulid.MustNew("usr_"), false), false, "")
	theirs := subscribe(t, b, other.conn(pulid.MustNew("usr_"), false), false, "")

	require.True(t, b.Enqueue(invalidation(a, "shipments", pulid.Nil)))

	frame := next(t, mine)
	assert.Equal(t, services.RealtimeEventInvalidation, frame.Event)
	assert.Contains(t, string(frame.Data), `"secret":true`)
	_, ok := parseCursor(frame.ID)
	assert.True(t, ok, "frame carries a resumable cursor: %q", frame.ID)

	silent(t, theirs, services.RealtimeEventInvalidation)
}

func TestBroker_AudienceReachesOnlyThatUser(t *testing.T) {
	b, _ := newTestBroker(t, nil)
	tn := newTenant()
	target, bystander := pulid.MustNew("usr_"), pulid.MustNew("usr_")

	targetStream := subscribe(t, b, tn.conn(target, false), false, "")
	bystanderStream := subscribe(t, b, tn.conn(bystander, false), false, "")

	require.True(t, b.Enqueue(invalidation(tn, "notifications", target)))

	assert.Contains(t, string(next(t, targetStream).Data), "notifications")
	silent(t, bystanderStream, services.RealtimeEventInvalidation)
}

func TestBroker_PortalReadersGetTheRedactedPayload(t *testing.T) {
	b, _ := newTestBroker(t, nil)
	tn := newTenant()
	driver := pulid.MustNew("usr_")
	portal := subscribe(t, b, tn.conn(driver, true), false, "")

	require.True(t, b.Enqueue(invalidation(tn, "shipments", pulid.Nil)))
	frame := next(t, portal)
	assert.NotContains(t, string(frame.Data), "secret")

	require.True(t, b.Enqueue(invalidation(tn, "notifications", driver)))
	assert.Contains(t, string(next(t, portal).Data), `"secret":true`,
		"an event addressed to the reader arrives whole")

	internalOnly := invalidation(tn, "shipments", pulid.Nil)
	internalOnly.PortalPayload = nil
	require.True(t, b.Enqueue(internalOnly))
	silent(t, portal, services.RealtimeEventInvalidation)
}

func TestBroker_ResumesFromLastEventIDWithoutDuplicates(t *testing.T) {
	b, _ := newTestBroker(t, nil)
	tn := newTenant()
	user := pulid.MustNew("usr_")

	first := subscribe(t, b, tn.conn(user, false), false, "")
	require.True(t, b.Enqueue(invalidation(tn, "one", pulid.Nil)))
	seen := next(t, first)
	first.Close()

	require.True(t, b.Enqueue(invalidation(tn, "two", pulid.Nil)))
	require.True(t, b.Enqueue(invalidation(tn, "three", pulid.Nil)))
	time.Sleep(200 * time.Millisecond)

	resumed := subscribe(t, b, tn.conn(user, false), false, seen.ID)
	prelude := resumed.Prelude()
	require.GreaterOrEqual(t, len(prelude), 3)

	var ready services.RealtimeReadyEvent
	require.Equal(t, services.RealtimeEventReady, prelude[0].Event)
	require.NoError(t, sonic.Unmarshal(prelude[0].Data, &ready))
	assert.True(t, ready.Resumed)

	replayed := make([]string, 0, 2)
	for _, frame := range prelude[1:] {
		require.NotEqual(t, services.RealtimeEventReset, frame.Event)
		replayed = append(replayed, string(frame.Data))
	}
	require.Len(t, replayed, 2)
	assert.Contains(t, replayed[0], `"two"`)
	assert.Contains(t, replayed[1], `"three"`)

	require.True(t, b.Enqueue(invalidation(tn, "four", pulid.Nil)))
	assert.Contains(t, string(next(t, resumed).Data), `"four"`,
		"the first live frame follows the replay, with nothing repeated")
}

func TestBroker_UnknownCursorAsksForReset(t *testing.T) {
	b, _ := newTestBroker(t, nil)
	tn := newTenant()

	stream := subscribe(t, b, tn.conn(pulid.MustNew("usr_"), false), false, "not-a-cursor")
	prelude := stream.Prelude()
	require.Len(t, prelude, 2)
	assert.Equal(t, services.RealtimeEventReady, prelude[0].Event)
	assert.Equal(t, services.RealtimeEventReset, prelude[1].Event)
}

func TestBroker_TrimmedCursorAsksForReset(t *testing.T) {
	b, client := newTestBroker(t, nil)
	tn := newTenant()
	shard := shardFor(tenantKey(tn.org, tn.bu), 4)

	require.True(t, b.Enqueue(invalidation(tn, "kept", pulid.Nil)))
	require.Eventually(t, func() bool {
		n, _ := client.XLen(t.Context(), streamKey(shard)).Result()
		return n > 0
	}, waitFor, 20*time.Millisecond)

	stale := formatCursor(shard, "1-0")
	stream := subscribe(t, b, tn.conn(pulid.MustNew("usr_"), false), false, stale)
	prelude := stream.Prelude()
	require.Len(t, prelude, 2)
	assert.Equal(t, services.RealtimeEventReset, prelude[1].Event)
}

func TestBroker_UserPresenceSnapshotAndTransitions(t *testing.T) {
	b, _ := newTestBroker(t, nil)
	tn := newTenant()

	aliceConn := tn.conn(pulid.MustNew("usr_"), false)
	alice := subscribe(t, b, aliceConn, true, "")

	prelude := alice.Prelude()
	var snapshot services.RealtimePresenceSnapshot
	found := false
	for _, frame := range prelude {
		if frame.Event == services.RealtimeEventPresenceState {
			require.NoError(t, sonic.Unmarshal(frame.Data, &snapshot))
			found = true
		}
	}
	require.True(t, found)
	assert.Equal(t, services.RealtimeScopeUsers, snapshot.Scope)
	require.Len(t, snapshot.Members, 1)
	assert.Equal(t, aliceConn.ConnectionID, snapshot.Members[0].ConnectionID)

	bobConn := tn.conn(pulid.MustNew("usr_"), false)
	bob := subscribe(t, b, bobConn, true, "")

	var entered services.RealtimePresenceEvent
	for {
		require.NoError(t, sonic.Unmarshal(nextOf(t, alice, services.RealtimeEventPresence).Data, &entered))
		if entered.ConnectionID == bobConn.ConnectionID {
			break
		}
	}
	assert.Equal(t, services.RealtimePresenceEnter, entered.Action)
	assert.Equal(t, bobConn.UserID.String(), entered.UserID)

	bob.Close()
	var left services.RealtimePresenceEvent
	for {
		require.NoError(t, sonic.Unmarshal(nextOf(t, alice, services.RealtimeEventPresence).Data, &left))
		if left.ConnectionID == bobConn.ConnectionID && left.Action == services.RealtimePresenceLeave {
			break
		}
	}
}

func TestBroker_ScopedEventsReachOnlyJoinedConnections(t *testing.T) {
	b, _ := newTestBroker(t, nil)
	tn := newTenant()
	scope := "shipment-comments:" + pulid.MustNew("shp_").String()

	viewerConn := tn.conn(pulid.MustNew("usr_"), false)
	viewer := subscribe(t, b, viewerConn, false, "")
	outsider := subscribe(t, b, tn.conn(pulid.MustNew("usr_"), false), false, "")

	snapshot, err := b.JoinPresence(t.Context(), viewerConn, scope)
	require.NoError(t, err)
	require.Len(t, snapshot.Members, 1)
	nextOf(t, viewer, services.RealtimeEventPresence)

	typistConn := tn.conn(pulid.MustNew("usr_"), false)
	_ = subscribe(t, b, typistConn, false, "")
	_, err = b.JoinPresence(t.Context(), typistConn, scope)
	require.NoError(t, err)

	b.Enqueue(&services.RealtimeEnvelope{
		OrganizationID: tn.org,
		BusinessUnitID: tn.bu,
		Event:          services.RealtimeEventTyping,
		Scope:          scope,
		Payload:        []byte(`{"scope":"` + scope + `"}`),
		Ephemeral:      true,
	})

	nextOf(t, viewer, services.RealtimeEventTyping)
	silent(t, outsider, services.RealtimeEventTyping)
	silent(t, outsider, services.RealtimeEventPresence)

	require.NoError(t, b.LeavePresence(t.Context(), viewerConn, scope))
	require.Eventually(t, func() bool {
		return !slices.Contains(viewer.(*subscriber).joinedScopes(), scope)
	}, waitFor, 20*time.Millisecond)
}

func TestBroker_SweepAnnouncesDeadMembers(t *testing.T) {
	b, client := newTestBroker(t, nil)
	tn := newTenant()
	watcherConn := tn.conn(pulid.MustNew("usr_"), false)
	watcher := subscribe(t, b, watcherConn, true, "")

	ghost := pulid.MustNew("rtc_").String()
	tenant := tenantKey(tn.org, tn.bu)
	ctx := t.Context()
	require.NoError(t, client.ZAdd(ctx, presenceExpiryKey(tenant, services.RealtimeScopeUsers), redis.Z{
		Score:  float64(time.Now().Add(-time.Minute).UnixMilli()),
		Member: ghost,
	}).Err())
	require.NoError(t, client.HSet(ctx, presenceDataKey(tenant, services.RealtimeScopeUsers),
		ghost, `{"u":"usr_ghost","n":"Ghost"}`).Err())

	b.sweep()

	var left services.RealtimePresenceEvent
	for {
		require.NoError(t, sonic.Unmarshal(nextOf(t, watcher, services.RealtimeEventPresence).Data, &left))
		if left.ConnectionID == ghost {
			break
		}
	}
	assert.Equal(t, services.RealtimePresenceLeave, left.Action)
	assert.Equal(t, "usr_ghost", left.UserID)

	members, err := b.members(ctx, tenant, services.RealtimeScopeUsers)
	require.NoError(t, err)
	for _, member := range members.Members {
		assert.NotEqual(t, ghost, member.ConnectionID)
	}
}

func TestBroker_SlowReaderIsClosedWithOverflow(t *testing.T) {
	b, _ := newTestBroker(t, func(c *config.RealtimeConfig) { c.SubscriberBuffer = 8 })
	tn := newTenant()
	stream := subscribe(t, b, tn.conn(pulid.MustNew("usr_"), false), false, "")

	for i := range 32 {
		b.Enqueue(invalidation(tn, "burst-"+strconv.Itoa(i), pulid.Nil))
	}

	require.Eventually(t, func() bool {
		return stream.Reason() == services.RealtimeCloseOverflow
	}, waitFor, 20*time.Millisecond)
}

func TestBroker_PerUserConnectionLimit(t *testing.T) {
	b, _ := newTestBroker(t, func(c *config.RealtimeConfig) { c.MaxConnectionsPerUser = 1 })
	tn := newTenant()
	user := pulid.MustNew("usr_")

	first := subscribe(t, b, tn.conn(user, false), false, "")
	_, err := b.Subscribe(t.Context(), &services.RealtimeSubscribeRequest{
		Connection: tn.conn(user, false),
	})
	require.Error(t, err)

	first.Close()
	_ = subscribe(t, b, tn.conn(user, false), false, "")
}

func TestBroker_DrainClosesStreamsWithShutdown(t *testing.T) {
	b, _ := newTestBroker(t, nil)
	tn := newTenant()
	stream := subscribe(t, b, tn.conn(pulid.MustNew("usr_"), false), false, "")

	b.Drain()

	select {
	case _, ok := <-stream.Frames():
		assert.False(t, ok)
	case <-time.After(waitFor):
		t.Fatal("stream was not closed")
	}
	assert.Equal(t, services.RealtimeCloseShutdown, stream.Reason())

	_, err := b.Subscribe(t.Context(), &services.RealtimeSubscribeRequest{
		Connection: tn.conn(pulid.MustNew("usr_"), false),
	})
	require.Error(t, err)
}

func TestBroker_FreshStreamsCarryAResumableCursor(t *testing.T) {
	b, _ := newTestBroker(t, nil)
	tn := newTenant()
	user := pulid.MustNew("usr_")

	require.True(t, b.Enqueue(invalidation(tn, "before", pulid.Nil)))
	time.Sleep(200 * time.Millisecond)

	quiet := subscribe(t, b, tn.conn(user, false), false, "")
	var ready services.RealtimeReadyEvent
	require.NoError(t, sonic.Unmarshal(quiet.Prelude()[0].Data, &ready))
	assert.False(t, ready.Resumed)
	_, ok := parseCursor(ready.Cursor)
	require.True(t, ok, "ready carries a cursor: %q", ready.Cursor)
	quiet.Close()

	require.True(t, b.Enqueue(invalidation(tn, "missed", pulid.Nil)))
	time.Sleep(200 * time.Millisecond)

	resumed := subscribe(t, b, tn.conn(user, false), false, ready.Cursor)
	prelude := resumed.Prelude()
	require.Len(t, prelude, 2, "ready, then exactly the one event the quiet tab missed")
	assert.Contains(t, string(prelude[1].Data), `"missed"`)
}
