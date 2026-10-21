package authhandler

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"

	"github.com/emoss08/trenova/internal/api/csrf"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/modules/identity/domain/session"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newTestConfig() *config.Config {
	return &config.Config{
		App: config.AppConfig{Debug: true},
		Security: config.SecurityConfig{
			Session: config.SessionConfig{
				Name:     "session_id",
				Secret:   "test-session-secret",
				Path:     "/",
				Domain:   "localhost",
				Secure:   false,
				HTTPOnly: true,
			},
			CSRF: config.CSRFConfig{
				TokenName:  "csrf_token",
				HeaderName: "X-CSRF-Token",
			},
		},
	}
}

func newTestHandler(svc *tenantportsmock.MockAuthService) (*Handler, *gin.Engine) {
	gin.SetMode(gin.TestMode)
	cfg := newTestConfig()
	eh := helpers.NewErrorHandler(helpers.ErrorHandlerParams{
		Logger: zap.NewNop(),
		Config: cfg,
	})
	h := New(Params{
		Service:      svc,
		Logger:       zap.NewNop(),
		Config:       cfg,
		ErrorHandler: eh,
	})
	r := gin.New()
	h.RegisterRoutes(&r.RouterGroup)
	return h, r
}

func TestLogin_Success(t *testing.T) {
	t.Parallel()

	userID := pulid.MustNew("usr_")
	buID := pulid.MustNew("bu_")
	orgID := pulid.MustNew("org_")
	sessionID := pulid.MustNew("ses_")

	svc := tenantportsmock.NewMockAuthService(t)
	svc.On("Login", mock.Anything, mock.Anything).Return(&tenantports.LoginResponse{
		User: &tenant.User{
			ID:                    userID,
			BusinessUnitID:        buID,
			CurrentOrganizationID: orgID,
			EmailAddress:          "test@example.com",
		},
		ExpiresAt:    9999999999,
		SessionID:    sessionID.String(),
		SessionToken: sessionID.String() + ".opaque-session-secret",
	}, nil)

	_, r := newTestHandler(svc)

	body, _ := json.Marshal(map[string]string{
		"emailAddress": "test@example.com",
		"password":     "password123",
	})

	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp tenantports.LoginResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, sessionID.String(), resp.SessionID)
	assert.Equal(t, int64(9999999999), resp.ExpiresAt)
	assert.Equal(t, csrf.Token(sessionID.String(), "test-session-secret"), resp.CSRFToken)
	assert.Equal(t, "test@example.com", resp.User.EmailAddress)

	cookies := w.Result().Cookies()
	var found bool
	for _, c := range cookies {
		if c.Name == "session_id" {
			found = true
			assert.Equal(t, sessionID.String()+".opaque-session-secret", c.Value)
			assert.True(t, c.HttpOnly)
		}
		assert.NotEqual(t, "csrf_token", c.Name)
	}
	assert.True(t, found, "session cookie should be set")
}

func TestCSRFToken_Success(t *testing.T) {
	t.Parallel()

	sessionID := pulid.MustNew("ses_")

	svc := tenantportsmock.NewMockAuthService(t)
	svc.On("AuthenticateSession", mock.Anything, sessionID.String()).Return(&session.Session{
		ID: sessionID,
	}, nil)

	_, r := newTestHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/auth/csrf", nil)
	req.AddCookie(&http.Cookie{
		Name:  "session_id",
		Value: sessionID.String(),
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "X-CSRF-Token", resp["headerName"])
	assert.Equal(t, csrf.Token(sessionID.String(), "test-session-secret"), resp["csrfToken"])
}

func TestLogin_InvalidJSON(t *testing.T) {
	t.Parallel()

	svc := tenantportsmock.NewMockAuthService(t)
	_, r := newTestHandler(svc)

	req := httptest.NewRequest(
		http.MethodPost,
		"/auth/login",
		bytes.NewReader([]byte("{invalid json")),
	)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.GreaterOrEqual(t, w.Code, 400)
}

func TestRedirectSSOError_UsesRelativeFallbackWithoutAllowedOrigins(t *testing.T) {
	t.Parallel()

	svc := tenantportsmock.NewMockAuthService(t)
	h, r := newTestHandler(svc)

	r.GET("/sso/error", func(c *gin.Context) {
		h.redirectSSOError(c, nil, errors.New("callback failed"))
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/sso/error", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusFound, w.Code)
	assert.Contains(t, w.Header().Get("Location"), "/login?sso_error=")
}

func TestResolveSSOErrorOrigin_PrefersReturnToOrigin(t *testing.T) {
	t.Parallel()

	svc := tenantportsmock.NewMockAuthService(t)
	h, _ := newTestHandler(svc)

	origin := h.resolveSSOErrorOrigin(&tenantports.SSOLoginState{
		ReturnTo: "https://app.example.test/shipments",
	})

	assert.Equal(t, "https://app.example.test", origin)
}

func TestLogin_ServiceError(t *testing.T) {
	t.Parallel()

	svc := tenantportsmock.NewMockAuthService(t)
	svc.On("Login", mock.Anything, mock.Anything).Return(nil, errors.New("invalid credentials"))

	_, r := newTestHandler(svc)

	body, _ := json.Marshal(map[string]string{
		"emailAddress": "test@example.com",
		"password":     "wrongpass",
	})

	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.GreaterOrEqual(t, w.Code, 400)
}

func TestLogout_Success(t *testing.T) {
	t.Parallel()

	sessionID := pulid.MustNew("ses_")

	svc := tenantportsmock.NewMockAuthService(t)
	svc.On("AuthenticateSession", mock.Anything, sessionID.String()).
		Return(&session.Session{ID: sessionID}, nil)
	svc.On("Logout", mock.Anything, sessionID).Return(nil)

	_, r := newTestHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	req.AddCookie(&http.Cookie{
		Name:  "session_id",
		Value: sessionID.String(),
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	svc.AssertCalled(t, "Logout", mock.Anything, sessionID)

	cookies := w.Result().Cookies()
	for _, c := range cookies {
		if c.Name == "session_id" {
			assert.Equal(t, "", c.Value)
			assert.Less(t, c.MaxAge, 0)
			break
		}
	}
}

func TestLogout_NoCookie(t *testing.T) {
	t.Parallel()

	svc := tenantportsmock.NewMockAuthService(t)
	_, r := newTestHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestValidateSession_Valid(t *testing.T) {
	t.Parallel()

	sessionID := pulid.MustNew("ses_")

	svc := tenantportsmock.NewMockAuthService(t)
	svc.On("AuthenticateSession", mock.Anything, mock.Anything).
		Return(&session.Session{ID: sessionID}, nil)

	_, r := newTestHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/auth/validate-session", nil)
	req.AddCookie(&http.Cookie{
		Name:  "session_id",
		Value: sessionID.String(),
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]bool
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp["valid"])
}

func TestValidateSession_NoCookie(t *testing.T) {
	t.Parallel()

	svc := tenantportsmock.NewMockAuthService(t)
	_, r := newTestHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/auth/validate-session", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var resp map[string]bool
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.False(t, resp["valid"])
}

func TestValidateSession_InvalidSession(t *testing.T) {
	t.Parallel()

	sessionID := pulid.MustNew("ses_")

	svc := tenantportsmock.NewMockAuthService(t)
	svc.On("AuthenticateSession", mock.Anything, mock.Anything).
		Return(nil, errors.New("session expired"))

	_, r := newTestHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/auth/validate-session", nil)
	req.AddCookie(&http.Cookie{
		Name:  "session_id",
		Value: sessionID.String(),
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var resp map[string]bool
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.False(t, resp["valid"])
}

func TestKeepSessionAlive_RecordsActivityOnTheCookieSession(t *testing.T) {
	t.Parallel()

	sessionID := pulid.MustNew("ses_")

	svc := tenantportsmock.NewMockAuthService(t)
	svc.On("KeepSessionAlive", mock.Anything, "token-value").
		Return(&session.Session{ID: sessionID}, nil).
		Once()

	_, r := newTestHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/auth/session/keepalive", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "token-value"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Empty(t, w.Result().Cookies())
}

func TestKeepSessionAlive_NoCookieIsUnauthorized(t *testing.T) {
	t.Parallel()

	svc := tenantportsmock.NewMockAuthService(t)
	_, r := newTestHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/auth/session/keepalive", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestKeepSessionAlive_EndedSessionClearsTheCookie(t *testing.T) {
	t.Parallel()

	svc := tenantportsmock.NewMockAuthService(t)
	svc.On("KeepSessionAlive", mock.Anything, "token-value").
		Return(nil, errortypes.NewAuthenticationError("Session has expired. Please login again."))

	_, r := newTestHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/auth/session/keepalive", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "token-value"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	cookies := w.Result().Cookies()
	require.Len(t, cookies, 1)
	assert.Equal(t, "session_id", cookies[0].Name)
	assert.Empty(t, cookies[0].Value)
	assert.Negative(t, cookies[0].MaxAge)
}

func TestLogout_RefusesACookieThatDoesNotAuthenticate(t *testing.T) {
	t.Parallel()

	sessionID := pulid.MustNew("ses_")

	svc := tenantportsmock.NewMockAuthService(t)
	svc.On("AuthenticateSession", mock.Anything, sessionID.String()).
		Return(nil, errortypes.NewAuthenticationError("Session is invalid. Please login again."))

	_, r := newTestHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: sessionID.String()})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	svc.AssertNotCalled(t, "Logout", mock.Anything, mock.Anything)
}

func TestSSOCallbackSendsAnUntrustedFederationToTheSecondFactorWithoutASession(t *testing.T) {
	t.Parallel()

	svc := tenantportsmock.NewMockAuthService(t)
	h, r := newTestHandler(svc)
	loginState := &tenantports.SSOLoginState{
		Provider:         tenant.SSOProviderOkta,
		OrganizationSlug: "acme",
		ReturnTo:         "https://app.example.test/",
	}
	svc.EXPECT().GetSSOLoginState(mock.Anything, "state-1").Return(loginState, nil)
	svc.EXPECT().HandleSSOCallback(mock.Anything, tenantports.SSOCallbackRequest{
		State: "state-1",
		Code:  "code-1",
	}).Return(&tenantports.SSOCallbackResponse{
		LoginResponse: &tenantports.LoginResponse{
			MFARequired:       true,
			MFAChallengeToken: "challenge/token+1",
			ExpiresAt:         1_800_000_300,
		},
		RedirectTo: loginState.ReturnTo,
	}, nil)

	r.GET("/sso/okta/callback", h.ssoCallback(tenant.SSOProviderOkta))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/sso/okta/callback?state=state-1&code=code-1", nil))

	assert.Equal(t, http.StatusFound, w.Code)
	location, err := url.Parse(w.Header().Get("Location"))
	require.NoError(t, err)
	assert.Equal(t, "https://app.example.test/login/acme", location.Scheme+"://"+location.Host+location.Path)
	assert.Empty(t, location.RawQuery)
	fragment, err := url.ParseQuery(location.EscapedFragment())
	require.NoError(t, err)
	assert.Equal(t, "challenge/token+1", fragment.Get("mfa_challenge"))
	assert.Equal(t, "1800000300", fragment.Get("mfa_expires"))
	assert.Empty(t, w.Header().Values("Set-Cookie"))
}
