package supportaccessmiddleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/cloud/domain/supportaccess"
	supportctx "github.com/emoss08/trenova/internal/cloud/supportaccess"
	"github.com/emoss08/trenova/internal/cloud/supportaccess/supportaccessservice"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const cookieName = "trenova_support_session"

type fakeResolver struct {
	mu       sync.Mutex
	active   *supportctx.Active
	err      error
	refusals []*supportaccessservice.RefusalRequest
}

func (*fakeResolver) Enabled() bool      { return true }
func (*fakeResolver) CookieName() string { return cookieName }

func (f *fakeResolver) Resolve(
	context.Context,
	supportaccessservice.StaffContext,
	string,
) (*supportctx.Active, *supportaccess.Session, error) {
	if f.err != nil {
		return nil, nil, f.err
	}
	return f.active, &supportaccess.Session{}, nil
}

func (f *fakeResolver) RecordRefusal(_ context.Context, req *supportaccessservice.RefusalRequest) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.refusals = append(f.refusals, req)
}

type observed struct {
	orgID     pulid.ID
	userID    pulid.ID
	scopeOrg  pulid.ID
	hasActive bool
}

func newRouter(t *testing.T, resolver *fakeResolver, staffOrg pulid.ID, seen *observed) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	m := &Middleware{
		service: resolver,
		session: &config.SessionConfig{Path: "/", Secure: true, SameSite: "lax"},
		eh: helpers.NewErrorHandler(helpers.ErrorHandlerParams{
			Logger: zap.NewNop(),
			Config: &config.Config{App: config.AppConfig{Debug: true}},
		}),
	}

	engine := gin.New()
	engine.Use(func(c *gin.Context) {
		authctx.SetSessionAuthContext(c, authctx.SessionAuthContextParams{
			SessionID:          pulid.MustNew("ses_"),
			UserID:             pulid.MustNew("usr_"),
			BusinessUnitID:     pulid.MustNew("bu_"),
			OrganizationID:     staffOrg,
			AuthenticatorAAL:   2,
			MFAAuthenticatedAt: 1,
		})
		c.Next()
	})
	engine.Use(m.Handle())

	record := func(c *gin.Context) {
		authCtx := authctx.GetAuthContext(c)
		seen.orgID = authCtx.OrganizationID
		seen.userID = authCtx.UserID
		if tenant, ok := dbscope.From(c.Request.Context()).Tenant(); ok {
			seen.scopeOrg = tenant.OrganizationID
		}
		_, seen.hasActive = supportctx.ActiveFrom(c.Request.Context())
		c.Status(http.StatusOK)
	}
	engine.GET("/api/v1/shipments/", record)
	engine.POST("/api/v1/shipments/", record)
	engine.POST("/api/v1/shipments/calculate-totals", record)
	engine.POST("/api/v1/users/me/change-password/", record)
	engine.POST("/api/v1/api-keys/", record)
	engine.POST("/graphql", record)
	engine.GET("/api/v1/support/sessions/current/", record)

	return engine
}

func activeSession(writeUntil int64) *supportctx.Active {
	return &supportctx.Active{
		SessionID:       pulid.MustNew("sps_"),
		OrganizationID:  pulid.MustNew("org_"),
		BusinessUnitID:  pulid.MustNew("bu_"),
		PrincipalUserID: pulid.MustNew("usr_"),
		StaffName:       "Jordan Lee",
		GrantMode:       supportaccess.AccessModeReadWrite,
		ElevatedUntil:   writeUntil,
		Now:             1000,
	}
}

func do(engine *gin.Engine, method, path string, withCookie bool) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, http.NoBody)
	if withCookie {
		req.AddCookie(&http.Cookie{Name: cookieName, Value: "v1.token"})
	}
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)
	return recorder
}

func TestWithoutACookieTheRequestStaysInTheStaffOrganization(t *testing.T) {
	t.Parallel()
	staffOrg := pulid.MustNew("org_")
	seen := &observed{}
	engine := newRouter(t, &fakeResolver{active: activeSession(0)}, staffOrg, seen)

	resp := do(engine, http.MethodPost, "/api/v1/shipments/", false)
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, staffOrg, seen.orgID)
	assert.False(t, seen.hasActive)
}

func TestReadsEnterTheTargetTenantAsTheSupportPrincipal(t *testing.T) {
	t.Parallel()
	active := activeSession(0)
	seen := &observed{}
	engine := newRouter(t, &fakeResolver{active: active}, pulid.MustNew("org_"), seen)

	resp := do(engine, http.MethodGet, "/api/v1/shipments/", true)
	require.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, active.OrganizationID, seen.orgID)
	assert.Equal(t, active.OrganizationID, seen.scopeOrg)
	assert.Equal(t, active.PrincipalUserID, seen.userID)
	assert.True(t, seen.hasActive)
	assert.Equal(t, "read_only", resp.Header().Get(HeaderSessionState))
}

func TestReadOnlySessionsRefuseWrites(t *testing.T) {
	t.Parallel()
	resolver := &fakeResolver{active: activeSession(0)}
	seen := &observed{}
	engine := newRouter(t, resolver, pulid.MustNew("org_"), seen)

	resp := do(engine, http.MethodPost, "/api/v1/shipments/", true)
	assert.Equal(t, http.StatusForbidden, resp.Code)
	assert.True(t, seen.orgID.IsNil(), "the handler must not run")
	require.Len(t, resolver.refusals, 1)
	assert.False(t, resolver.refusals[0].Denied)

	resp = do(engine, http.MethodPost, "/api/v1/shipments/calculate-totals", true)
	assert.Equal(t, http.StatusOK, resp.Code, "a computation that saves nothing stays available")

	resp = do(engine, http.MethodPost, "/graphql", true)
	assert.Equal(t, http.StatusOK, resp.Code, "GraphQL mutations are refused by the extension")
}

func TestElevatedSessionsWriteButNeverTouchTheDenyList(t *testing.T) {
	t.Parallel()
	resolver := &fakeResolver{active: activeSession(2000)}
	seen := &observed{}
	engine := newRouter(t, resolver, pulid.MustNew("org_"), seen)

	resp := do(engine, http.MethodPost, "/api/v1/shipments/", true)
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "read_write", resp.Header().Get(HeaderSessionState))

	for _, path := range []string{"/api/v1/users/me/change-password/", "/api/v1/api-keys/"} {
		resp = do(engine, http.MethodPost, path, true)
		assert.Equal(t, http.StatusForbidden, resp.Code, path)
	}
	require.Len(t, resolver.refusals, 2)
	assert.True(t, resolver.refusals[0].Denied)
}

func TestAnEndedSessionClearsTheCookieAndRefuses(t *testing.T) {
	t.Parallel()
	resolver := &fakeResolver{
		err: &supportaccessservice.SessionEndedError{Reason: supportaccess.EndReasonGrantRevoked},
	}
	seen := &observed{}
	engine := newRouter(t, resolver, pulid.MustNew("org_"), seen)

	resp := do(engine, http.MethodGet, "/api/v1/shipments/", true)
	assert.Equal(t, http.StatusForbidden, resp.Code)
	assert.Equal(t, StateEnded, resp.Header().Get(HeaderSessionState))

	cleared := false
	for _, cookie := range resp.Result().Cookies() {
		if cookie.Name == cookieName && cookie.MaxAge < 0 {
			cleared = true
		}
	}
	assert.True(t, cleared)
}

func TestStaffRoutesRunAsTheStaffMember(t *testing.T) {
	t.Parallel()
	staffOrg := pulid.MustNew("org_")
	seen := &observed{}
	engine := newRouter(t, &fakeResolver{active: activeSession(0)}, staffOrg, seen)

	resp := do(engine, http.MethodGet, "/api/v1/support/sessions/current/", true)
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, staffOrg, seen.orgID)
	assert.False(t, seen.hasActive)
}
