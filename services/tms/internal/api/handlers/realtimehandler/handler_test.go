package realtimehandler

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/api/sessionwatch"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/sessionports"
	"github.com/emoss08/trenova/internal/testutil/mocks/permissionportsmock"
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
	prelude []realtimeports.RealtimeFrame
	frames  chan realtimeports.RealtimeFrame
	reason  string
	once    sync.Once
	closed  bool
}

func (s *stubStream) ConnectionID() string                       { return s.id }
func (s *stubStream) Prelude() []realtimeports.RealtimeFrame     { return s.prelude }
func (s *stubStream) Frames() <-chan realtimeports.RealtimeFrame { return s.frames }
func (s *stubStream) Reason() string                             { return s.reason }

func (s *stubStream) Close() { s.once.Do(func() { s.closed = true }) }

type stubGateway struct {
	mu       sync.Mutex
	stream   *stubStream
	openErr  error
	opened   *realtimeports.OpenRealtimeStreamRequest
	joined   *realtimeports.RealtimeScopeRequest
	left     *realtimeports.RealtimeScopeRequest
	typing   *realtimeports.RealtimeTypingRequest
	snapshot *realtimeports.RealtimePresenceSnapshot
}

func (g *stubGateway) Open(
	_ context.Context,
	req *realtimeports.OpenRealtimeStreamRequest,
) (realtimeports.RealtimeStream, error) {
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
	req *realtimeports.RealtimeScopeRequest,
) (*realtimeports.RealtimePresenceSnapshot, error) {
	g.joined = req
	return g.snapshot, nil
}

func (g *stubGateway) LeavePresence(
	_ context.Context,
	req *realtimeports.RealtimeScopeRequest,
) error {
	g.left = req
	return nil
}

func (g *stubGateway) Drain() {}

type authMode int

const (
	authUser authMode = iota
	authPortal
	authAPIKey
	authSession
)

var (
	testUserID = pulid.MustNew("usr_")
	testBUID   = pulid.MustNew("bu_")
	testOrgID  = pulid.MustNew("org_")

	testSessionID = pulid.MustNew("ses_")
)

const testSessionSecretHash = "test-session-secret-hash"

type stubLiveness struct {
	mu      sync.Mutex
	ended   bool
	rotated bool
}

func (l *stubLiveness) Standings(
	_ context.Context,
	sessionIDs []pulid.ID,
) ([]sessionports.SessionStanding, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	secrets := []string{testSessionSecretHash}
	if l.rotated {
		secrets = []string{"a-newer-secret-hash"}
	}
	standings := make([]sessionports.SessionStanding, 0, len(sessionIDs))
	for _, sessionID := range sessionIDs {
		standings = append(standings, sessionports.SessionStanding{
			ID:           sessionID,
			Ended:        l.ended,
			SecretsKnown: !l.ended,
			SecretHashes: secrets,
		})
	}
	return standings, nil
}

func (l *stubLiveness) end() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.ended = true
}

func (l *stubLiveness) rotate() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.rotated = true
}

func newTestRouter(t *testing.T, gateway *stubGateway, mode authMode) *gin.Engine {
	t.Helper()

	return newTestRouterWithSessions(
		t,
		gateway,
		mode,
		sessionwatch.NewWatch(&stubLiveness{}, time.Minute, zap.NewNop()),
	)
}

func newTestRouterWithSessions(
	t *testing.T,
	gateway *stubGateway,
	mode authMode,
	sessions *sessionwatch.Watch,
) *gin.Engine {
	t.Helper()

	gin.SetMode(gin.TestMode)
	cfg := &config.Config{App: appconfig.Config{Debug: true}}
	eh := helpers.NewErrorHandler(
		helpers.ErrorHandlerParams{Logger: zap.NewNop(), AppConfig: &cfg.App},
	)

	engine := permissionportsmock.NewMockPermissionEngine(t)
	engine.On("Check", mock.Anything, mock.Anything).
		Return(&permissionports.PermissionCheckResult{Allowed: true}, nil).Maybe()

	h := New(Params{
		Gateway:      gateway,
		ErrorHandler: eh,
		PermissionMiddleware: middleware.NewPermissionMiddleware(
			middleware.PermissionMiddlewareParams{PermissionEngine: engine, ErrorHandler: eh},
		),
		RealtimeConfig: &cfg.Realtime,
		Sessions:       sessions,
		Logger:         zap.NewNop(),
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
		case authSession:
			authctx.SetSessionAuthContext(c, authctx.SessionAuthContextParams{
				SessionID:         testSessionID,
				SessionSecretHash: testSessionSecretHash,
				UserID:            testUserID,
				BusinessUnitID:    testBUID,
				OrganizationID:    testOrgID,
			})
		}
		c.Next()
	})
	h.RegisterRoutes(api)

	return r
}

func TestStream_WritesPreludeLiveFramesAndCloseReason(t *testing.T) {
	t.Parallel()

	frames := make(chan realtimeports.RealtimeFrame, 2)
	frames <- realtimeports.RealtimeFrame{
		ID:    "3.100-0",
		Event: realtimeports.RealtimeEventInvalidation,
		Data:  []byte(`{"resource":"shipments"}`),
	}
	close(frames)

	stream := &stubStream{
		id: "rtc_1",
		prelude: []realtimeports.RealtimeFrame{
			{Event: realtimeports.RealtimeEventReady, Data: []byte(`{"connectionId":"rtc_1"}`)},
		},
		frames: frames,
		reason: realtimeports.RealtimeCloseShutdown,
	}
	gateway := &stubGateway{stream: stream}
	router := newTestRouter(t, gateway, authUser)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/realtime/stream/?presence=users", nil)
	req.Header.Set(helpers.LastEventIDHeader, "3.99-0")
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

func streamUntilClosed(
	t *testing.T,
	change func(*stubLiveness),
) string {
	t.Helper()

	stream := &stubStream{
		id:     "rtc_1",
		frames: make(chan realtimeports.RealtimeFrame),
	}
	liveness := &stubLiveness{}
	sessions := sessionwatch.NewWatch(liveness, time.Minute, zap.NewNop())
	router := newTestRouterWithSessions(t, &stubGateway{stream: stream}, authSession, sessions)

	w := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		defer close(done)
		router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/realtime/stream/", nil))
	}()

	sessions.Sweep(t.Context())
	select {
	case <-done:
		t.Fatal("the stream closed while its session still held")
	case <-time.After(50 * time.Millisecond):
	}

	change(liveness)
	deadline := time.After(5 * time.Second)
	for {
		sessions.Sweep(t.Context())
		select {
		case <-done:
			assert.True(t, stream.closed)
			return w.Body.String()
		case <-deadline:
			t.Fatal("the stream outlived its session")
		case <-time.After(10 * time.Millisecond):
		}
	}
}

func TestStream_ClosesOnceItsSessionEnds(t *testing.T) {
	t.Parallel()

	body := streamUntilClosed(t, (*stubLiveness).end)

	assert.Contains(t, body, "event: close\ndata: {\"reason\":\"session_ended\"}")
}

func TestStream_AsksForAReconnectOnceItsSecretIsSuperseded(t *testing.T) {
	t.Parallel()

	body := streamUntilClosed(t, (*stubLiveness).rotate)

	assert.Contains(t, body, "event: close\ndata: {\"reason\":\"rotate\"}")
}

func TestStream_PortalUsersAreMarked(t *testing.T) {
	t.Parallel()

	frames := make(chan realtimeports.RealtimeFrame)
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

func TestJoinView_ScopesToTheTablesResource(t *testing.T) {
	t.Parallel()

	gateway := &stubGateway{
		snapshot: &realtimeports.RealtimePresenceSnapshot{Scope: "view:customer"},
	}
	router := newTestRouter(t, gateway, authUser)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/realtime/presence/customer/",
		bytes.NewBufferString(`{"connectionId":"rtc_01J00000000000000000000000"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, gateway.joined)
	assert.Equal(t, "view:customer", gateway.joined.Scope)
}

func TestJoinRecord_ScopesToTheResourceAndRecord(t *testing.T) {
	t.Parallel()

	customerID := pulid.MustNew("cus_")
	gateway := &stubGateway{snapshot: &realtimeports.RealtimePresenceSnapshot{}}
	router := newTestRouter(t, gateway, authUser)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/realtime/presence/customer/"+customerID.String()+"/",
		bytes.NewBufferString(`{"connectionId":"rtc_01J00000000000000000000000"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, gateway.joined)
	assert.Equal(t, "record:customer:"+customerID.String(), gateway.joined.Scope)
}

func TestLeaveRecord(t *testing.T) {
	t.Parallel()

	customerID := pulid.MustNew("cus_")
	gateway := &stubGateway{}
	router := newTestRouter(t, gateway, authUser)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(
		http.MethodDelete,
		"/api/v1/realtime/presence/customer/"+customerID.String()+
			"/?connectionId=rtc_01J00000000000000000000000",
		nil,
	))

	assert.Equal(t, http.StatusNoContent, w.Code)
	require.NotNil(t, gateway.left)
	assert.Equal(t, "record:customer:"+customerID.String(), gateway.left.Scope)
}

func TestPresence_RefusesMalformedResourceAndRecord(t *testing.T) {
	t.Parallel()

	for name, path := range map[string]string{
		"resource with a colon": "/api/v1/realtime/presence/customer:x/",
		"resource in capitals":  "/api/v1/realtime/presence/Customer/",
		"record not an id":      "/api/v1/realtime/presence/customer/not-an-id/",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			gateway := &stubGateway{}
			router := newTestRouter(t, gateway, authUser)
			req := httptest.NewRequest(
				http.MethodPost,
				path,
				bytes.NewBufferString(`{"connectionId":"rtc_01J00000000000000000000000"}`),
			)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code)
			assert.Nil(t, gateway.joined)
		})
	}
}
