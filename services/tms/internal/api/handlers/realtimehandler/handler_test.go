package realtimehandler

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	servicesport "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type stubStream struct {
	id      string
	prelude []servicesport.RealtimeFrame
	frames  chan servicesport.RealtimeFrame
	reason  string
	once    sync.Once
	closed  bool
}

func (s *stubStream) ConnectionID() string                      { return s.id }
func (s *stubStream) Prelude() []servicesport.RealtimeFrame     { return s.prelude }
func (s *stubStream) Frames() <-chan servicesport.RealtimeFrame { return s.frames }
func (s *stubStream) Reason() string                            { return s.reason }
func (s *stubStream) Close()                                    { s.once.Do(func() { s.closed = true }) }

type stubGateway struct {
	mu       sync.Mutex
	stream   *stubStream
	openErr  error
	opened   *servicesport.OpenRealtimeStreamRequest
	joined   *servicesport.RealtimeScopeRequest
	left     *servicesport.RealtimeScopeRequest
	typing   *servicesport.RealtimeTypingRequest
	snapshot *servicesport.RealtimePresenceSnapshot
}

func (g *stubGateway) Open(
	_ context.Context,
	req *servicesport.OpenRealtimeStreamRequest,
) (servicesport.RealtimeStream, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.opened = req
	if g.openErr != nil {
		return nil, g.openErr
	}
	return g.stream, nil
}

func (g *stubGateway) JoinPresence(
	_ context.Context,
	req *servicesport.RealtimeScopeRequest,
) (*servicesport.RealtimePresenceSnapshot, error) {
	g.joined = req
	return g.snapshot, nil
}

func (g *stubGateway) LeavePresence(_ context.Context, req *servicesport.RealtimeScopeRequest) error {
	g.left = req
	return nil
}

func (g *stubGateway) SignalTyping(_ context.Context, req *servicesport.RealtimeTypingRequest) error {
	g.typing = req
	return nil
}

func (g *stubGateway) Drain() {}

type authMode int

const (
	authUser authMode = iota
	authPortal
	authAPIKey
)

var (
	testUserID = pulid.MustNew("usr_")
	testBUID   = pulid.MustNew("bu_")
	testOrgID  = pulid.MustNew("org_")
)

func newTestRouter(t *testing.T, gateway *stubGateway, mode authMode) *gin.Engine {
	t.Helper()

	gin.SetMode(gin.TestMode)
	cfg := &config.Config{App: config.AppConfig{Debug: true}}
	eh := helpers.NewErrorHandler(helpers.ErrorHandlerParams{Logger: zap.NewNop(), Config: cfg})

	engine := mocks.NewMockPermissionEngine(t)
	engine.On("Check", mock.Anything, mock.Anything).
		Return(&servicesport.PermissionCheckResult{Allowed: true}, nil).Maybe()

	h := New(Params{
		Gateway:      gateway,
		ErrorHandler: eh,
		PermissionMiddleware: middleware.NewPermissionMiddleware(
			middleware.PermissionMiddlewareParams{PermissionEngine: engine, ErrorHandler: eh},
		),
		Config: cfg,
		Logger: zap.NewNop(),
	})

	r := gin.New()
	api := r.Group("/api/v1")
	api.Use(func(c *gin.Context) {
		switch mode {
		case authAPIKey:
			authctx.SetAPIKeyContext(c, pulid.MustNew("key_"), testBUID, testOrgID)
		case authPortal:
			authctx.SetSessionAuthContext(c, authctx.SessionAuthContextParams{
				UserID:         testUserID,
				BusinessUnitID: testBUID,
				OrganizationID: testOrgID,
				IsPortalUser:   true,
			})
		case authUser:
			authctx.SetAuthContext(c, testUserID, testBUID, testOrgID)
		}
		c.Next()
	})
	h.RegisterRoutes(api)

	return r
}

func TestStream_WritesPreludeLiveFramesAndCloseReason(t *testing.T) {
	t.Parallel()

	frames := make(chan servicesport.RealtimeFrame, 2)
	frames <- servicesport.RealtimeFrame{
		ID:    "3.100-0",
		Event: servicesport.RealtimeEventInvalidation,
		Data:  []byte(`{"resource":"shipments"}`),
	}
	close(frames)

	stream := &stubStream{
		id: "rtc_1",
		prelude: []servicesport.RealtimeFrame{
			{Event: servicesport.RealtimeEventReady, Data: []byte(`{"connectionId":"rtc_1"}`)},
		},
		frames: frames,
		reason: servicesport.RealtimeCloseShutdown,
	}
	gateway := &stubGateway{stream: stream}
	router := newTestRouter(t, gateway, authUser)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/realtime/stream/?presence=users", nil)
	req.Header.Set(lastEventIDHeader, "3.99-0")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))

	body := w.Body.String()
	ready := strings.Index(body, "event: ready\n")
	live := strings.Index(body, "id: 3.100-0\nevent: resource.invalidation\n")
	closing := strings.Index(body, "event: close\ndata: {\"reason\":\"shutdown\"}")
	require.GreaterOrEqual(t, ready, 0, body)
	require.Greater(t, live, ready, body)
	require.Greater(t, closing, live, body)

	require.NotNil(t, gateway.opened)
	assert.Equal(t, testUserID, gateway.opened.UserID)
	assert.Equal(t, testOrgID, gateway.opened.OrganizationID)
	assert.Equal(t, testBUID, gateway.opened.BusinessUnitID)
	assert.True(t, gateway.opened.JoinUsers)
	assert.False(t, gateway.opened.Portal)
	assert.Equal(t, "3.99-0", gateway.opened.LastEventID)
	assert.True(t, stream.closed)
}

func TestStream_PortalUsersAreMarked(t *testing.T) {
	t.Parallel()

	frames := make(chan servicesport.RealtimeFrame)
	close(frames)
	gateway := &stubGateway{stream: &stubStream{id: "rtc_2", frames: frames}}
	router := newTestRouter(t, gateway, authPortal)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/realtime/stream/", nil))

	require.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, gateway.opened)
	assert.True(t, gateway.opened.Portal)
	assert.False(t, gateway.opened.JoinUsers)
}

func TestStream_RejectsAPIKeys(t *testing.T) {
	t.Parallel()

	gateway := &stubGateway{}
	router := newTestRouter(t, gateway, authAPIKey)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/realtime/stream/", nil))

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Nil(t, gateway.opened)
}

func TestStream_CapacityAnswersWithRetryAfter(t *testing.T) {
	t.Parallel()

	gateway := &stubGateway{openErr: errortypes.NewRateLimitError("realtime", "full")}
	router := newTestRouter(t, gateway, authUser)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/realtime/stream/", nil))

	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.Equal(t, "5", w.Header().Get("Retry-After"))
}

func TestJoinShipmentComments_ScopesToTheShipment(t *testing.T) {
	t.Parallel()

	shipmentID := pulid.MustNew("shp_")
	gateway := &stubGateway{snapshot: &servicesport.RealtimePresenceSnapshot{
		Scope: "shipment-comments:" + shipmentID.String(),
		Members: []servicesport.RealtimePresenceMember{
			{UserID: "usr_other", ConnectionID: "rtc_other", Name: "Other"},
		},
	}}
	router := newTestRouter(t, gateway, authUser)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/shipments/"+shipmentID.String()+"/comments/presence/",
		bytes.NewBufferString(`{"connectionId":"rtc_01J00000000000000000000000"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, gateway.joined)
	assert.Equal(t, "shipment-comments:"+shipmentID.String(), gateway.joined.Scope)
	assert.Equal(t, "rtc_01J00000000000000000000000", gateway.joined.ConnectionID)
	assert.Equal(t, testUserID, gateway.joined.UserID)

	var snapshot servicesport.RealtimePresenceSnapshot
	require.NoError(t, sonic.Unmarshal(w.Body.Bytes(), &snapshot))
	require.Len(t, snapshot.Members, 1)
	assert.Equal(t, "Other", snapshot.Members[0].Name)
}

func TestJoinShipmentComments_RejectsMalformedShipmentID(t *testing.T) {
	t.Parallel()

	gateway := &stubGateway{}
	router := newTestRouter(t, gateway, authUser)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/shipments/shp_01J0000000000000000000000|x/comments/presence/",
		bytes.NewBufferString(`{"connectionId":"rtc_01J00000000000000000000000"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Nil(t, gateway.joined)
}

func TestLeaveShipmentComments(t *testing.T) {
	t.Parallel()

	shipmentID := pulid.MustNew("shp_")
	gateway := &stubGateway{}
	router := newTestRouter(t, gateway, authUser)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(
		http.MethodDelete,
		"/api/v1/shipments/"+shipmentID.String()+
			"/comments/presence/?connectionId=rtc_01J00000000000000000000000",
		nil,
	))

	assert.Equal(t, http.StatusNoContent, w.Code)
	require.NotNil(t, gateway.left)
	assert.Equal(t, "rtc_01J00000000000000000000000", gateway.left.ConnectionID)
}

func TestTypingShipmentComments_PassesStop(t *testing.T) {
	t.Parallel()

	shipmentID := pulid.MustNew("shp_")
	gateway := &stubGateway{}
	router := newTestRouter(t, gateway, authUser)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/shipments/"+shipmentID.String()+"/comments/typing/",
		bytes.NewBufferString(`{"connectionId":"rtc_01J00000000000000000000000","stop":true}`),
	)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	require.NotNil(t, gateway.typing)
	assert.True(t, gateway.typing.Stop)
	assert.Equal(t, "shipment-comments:"+shipmentID.String(), gateway.typing.Scope)
}
