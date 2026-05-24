package middleware

import (
	"encoding/json" //nolint:depguard // test response decoding
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestRequestTimeoutMiddleware_ReturnsGatewayTimeoutWithHeaders(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	cfg := newRequestTimeoutTestConfig(20 * time.Millisecond)
	router := gin.New()
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"https://cloud.trenova.app"},
		AllowMethods:     []string{http.MethodGet},
		AllowHeaders:     []string{"Content-Type"},
		AllowCredentials: true,
	}))
	router.Use(NewSecurityHeadersMiddleware(cfg))
	router.Use(NewRequestTimeoutMiddleware(cfg, newRequestTimeoutTestErrorHandler(cfg)))
	router.GET("/slow", func(c *gin.Context) {
		time.Sleep(200 * time.Millisecond)
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/slow", nil)
	req.Header.Set("Origin", "https://cloud.trenova.app")
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusGatewayTimeout, w.Code)
	assert.Equal(t, "https://cloud.trenova.app", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))

	var body helpers.ProblemDetail
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, http.StatusGatewayTimeout, body.Status)
	assert.Equal(t, "Gateway Timeout", body.Title)
	assert.Contains(t, body.Type, "request-timeout")
}

func TestRequestTimeoutMiddleware_SkipsLiveAndWebSocketRoutes(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	cfg := newRequestTimeoutTestConfig(time.Nanosecond)
	router := gin.New()
	router.Use(NewRequestTimeoutMiddleware(cfg, newRequestTimeoutTestErrorHandler(cfg)))
	router.GET("/api/v1/realtime/live", func(c *gin.Context) {
		_, hasDeadline := c.Request.Context().Deadline()
		c.JSON(http.StatusOK, gin.H{"hasDeadline": hasDeadline})
	})
	router.GET("/api/v1/realtime/ws/connect", func(c *gin.Context) {
		_, hasDeadline := c.Request.Context().Deadline()
		c.JSON(http.StatusOK, gin.H{"hasDeadline": hasDeadline})
	})

	for _, path := range []string{"/api/v1/realtime/live", "/api/v1/realtime/ws/connect"} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		if path == "/api/v1/realtime/ws/connect" {
			req.Header.Set("Upgrade", "websocket")
		}

		router.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		assert.JSONEq(t, `{"hasDeadline":false}`, w.Body.String())
	}
}

func newRequestTimeoutTestConfig(timeout time.Duration) *config.Config {
	return &config.Config{
		App: config.AppConfig{
			Name:    "trenova",
			Env:     config.EnvTest,
			Version: "test",
			Debug:   true,
		},
		Server: config.ServerConfig{
			RequestTimeout: timeout,
		},
	}
}

func newRequestTimeoutTestErrorHandler(cfg *config.Config) *helpers.ErrorHandler {
	return helpers.NewErrorHandler(helpers.ErrorHandlerParams{
		Logger: zap.NewNop(),
		Config: cfg,
	})
}
