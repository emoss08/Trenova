package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/securityconfig"
	"github.com/emoss08/trenova/internal/infrastructure/ratelimit"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type recordingStore struct {
	inner    kernelports.RateLimitStore
	requests [][]kernelports.RateLimitRequest
	err      error
}

func (s *recordingStore) Check(
	ctx context.Context,
	requests []kernelports.RateLimitRequest,
) ([]kernelports.RateLimitDecision, error) {
	s.requests = append(s.requests, requests)
	if s.err != nil {
		return nil, s.err
	}
	return s.inner.Check(ctx, requests)
}

func newTestRateLimiter(
	rateLimitCfg securityconfig.RateLimitConfig,
	store kernelports.RateLimitStore,
) *RateLimiter {
	cfg := &config.Config{
		App: appconfig.Config{Debug: true},
		Security: securityconfig.Config{
			RateLimit: rateLimitCfg,
		},
	}

	return NewRateLimiter(RateLimiterParams{
		Config: cfg,
		Store:  store,
		ErrorHandler: helpers.NewErrorHandler(helpers.ErrorHandlerParams{
			Logger:    zap.NewNop(),
			AppConfig: &cfg.App,
		}),
		Logger: zap.NewNop(),
	})
}

func newMemoryStore() *ratelimit.MemoryStore {
	return ratelimit.NewMemoryStore(ratelimit.MemoryStoreOptions{})
}

func serve(router *gin.Engine, method, path, remoteAddr string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, nil)
	req.RemoteAddr = remoteAddr
	router.ServeHTTP(w, req)
	return w
}

func apiKeyRouter(
	limiter *RateLimiter,
	principal func(c *gin.Context),
) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		principal(c)
		c.Next()
	}, limiter.ByAPIKey(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	return router
}

func TestRateLimiter_AllowsRequestsWhenDisabled(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	limiter := newTestRateLimiter(securityconfig.RateLimitConfig{
		Enabled:   false,
		Anonymous: securityconfig.RateLimitScopeConfig{RequestsPerMinute: 1, BurstSize: 1},
	}, newMemoryStore())

	router := gin.New()
	called := 0
	router.GET("/test", limiter.ByClientIP(), func(c *gin.Context) {
		called++
		c.Status(http.StatusOK)
	})

	for range 3 {
		w := serve(router, http.MethodGet, "/test", "192.0.2.10:1234")
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Empty(t, w.Header().Get(HeaderRateLimitLimit))
	}
	assert.Equal(t, 3, called)
}

func TestRateLimiter_ByClientIP_ThrottlesAfterBurst(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	limiter := newTestRateLimiter(securityconfig.RateLimitConfig{
		Enabled:   true,
		Anonymous: securityconfig.RateLimitScopeConfig{RequestsPerMinute: 60, BurstSize: 2},
	}, newMemoryStore())

	router := gin.New()
	router.GET("/test", limiter.ByClientIP(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	first := serve(router, http.MethodGet, "/test", "192.0.2.20:1234")
	assert.Equal(t, http.StatusOK, first.Code)
	assert.Equal(t, "2", first.Header().Get(HeaderRateLimitLimit))
	assert.Equal(t, "1", first.Header().Get(HeaderRateLimitRemaining))
	assert.Equal(t, "60;w=60;burst=2", first.Header().Get(HeaderRateLimitPolicy))

	second := serve(router, http.MethodGet, "/test", "192.0.2.20:1234")
	assert.Equal(t, http.StatusOK, second.Code)
	assert.Equal(t, "0", second.Header().Get(HeaderRateLimitRemaining))

	third := serve(router, http.MethodGet, "/test", "192.0.2.20:1234")
	assert.Equal(t, http.StatusTooManyRequests, third.Code)
	assert.Equal(t, "1", third.Header().Get(HeaderRetryAfter))
	assert.Equal(t, "anonymous", third.Header().Get(HeaderRateLimitScope))

	other := serve(router, http.MethodGet, "/test", "192.0.2.21:1234")
	assert.Equal(t, http.StatusOK, other.Code, "buckets are per client IP")
}

func TestRateLimiter_ByAPIKey_KeysAPIKeysSeparately(t *testing.T) {
	t.Parallel()

	store := &recordingStore{inner: newMemoryStore()}
	limiter := newTestRateLimiter(securityconfig.RateLimitConfig{
		Enabled: true,
		APIKey:  securityconfig.RateLimitScopeConfig{RequestsPerMinute: 60, BurstSize: 1},
		Tenant:  securityconfig.RateLimitScopeConfig{RequestsPerMinute: 600, BurstSize: 100},
	}, store)

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	keyA := pulid.MustNew("ak_")
	keyB := pulid.MustNew("ak_")

	routerA := apiKeyRouter(limiter, func(c *gin.Context) {
		authctx.SetAPIKeyContext(c, keyA, buID, orgID)
	})
	routerB := apiKeyRouter(limiter, func(c *gin.Context) {
		authctx.SetAPIKeyContext(c, keyB, buID, orgID)
	})

	assert.Equal(t, http.StatusOK, serve(routerA, http.MethodGet, "/test", "10.0.0.1:1").Code)
	limited := serve(routerA, http.MethodGet, "/test", "10.0.0.2:1")
	assert.Equal(t, http.StatusTooManyRequests, limited.Code,
		"the same key is throttled regardless of source address")
	assert.Equal(t, "apiKey", limited.Header().Get(HeaderRateLimitScope))

	assert.Equal(t, http.StatusOK, serve(routerB, http.MethodGet, "/test", "10.0.0.1:1").Code,
		"a different key in the same tenant has its own bucket")

	require.NotEmpty(t, store.requests)
	last := store.requests[len(store.requests)-1]
	require.Len(t, last, 2)
	assert.Equal(t, "{"+orgID.String()+"}:apikey:"+keyB.String(), last[0].Key)
	assert.Equal(t, "{"+orgID.String()+"}:tenant", last[1].Key)
}

func TestRateLimiter_ByAPIKey_LeavesFirstPartyPrincipalsUnmetered(t *testing.T) {
	t.Parallel()

	store := &recordingStore{inner: newMemoryStore()}
	limiter := newTestRateLimiter(securityconfig.RateLimitConfig{
		Enabled:   true,
		Anonymous: securityconfig.RateLimitScopeConfig{RequestsPerMinute: 60, BurstSize: 1},
		APIKey:    securityconfig.RateLimitScopeConfig{RequestsPerMinute: 60, BurstSize: 1},
		Tenant:    securityconfig.RateLimitScopeConfig{RequestsPerMinute: 60, BurstSize: 1},
	}, store)

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")

	principals := map[string]func(c *gin.Context){
		"session": func(c *gin.Context) {
			authctx.SetAuthContext(c, userID, buID, orgID)
		},
		"capture device": func(c *gin.Context) {
			authctx.SetCaptureDeviceContext(c, pulid.MustNew("cdev_"), userID, buID, orgID)
		},
		"no principal": func(*gin.Context) {},
	}

	for name, principal := range principals {
		router := apiKeyRouter(limiter, principal)
		for range 5 {
			w := serve(router, http.MethodGet, "/test", "10.0.0.1:1")
			assert.Equal(t, http.StatusOK, w.Code, name)
			assert.Empty(t, w.Header().Get(HeaderRateLimitLimit), name)
		}
	}
	assert.Empty(t, store.requests, "only API keys reach the rate limit store")
}

func TestRateLimiter_ByAPIKey_TenantCeilingSpansKeys(t *testing.T) {
	t.Parallel()

	limiter := newTestRateLimiter(securityconfig.RateLimitConfig{
		Enabled: true,
		APIKey:  securityconfig.RateLimitScopeConfig{RequestsPerMinute: 600, BurstSize: 100},
		Tenant:  securityconfig.RateLimitScopeConfig{RequestsPerMinute: 60, BurstSize: 2},
	}, newMemoryStore())

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	keyA := apiKeyRouter(limiter, func(c *gin.Context) {
		authctx.SetAPIKeyContext(c, pulid.MustNew("ak_"), buID, orgID)
	})
	keyB := apiKeyRouter(limiter, func(c *gin.Context) {
		authctx.SetAPIKeyContext(c, pulid.MustNew("ak_"), buID, orgID)
	})
	session := apiKeyRouter(limiter, func(c *gin.Context) {
		authctx.SetAuthContext(c, pulid.MustNew("usr_"), buID, orgID)
	})

	assert.Equal(t, http.StatusOK, serve(keyA, http.MethodGet, "/test", "10.0.0.1:1").Code)
	assert.Equal(t, http.StatusOK, serve(keyB, http.MethodGet, "/test", "10.0.0.1:1").Code)

	limited := serve(keyA, http.MethodGet, "/test", "10.0.0.1:1")
	assert.Equal(t, http.StatusTooManyRequests, limited.Code)
	assert.Equal(t, "tenant", limited.Header().Get(HeaderRateLimitScope))

	assert.Equal(t, http.StatusOK, serve(session, http.MethodGet, "/test", "10.0.0.1:1").Code,
		"sessions in a tenant whose API keys are throttled keep working")
}

func TestRateLimiter_ByPublicToken_HashesTokenAndIsolatesBuckets(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	store := &recordingStore{inner: newMemoryStore()}
	limiter := newTestRateLimiter(securityconfig.RateLimitConfig{
		Enabled:     true,
		PublicToken: securityconfig.RateLimitScopeConfig{RequestsPerMinute: 60, BurstSize: 1},
	}, store)

	router := gin.New()
	router.GET("/links/:token/", limiter.ByPublicToken("token"), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	assert.Equal(
		t,
		http.StatusOK,
		serve(router, http.MethodGet, "/links/secret-a/", "10.0.0.1:1").Code,
	)
	assert.Equal(t, http.StatusTooManyRequests,
		serve(router, http.MethodGet, "/links/secret-a/", "10.0.0.2:1").Code)
	assert.Equal(
		t,
		http.StatusOK,
		serve(router, http.MethodGet, "/links/secret-b/", "10.0.0.1:1").Code,
	)

	for _, batch := range store.requests {
		require.Len(t, batch, 1)
		assert.NotContains(t, batch[0].Key, "secret-")
		assert.Contains(t, batch[0].Key, "token:")
	}
}

func TestRateLimiter_ExemptPathPrefixesSkipTheStore(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	store := &recordingStore{inner: newMemoryStore()}
	limiter := newTestRateLimiter(securityconfig.RateLimitConfig{
		Enabled: true,
		Anonymous: securityconfig.RateLimitScopeConfig{
			RequestsPerMinute: 60,
			BurstSize:         1,
		},
		ExemptPathPrefixes: []string{"/api/v1/webhooks/"},
	}, store)

	router := gin.New()
	router.POST("/api/v1/webhooks/samsara/abc/", limiter.ByClientIP(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	for range 3 {
		assert.Equal(t, http.StatusOK,
			serve(router, http.MethodPost, "/api/v1/webhooks/samsara/abc/", "10.0.0.1:1").Code)
	}
	assert.Empty(t, store.requests)
}

func TestRateLimiter_StoreDeniedRespondsTooManyRequests(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	store := &recordingStore{inner: newMemoryStore(), err: ratelimit.ErrStoreDenied}
	limiter := newTestRateLimiter(securityconfig.RateLimitConfig{
		Enabled:   true,
		Anonymous: securityconfig.RateLimitScopeConfig{RequestsPerMinute: 60, BurstSize: 10},
	}, store)

	router := gin.New()
	router.GET("/test", limiter.ByClientIP(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := serve(router, http.MethodGet, "/test", "10.0.0.1:1")
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.Equal(t, "1", w.Header().Get(HeaderRetryAfter))
}

func TestRateLimiter_UnexpectedStoreErrorAllowsRequest(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	store := &recordingStore{inner: newMemoryStore(), err: errors.New("boom")}
	limiter := newTestRateLimiter(securityconfig.RateLimitConfig{
		Enabled:   true,
		Anonymous: securityconfig.RateLimitScopeConfig{RequestsPerMinute: 60, BurstSize: 1},
	}, store)

	router := gin.New()
	router.GET("/test", limiter.ByClientIP(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	for range 3 {
		assert.Equal(t, http.StatusOK, serve(router, http.MethodGet, "/test", "10.0.0.1:1").Code)
	}
}

func TestRateLimiter_RetryAfterRoundsUpToWholeSeconds(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	current := time.Unix(1_700_000_000, 0)
	store := ratelimit.NewMemoryStore(ratelimit.MemoryStoreOptions{
		Now: func() time.Time { return current },
	})
	limiter := newTestRateLimiter(securityconfig.RateLimitConfig{
		Enabled:   true,
		Anonymous: securityconfig.RateLimitScopeConfig{RequestsPerMinute: 4, BurstSize: 1},
	}, store)

	router := gin.New()
	router.GET("/test", limiter.ByClientIP(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	assert.Equal(t, http.StatusOK, serve(router, http.MethodGet, "/test", "10.0.0.1:1").Code)

	current = current.Add(500 * time.Millisecond)
	w := serve(router, http.MethodGet, "/test", "10.0.0.1:1")
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.Equal(t, "15", w.Header().Get(HeaderRetryAfter))

	current = current.Add(15 * time.Second)
	assert.Equal(t, http.StatusOK, serve(router, http.MethodGet, "/test", "10.0.0.1:1").Code)
}
