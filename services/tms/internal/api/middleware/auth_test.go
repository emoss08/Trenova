package middleware

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/securityconfig"
	"github.com/emoss08/trenova/internal/modules/identity/domain/session"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/sessionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

func newTestAuthConfig() *config.Config {
	return &config.Config{
		App: appconfig.Config{Debug: true},
		Security: securityconfig.Config{
			Session: securityconfig.SessionConfig{
				Name:     "trenova_session",
				Path:     "/",
				Domain:   "",
				Secure:   false,
				HTTPOnly: true,
			},
		},
	}
}

func newAuthMiddleware(cfg *config.Config, svc *tenantportsmock.MockAuthService) *AuthMiddleware {
	return NewAuthMiddleware(AuthMiddlewareParams{
		SecurityConfig: &cfg.Security,
		Service:        svc,
		ErrorHandler: helpers.NewErrorHandler(helpers.ErrorHandlerParams{
			Logger:    zap.NewNop(),
			AppConfig: &cfg.App,
		}),
	})
}

func TestRequireAuth_ValidSession(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	cfg := newTestAuthConfig()
	userID := pulid.MustNew("usr_")
	buID := pulid.MustNew("bu_")
	orgID := pulid.MustNew("org_")
	sessionID := pulid.MustNew("ses_")

	authSvc := tenantportsmock.NewMockAuthService(t)
	authSvc.On("AuthenticateSession", mock.Anything, sessionID.String()).Return(&session.Session{
		ID:             sessionID,
		UserID:         userID,
		BusinessUnitID: buID,
		OrganizationID: orgID,
	}, nil)

	am := newAuthMiddleware(cfg, authSvc)

	var gotUserID, gotBuID, gotOrgID pulid.ID
	var userOK, buOK, orgOK bool

	r := gin.New()
	r.GET("/test", am.RequireAuth(), func(c *gin.Context) {
		gotUserID, userOK = authctx.GetUserID(c)
		gotBuID, buOK = authctx.GetBusinessUnitID(c)
		gotOrgID, orgOK = authctx.GetOrganizationID(c)
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(&http.Cookie{Name: cfg.Security.Session.Name, Value: sessionID.String()})
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, userOK)
	assert.Equal(t, userID, gotUserID)
	assert.True(t, buOK)
	assert.Equal(t, buID, gotBuID)
	assert.True(t, orgOK)
	assert.Equal(t, orgID, gotOrgID)
	authSvc.AssertExpectations(t)
}

func TestRequireAuth_NoCookie(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	cfg := newTestAuthConfig()
	authSvc := tenantportsmock.NewMockAuthService(t)
	am := newAuthMiddleware(cfg, authSvc)

	handlerCalled := false
	r := gin.New()
	r.GET("/test", am.RequireAuth(), func(c *gin.Context) {
		handlerCalled = true
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	assert.False(t, handlerCalled)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	authSvc.AssertNotCalled(t, "AuthenticateSession")
}

func TestRequireAuth_InvalidSessionID(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	cfg := newTestAuthConfig()
	authSvc := tenantportsmock.NewMockAuthService(t)
	authSvc.On("AuthenticateSession", mock.Anything, "short").
		Return(nil, errortypes.NewAuthenticationError("Session is invalid. Please login again."))
	am := newAuthMiddleware(cfg, authSvc)

	handlerCalled := false
	r := gin.New()
	r.GET("/test", am.RequireAuth(), func(c *gin.Context) {
		handlerCalled = true
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(&http.Cookie{Name: cfg.Security.Session.Name, Value: "short"})
	r.ServeHTTP(w, req)

	assert.False(t, handlerCalled)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	authSvc.AssertExpectations(t)
}

func TestRequireAuth_SessionValidationFails(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	cfg := newTestAuthConfig()
	sessionID := pulid.MustNew("ses_")

	authSvc := tenantportsmock.NewMockAuthService(t)
	authSvc.On("AuthenticateSession", mock.Anything, sessionID.String()).
		Return(nil, errors.New("session expired"))

	am := newAuthMiddleware(cfg, authSvc)

	handlerCalled := false
	r := gin.New()
	r.GET("/test", am.RequireAuth(), func(c *gin.Context) {
		handlerCalled = true
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(&http.Cookie{Name: cfg.Security.Session.Name, Value: sessionID.String()})
	r.ServeHTTP(w, req)

	assert.False(t, handlerCalled)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	authSvc.AssertExpectations(t)
}

func requestWithFailingSession(t *testing.T, failure error) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)

	cfg := newTestAuthConfig()
	token := pulid.MustNew("ses_").String() + ".secret"
	authSvc := tenantportsmock.NewMockAuthService(t)
	authSvc.EXPECT().AuthenticateSession(mock.Anything, token).Return(nil, failure).Once()

	r := gin.New()
	r.GET("/test", newAuthMiddleware(cfg, authSvc).RequireAuth(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.AddCookie(&http.Cookie{Name: cfg.Security.Session.Name, Value: token})
	r.ServeHTTP(w, req)
	return w
}

func TestRequireAuth_ClearsTheCookieOfASessionThatEnded(t *testing.T) {
	t.Parallel()

	w := requestWithFailingSession(t, errortypes.NewAuthenticationError("Session has expired."))

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	cookies := w.Result().Cookies()
	assert.Len(t, cookies, 1)
	assert.Equal(t, "trenova_session", cookies[0].Name)
	assert.Negative(t, cookies[0].MaxAge)
}

func TestRequireAuth_KeepsTheCookieWhenOnlyTheSecretIsStale(t *testing.T) {
	t.Parallel()

	w := requestWithFailingSession(
		t,
		errortypes.NewAuthenticationError("Session is invalid. Please login again.").
			WithInternal(sessionports.ErrSessionSecretMismatch),
	)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Empty(t, w.Result().Cookies())
}

func TestRequireAuth_AnUnconfirmedSessionIsRetryableAndKeepsItsCookie(t *testing.T) {
	t.Parallel()

	w := requestWithFailingSession(
		t,
		fmt.Errorf("%w: database unavailable", sessionports.ErrSessionAccessUnconfirmed),
	)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Empty(t, w.Result().Cookies())
}

func requestWithGracedSecret(t *testing.T, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)

	cfg := newTestAuthConfig()
	token := pulid.MustNew("ses_").String() + ".secret"
	authSvc := tenantportsmock.NewMockAuthService(t)
	authSvc.EXPECT().AuthenticateSession(mock.Anything, token).Return(&session.Session{
		ID:                      pulid.MustNew("ses_"),
		UserID:                  pulid.MustNew("usr_"),
		OrganizationID:          pulid.MustNew("org_"),
		BusinessUnitID:          pulid.MustNew("bu_"),
		SecretHash:              "current-secret-hash",
		PreviousSecretHash:      "previous-secret-hash",
		AuthenticatedSecretHash: "previous-secret-hash",
	}, nil).Once()

	r := gin.New()
	handler := func(c *gin.Context) { c.Status(http.StatusOK) }
	r.GET("/records", newAuthMiddleware(cfg, authSvc).RequireAuth(), handler)
	r.POST("/records", newAuthMiddleware(cfg, authSvc).RequireAuth(), handler)
	r.POST("/graphql", newAuthMiddleware(cfg, authSvc).RequireAuth(), func(c *gin.Context) {
		assert.True(t, authctx.UsesSupersededSessionSecret(c.Request.Context()))
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, nil)
	req.AddCookie(&http.Cookie{Name: cfg.Security.Session.Name, Value: token})
	r.ServeHTTP(w, req)
	return w
}

func TestRequireAuth_APreviousSecretMayStillRead(t *testing.T) {
	t.Parallel()

	w := requestWithGracedSecret(t, http.MethodGet, "/records")

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRequireAuth_APreviousSecretCannotWrite(t *testing.T) {
	t.Parallel()

	w := requestWithGracedSecret(t, http.MethodPost, "/records")

	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), "session-context-stale")
	assert.Empty(t, w.Result().Cookies())
}

func TestRequireAuth_APreviousSecretReachesGraphQLMarkedAsSuperseded(t *testing.T) {
	t.Parallel()

	w := requestWithGracedSecret(t, http.MethodPost, "/graphql")

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRequireAuth_ValidBearerAPIKeyDoesNotSetUserContext(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	cfg := newTestAuthConfig()
	apiKeyID := pulid.MustNew("ak_")
	buID := pulid.MustNew("bu_")
	orgID := pulid.MustNew("org_")

	authSvc := tenantportsmock.NewMockAuthService(t)
	authSvc.
		On("AuthenticateAPIKey", mock.Anything, "trv_test.secret", "192.0.2.1", "").
		Return(&tenantports.AuthenticatedPrincipal{
			Type:           permissionports.PrincipalTypeAPIKey,
			PrincipalID:    apiKeyID,
			BusinessUnitID: buID,
			OrganizationID: orgID,
		}, nil)

	am := newAuthMiddleware(cfg, authSvc)

	var authContext *authctx.AuthContext

	r := gin.New()
	r.GET("/test", am.RequireAuth(), func(c *gin.Context) {
		authContext = authctx.GetAuthContext(c)
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer trv_test.secret")
	req.RemoteAddr = "192.0.2.1:1234"
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	if assert.NotNil(t, authContext) {
		assert.Equal(t, authctx.PrincipalTypeAPIKey, authContext.PrincipalType)
		assert.Equal(t, apiKeyID, authContext.PrincipalID)
		assert.Equal(t, apiKeyID, authContext.APIKeyID)
		assert.Equal(t, buID, authContext.BusinessUnitID)
		assert.Equal(t, orgID, authContext.OrganizationID)
		assert.True(t, authContext.UserID.IsNil())
	}
	authSvc.AssertExpectations(t)
}

func TestRequireAuth_BearerTakesPrecedenceOverSessionCookie(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	cfg := newTestAuthConfig()
	apiKeyID := pulid.MustNew("ak_")
	sessionID := pulid.MustNew("ses_")

	authSvc := tenantportsmock.NewMockAuthService(t)
	authSvc.
		On("AuthenticateAPIKey", mock.Anything, "trv_test.secret", "192.0.2.1", "").
		Return(&tenantports.AuthenticatedPrincipal{
			Type:           permissionports.PrincipalTypeAPIKey,
			PrincipalID:    apiKeyID,
			BusinessUnitID: pulid.MustNew("bu_"),
			OrganizationID: pulid.MustNew("org_"),
		}, nil)

	am := newAuthMiddleware(cfg, authSvc)

	r := gin.New()
	r.GET("/test", am.RequireAuth(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer trv_test.secret")
	req.RemoteAddr = "192.0.2.1:1234"
	req.AddCookie(&http.Cookie{Name: cfg.Security.Session.Name, Value: sessionID.String()})
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	authSvc.AssertNotCalled(t, "AuthenticateSession", mock.Anything, sessionID.String())
	authSvc.AssertExpectations(t)
}
