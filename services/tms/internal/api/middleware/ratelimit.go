package middleware

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config/managedconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/tenantlimitsconfig"
	"github.com/emoss08/trenova/internal/infrastructure/observability/platformmetrics"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/infrastructure/config/securityconfig"
	"github.com/emoss08/trenova/internal/infrastructure/observability/httpmetrics"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/internal/infrastructure/ratelimit"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/hashutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	HeaderRateLimitLimit     = "X-RateLimit-Limit"
	HeaderRateLimitRemaining = "X-RateLimit-Remaining"
	HeaderRateLimitReset     = "X-RateLimit-Reset"
	HeaderRateLimitPolicy    = "X-RateLimit-Policy"
	HeaderRateLimitScope     = "X-RateLimit-Scope"
	HeaderRetryAfter         = "Retry-After"

	rateLimitPeriod = time.Minute

	RateLimitScopeInstance = securityconfig.RateLimitScope("instance")
	instanceRequestsKey    = "instance:requests"
)

var tenancyScopes = map[securityconfig.RateLimitScope]string{ //nolint:exhaustive // only the buckets the tenancy metrics count
	securityconfig.RateLimitScopeTenant: platformmetrics.LimitScopeOrganization,
	RateLimitScopeInstance:              platformmetrics.LimitScopeInstance,
}

type RateLimiterParams struct {
	fx.In

	SecurityConfig *securityconfig.Config
	Store          kernelports.RateLimitStore
	ErrorHandler   *helpers.ErrorHandler
	Logger         *zap.Logger
	Metrics        *metrics.Registry
	Managed        *managedconfig.Config
	TenantLimits   *tenantlimitsconfig.Config
}

type RateLimiter struct {
	enabled        bool
	everyPrincipal bool
	tenancy        *platformmetrics.Tenancy
	store          kernelports.RateLimitStore
	errorHandler   *helpers.ErrorHandler
	logger         *zap.Logger
	metrics        *httpmetrics.RateLimit
	policies       map[securityconfig.RateLimitScope]kernelports.RateLimitPolicy
	exemptPaths    []string
}

type scopedRequest struct {
	scope   securityconfig.RateLimitScope
	request kernelports.RateLimitRequest
}

func NewRateLimiter(p RateLimiterParams) *RateLimiter {
	cfg := securityconfig.RateLimitConfig{}
	cfg = p.SecurityConfig.RateLimit

	logger := p.Logger

	var rateLimitMetrics *httpmetrics.RateLimit
	rateLimitMetrics = p.Metrics.RateLimit

	scopes := []securityconfig.RateLimitScope{
		securityconfig.RateLimitScopeAnonymous,
		securityconfig.RateLimitScopeAPIKey,
		securityconfig.RateLimitScopeTenant,
		securityconfig.RateLimitScopePublicToken,
		securityconfig.RateLimitScopeSCIM,
	}
	policies := make(map[securityconfig.RateLimitScope]kernelports.RateLimitPolicy, len(scopes))
	for _, scope := range scopes {
		policy := cfg.ScopePolicy(scope)
		if !policy.Enabled {
			continue
		}
		policies[scope] = kernelports.RateLimitPolicy{
			Rate:   policy.RequestsPerMinute,
			Period: rateLimitPeriod,
			Burst:  policy.BurstSize,
		}
	}

	everyPrincipal := p.TenantLimits.Requests.Organizations.On(p.Managed.SharedTenancy())
	if everyPrincipal {
		rate, burst := p.TenantLimits.InstanceRequests()
		policies[RateLimitScopeInstance] = kernelports.RateLimitPolicy{
			Rate:   rate,
			Period: rateLimitPeriod,
			Burst:  burst,
		}
	}

	exempt := make([]string, 0, len(cfg.ExemptPathPrefixes))
	for _, prefix := range cfg.ExemptPathPrefixes {
		if trimmed := strings.TrimSpace(prefix); trimmed != "" {
			exempt = append(exempt, trimmed)
		}
	}

	return &RateLimiter{
		enabled:        cfg.Enabled,
		everyPrincipal: everyPrincipal,
		tenancy:        p.Metrics.Tenancy,
		store:          p.Store,
		errorHandler:   p.ErrorHandler,
		logger:         logger.Named("ratelimit"),
		metrics:        rateLimitMetrics,
		policies:       policies,
		exemptPaths:    exempt,
	}
}

func (m *RateLimiter) ByClientIP() gin.HandlerFunc {
	return func(c *gin.Context) {
		if m.skip(c) {
			c.Next()
			return
		}

		if m.enforce(c, m.anonymousRequests(c)) {
			c.Next()
		}
	}
}

func (m *RateLimiter) ByPrincipal() gin.HandlerFunc {
	return func(c *gin.Context) {
		if m.skip(c) {
			c.Next()
			return
		}

		if m.enforce(c, m.principalRequests(c)) {
			c.Next()
		}
	}
}

func (m *RateLimiter) ByPublicToken(param string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if m.skip(c) {
			c.Next()
			return
		}

		if m.enforce(c, m.publicTokenRequests(c, param)) {
			c.Next()
		}
	}
}

func (m *RateLimiter) Allow(
	c *gin.Context,
	scope securityconfig.RateLimitScope,
	key string,
) bool {
	if m.skip(c) {
		return true
	}
	policy, ok := m.policies[scope]
	if !ok {
		return true
	}

	return m.enforce(c, []scopedRequest{{
		scope: scope,
		request: kernelports.RateLimitRequest{
			Key:    string(scope) + ":" + key,
			Policy: policy,
			Cost:   1,
		},
	}})
}

func (m *RateLimiter) AllowClientIP(c *gin.Context) bool {
	if m.skip(c) {
		return true
	}

	return m.enforce(c, m.anonymousRequests(c))
}

func (m *RateLimiter) skip(c *gin.Context) bool {
	if !m.enabled {
		return true
	}
	path := c.Request.URL.Path
	for _, prefix := range m.exemptPaths {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

func (m *RateLimiter) anonymousRequests(c *gin.Context) []scopedRequest {
	policy, ok := m.policies[securityconfig.RateLimitScopeAnonymous]
	if !ok {
		return nil
	}
	return []scopedRequest{{
		scope: securityconfig.RateLimitScopeAnonymous,
		request: kernelports.RateLimitRequest{
			Key:    "anon:" + c.ClientIP(),
			Policy: policy,
			Cost:   1,
		},
	}}
}

func (m *RateLimiter) principalRequests(c *gin.Context) []scopedRequest {
	authCtx := authctx.GetAuthContext(c)
	if authCtx == nil || authCtx.OrganizationID.IsNil() {
		return nil
	}
	apiKey := authCtx.IsAPIKey() && authCtx.APIKeyID.IsNotNil()
	if !apiKey && !m.everyPrincipal {
		return nil
	}

	orgTag := "{" + authCtx.OrganizationID.String() + "}"
	requests := make([]scopedRequest, 0, 3)
	if policy, ok := m.policies[securityconfig.RateLimitScopeAPIKey]; ok && apiKey {
		requests = append(requests, scopedRequest{
			scope: securityconfig.RateLimitScopeAPIKey,
			request: kernelports.RateLimitRequest{
				Key:    orgTag + ":apikey:" + authCtx.APIKeyID.String(),
				Policy: policy,
				Cost:   1,
			},
		})
	}

	if policy, ok := m.policies[securityconfig.RateLimitScopeTenant]; ok {
		requests = append(requests, scopedRequest{
			scope: securityconfig.RateLimitScopeTenant,
			request: kernelports.RateLimitRequest{
				Key:    orgTag + ":tenant",
				Policy: policy,
				Cost:   1,
			},
		})
	}

	if policy, ok := m.policies[RateLimitScopeInstance]; ok {
		requests = append(requests, scopedRequest{
			scope: RateLimitScopeInstance,
			request: kernelports.RateLimitRequest{
				Key:    instanceRequestsKey,
				Policy: policy,
				Cost:   1,
			},
		})
	}

	return requests
}

func (m *RateLimiter) publicTokenRequests(c *gin.Context, param string) []scopedRequest {
	policy, ok := m.policies[securityconfig.RateLimitScopePublicToken]
	if !ok {
		return nil
	}
	token := c.Param(param)
	if token == "" {
		return m.anonymousRequests(c)
	}
	return []scopedRequest{{
		scope: securityconfig.RateLimitScopePublicToken,
		request: kernelports.RateLimitRequest{
			Key:    "token:" + hashutils.SHA256Hex(token),
			Policy: policy,
			Cost:   1,
		},
	}}
}

func (m *RateLimiter) enforce(c *gin.Context, requests []scopedRequest) bool {
	if len(requests) == 0 {
		return true
	}

	storeRequests := make([]kernelports.RateLimitRequest, len(requests))
	for i := range requests {
		storeRequests[i] = requests[i].request
	}

	decisions, err := m.store.Check(c.Request.Context(), storeRequests)
	if err != nil {
		return m.handleStoreError(c, err)
	}
	if len(decisions) != len(requests) {
		m.logger.Error(
			"rate limit store returned a mismatched decision count",
			zap.Int("expected", len(requests)),
			zap.Int("actual", len(decisions)),
		)
		return true
	}

	tightest := -1
	denied := -1
	for i := range decisions {
		m.recordDecision(requests[i].scope, decisions[i].Allowed)
		if !decisions[i].Allowed {
			if denied < 0 || decisions[i].RetryAfter > decisions[denied].RetryAfter {
				denied = i
			}
			continue
		}
		if tightest < 0 || decisions[i].Remaining < decisions[tightest].Remaining {
			tightest = i
		}
	}

	if denied >= 0 {
		m.reject(c, requests[denied], decisions[denied])
		return false
	}

	m.writeHeaders(c, requests[tightest], decisions[tightest])
	return true
}

func (m *RateLimiter) handleStoreError(c *gin.Context, err error) bool {
	if errors.Is(err, ratelimit.ErrStoreDenied) {
		c.Header(HeaderRetryAfter, "1")
		m.errorHandler.HandleError(
			c,
			errortypes.NewRateLimitError("request", "Too many requests"),
		)
		return false
	}

	if c.Request.Context().Err() != nil {
		return true
	}

	m.logger.Error("rate limit check failed; allowing request", zap.Error(err))
	return true
}

func (m *RateLimiter) reject(
	c *gin.Context,
	req scopedRequest,
	decision kernelports.RateLimitDecision,
) {
	m.writeHeaders(c, req, decision)
	c.Header(HeaderRateLimitScope, string(req.scope))
	if scope, ok := tenancyScopes[req.scope]; ok {
		m.tenancy.RecordLimitRefusal("requests", scope)
	}
	c.Header(HeaderRetryAfter, strconv.Itoa(timeutils.CeilSeconds(decision.RetryAfter)))
	m.errorHandler.HandleError(
		c,
		errortypes.NewRateLimitError("request", "Too many requests"),
	)
}

func (m *RateLimiter) writeHeaders(
	c *gin.Context,
	req scopedRequest,
	decision kernelports.RateLimitDecision,
) {
	c.Header(HeaderRateLimitLimit, strconv.Itoa(decision.Limit))
	c.Header(HeaderRateLimitRemaining, strconv.Itoa(decision.Remaining))
	c.Header(HeaderRateLimitReset, strconv.Itoa(timeutils.CeilSeconds(decision.ResetAfter)))
	c.Header(HeaderRateLimitPolicy, formatRateLimitPolicy(req.request.Policy))
}

func (m *RateLimiter) recordDecision(scope securityconfig.RateLimitScope, allowed bool) {
	if m.metrics == nil {
		return
	}
	m.metrics.RecordDecision(string(scope), allowed)
}

func formatRateLimitPolicy(policy kernelports.RateLimitPolicy) string {
	return strconv.Itoa(policy.Rate) +
		";w=" + strconv.Itoa(int(policy.Period/time.Second)) +
		";burst=" + strconv.Itoa(policy.Burst)
}
