package middleware

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/internal/infrastructure/ratelimit"
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

	Config       *config.Config
	Store        repositories.RateLimitStore
	ErrorHandler *helpers.ErrorHandler
	Logger       *zap.Logger
	Metrics      *metrics.Registry `optional:"true"`
}

type RateLimiter struct {
	enabled      bool
	store        repositories.RateLimitStore
	errorHandler *helpers.ErrorHandler
	logger       *zap.Logger
	metrics      *metrics.RateLimit
	policies     map[config.RateLimitScope]repositories.RateLimitPolicy
	exemptPaths  []string
}

type scopedRequest struct {
	scope   config.RateLimitScope
	request repositories.RateLimitRequest
}

func NewRateLimiter(p RateLimiterParams) *RateLimiter {
	cfg := config.RateLimitConfig{}
	if p.Config != nil {
		cfg = p.Config.Security.RateLimit
	}

	logger := p.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	var rateLimitMetrics *metrics.RateLimit
	if p.Metrics != nil {
		rateLimitMetrics = p.Metrics.RateLimit
	}

	scopes := []config.RateLimitScope{
		config.RateLimitScopeAnonymous,
		config.RateLimitScopeUser,
		config.RateLimitScopeAPIKey,
		config.RateLimitScopeTenant,
		config.RateLimitScopePublicToken,
	}
	policies := make(map[config.RateLimitScope]repositories.RateLimitPolicy, len(scopes))
	for _, scope := range scopes {
		policy := cfg.ScopePolicy(scope)
		if !policy.Enabled {
			continue
		}
		policies[scope] = repositories.RateLimitPolicy{
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
	policy, ok := m.policies[config.RateLimitScopeAnonymous]
	if !ok {
		return nil
	}
	return []scopedRequest{{
		scope: config.RateLimitScopeAnonymous,
		request: repositories.RateLimitRequest{
			Key:    "anon:" + c.ClientIP(),
			Policy: policy,
			Cost:   1,
		},
	}}
}

func (m *RateLimiter) principalRequests(c *gin.Context) []scopedRequest {
	authCtx := authctx.GetAuthContext(c)
	if authCtx.OrganizationID.IsNil() {
		return m.anonymousRequests(c)
	}

	orgTag := "{" + authCtx.OrganizationID.String() + "}"
	requests := make([]scopedRequest, 0, 2)

	if authCtx.IsAPIKey() && authCtx.APIKeyID.IsNotNil() {
		if policy, ok := m.policies[config.RateLimitScopeAPIKey]; ok {
			requests = append(requests, scopedRequest{
				scope: config.RateLimitScopeAPIKey,
				request: repositories.RateLimitRequest{
					Key:    orgTag + ":apikey:" + authCtx.APIKeyID.String(),
					Policy: policy,
					Cost:   1,
				},
			})
		}
	} else if authCtx.UserID.IsNotNil() {
		if policy, ok := m.policies[config.RateLimitScopeUser]; ok {
			requests = append(requests, scopedRequest{
				scope: config.RateLimitScopeUser,
				request: repositories.RateLimitRequest{
					Key:    orgTag + ":user:" + authCtx.UserID.String(),
					Policy: policy,
					Cost:   1,
				},
			})
		}
	}

	if policy, ok := m.policies[config.RateLimitScopeTenant]; ok {
		requests = append(requests, scopedRequest{
			scope: config.RateLimitScopeTenant,
			request: repositories.RateLimitRequest{
				Key:    orgTag + ":tenant",
				Policy: policy,
				Cost:   1,
			},
		})
	}

	return requests
}

func (m *RateLimiter) publicTokenRequests(c *gin.Context, param string) []scopedRequest {
	policy, ok := m.policies[config.RateLimitScopePublicToken]
	if !ok {
		return nil
	}
	token := c.Param(param)
	if token == "" {
		return m.anonymousRequests(c)
	}
	return []scopedRequest{{
		scope: config.RateLimitScopePublicToken,
		request: repositories.RateLimitRequest{
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

	storeRequests := make([]repositories.RateLimitRequest, len(requests))
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
	decision repositories.RateLimitDecision,
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
	decision repositories.RateLimitDecision,
) {
	c.Header(HeaderRateLimitLimit, strconv.Itoa(decision.Limit))
	c.Header(HeaderRateLimitRemaining, strconv.Itoa(decision.Remaining))
	c.Header(HeaderRateLimitReset, strconv.Itoa(timeutils.CeilSeconds(decision.ResetAfter)))
	c.Header(HeaderRateLimitPolicy, formatRateLimitPolicy(req.request.Policy))
}

func (m *RateLimiter) recordDecision(scope config.RateLimitScope, allowed bool) {
	if m.metrics == nil {
		return
	}
	m.metrics.RecordDecision(string(scope), allowed)
}

func formatRateLimitPolicy(policy repositories.RateLimitPolicy) string {
	return strconv.Itoa(policy.Rate) +
		";w=" + strconv.Itoa(int(policy.Period/time.Second)) +
		";burst=" + strconv.Itoa(policy.Burst)
}
