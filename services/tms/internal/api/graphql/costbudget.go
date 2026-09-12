package graphql

import (
	"context"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/errcode"
	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/graphql/querycost"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/vektah/gqlparser/v2/gqlerror"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

const (
	costBudgetExtensionName   = "OperationCostBudget"
	CostBudgetErrorCode       = "OPERATION_COST_BUDGET_EXCEEDED"
	rejectionReasonCostBudget = "cost_budget"
	minCostBudgetRetryAfter   = time.Second
)

type CostBudgetParams struct {
	fx.In

	Config  *config.Config
	Logger  *zap.Logger
	Metrics *metrics.Registry
}

type CostBudgetExtension struct {
	cfg     config.GraphQLCostBudgetConfig
	metrics *metrics.GraphQL
	l       *zap.Logger
	now     func() time.Time

	mu          sync.Mutex
	budgets     map[string]*costBudget
	lastCleanup time.Time
}

type costBudget struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

var _ interface {
	graphql.HandlerExtension
	graphql.OperationContextMutator
} = (*CostBudgetExtension)(nil)

func NewCostBudgetExtension(p CostBudgetParams) *CostBudgetExtension {
	return newCostBudgetExtension(
		p.Config.Security.GraphQL.CostBudget,
		p.Metrics.GraphQL,
		p.Logger.Named("api.graphql.costbudget"),
	)
}

func newCostBudgetExtension(
	cfg config.GraphQLCostBudgetConfig,
	graphQLMetrics *metrics.GraphQL,
	logger *zap.Logger,
) *CostBudgetExtension {
	return &CostBudgetExtension{
		cfg:     cfg,
		metrics: graphQLMetrics,
		l:       logger,
		now:     time.Now,
		budgets: make(map[string]*costBudget),
	}
}

func (*CostBudgetExtension) ExtensionName() string {
	return costBudgetExtensionName
}

func (*CostBudgetExtension) Validate(graphql.ExecutableSchema) error {
	return nil
}

func (e *CostBudgetExtension) MutateOperationContext(
	ctx context.Context,
	opCtx *graphql.OperationContext,
) *gqlerror.Error {
	if !e.cfg.Enabled {
		return nil
	}

	cost, ok := operationCost(opCtx)
	if !ok || cost <= 0 {
		return nil
	}

	authCtx, ok := gqlctx.AuthContext(ctx)
	if !ok || authCtx == nil {
		return nil
	}

	key := authCtx.OrganizationID.String() + "|" + authCtx.UserID.String()
	retryAfter, allowed := e.allow(key, cost)
	if allowed {
		return nil
	}

	e.metrics.RecordRejection(rejectionReasonCostBudget)
	if status, found := gqlctx.ResponseStatusFrom(ctx); found {
		status.Override(http.StatusTooManyRequests, retryAfter)
	}
	e.l.Warn("GraphQL operation exceeded the cost budget",
		zap.String("operation", opCtx.OperationName),
		zap.Int("cost", cost),
		zap.Duration("retry_after", retryAfter),
		zap.String("request_id", gqlctx.RequestID(ctx)),
	)

	err := gqlerror.Errorf(
		"operation cost %d exceeds the remaining request budget; retry in %s",
		cost,
		retryAfter,
	)
	errcode.Set(err, CostBudgetErrorCode)

	return err
}

func (e *CostBudgetExtension) allow(key string, cost int) (time.Duration, bool) {
	now := e.now()

	e.mu.Lock()
	defer e.mu.Unlock()

	e.cleanupLocked(now)

	budget, ok := e.budgets[key]
	if !ok {
		budget = &costBudget{limiter: rate.NewLimiter(e.refillRate(), e.burst())}
		e.budgets[key] = budget
	}
	budget.lastSeen = now

	if budget.limiter.AllowN(now, cost) {
		return 0, true
	}

	return retryAfterFor(budget.limiter, now, cost), false
}

func (e *CostBudgetExtension) cleanupLocked(now time.Time) {
	interval := e.cfg.GetCleanupInterval()
	if !e.lastCleanup.IsZero() && now.Sub(e.lastCleanup) < interval {
		return
	}

	for key, budget := range e.budgets {
		if now.Sub(budget.lastSeen) > interval {
			delete(e.budgets, key)
		}
	}
	e.lastCleanup = now
}

func (e *CostBudgetExtension) refillRate() rate.Limit {
	return rate.Limit(float64(e.cfg.GetPointsPerMinute()) / time.Minute.Seconds())
}

func (e *CostBudgetExtension) burst() int {
	return max(e.cfg.GetBurst(), querycost.MaxOperationCost)
}

func retryAfterFor(limiter *rate.Limiter, now time.Time, cost int) time.Duration {
	deficit := float64(cost) - limiter.TokensAt(now)
	if deficit <= 0 {
		return minCostBudgetRetryAfter
	}

	seconds := math.Ceil(deficit / float64(limiter.Limit()))
	retryAfter := time.Duration(seconds) * time.Second
	if retryAfter < minCostBudgetRetryAfter {
		return minCostBudgetRetryAfter
	}

	return retryAfter
}
