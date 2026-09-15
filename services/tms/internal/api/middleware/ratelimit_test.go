package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/ratelimit"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type recordingStore struct {
	inner    repositories.RateLimitStore
	requests [][]repositories.RateLimitRequest
	err      error
}

func (s *recordingStore) Check(
	ctx context.Context,
	requests []repositories.RateLimitRequest,
) ([]repositories.RateLimitDecision, error) {
	s.requests = append(s.requests, requests)
	if s.err != nil {
		return nil, s.err
	}
	return s.inner.Check(ctx, requests)
}

func newTestRateLimiter(
	rateLimitCfg config.RateLimitConfig,
	store repositories.RateLimitStore,
) *RateLimiter {
	cfg := &config.Config{
		App: config.AppConfig{Debug: true},
		Security: config.SecurityConfig{
			RateLimit: rateLimitCfg,
		},
	}

	return NewRateLimiter(RateLimiterParams{
		Config: cfg,
		Store:  store,
		ErrorHandler: helpers.NewErrorHandler(helpers.ErrorHandlerParams{
			Logger: zap.NewNop(),
			Config: cfg,
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

func principalRouter(
	limiter *RateLimiter,
	principal func(c *gin.Context),
) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		principal(c)
		c.Next()
	}, limiter.ByPrincipal(), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	return router
}

func TestRateLimiter_AllowsRequestsWhenDisabled(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	limiter := newTestRateLimiter(config.RateLimitConfig{
		Enabled:           false,
		RequestsPerMinute: 1,
		BurstSize:         1,
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

	limiter := newTestRateLimiter(config.RateLimitConfig{
		Enabled:           true,
		RequestsPerMinute: 60,
		BurstSize:         2,
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

func TestRateLimiter_ByPrincipal_KeysAPIKeysSeparately(t *testing.T) {
	t.Parallel()

	store := &recordingStore{inner: newMemoryStore()}
	limiter := newTestRateLimiter(config.RateLimitConfig{
		Enabled: true,
		APIKey:  config.RateLimitScopeConfig{RequestsPerMinute: 60, BurstSize: 1},
		Tenant:  config.RateLimitScopeConfig{RequestsPerMinute: 600, BurstSize: 100},
	}, store)

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	keyA := pulid.MustNew("ak_")
	keyB := pulid.MustNew("ak_")

	routerA := principalRouter(limiter, func(c *gin.Context) {
		authctx.SetAPIKeyContext(c, keyA, buID, orgID)
	})
	routerB := principalRouter(limiter, func(c *gin.Context) {
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

func TestRateLimiter_ByPrincipal_KeysUsersByOrganization(t *testing.T) {
	t.Parallel()

	store := &recordingStore{inner: newMemoryStore()}
	limiter := newTestRateLimiter(config.RateLimitConfig{
		Enabled: true,
		User:    config.RateLimitScopeConfig{RequestsPerMinute: 60, BurstSize: 1},
		Tenant:  config.RateLimitScopeConfig{Disabled: true},
	}, store)

	userID := pulid.MustNew("usr_")
	buID := pulid.MustNew("bu_")
	orgA := pulid.MustNew("org_")
	orgB := pulid.MustNew("org_")

	routerA := principalRouter(limiter, func(c *gin.Context) {
		authctx.SetAuthContext(c, userID, buID, orgA)
	})
	routerB := principalRouter(limiter, func(c *gin.Context) {
		authctx.SetAuthContext(c, userID, buID, orgB)
	})

	assert.Equal(t, http.StatusOK, serve(routerA, http.MethodGet, "/test", "10.0.0.1:1").Code)
	assert.Equal(t, http.StatusTooManyRequests,
		serve(routerA, http.MethodGet, "/test", "10.0.0.1:1").Code)
	assert.Equal(t, http.StatusOK, serve(routerB, http.MethodGet, "/test", "10.0.0.1:1").Code,
		"the same user in another organization has its own bucket")

	require.NotEmpty(t, store.requests)
	last := store.requests[len(store.requests)-1]
	require.Len(t, last, 1, "the tenant scope is disabled")
	assert.Equal(t, "{"+orgB.String()+"}:user:"+userID.String(), last[0].Key)
}

func TestRateLimiter_ByPrincipal_TenantCeilingSpansPrincipals(t *testing.T) {
	t.Parallel()

	limiter := newTestRateLimiter(config.RateLimitConfig{
		Enabled: true,
		User:    config.RateLimitScopeConfig{RequestsPerMinute: 600, BurstSize: 100},
		APIKey:  config.RateLimitScopeConfig{RequestsPerMinute: 600, BurstSize: 100},
		Tenant:  config.RateLimitScopeConfig{RequestsPerMinute: 60, BurstSize: 2},
	}, newMemoryStore())

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	userRouter := principalRouter(limiter, func(c *gin.Context) {
		authctx.SetAuthContext(c, pulid.MustNew("usr_"), buID, orgID)
	})
	keyRouter := principalRouter(limiter, func(c *gin.Context) {
		authctx.SetAPIKeyContext(c, pulid.MustNew("ak_"), buID, orgID)
	})

	assert.Equal(t, http.StatusOK, serve(userRouter, http.MethodGet, "/test", "10.0.0.1:1").Code)
	assert.Equal(t, http.StatusOK, serve(keyRouter, http.MethodGet, "/test", "10.0.0.1:1").Code)

	limited := serve(userRouter, http.MethodGet, "/test", "10.0.0.1:1")
	assert.Equal(t, http.StatusTooManyRequests, limited.Code)
	assert.Equal(t, "tenant", limited.Header().Get(HeaderRateLimitScope))
}

func TestRateLimiter_ByPrincipal_FallsBackToClientIPWithoutTenant(t *testing.T) {
	t.Parallel()

	store := &recordingStore{inner: newMemoryStore()}
	limiter := newTestRateLimiter(config.RateLimitConfig{
		Enabled:           true,
		RequestsPerMinute: 60,
		BurstSize:         5,
	}, store)

	router := principalRouter(limiter, func(*gin.Context) {})
	assert.Equal(t, http.StatusOK, serve(router, http.MethodGet, "/test", "10.0.0.9:1").Code)

	require.Len(t, store.requests, 1)
	require.Len(t, store.requests[0], 1)
	assert.Equal(t, "anon:10.0.0.9", store.requests[0][0].Key)
}

func TestRateLimiter_ByPublicToken_HashesTokenAndIsolatesBuckets(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	store := &recordingStore{inner: newMemoryStore()}
	limiter := newTestRateLimiter(config.RateLimitConfig{
		Enabled:     true,
		PublicToken: config.RateLimitScopeConfig{RequestsPerMinute: 60, BurstSize: 1},
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
	limiter := newTestRateLimiter(config.RateLimitConfig{
		Enabled:            true,
		RequestsPerMinute:  60,
		BurstSize:          1,
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
	limiter := newTestRateLimiter(config.RateLimitConfig{
		Enabled:           true,
		RequestsPerMinute: 60,
		BurstSize:         10,
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
	limiter := newTestRateLimiter(config.RateLimitConfig{
		Enabled:           true,
		RequestsPerMinute: 60,
		BurstSize:         1,
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
	limiter := newTestRateLimiter(config.RateLimitConfig{
		Enabled:           true,
		RequestsPerMinute: 4,
		BurstSize:         1,
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
