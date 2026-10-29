package inboxpresencehandler

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/testutil/mocks/permissionportsmock"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type stubGateway struct {
	joined   *realtimeports.RealtimeScopeRequest
	left     *realtimeports.RealtimeScopeRequest
	snapshot *realtimeports.RealtimePresenceSnapshot
}

func (g *stubGateway) Open(
	context.Context,
	*realtimeports.OpenRealtimeStreamRequest,
) (realtimeports.RealtimeStream, error) {
	return nil, nil
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

func (g *stubGateway) SignalTyping(context.Context, *realtimeports.RealtimeTypingRequest) error {
	return nil
}

func (g *stubGateway) Drain() {}

var (
	testUserID = pulid.MustNew("usr_")
	testBUID   = pulid.MustNew("bu_")
	testOrgID  = pulid.MustNew("org_")
)

func newRouter(t *testing.T, gateway *stubGateway, allowed bool) *gin.Engine {
	t.Helper()

	gin.SetMode(gin.TestMode)
	cfg := &config.Config{App: appconfig.Config{Debug: true}}
	eh := helpers.NewErrorHandler(
		helpers.ErrorHandlerParams{Logger: zap.NewNop(), AppConfig: &cfg.App},
	)

	engine := permissionportsmock.NewMockPermissionEngine(t)
	engine.On("Check", mock.Anything, mock.MatchedBy(func(req *permissionports.PermissionCheckRequest) bool {
		return req.Resource == permission.ResourceInboundMessage.String() &&
			req.Operation == permission.OpRead
	})).
		Return(&permissionports.PermissionCheckResult{Allowed: allowed}, nil).
		Maybe()

	h := New(Params{
		Gateway:      gateway,
		ErrorHandler: eh,
		PermissionMiddleware: middleware.NewPermissionMiddleware(
			middleware.PermissionMiddlewareParams{PermissionEngine: engine, ErrorHandler: eh},
		),
	})

	r := gin.New()
	api := r.Group("/api/v1")
	api.Use(func(c *gin.Context) {
		authctx.SetAuthContext(c, testUserID, testBUID, testOrgID)
		c.Next()
	})
	h.RegisterRoutes(api)

	return r
}

func TestJoinCarriesWhatTheCallerIsDoing(t *testing.T) {
	t.Parallel()

	gateway := &stubGateway{snapshot: &realtimeports.RealtimePresenceSnapshot{
		Scope: Scope,
		Members: []realtimeports.RealtimePresenceMember{{
			UserID:       testUserID.String(),
			ConnectionID: "rtc_01J00000000000000000000000",
			Name:         "Dana",
			MessageID:    "imsg_01J00000000000000000000000",
			Activity:     realtimeports.RealtimeActivityWriting,
		}},
	}}
	r := newRouter(t, gateway, true)

	body := `{"connectionId":"rtc_01J00000000000000000000000",` +
		`"messageId":" imsg_01J00000000000000000000000 ","activity":"writing"}`
	req := httptest.NewRequestWithContext(
		t.Context(), http.MethodPost, "/api/v1/inbox/presence/", bytes.NewBufferString(body),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.NotNil(t, gateway.joined)
	assert.Equal(t, Scope, gateway.joined.Scope)
	assert.Equal(t, testUserID, gateway.joined.UserID)
	assert.Equal(t, testOrgID, gateway.joined.OrganizationID)
	assert.Equal(t, "rtc_01J00000000000000000000000", gateway.joined.ConnectionID)
	assert.Equal(t, "imsg_01J00000000000000000000000", gateway.joined.State.MessageID)
	assert.Equal(t, realtimeports.RealtimeActivityWriting, gateway.joined.State.Activity)

	var snapshot realtimeports.RealtimePresenceSnapshot
	require.NoError(t, sonic.Unmarshal(rec.Body.Bytes(), &snapshot))
	require.Len(t, snapshot.Members, 1)
	assert.Equal(t, "writing", snapshot.Members[0].Activity)
	assert.Equal(t, "imsg_01J00000000000000000000000", snapshot.Members[0].MessageID)
}

func TestJoinWithNullsClearsTheState(t *testing.T) {
	t.Parallel()

	gateway := &stubGateway{snapshot: &realtimeports.RealtimePresenceSnapshot{Scope: Scope}}
	r := newRouter(t, gateway, true)

	body := `{"connectionId":"rtc_01J00000000000000000000000","messageId":null,"activity":null}`
	req := httptest.NewRequestWithContext(
		t.Context(), http.MethodPost, "/api/v1/inbox/presence/", bytes.NewBufferString(body),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.NotNil(t, gateway.joined)
	assert.Equal(t, realtimeports.RealtimePresenceState{}, gateway.joined.State)
}

func TestLeave(t *testing.T) {
	t.Parallel()

	gateway := &stubGateway{}
	r := newRouter(t, gateway, true)

	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodDelete,
		"/api/v1/inbox/presence/?connectionId=rtc_01J00000000000000000000000",
		nil,
	)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNoContent, rec.Code)
	require.NotNil(t, gateway.left)
	assert.Equal(t, Scope, gateway.left.Scope)
	assert.Equal(t, "rtc_01J00000000000000000000000", gateway.left.ConnectionID)
}

func TestPresenceNeedsInboundMessageRead(t *testing.T) {
	t.Parallel()

	gateway := &stubGateway{}
	r := newRouter(t, gateway, false)

	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPost,
		"/api/v1/inbox/presence/",
		bytes.NewBufferString(`{"connectionId":"rtc_01J00000000000000000000000"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
	assert.Nil(t, gateway.joined)
}
