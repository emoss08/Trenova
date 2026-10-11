package sim

import (
	"net/http"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/emoss08/trenova/samsara-sim/internal/config"
)

var retryAfterPattern = regexp.MustCompile(`^[0-9]+\.[0-9]{5}$`)

type fakeWallClock struct {
	mu  sync.Mutex
	now time.Time
}

func newFakeWallClock() *fakeWallClock {
	return &fakeWallClock{now: time.Date(2026, time.May, 1, 12, 0, 0, 0, time.UTC)}
}

func (c *fakeWallClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.now
}

func (c *fakeWallClock) Advance(duration time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.now = c.now.Add(duration)
}

func TestEndpointRateLimitLevels(t *testing.T) {
	t.Parallel()

	tests := []struct {
		method string
		path   string
		want   rateLimitRule
	}{
		{http.MethodGet, "/addresses", rateLevelTwo},
		{http.MethodPost, "/addresses", rateLevelOne},
		{http.MethodGet, "/addresses/{addressId}", rateLegacyTierOne},
		{http.MethodPatch, "/addresses/{id}", rateLevelOne},
		{http.MethodDelete, "/addresses/{id}", rateLevelOne},
		{http.MethodGet, "/fleet/vehicles", rateLegacyTierOne},
		{http.MethodGet, "/fleet/vehicles/stats", rateLegacyTierTwo},
		{http.MethodGet, "/fleet/vehicles/stats/feed", rateLegacyTierTwo},
		{http.MethodGet, "/fleet/vehicles/{id}", rateLegacyTierOne},
		{http.MethodGet, "/fleet/vehicles/locations", rateLegacyTierOne},
		{http.MethodGet, "/fleet/dvirs/history", rateLegacyTierTwo},
		{http.MethodGet, "/fleet/equipment/stats", rateLegacyTierOne},
		{http.MethodGet, "/fleet/equipment/stats/history", rateLevelThree},
		{http.MethodGet, "/fleet/hos/clocks", rateLegacyTierOne},
		{http.MethodGet, "/fleet/hos/logs", rateLevelTwo},
		{http.MethodGet, "/fleet/routes/{id}", rateLegacyTierOne},
		{http.MethodDelete, "/fleet/routes/{id}", rateLevelOne},
		{http.MethodGet, "/assets/location-and-speed/stream", rateLevelTwo},
		{http.MethodPut, "/tags/{id}", rateLevelOne},
		{http.MethodGet, "/v1/fleet/messages", rateLevelOne},
		{http.MethodPost, "/v1/fleet/messages", rateLevelOne},
		{http.MethodGet, "/readings/latest", rateLevelThree},
		{http.MethodPost, "/functions/{name}/runs", rateFunctionRuns},
		{http.MethodGet, "/form-templates", rateLevelTwo},
		{http.MethodGet, "/some/undocumented/read", rateLevelTwo},
		{http.MethodPost, "/some/undocumented/write", rateLevelOne},
		{http.MethodPatch, "/webhooks/{id}", rateLevelOne},
	}
	for _, testCase := range tests {
		key := routeKey(testCase.method, testCase.path)
		if got := endpointRateLimitRule(key, testCase.method); got != testCase.want {
			t.Fatalf(
				"expected %s %s to be %s, got %s",
				testCase.method,
				testCase.path,
				testCase.want.Name,
				got.Name,
			)
		}
	}
}

func TestEndpointRateLimitTableHasNoConflictingDuplicates(t *testing.T) {
	t.Parallel()

	if len(endpointRateLimits.byKey) < 140 {
		t.Fatalf(
			"expected the documented endpoint table, got %d entries",
			len(endpointRateLimits.byKey),
		)
	}
	if routeKey(http.MethodGet, "/v1/fleet/assets/{assetId}/reefer") !=
		routeKey(http.MethodGet, "/v1/fleet/assets/{asset_id}/reefer") {
		t.Fatal("expected path parameter names to be ignored when matching")
	}
	if patternRouteKey("GET /fleet/routes/{id}", http.MethodHead) !=
		routeKey(http.MethodGet, "/fleet/routes/{routeId}") {
		t.Fatal("expected mux patterns to normalize to table keys")
	}
}

func TestRateLimiterLevelTwoBurstAndDecimalRetryAfter(t *testing.T) {
	t.Parallel()

	clock := newFakeWallClock()
	limiter := newRateLimiter(1, clock.Now)
	key := routeKey(http.MethodGet, "/fleet/drivers")
	for idx := 0; idx < 5; idx++ {
		if decision := limiter.allow("token", key, rateLevelTwo); !decision.Allowed {
			t.Fatalf("expected request %d to be allowed", idx+1)
		}
	}
	denied := limiter.allow("token", key, rateLevelTwo)
	if denied.Allowed {
		t.Fatal("expected the sixth request within a second to be limited")
	}
	if denied.Scope != rateLimitScopeEndpoint || denied.RetryAfter != 200*time.Millisecond {
		t.Fatalf(
			"expected endpoint limit with 200ms wait, got %s %s",
			denied.Scope,
			denied.RetryAfter,
		)
	}
	if got := formatRetryAfter(denied.RetryAfter); got != "0.20000" {
		t.Fatalf("expected Retry-After 0.20000, got %s", got)
	}

	clock.Advance(80 * time.Millisecond)
	partial := limiter.allow("token", key, rateLevelTwo)
	if partial.Allowed || partial.RetryAfter != 120*time.Millisecond {
		t.Fatalf("expected 120ms remaining wait, got %v %s", partial.Allowed, partial.RetryAfter)
	}
	if got := formatRetryAfter(partial.RetryAfter); got != "0.12000" {
		t.Fatalf("expected Retry-After 0.12000, got %s", got)
	}

	clock.Advance(120 * time.Millisecond)
	if !limiter.allow("token", key, rateLevelTwo).Allowed {
		t.Fatal("expected a token to refill after 200ms")
	}
	if limiter.allow("token", key, rateLevelTwo).Allowed {
		t.Fatal("expected only one refilled token")
	}
}

func TestRateLimiterLevelOneIsPerMinute(t *testing.T) {
	t.Parallel()

	clock := newFakeWallClock()
	limiter := newRateLimiter(1, clock.Now)
	key := routeKey(http.MethodPost, "/addresses")
	for idx := 0; idx < 100; idx++ {
		if !limiter.allow("token", key, rateLevelOne).Allowed {
			t.Fatalf("expected write %d to be allowed", idx+1)
		}
		clock.Advance(5 * time.Millisecond)
	}
	denied := limiter.allow("token", key, rateLevelOne)
	if denied.Allowed {
		t.Fatal("expected the 101st write within a minute to be limited")
	}
	if denied.RetryAfter <= 0 || denied.RetryAfter > 600*time.Millisecond {
		t.Fatalf("expected a sub-second refill wait at 100/min, got %s", denied.RetryAfter)
	}
	if !retryAfterPattern.MatchString(formatRetryAfter(denied.RetryAfter)) {
		t.Fatalf("expected decimal Retry-After, got %s", formatRetryAfter(denied.RetryAfter))
	}
}

func TestRateLimiterPerTokenAndPerOrganization(t *testing.T) {
	t.Parallel()

	clock := newFakeWallClock()
	limiter := newRateLimiter(1, clock.Now)
	for idx := 0; idx < 150; idx++ {
		if !limiter.allow("token-a", "", rateLimitRule{}).Allowed {
			t.Fatalf("expected token-a request %d to be allowed", idx+1)
		}
	}
	tokenDenied := limiter.allow("token-a", "", rateLimitRule{})
	if tokenDenied.Allowed || tokenDenied.Scope != rateLimitScopeToken {
		t.Fatalf("expected the per-token limit, got %+v", tokenDenied)
	}

	for idx := 0; idx < 50; idx++ {
		if !limiter.allow("token-b", "", rateLimitRule{}).Allowed {
			t.Fatalf("expected token-b request %d to fit the organization budget", idx+1)
		}
	}
	orgDenied := limiter.allow("token-b", "", rateLimitRule{})
	if orgDenied.Allowed || orgDenied.Scope != rateLimitScopeOrg {
		t.Fatalf("expected the shared organization limit, got %+v", orgDenied)
	}
	if orgDenied.RetryAfter != 5*time.Millisecond {
		t.Fatalf("expected a 5ms organization refill wait, got %s", orgDenied.RetryAfter)
	}

	clock.Advance(time.Second)
	if !limiter.allow("token-b", "", rateLimitRule{}).Allowed {
		t.Fatal("expected the organization budget to refill")
	}
}

func TestRateLimiterEndpointLimitIsSharedAcrossTokens(t *testing.T) {
	t.Parallel()

	clock := newFakeWallClock()
	limiter := newRateLimiter(1, clock.Now)
	key := routeKey(http.MethodGet, "/fleet/routes")
	for idx := 0; idx < 5; idx++ {
		if !limiter.allow("token-a", key, rateLevelTwo).Allowed {
			t.Fatalf("expected request %d to be allowed", idx+1)
		}
	}
	if limiter.allow("token-b", key, rateLevelTwo).Allowed {
		t.Fatal("expected the endpoint budget to be per organization, not per token")
	}
	other := routeKey(http.MethodGet, "/fleet/drivers")
	if !limiter.allow("token-b", other, rateLevelTwo).Allowed {
		t.Fatal("expected another endpoint to keep its own budget")
	}
}

func TestRateLimiterDeniedRequestsConsumeNothing(t *testing.T) {
	t.Parallel()

	clock := newFakeWallClock()
	limiter := newRateLimiter(1, clock.Now)
	key := routeKey(http.MethodGet, "/fleet/routes")
	for idx := 0; idx < 5; idx++ {
		limiter.allow("token", key, rateLevelTwo)
	}
	for idx := 0; idx < 100; idx++ {
		if limiter.allow("token", key, rateLevelTwo).Allowed {
			t.Fatal("expected the endpoint to stay limited")
		}
	}
	for idx := 0; idx < 145; idx++ {
		if !limiter.allow("token", "", rateLimitRule{}).Allowed {
			t.Fatalf("expected denied requests not to drain the token budget (%d)", idx)
		}
	}
}

func TestRateLimiterMultiplierScalesBudgets(t *testing.T) {
	t.Parallel()

	clock := newFakeWallClock()
	limiter := newRateLimiter(4, clock.Now)
	key := routeKey(http.MethodGet, "/fleet/routes")
	for idx := 0; idx < 20; idx++ {
		if !limiter.allow("token", key, rateLevelTwo).Allowed {
			t.Fatalf("expected request %d within the 4x budget", idx+1)
		}
	}
	denied := limiter.allow("token", key, rateLevelTwo)
	if denied.Allowed || denied.RetryAfter != 50*time.Millisecond {
		t.Fatalf("expected 4x refill rate, got %+v", denied)
	}
	if err := limiter.exceededError(&denied, http.MethodGet, "/fleet/routes"); !strings.Contains(
		err.Error(),
		"20 requests/sec",
	) {
		t.Fatalf("expected the effective budget in the message, got %q", err.Error())
	}

	slow := newRateLimiter(0.001, clock.Now)
	runsKey := routeKey(http.MethodPost, "/functions/{name}/runs")
	if !slow.allow("token", runsKey, rateFunctionRuns).Allowed {
		t.Fatal("expected a minimum burst of one request")
	}
}

func TestRateLimiterConcurrentAccessIsExact(t *testing.T) {
	t.Parallel()

	clock := newFakeWallClock()
	limiter := newRateLimiter(1, clock.Now)
	var allowed atomic.Int64
	var group sync.WaitGroup
	for worker := 0; worker < 16; worker++ {
		group.Add(1)
		go func() {
			defer group.Done()
			for idx := 0; idx < 25; idx++ {
				if limiter.allow("token", "", rateLimitRule{}).Allowed {
					allowed.Add(1)
				}
			}
		}()
	}
	group.Wait()
	if allowed.Load() != 150 {
		t.Fatalf("expected exactly 150 allowed requests, got %d", allowed.Load())
	}
}

func TestServerRateLimitReturns429WithDecimalRetryAfter(t *testing.T) {
	t.Parallel()

	srv := newEventTestServer(t, "")
	clock := newFakeWallClock()
	srv.rateLimiter = newRateLimiter(1, clock.Now)

	for idx := 0; idx < 5; idx++ {
		response := performAuthorizedRequest(srv, http.MethodGet, "/fleet/routes")
		if response.Code != http.StatusOK {
			t.Fatalf("expected request %d to succeed, got %d", idx+1, response.Code)
		}
		if response.Header().Get("X-RateLimit-Limit") != "" {
			t.Fatal("expected no undocumented X-RateLimit headers")
		}
	}

	response := performAuthorizedRequest(srv, http.MethodGet, "/fleet/routes")
	if response.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 on the sixth Level Two request, got %d", response.Code)
	}
	if got := response.Header().Get("Retry-After"); got != "0.20000" {
		t.Fatalf("expected Retry-After 0.20000, got %q", got)
	}
	for _, header := range []string{"X-RateLimit-Limit", "X-RateLimit-Remaining", "X-RateLimit-Reset"} {
		if response.Header().Get(header) != "" {
			t.Fatalf("expected %s to be absent", header)
		}
	}
	message := mustReadAPIError(t, response.Body.Bytes())
	if !strings.HasPrefix(message, "Exceeded rate limit") ||
		!strings.Contains(message, "GET /fleet/routes") ||
		!strings.Contains(message, "Level Two") {
		t.Fatalf("expected an informative rate-limit message, got %q", message)
	}

	readOnly := performRequestWithToken(
		srv,
		http.MethodGet,
		"/fleet/routes",
		"dev-samsara-token-readonly",
		nil,
	)
	if readOnly.Code != http.StatusTooManyRequests {
		t.Fatalf("expected the endpoint limit to apply across tokens, got %d", readOnly.Code)
	}

	for idx := 0; idx < 40; idx++ {
		control := performAuthorizedRequest(srv, http.MethodGet, "/_sim/time")
		if control.Code != http.StatusOK {
			t.Fatalf("expected control plane to stay unlimited, got %d", control.Code)
		}
	}

	clock.Advance(200 * time.Millisecond)
	if retry := performAuthorizedRequest(
		srv,
		http.MethodGet,
		"/fleet/routes",
	); retry.Code != http.StatusOK {
		t.Fatalf("expected the request to succeed after Retry-After, got %d", retry.Code)
	}
}

func TestServerRateLimitCanBeDisabled(t *testing.T) {
	t.Parallel()

	srv := newEventTestServer(t, "")
	if srv.rateLimiter != nil {
		t.Fatal("expected the test server to have rate limiting disabled")
	}
	for idx := 0; idx < 20; idx++ {
		if response := performAuthorizedRequest(
			srv,
			http.MethodGet,
			"/fleet/routes",
		); response.Code != http.StatusOK {
			t.Fatalf("expected no limiting when disabled, got %d", response.Code)
		}
	}

	enabled := newDefaultFixtureServer(t, func(cfg *config.Config) {
		cfg.RateLimits.Enabled = true
	})
	if enabled.rateLimiter == nil {
		t.Fatal("expected rateLimits.enabled to install the limiter")
	}
}
