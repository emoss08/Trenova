package sim

import (
	"fmt"
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type rateLimitRule struct {
	Name     string
	Requests int
	Per      time.Duration
}

var (
	rateLevelOne       = rateLimitRule{Name: "Level One", Requests: 100, Per: time.Minute}
	rateLevelTwo       = rateLimitRule{Name: "Level Two", Requests: 5, Per: time.Second}
	rateLevelThree     = rateLimitRule{Name: "Level Three", Requests: 10, Per: time.Second}
	rateLegacyTierOne  = rateLimitRule{Name: "Legacy Tier 1", Requests: 25, Per: time.Second}
	rateLegacyTierTwo  = rateLimitRule{Name: "Legacy Tier 2", Requests: 50, Per: time.Second}
	rateFunctionRuns   = rateLimitRule{Name: "Function runs", Requests: 2, Per: time.Minute}
	ratePerToken       = rateLimitRule{Name: "API token", Requests: 150, Per: time.Second}
	ratePerOrgAllToken = rateLimitRule{Name: "organization", Requests: 200, Per: time.Second}
)

const (
	rateLimitScopeToken    = "token"
	rateLimitScopeOrg      = "organization"
	rateLimitScopeEndpoint = "endpoint"
	retryAfterPrecision    = 5
	tokenEpsilon           = 1e-9
	nanosecondEpsilon      = 1e-3
)

var endpointRateLimits = newRouteTable([]routeEntry[rateLimitRule]{
	{http.MethodGet, "/addresses", rateLevelTwo},
	{http.MethodPost, "/addresses", rateLevelOne},
	{http.MethodGet, "/addresses/{id}", rateLegacyTierOne},
	{http.MethodPatch, "/addresses/{id}", rateLevelOne},
	{http.MethodDelete, "/addresses/{id}", rateLevelOne},
	{http.MethodGet, "/dvirs/stream", rateLevelTwo},
	{http.MethodGet, "/fleet/document-types", rateLevelTwo},
	{http.MethodGet, "/fleet/documents", rateLevelTwo},
	{http.MethodPost, "/fleet/documents", rateLevelOne},
	{http.MethodPost, "/fleet/documents/pdfs", rateLevelOne},
	{http.MethodGet, "/fleet/documents/pdfs/{id}", rateLegacyTierOne},
	{http.MethodGet, "/fleet/documents/{id}", rateLegacyTierOne},
	{http.MethodDelete, "/fleet/documents/{id}", rateLevelOne},
	{http.MethodGet, "/fleet/drivers", rateLevelTwo},
	{http.MethodPost, "/fleet/drivers", rateLevelOne},
	{http.MethodGet, "/fleet/drivers/efficiency", rateLevelTwo},
	{http.MethodGet, "/fleet/drivers/tachograph-activity/history", rateLevelTwo},
	{http.MethodGet, "/fleet/drivers/tachograph-files/history", rateLevelTwo},
	{http.MethodGet, "/fleet/drivers/vehicle-assignments", rateLegacyTierOne},
	{http.MethodGet, "/fleet/drivers/{id}", rateLegacyTierOne},
	{http.MethodPatch, "/fleet/drivers/{id}", rateLevelOne},
	{http.MethodPost, "/fleet/dvirs", rateLevelOne},
	{http.MethodGet, "/fleet/dvirs/history", rateLegacyTierTwo},
	{http.MethodPatch, "/fleet/dvirs/{id}", rateLevelOne},
	{http.MethodGet, "/fleet/equipment", rateLevelTwo},
	{http.MethodGet, "/fleet/equipment/locations", rateLevelTwo},
	{http.MethodGet, "/fleet/equipment/locations/feed", rateLevelTwo},
	{http.MethodGet, "/fleet/equipment/locations/history", rateLevelTwo},
	{http.MethodGet, "/fleet/equipment/stats", rateLegacyTierOne},
	{http.MethodGet, "/fleet/equipment/stats/feed", rateLevelThree},
	{http.MethodGet, "/fleet/equipment/stats/history", rateLevelThree},
	{http.MethodGet, "/fleet/equipment/{id}", rateLevelTwo},
	{http.MethodGet, "/fleet/hos/clocks", rateLegacyTierOne},
	{http.MethodGet, "/fleet/hos/daily-logs", rateLevelTwo},
	{http.MethodGet, "/fleet/hos/logs", rateLevelTwo},
	{http.MethodGet, "/fleet/routes", rateLevelTwo},
	{http.MethodPost, "/fleet/routes", rateLevelOne},
	{http.MethodGet, "/fleet/routes/audit-logs/feed", rateLevelTwo},
	{http.MethodGet, "/fleet/routes/{id}", rateLegacyTierOne},
	{http.MethodPatch, "/fleet/routes/{id}", rateLevelOne},
	{http.MethodDelete, "/fleet/routes/{id}", rateLevelOne},
	{http.MethodGet, "/fleet/trailers", rateLevelTwo},
	{http.MethodPost, "/fleet/trailers", rateLevelOne},
	{http.MethodGet, "/fleet/trailers/{id}", rateLevelTwo},
	{http.MethodPatch, "/fleet/trailers/{id}", rateLevelOne},
	{http.MethodDelete, "/fleet/trailers/{id}", rateLevelOne},
	{http.MethodGet, "/fleet/vehicles", rateLegacyTierOne},
	{http.MethodGet, "/fleet/vehicles/driver-assignments", rateLevelTwo},
	{http.MethodGet, "/fleet/vehicles/locations", rateLegacyTierOne},
	{http.MethodGet, "/fleet/vehicles/locations/feed", rateLegacyTierTwo},
	{http.MethodGet, "/fleet/vehicles/locations/history", rateLegacyTierTwo},
	{http.MethodGet, "/fleet/vehicles/stats", rateLegacyTierTwo},
	{http.MethodGet, "/fleet/vehicles/stats/feed", rateLegacyTierTwo},
	{http.MethodGet, "/fleet/vehicles/stats/history", rateLegacyTierTwo},
	{http.MethodGet, "/fleet/vehicles/tachograph-files/history", rateLevelTwo},
	{http.MethodGet, "/fleet/vehicles/{id}", rateLegacyTierOne},
	{http.MethodPatch, "/fleet/vehicles/{id}", rateLevelOne},
	{http.MethodPost, "/ifta-detail/csv", rateLevelOne},
	{http.MethodGet, "/ifta-detail/csv/{id}", rateLevelTwo},
	{http.MethodGet, "/assets/location-and-speed/stream", rateLevelTwo},
	{http.MethodGet, "/tags", rateLevelTwo},
	{http.MethodPost, "/tags", rateLevelOne},
	{http.MethodGet, "/tags/{id}", rateLevelTwo},
	{http.MethodPatch, "/tags/{id}", rateLevelOne},
	{http.MethodPut, "/tags/{id}", rateLevelOne},
	{http.MethodDelete, "/tags/{id}", rateLevelOne},
	{http.MethodGet, "/v1/fleet/messages", rateLevelOne},
	{http.MethodPost, "/v1/fleet/messages", rateLevelOne},
	{http.MethodGet, "/agent-studio/voice-sessions", rateLevelOne},
	{http.MethodGet, "/agent-studio/voice-sessions/stream", rateLevelOne},
	{http.MethodGet, "/alerts/incidents/stream", rateLevelThree},
	{http.MethodGet, "/assets/inputs/stream", rateLevelThree},
	{http.MethodGet, "/beta/fleet/drivers/efficiency", rateLegacyTierTwo},
	{http.MethodGet, "/coaching/driver-coach-assignments", rateLevelThree},
	{http.MethodPut, "/coaching/driver-coach-assignments", rateLevelThree},
	{http.MethodGet, "/defects/{id}", rateLevelThree},
	{http.MethodGet, "/driver-efficiency/drivers", rateLevelThree},
	{http.MethodPatch, "/driver-trailer-assignments", rateLevelTwo},
	{http.MethodPost, "/driver-trailer-assignments", rateLevelTwo},
	{http.MethodGet, "/dvirs/{id}", rateLevelThree},
	{http.MethodPost, "/fleet/drivers/workflow-assignments", rateLevelThree},
	{http.MethodGet, "/fleet/drivers/workflows", rateLevelThree},
	{http.MethodGet, "/fleet/installer/photo-uploads", rateLevelOne},
	{http.MethodGet, "/fleet/reports/ifta/vehicle", rateLegacyTierOne},
	{http.MethodGet, "/fleet/reports/vehicle/idling", rateLegacyTierOne},
	{http.MethodGet, "/fleet/reports/vehicles/fuel-energy", rateLegacyTierOne},
	{http.MethodGet, "/fleet/trailers/stats", rateLegacyTierOne},
	{http.MethodGet, "/fleet/trailers/stats/feed", rateLegacyTierOne},
	{http.MethodGet, "/fleet/trailers/stats/history", rateLevelThree},
	{http.MethodGet, "/form-submissions/pdf-exports", rateLevelOne},
	{http.MethodPost, "/form-submissions/pdf-exports", rateLevelOne},
	{http.MethodGet, "/form-submissions", rateLevelTwo},
	{http.MethodPost, "/form-submissions", rateLevelOne},
	{http.MethodPatch, "/form-submissions", rateLevelOne},
	{http.MethodGet, "/form-submissions/stream", rateLevelTwo},
	{http.MethodGet, "/form-templates", rateLevelTwo},
	{http.MethodGet, "/functions-storage/files", rateLevelOne},
	{http.MethodGet, "/functions-storage/ls", rateLevelOne},
	{http.MethodGet, "/functions/{name}", rateLevelOne},
	{http.MethodGet, "/functions/{name}/logs", rateLevelOne},
	{http.MethodPost, "/functions/{name}/runs", rateFunctionRuns},
	{http.MethodGet, "/functions/{name}/runs/{correlationId}", rateLevelOne},
	{http.MethodPatch, "/hos/daily-logs/log-meta-data", rateLevelTwo},
	{http.MethodGet, "/hub/capacities", rateLevelThree},
	{http.MethodGet, "/hub/customProperties", rateLevelThree},
	{http.MethodGet, "/hub/locations", rateLevelThree},
	{http.MethodPost, "/hub/plan", rateLevelTwo},
	{http.MethodGet, "/hub/plan/orders", rateLevelThree},
	{http.MethodPost, "/hub/plan/orders", rateLevelTwo},
	{http.MethodDelete, "/hub/plan/orders", rateLevelTwo},
	{http.MethodGet, "/hub/plan/routes", rateLevelThree},
	{http.MethodGet, "/hub/plans", rateLevelThree},
	{http.MethodGet, "/hub/route-templates", rateLevelThree},
	{http.MethodPost, "/hub/route-templates", rateLevelTwo},
	{http.MethodPatch, "/hub/route-templates", rateLevelTwo},
	{http.MethodDelete, "/hub/route-templates", rateLevelTwo},
	{http.MethodGet, "/hub/skills", rateLevelThree},
	{http.MethodGet, "/hubs", rateLevelThree},
	{http.MethodPost, "/maintenance/preventive/resolve", rateLevelTwo},
	{http.MethodPatch, "/maintenance/preventive/upcoming", rateLevelTwo},
	{http.MethodPost, "/maintenance/purchase-orders", rateLevelTwo},
	{http.MethodPatch, "/maintenance/purchase-orders", rateLevelTwo},
	{http.MethodDelete, "/maintenance/purchase-orders", rateLevelTwo},
	{http.MethodPost, "/maintenance/warranties", rateLevelTwo},
	{http.MethodPatch, "/maintenance/warranties", rateLevelTwo},
	{http.MethodDelete, "/maintenance/warranties", rateLevelTwo},
	{http.MethodPost, "/maintenance/warranties/assets/replace", rateLevelTwo},
	{http.MethodPost, "/maintenance/warranty-claims", rateLevelTwo},
	{http.MethodPatch, "/maintenance/warranty-claims", rateLevelTwo},
	{http.MethodDelete, "/maintenance/warranty-claims", rateLevelTwo},
	{http.MethodGet, "/preferred-stations", rateLevelOne},
	{http.MethodGet, "/preferred-stations/{id}", rateLevelOne},
	{http.MethodPost, "/readings", rateLevelThree},
	{http.MethodGet, "/readings/definitions", rateLevelThree},
	{http.MethodGet, "/readings/history", rateLevelThree},
	{http.MethodGet, "/readings/latest", rateLevelThree},
	{http.MethodGet, "/ridership/passengers/{id}", rateLevelThree},
	{http.MethodGet, "/ridership/route-setups/{routeId}", rateLevelThree},
	{http.MethodPatch, "/safety-events/batch", rateLevelTwo},
	{http.MethodGet, "/safety-scores/drivers", rateLevelOne},
	{http.MethodGet, "/safety-scores/tag-group", rateLevelOne},
	{http.MethodGet, "/safety-scores/tags", rateLevelOne},
	{http.MethodGet, "/safety-scores/vehicles", rateLevelOne},
	{http.MethodPost, "/training-assignments", rateLevelThree},
	{http.MethodPatch, "/training-assignments", rateLevelThree},
	{http.MethodDelete, "/training-assignments", rateLevelThree},
	{http.MethodGet, "/v1/fleet/assets/{assetId}/locations", rateLegacyTierOne},
	{http.MethodGet, "/v1/fleet/assets/{assetId}/reefer", rateLegacyTierOne},
	{http.MethodGet, "/v1/fleet/locations", rateLegacyTierTwo},
	{http.MethodGet, "/v1/fleet/trailers/assignments", rateLegacyTierOne},
	{http.MethodGet, "/v1/fleet/trailers/{trailerId}/assignments", rateLegacyTierOne},
})

func endpointRateLimitRule(key, method string) rateLimitRule {
	if rule, ok := endpointRateLimits.lookup(key); ok {
		return rule
	}
	if isSafeHTTPMethod(method) {
		return rateLevelTwo
	}
	return rateLevelOne
}

type tokenBucket struct {
	tokens   float64
	capacity float64
	rate     float64
	updated  time.Time
}

func newTokenBucket(rule rateLimitRule, multiplier float64, now time.Time) *tokenBucket {
	capacity := math.Max(1, float64(rule.Requests)*multiplier)
	return &tokenBucket{
		tokens:   capacity,
		capacity: capacity,
		rate:     capacity / rule.Per.Seconds(),
		updated:  now,
	}
}

func (b *tokenBucket) refill(now time.Time) {
	elapsed := now.Sub(b.updated)
	if elapsed <= 0 {
		return
	}
	b.tokens = math.Min(b.capacity, b.tokens+elapsed.Seconds()*b.rate)
	b.updated = now
}

func (b *tokenBucket) wait() time.Duration {
	if b.tokens >= 1-tokenEpsilon {
		return 0
	}
	nanos := (1 - b.tokens) / b.rate * float64(time.Second)
	return time.Duration(math.Ceil(nanos - nanosecondEpsilon))
}

type rateLimitDecision struct {
	Allowed    bool
	RetryAfter time.Duration
	Scope      string
	Rule       rateLimitRule
}

type rateLimiter struct {
	mu         sync.Mutex
	multiplier float64
	now        func() time.Time
	org        *tokenBucket
	tokens     map[string]*tokenBucket
	endpoints  map[string]*tokenBucket
}

func newRateLimiter(multiplier float64, now func() time.Time) *rateLimiter {
	if multiplier <= 0 || math.IsNaN(multiplier) || math.IsInf(multiplier, 0) {
		multiplier = 1
	}
	if now == nil {
		now = time.Now
	}
	return &rateLimiter{
		multiplier: multiplier,
		now:        now,
		tokens:     map[string]*tokenBucket{},
		endpoints:  map[string]*tokenBucket{},
	}
}

func (l *rateLimiter) allow(
	token, endpointKey string,
	endpointRule rateLimitRule,
) rateLimitDecision {
	now := l.now()

	l.mu.Lock()
	defer l.mu.Unlock()

	if l.org == nil {
		l.org = newTokenBucket(ratePerOrgAllToken, l.multiplier, now)
	}
	perToken := l.tokens[token]
	if perToken == nil {
		perToken = newTokenBucket(ratePerToken, l.multiplier, now)
		l.tokens[token] = perToken
	}
	var endpointBucket *tokenBucket
	if endpointKey != "" {
		endpointBucket = l.endpoints[endpointKey]
		if endpointBucket == nil {
			endpointBucket = newTokenBucket(endpointRule, l.multiplier, now)
			l.endpoints[endpointKey] = endpointBucket
		}
	}

	decision := rateLimitDecision{Allowed: true}
	l.consider(&decision, perToken, now, rateLimitScopeToken, ratePerToken)
	l.consider(&decision, l.org, now, rateLimitScopeOrg, ratePerOrgAllToken)
	if endpointBucket != nil {
		l.consider(&decision, endpointBucket, now, rateLimitScopeEndpoint, endpointRule)
	}
	if !decision.Allowed {
		return decision
	}

	perToken.tokens--
	l.org.tokens--
	if endpointBucket != nil {
		endpointBucket.tokens--
	}
	return decision
}

func (l *rateLimiter) consider(
	decision *rateLimitDecision,
	bucket *tokenBucket,
	now time.Time,
	scope string,
	rule rateLimitRule,
) {
	bucket.refill(now)
	wait := bucket.wait()
	if wait <= 0 || wait <= decision.RetryAfter {
		return
	}
	decision.Allowed = false
	decision.RetryAfter = wait
	decision.Scope = scope
	decision.Rule = rule
}

func (l *rateLimiter) effectiveRequests(rule rateLimitRule) float64 {
	return math.Max(1, float64(rule.Requests)*l.multiplier)
}

func (l *rateLimiter) exceededError(
	decision *rateLimitDecision,
	method, routePattern string,
) error {
	requests := strconv.FormatFloat(l.effectiveRequests(decision.Rule), 'f', -1, 64)
	unit := "sec"
	if decision.Rule.Per >= time.Minute {
		unit = "min"
	}
	switch decision.Scope {
	case rateLimitScopeToken:
		return fmt.Errorf(
			"%w: each API token is limited to %s requests/%s",
			ErrRateLimitExceeded,
			requests,
			unit,
		)
	case rateLimitScopeOrg:
		return fmt.Errorf(
			"%w: the organization is limited to %s requests/%s across all API tokens",
			ErrRateLimitExceeded,
			requests,
			unit,
		)
	default:
		return fmt.Errorf(
			"%w: %s %s is limited to %s requests/%s per organization (%s)",
			ErrRateLimitExceeded,
			method,
			routePattern,
			requests,
			unit,
			decision.Rule.Name,
		)
	}
}

func formatRetryAfter(wait time.Duration) string {
	scale := math.Pow10(retryAfterPrecision)
	seconds := math.Max(math.Ceil(wait.Seconds()*scale)/scale, 1/scale)
	return strconv.FormatFloat(seconds, 'f', retryAfterPrecision, 64)
}
