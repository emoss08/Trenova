package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	csrfutil "github.com/emoss08/trenova/internal/api/csrf"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestCSRFMiddleware_AllowsSafeMethodWithoutToken(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	router := newCSRFMiddlewareRouter(t, authctx.PrincipalTypeUser)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCSRFMiddleware_AllowsAPIKeyWithoutToken(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	router := newCSRFMiddlewareRouter(t, authctx.PrincipalTypeAPIKey)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCSRFMiddleware_RejectsSessionUnsafeMethodWithoutToken(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	router := newCSRFMiddlewareRouter(t, authctx.PrincipalTypeUser)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "session-value"})
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestCSRFMiddleware_AllowsSessionUnsafeMethodWithToken(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	router := newCSRFMiddlewareRouter(t, authctx.PrincipalTypeUser)
	token := csrfutil.Token("session-value", "test-session-secret")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req.Header.Set("X-CSRF-Token", token)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "session-value"})
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCSRFBrowserGuard_AllowsTrustedOrigin(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	router := newCSRFBrowserGuardRouter(t, "enforce")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req.Header.Set("Origin", "https://app.example.test")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCSRFBrowserGuard_RejectsUntrustedOriginInEnforceMode(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	router := newCSRFBrowserGuardRouter(t, "enforce")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req.Header.Set("Origin", "https://evil.example.test")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestCSRFBrowserGuard_RejectsCrossSiteFetchMetadataInEnforceMode(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	router := newCSRFBrowserGuardRouter(t, "enforce")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestCSRFBrowserGuard_ReportModeAllowsViolation(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	router := newCSRFBrowserGuardRouter(t, "report")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req.Header.Set("Origin", "https://evil.example.test")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCSRFBrowserGuard_RejectsMissingOriginAndRefererWithSessionCookie(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	router := newCSRFBrowserGuardRouter(t, "enforce")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "session-value"})
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func newCSRFMiddlewareRouter(t *testing.T, principalType string) *gin.Engine {
	t.Helper()

	cfg := &config.Config{
		App: config.AppConfig{Debug: true},
		Security: config.SecurityConfig{
			Session: config.SessionConfig{
				Name:   "session_id",
				Secret: "test-session-secret",
			},
			CSRF: config.CSRFConfig{
				HeaderName: "X-CSRF-Token",
				TokenName:  "csrf_token",
				BrowserGuard: config.CSRFBrowserGuardConfig{
					Mode: "report",
				},
			},
		},
	}
	middleware := NewCSRFMiddleware(
		cfg,
		helpers.NewErrorHandler(helpers.ErrorHandlerParams{
			Logger: zap.NewNop(),
			Config: cfg,
		}),
	)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		switch principalType {
		case authctx.PrincipalTypeAPIKey:
			authctx.SetAPIKeyContext(
				c,
				pulid.MustNew("ak_"),
				pulid.MustNew("bu_"),
				pulid.MustNew("org_"),
			)
		default:
			authctx.SetAuthContext(
				c,
				pulid.MustNew("usr_"),
				pulid.MustNew("bu_"),
				pulid.MustNew("org_"),
			)
		}
		c.Next()
	})
	router.Use(middleware.RequireToken())
	router.Any("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	return router
}

func newCSRFBrowserGuardRouter(t *testing.T, mode string) *gin.Engine {
	t.Helper()

	cfg := &config.Config{
		Server: config.ServerConfig{
			CORS: config.CORSConfig{
				AllowedOrigins: []string{"https://app.example.test"},
			},
		},
		Security: config.SecurityConfig{
			Session: config.SessionConfig{
				Name: "session_id",
			},
			CSRF: config.CSRFConfig{
				HeaderName: "X-CSRF-Token",
				TokenName:  "csrf_token",
				BrowserGuard: config.CSRFBrowserGuardConfig{
					Mode: mode,
				},
			},
		},
	}

	router := gin.New()
	router.Use(NewCSRFBrowserGuard(
		cfg,
		helpers.NewErrorHandler(helpers.ErrorHandlerParams{
			Logger: zap.NewNop(),
			Config: cfg,
		}),
		zap.NewNop(),
	).Guard())
	router.Any("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	return router
}
