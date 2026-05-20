package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestRateLimiter_AllowsRequestsWhenDisabled(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	limiter := newTestRateLimiter(&config.RateLimitConfig{
		Enabled:           false,
		RequestsPerMinute: 1,
		BurstSize:         1,
	})

	router := gin.New()
	called := 0
	router.GET("/test", limiter.Middleware(), func(c *gin.Context) {
		called++
		c.Status(http.StatusOK)
	})

	for range 3 {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.0.2.10:1234"
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	}
	assert.Equal(t, 3, called)
}

func TestRateLimiter_ThrottlesAfterBurst(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	limiter := newTestRateLimiter(&config.RateLimitConfig{
		Enabled:           true,
		RequestsPerMinute: 60,
		BurstSize:         1,
		CleanupInterval:   time.Minute,
	})

	router := gin.New()
	router.GET("/test", limiter.Middleware(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	first := httptest.NewRecorder()
	firstReq := httptest.NewRequest(http.MethodGet, "/test", nil)
	firstReq.RemoteAddr = "192.0.2.20:1234"
	router.ServeHTTP(first, firstReq)
	assert.Equal(t, http.StatusOK, first.Code)

	second := httptest.NewRecorder()
	secondReq := httptest.NewRequest(http.MethodGet, "/test", nil)
	secondReq.RemoteAddr = "192.0.2.20:1234"
	router.ServeHTTP(second, secondReq)

	assert.Equal(t, http.StatusTooManyRequests, second.Code)
	assert.Equal(t, "60", second.Header().Get("Retry-After"))
}

func TestRateLimiter_CleansUpIdleClients(t *testing.T) {
	t.Parallel()

	limiter := newTestRateLimiter(&config.RateLimitConfig{
		Enabled:           true,
		RequestsPerMinute: 60,
		BurstSize:         1,
		CleanupInterval:   time.Second,
	})

	currentTime := time.Unix(100, 0)
	limiter.now = func() time.Time { return currentTime }

	_ = limiter.limiterFor("192.0.2.30")
	assert.Len(t, limiter.clients, 1)

	currentTime = currentTime.Add(2 * time.Second)
	_ = limiter.limiterFor("192.0.2.31")

	assert.Len(t, limiter.clients, 1)
	assert.Contains(t, limiter.clients, "192.0.2.31")
}

func newTestRateLimiter(rateLimitCfg *config.RateLimitConfig) *RateLimiter {
	cfg := &config.Config{
		App: config.AppConfig{Debug: true},
		Security: config.SecurityConfig{
			RateLimit: *rateLimitCfg,
		},
	}

	return NewRateLimiter(
		cfg,
		helpers.NewErrorHandler(helpers.ErrorHandlerParams{
			Logger: zap.NewNop(),
			Config: cfg,
		}),
	)
}
