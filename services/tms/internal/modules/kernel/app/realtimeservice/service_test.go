package realtimeservice

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/infrastructure/config/realtimeconfig"
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type recordingPublisher struct {
	full      bool
	enqueued  []*realtimeports.RealtimeEnvelope
	published []*realtimeports.RealtimeEnvelope
}

func (p *recordingPublisher) Enqueue(env *realtimeports.RealtimeEnvelope) bool {
	if p.full {
		return false
	}
	p.enqueued = append(p.enqueued, env)
	return true
}

func (p *recordingPublisher) Publish(_ context.Context, env *realtimeports.RealtimeEnvelope) error {
	p.published = append(p.published, env)
	return nil
}

func newTestService(pub realtimeports.RealtimePublisher, maxEntityBytes int) *Service {
	return New(Params{
		Logger:         zap.NewNop(),
		RealtimeConfig: &realtimeconfig.Config{MaxEntityBytes: maxEntityBytes},
		Publisher:      pub,
	}).(*Service)
}

func validRequest() *realtimeports.PublishResourceInvalidationRequest {
	return &realtimeports.PublishResourceInvalidationRequest{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		Resource:       "shipments",
		Action:         "updated",
		RecordID:       pulid.MustNew("shp_"),
		ActorUserID:    pulid.MustNew("usr_"),
		Entity:         map[string]any{"proNumber": "PRO-1", "secret": "rate"},
		Fields:         []string{"status"},
	}
}

func decode(t *testing.T, payload []byte) realtimeports.ResourceInvalidationEvent {
	t.Helper()
	var event realtimeports.ResourceInvalidationEvent
	require.NoError(t, sonic.Unmarshal(payload, &event))
	return event
}

func TestPublishResourceInvalidation_EnqueuesFullAndRedactedPayloads(t *testing.T) {
	t.Parallel()

	pub := &recordingPublisher{}
	svc := newTestService(pub, 0)
	req := validRequest()

	require.NoError(t, svc.PublishResourceInvalidation(t.Context(), req))
	require.Len(t, pub.enqueued, 1)
	env := pub.enqueued[0]

	assert.Equal(t, realtimeports.RealtimeEventInvalidation, env.Event)
	assert.Equal(t, req.OrganizationID, env.OrganizationID)
	assert.False(t, env.Ephemeral)

	full := decode(t, env.Payload)
	assert.Equal(t, "shipments.updated", full.Type)
	assert.Equal(t, req.RecordID.String(), full.RecordID)
	assert.Equal(t, req.ActorUserID.String(), full.ActorUserID)
	assert.NotNil(t, full.Entity)
	assert.True(t, strings.HasPrefix(full.EventID, "evt_"))

	redacted := decode(t, env.PortalPayload)
	assert.Nil(t, redacted.Entity)
	assert.Empty(t, redacted.Fields)
	assert.Equal(t, full.RecordID, redacted.RecordID)
	assert.NotContains(t, string(env.PortalPayload), "rate")
}

func TestPublishResourceInvalidation_AddressedEventsHaveNoPortalForm(t *testing.T) {
	t.Parallel()

	pub := &recordingPublisher{}
	svc := newTestService(pub, 0)
	req := validRequest()
	req.AudienceUserID = pulid.MustNew("usr_")

	require.NoError(t, svc.PublishResourceInvalidation(t.Context(), req))
	require.Len(t, pub.enqueued, 1)
	assert.Equal(t, req.AudienceUserID, pub.enqueued[0].AudienceUserID)
	assert.Nil(t, pub.enqueued[0].PortalPayload)
}

func TestPublishResourceInvalidation_DropsOversizedEntity(t *testing.T) {
	t.Parallel()

	pub := &recordingPublisher{}
	svc := newTestService(pub, 256)
	req := validRequest()
	req.Entity = map[string]any{"notes": strings.Repeat("x", 1024)}

	require.NoError(t, svc.PublishResourceInvalidation(t.Context(), req))
	event := decode(t, pub.enqueued[0].Payload)
	assert.Nil(t, event.Entity)
	assert.Equal(t, req.RecordID.String(), event.RecordID)
}

func TestPublishResourceInvalidation_ReportsBackpressure(t *testing.T) {
	t.Parallel()

	svc := newTestService(&recordingPublisher{full: true}, 0)
	err := svc.PublishResourceInvalidation(t.Context(), validRequest())
	require.ErrorIs(t, err, ErrPublishBackpressure)
}

func TestPublishResourceInvalidation_Validation(t *testing.T) {
	t.Parallel()

	svc := newTestService(&recordingPublisher{}, 0)

	require.Error(t, svc.PublishResourceInvalidation(t.Context(), nil))

	missingTenant := validRequest()
	missingTenant.BusinessUnitID = pulid.Nil
	require.Error(t, svc.PublishResourceInvalidation(t.Context(), missingTenant))

	missingAction := validRequest()
	missingAction.Action = ""
	require.Error(t, svc.PublishResourceInvalidation(t.Context(), missingAction))
}

type stubBroker struct {
	recordingPublisher
	conn        *realtimeports.RealtimeConnection
	connErr     error
	allow       bool
	released    []string
	throttled   []string
	joined      string
	joinedState realtimeports.RealtimePresenceState
	left        string
	subscribe   *realtimeports.RealtimeSubscribeRequest
}

func (b *stubBroker) Subscribe(
	_ context.Context,
	req *realtimeports.RealtimeSubscribeRequest,
) (realtimeports.RealtimeStream, error) {
	b.subscribe = req
	return nil, nil
}

func (b *stubBroker) Connection(
	context.Context,
	string,
) (*realtimeports.RealtimeConnection, error) {
	return b.conn, b.connErr
}

func (b *stubBroker) JoinPresence(
	_ context.Context,
	_ *realtimeports.RealtimeConnection,
	scope string,
	state realtimeports.RealtimePresenceState,
) (*realtimeports.RealtimePresenceSnapshot, error) {
	b.joined = scope
	b.joinedState = state
	return &realtimeports.RealtimePresenceSnapshot{Scope: scope}, nil
}

func (b *stubBroker) LeavePresence(
	_ context.Context,
	_ *realtimeports.RealtimeConnection,
	scope string,
) error {
	b.left = scope
	return nil
}

func (b *stubBroker) Throttle(_ context.Context, key string, _ time.Duration) (bool, error) {
	b.throttled = append(b.throttled, key)
	return b.allow, nil
}

func (b *stubBroker) Release(_ context.Context, key string) error {
	b.released = append(b.released, key)
	return nil
}

func (b *stubBroker) Drain() {}

type gatewayFixture struct {
	gateway *Gateway
	broker  *stubBroker
	conn    *realtimeports.RealtimeConnection
}

func newGatewayFixture(t *testing.T) *gatewayFixture {
	t.Helper()

	conn := &realtimeports.RealtimeConnection{
		ConnectionID:   pulid.MustNew("rtc_").String(),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		UserID:         pulid.MustNew("usr_"),
		Name:           "Dana",
	}
	broker := &stubBroker{conn: conn, allow: true}
	users := tenantportsmock.NewMockUserRepository(t)

	gateway := NewGateway(GatewayParams{
		Logger:         zap.NewNop(),
		RealtimeConfig: &realtimeconfig.Config{},
		Broker:         broker,
		UserRepo:       users,
	}).(*Gateway)

	return &gatewayFixture{gateway: gateway, broker: broker, conn: conn}
}

func (f *gatewayFixture) scope(scope string) realtimeports.RealtimeScopeRequest {
	return realtimeports.RealtimeScopeRequest{
		OrganizationID: f.conn.OrganizationID,
		BusinessUnitID: f.conn.BusinessUnitID,
		UserID:         f.conn.UserID,
		ConnectionID:   f.conn.ConnectionID,
		Scope:          scope,
	}
}

func TestGateway_OpenResolvesTheReadersName(t *testing.T) {
	t.Parallel()

	broker := &stubBroker{}
	users := tenantportsmock.NewMockUserRepository(t)
	userID := pulid.MustNew("usr_")
	users.On("GetByID", mock.Anything, mock.MatchedBy(func(req tenantports.GetUserByIDRequest) bool {
		return req.LookupUserID == userID
	})).
		Return(&tenant.User{ID: userID, Name: "Dana"}, nil)

	gateway := NewGateway(GatewayParams{
		Logger: zap.NewNop(), RealtimeConfig: &realtimeconfig.Config{}, Broker: broker, UserRepo: users,
	})

	_, err := gateway.Open(t.Context(), &realtimeports.OpenRealtimeStreamRequest{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		UserID:         userID,
		Portal:         true,
		JoinUsers:      true,
		LastEventID:    "1.1-0",
	})
	require.NoError(t, err)
	require.NotNil(t, broker.subscribe)
	assert.Equal(t, "Dana", broker.subscribe.Connection.Name)
	assert.True(t, pulid.LooksLike(broker.subscribe.Connection.ConnectionID))
	assert.False(t, broker.subscribe.JoinUsers, "portal users never join tenant presence")
	assert.Equal(t, "1.1-0", broker.subscribe.LastEventID)
}

func TestGateway_RejectsAnotherUsersConnection(t *testing.T) {
	t.Parallel()

	f := newGatewayFixture(t)
	req := f.scope("shipment-comments:shp_01J00000000000000000000000")
	req.UserID = pulid.MustNew("usr_")

	_, err := f.gateway.JoinPresence(t.Context(), &req)
	require.Error(t, err)
	assert.True(t, errortypes.IsAuthorizationError(err))
	assert.Empty(t, f.broker.joined)
}

func TestGateway_RejectsPortalConnections(t *testing.T) {
	t.Parallel()

	f := newGatewayFixture(t)
	f.conn.Portal = true
	req := f.scope("shipment-comments:shp_01J00000000000000000000000")

	_, err := f.gateway.JoinPresence(t.Context(), &req)
	require.Error(t, err)
	assert.Empty(t, f.broker.joined)
}

func TestGateway_ValidatesScope(t *testing.T) {
	t.Parallel()

	for name, scope := range map[string]string{
		"empty":    "",
		"reserved": realtimeports.RealtimeScopeUsers,
		"injected": "shipment-comments:x|y",
		"spaces":   "shipment comments",
		"too long": strings.Repeat("a", maxScopeLength+1),
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			f := newGatewayFixture(t)
			req := f.scope(scope)
			_, err := f.gateway.JoinPresence(t.Context(), &req)
			require.Error(t, err)
			assert.Empty(t, f.broker.joined)
		})
	}
}

func TestGateway_LeaveOfAnEndedConnectionIsNotAnError(t *testing.T) {
	t.Parallel()

	f := newGatewayFixture(t)
	f.broker.connErr = errortypes.NewNotFoundError("gone")
	req := f.scope("shipment-comments:shp_01J00000000000000000000000")

	require.NoError(t, f.gateway.LeavePresence(t.Context(), &req))
	assert.Empty(t, f.broker.left)
}

func TestGateway_TypingIsThrottledAndStampedWithTheCaller(t *testing.T) {
	t.Parallel()

	f := newGatewayFixture(t)
	scope := "shipment-comments:shp_01J00000000000000000000000"

	require.NoError(t, f.gateway.SignalTyping(t.Context(), &realtimeports.RealtimeTypingRequest{
		RealtimeScopeRequest: f.scope(scope),
	}))
	require.Len(t, f.broker.enqueued, 1)
	env := f.broker.enqueued[0]
	assert.Equal(t, realtimeports.RealtimeEventTyping, env.Event)
	assert.Equal(t, scope, env.Scope)
	assert.True(t, env.Ephemeral)
	assert.Nil(t, env.PortalPayload)

	var event realtimeports.RealtimeTypingEvent
	require.NoError(t, sonic.Unmarshal(env.Payload, &event))
	assert.Equal(t, f.conn.UserID.String(), event.UserID)
	assert.Equal(t, "Dana", event.Name)
	assert.False(t, event.Stop)

	f.broker.allow = false
	require.NoError(t, f.gateway.SignalTyping(t.Context(), &realtimeports.RealtimeTypingRequest{
		RealtimeScopeRequest: f.scope(scope),
	}))
	assert.Len(t, f.broker.enqueued, 1, "a throttled signal is not sent")

	require.NoError(t, f.gateway.SignalTyping(t.Context(), &realtimeports.RealtimeTypingRequest{
		RealtimeScopeRequest: f.scope(scope),
		Stop:                 true,
	}))
	require.Len(t, f.broker.enqueued, 2, "a stop is never throttled")
	assert.Len(t, f.broker.released, 1)
}

func TestGateway_JoinCarriesPresenceState(t *testing.T) {
	t.Parallel()

	f := newGatewayFixture(t)
	req := f.scope("inbox:messages")
	req.State = realtimeports.RealtimePresenceState{
		MessageID: "imsg_01J00000000000000000000000",
		Activity:  realtimeports.RealtimeActivityWriting,
	}

	_, err := f.gateway.JoinPresence(t.Context(), &req)
	require.NoError(t, err)
	assert.Equal(t, "inbox:messages", f.broker.joined)
	assert.Equal(t, req.State, f.broker.joinedState)
}

func TestGateway_ValidatesPresenceState(t *testing.T) {
	t.Parallel()

	for name, state := range map[string]realtimeports.RealtimePresenceState{
		"bad message id":         {MessageID: "not an id", Activity: realtimeports.RealtimeActivityViewing},
		"unknown activity":       {MessageID: "imsg_01J00000000000000000000000", Activity: "typing"},
		"activity without a msg": {Activity: realtimeports.RealtimeActivityViewing},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			f := newGatewayFixture(t)
			req := f.scope("inbox:messages")
			req.State = state
			_, err := f.gateway.JoinPresence(t.Context(), &req)
			require.Error(t, err)
			assert.Empty(t, f.broker.joined)
		})
	}
}
