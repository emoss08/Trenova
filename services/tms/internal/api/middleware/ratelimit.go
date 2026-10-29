package middleware

import (
	"errors"
	"strconv"
	"strings"
	"time"

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
)

type RateLimiterParams struct {
	fx.In

	SecurityConfig *securityconfig.Config
	Store          kernelports.RateLimitStore
	ErrorHandler   *helpers.ErrorHandler
	Logger         *zap.Logger
	Metrics        *metrics.Registry `optional:"true"`
}

type RateLimiter struct {
	enabled      bool
	store        kernelports.RateLimitStore
	errorHandler *helpers.ErrorHandler
	logger       *zap.Logger
	metrics      *httpmetrics.RateLimit
	policies     map[securityconfig.RateLimitScope]kernelports.RateLimitPolicy
	exemptPaths  []string
}

type scopedRequest struct {
	scope   securityconfig.RateLimitScope
	request kernelports.RateLimitRequest
}

func NewRateLimiter(p RateLimiterParams) *RateLimiter {
	cfg := securityconfig.RateLimitConfig{}
	if p.SecurityConfig != nil {
		cfg = p.SecurityConfig.RateLimit
	}

	logger := p.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	var rateLimitMetrics *httpmetrics.RateLimit
	if p.Metrics != nil {
		rateLimitMetrics = p.Metrics.RateLimit
	}

	scopes := []securityconfig.RateLimitScope{
		securityconfig.RateLimitScopeAnonymous,
		securityconfig.RateLimitScopeAPIKey,
		securityconfig.RateLimitScopeTenant,
		securityconfig.RateLimitScopePublicToken,
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

	exempt := make([]string, 0, len(cfg.ExemptPathPrefixes))
	for _, prefix := range cfg.ExemptPathPrefixes {
		if trimmed := strings.TrimSpace(prefix); trimmed != "" {
			exempt = append(exempt, trimmed)
		}
	}

	return &RateLimiter{
		enabled:      cfg.Enabled && p.Store != nil,
		store:        p.Store,
		errorHandler: p.ErrorHandler,
		logger:       logger.Named("ratelimit"),
		metrics:      rateLimitMetrics,
		policies:     policies,
		exemptPaths:  exempt,
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

func (m *RateLimiter) ByAPIKey() gin.HandlerFunc {
	return func(c *gin.Context) {
		if m.skip(c) {
			c.Next()
			return
		}

		if m.enforce(c, m.apiKeyRequests(c)) {
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

func (m *RateLimiter) apiKeyRequests(c *gin.Context) []scopedRequest {
	authCtx := authctx.GetAuthContext(c)
	if !authCtx.IsAPIKey() || authCtx.APIKeyID.IsNil() || authCtx.OrganizationID.IsNil() {
		return nil
	}

	orgTag := "{" + authCtx.OrganizationID.String() + "}"
	requests := make([]scopedRequest, 0, 2)
	if policy, ok := m.policies[securityconfig.RateLimitScopeAPIKey]; ok {
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
