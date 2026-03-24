package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/core/domain/session"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

func newTestAuthConfig() *config.Config {
	return &config.Config{
		App: config.AppConfig{Debug: true},
		Security: config.SecurityConfig{
			Session: config.SessionConfig{
				Name:     "trenova_session",
				Path:     "/",
				Domain:   "",
				Secure:   false,
				HTTPOnly: true,
			},
		},
	}
}

func newAuthMiddleware(cfg *config.Config, svc *mocks.MockAuthService) *AuthMiddleware {
	return NewAuthMiddleware(AuthMiddlewareParams{
		Config:  cfg,
		Service: svc,
		ErrorHandler: helpers.NewErrorHandler(helpers.ErrorHandlerParams{
			Logger: zap.NewNop(),
			Config: cfg,
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

	authSvc := mocks.NewMockAuthService(t)
	authSvc.On("ValidateSession", mock.Anything, sessionID).Return(&session.Session{
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
	authSvc := mocks.NewMockAuthService(t)
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
	authSvc.AssertNotCalled(t, "ValidateSession")
}

func TestRequireAuth_InvalidSessionID(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	cfg := newTestAuthConfig()
	authSvc := mocks.NewMockAuthService(t)
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
	authSvc.AssertNotCalled(t, "ValidateSession")
}

func TestRequireAuth_SessionValidationFails(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	cfg := newTestAuthConfig()
	sessionID := pulid.MustNew("ses_")

	authSvc := mocks.NewMockAuthService(t)
	authSvc.On("ValidateSession", mock.Anything, sessionID).
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

func TestRequireAuth_ValidBearerAPIKeyDoesNotSetUserContext(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	cfg := newTestAuthConfig()
	apiKeyID := pulid.MustNew("ak_")
	buID := pulid.MustNew("bu_")
	orgID := pulid.MustNew("org_")

	authSvc := mocks.NewMockAuthService(t)
	authSvc.
		On("AuthenticateAPIKey", mock.Anything, "trv_test.secret", "192.0.2.1", "").
		Return(&services.AuthenticatedPrincipal{
			Type:           services.PrincipalTypeAPIKey,
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

	authSvc := mocks.NewMockAuthService(t)
	authSvc.
		On("AuthenticateAPIKey", mock.Anything, "trv_test.secret", "192.0.2.1", "").
		Return(&services.AuthenticatedPrincipal{
			Type:           services.PrincipalTypeAPIKey,
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
	authSvc.AssertNotCalled(t, "ValidateSession", mock.Anything, sessionID)
	authSvc.AssertExpectations(t)
}
