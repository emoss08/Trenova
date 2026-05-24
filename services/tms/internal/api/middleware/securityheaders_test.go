package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

const (
	expectedAPIContentSecurityPolicy = "default-src 'none'; frame-ancestors 'none'; base-uri 'none'"
	expectedHSTSHeaderValue         = "max-age=31536000; includeSubDomains"
)

func TestSecurityHeadersMiddleware_AddsHeaders(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name   string
		method string
		path   string
		env    string
		status int
	}{
		{
			name:   "ok response without hsts outside staging and production",
			method: http.MethodGet,
			path:   "/ok",
			env:    config.EnvDevelopment,
			status: http.StatusOK,
		},
		{
			name:   "not found response includes security headers",
			method: http.MethodGet,
			path:   "/missing",
			env:    config.EnvDevelopment,
			status: http.StatusNotFound,
		},
		{
			name:   "error response includes security headers",
			method: http.MethodGet,
			path:   "/error",
			env:    config.EnvDevelopment,
			status: http.StatusInternalServerError,
		},
		{
			name:   "production response includes hsts",
			method: http.MethodGet,
			path:   "/ok",
			env:    config.EnvProduction,
			status: http.StatusOK,
		},
		{
			name:   "staging response includes hsts",
			method: http.MethodGet,
			path:   "/ok",
			env:    config.EnvStaging,
			status: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			router := newSecurityHeadersTestRouter(tt.env)
			w := httptest.NewRecorder()
			req := httptest.NewRequest(tt.method, tt.path, nil)

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.status, w.Code)
			assertSecurityHeaders(t, w.Header(), tt.env)
		})
	}
}

func TestSecurityHeadersMiddleware_PreservesCORSPreflightHeaders(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"https://cloud.trenova.app"},
		AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodOptions},
		AllowHeaders:     []string{"Content-Type", "X-CSRF-Token"},
		AllowCredentials: true,
		MaxAge:           time.Hour,
	}))
	router.Use(NewSecurityHeadersMiddleware(newSecurityHeadersTestConfig(config.EnvProduction)))
	router.GET("/ok", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/ok", nil)
	req.Header.Set("Origin", "https://cloud.trenova.app")
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	req.Header.Set("Access-Control-Request-Headers", "Content-Type, X-CSRF-Token")

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "https://cloud.trenova.app", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Contains(t, w.Header().Get("Access-Control-Allow-Headers"), "Content-Type")
	assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "GET")
	assert.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"))
}

func newSecurityHeadersTestRouter(env string) *gin.Engine {
	router := gin.New()
	router.Use(NewSecurityHeadersMiddleware(newSecurityHeadersTestConfig(env)))
	router.GET("/ok", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	router.GET("/error", func(c *gin.Context) {
		c.AbortWithStatus(http.StatusInternalServerError)
	})

	return router
}

func newSecurityHeadersTestConfig(env string) *config.Config {
	return &config.Config{
		App: config.AppConfig{Env: env},
	}
}

func assertSecurityHeaders(t *testing.T, headers http.Header, env string) {
	t.Helper()

	assert.Equal(t, "nosniff", headers.Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", headers.Get("X-Frame-Options"))
	assert.Equal(t, "no-referrer", headers.Get("Referrer-Policy"))
	assert.Equal(t, permissionsPolicyValue, headers.Get("Permissions-Policy"))
	assert.Equal(t, expectedAPIContentSecurityPolicy, headers.Get("Content-Security-Policy"))

	if env == config.EnvProduction || env == config.EnvStaging {
		assert.Equal(t, expectedHSTSHeaderValue, headers.Get("Strict-Transport-Security"))
		return
	}

	assert.Empty(t, headers.Get("Strict-Transport-Security"))
}
