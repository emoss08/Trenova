//revive:disable-next-line:var-naming
package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx/fxtest"
	"go.uber.org/zap"
)

func TestNewServer_UsesConfiguredHTTPTimeouts(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:              "127.0.0.1",
			Port:              8081,
			Mode:              "test",
			ReadTimeout:       11 * time.Second,
			ReadHeaderTimeout: 3 * time.Second,
			WriteTimeout:      17 * time.Second,
			IdleTimeout:       29 * time.Second,
			ShutdownTimeout:   5 * time.Second,
		},
	}

	server, err := NewServer(Params{
		Config: cfg,
		Logger: zap.NewNop(),
		LC:     fxtest.NewLifecycle(t),
	})
	require.NoError(t, err)

	assert.Equal(t, "127.0.0.1:8081", server.httpServer.Addr)
	assert.Equal(t, cfg.Server.ReadTimeout, server.httpServer.ReadTimeout)
	assert.Equal(t, cfg.Server.ReadHeaderTimeout, server.httpServer.ReadHeaderTimeout)
	assert.Equal(t, cfg.Server.WriteTimeout, server.httpServer.WriteTimeout)
	assert.Equal(t, cfg.Server.IdleTimeout, server.httpServer.IdleTimeout)
}

func TestNewServer_RejectsInvalidTrustedProxies(t *testing.T) {
	t.Parallel()

	_, err := NewServer(Params{
		Config: &config.Config{
			Server: config.ServerConfig{
				Host:           "127.0.0.1",
				Port:           8081,
				Mode:           "test",
				TrustedProxies: []string{"not-a-cidr"},
			},
		},
		Logger: zap.NewNop(),
		LC:     fxtest.NewLifecycle(t),
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "configure trusted proxies")
}

func TestClientIPResolution(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		server     config.ServerConfig
		remoteAddr string
		headers    map[string]string
		want       string
	}{
		{
			name:       "direct client cannot forge its own address",
			server:     config.ServerConfig{TrustedProxies: config.DefaultTrustedProxies},
			remoteAddr: "203.0.113.7:41000",
			headers:    map[string]string{"X-Forwarded-For": "198.51.100.99"},
			want:       "203.0.113.7",
		},
		{
			name:       "trusted proxy forwards the appended client address",
			server:     config.ServerConfig{TrustedProxies: config.DefaultTrustedProxies},
			remoteAddr: "172.18.0.5:41000",
			headers:    map[string]string{"X-Forwarded-For": "198.51.100.99, 203.0.113.7"},
			want:       "203.0.113.7",
		},
		{
			name: "cloudflare platform ignores a forged forwarded-for",
			server: config.ServerConfig{
				TrustedProxies:  config.DefaultTrustedProxies,
				TrustedPlatform: config.TrustedPlatformCloudflare,
			},
			remoteAddr: "172.18.0.5:41000",
			headers: map[string]string{
				"X-Forwarded-For":  "198.51.100.99",
				"CF-Connecting-IP": "203.0.113.7",
			},
			want: "203.0.113.7",
		},
		{
			name:       "trusting no proxy always reports the peer",
			server:     config.ServerConfig{TrustedProxies: []string{}},
			remoteAddr: "172.18.0.5:41000",
			headers:    map[string]string{"X-Forwarded-For": "198.51.100.99, 203.0.113.7"},
			want:       "172.18.0.5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gin.SetMode(gin.TestMode)

			router := gin.New()
			require.NoError(
				t,
				configureClientIPResolution(router, &config.Config{Server: tt.server}),
			)

			var got string
			router.GET("/probe", func(c *gin.Context) {
				got = c.ClientIP()
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, "/probe", nil)
			req.RemoteAddr = tt.remoteAddr
			for name, value := range tt.headers {
				req.Header.Set(name, value)
			}
			router.ServeHTTP(httptest.NewRecorder(), req)

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRateLimit_ForgedForwardedForCannotMintNewBuckets(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		Server: config.ServerConfig{
			TrustedProxies:  config.DefaultTrustedProxies,
			TrustedPlatform: config.TrustedPlatformCloudflare,
		},
		Security: config.SecurityConfig{
			RateLimit: config.RateLimitConfig{
				Enabled:           true,
				RequestsPerMinute: 60,
				BurstSize:         1,
				CleanupInterval:   time.Minute,
			},
		},
	}

	limiter := middleware.NewRateLimiter(cfg, helpers.NewErrorHandler(helpers.ErrorHandlerParams{
		Logger: zap.NewNop(),
		Config: cfg,
	}))

	router := gin.New()
	require.NoError(t, configureClientIPResolution(router, cfg))
	router.GET("/probe", limiter.Middleware(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	call := func(forged string) int {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/probe", nil)
		req.RemoteAddr = "172.18.0.5:41000"
		req.Header.Set("CF-Connecting-IP", "203.0.113.7")
		req.Header.Set("X-Forwarded-For", forged)
		router.ServeHTTP(w, req)
		return w.Code
	}

	assert.Equal(t, http.StatusOK, call("198.51.100.1"))
	assert.Equal(t, http.StatusTooManyRequests, call("198.51.100.2"))
	assert.Equal(t, http.StatusTooManyRequests, call("198.51.100.3"))
}
