package gqlext

import (
	"context"
	"net/http"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/errcode"
	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/graphql/querycost"
	"github.com/emoss08/trenova/internal/infrastructure/config/managedconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/securityconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/tenantlimitsconfig"
	"github.com/emoss08/trenova/internal/infrastructure/observability/httpmetrics"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/internal/infrastructure/observability/platformmetrics"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/vektah/gqlparser/v2/gqlerror"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	costBudgetExtensionName   = "OperationCostBudget"
	CostBudgetErrorCode       = "OPERATION_COST_BUDGET_EXCEEDED"
	rejectionReasonCostBudget = "cost_budget"
	minCostBudgetRetryAfter   = time.Second
	costBudgetPeriod          = time.Minute
)

type CostBudgetParams struct {
	fx.In

	SecurityConfig *securityconfig.Config
	Managed        *managedconfig.Config
	TenantLimits   *tenantlimitsconfig.Config
	Store          kernelports.RateLimitStore
	Logger         *zap.Logger
	Metrics        *metrics.Registry
}

type CostBudgetExtension struct {
	cfg           securityconfig.GraphQLCostBudgetConfig
	store         kernelports.RateLimitStore
	organizations bool
	metrics       *httpmetrics.GraphQL
	tenancy       *platformmetrics.Tenancy
	l             *zap.Logger
}

var _ interface {
	graphql.HandlerExtension
	graphql.OperationContextMutator
} = (*CostBudgetExtension)(nil)

func NewCostBudgetExtension(p CostBudgetParams) *CostBudgetExtension {
	return &CostBudgetExtension{
		cfg:           p.SecurityConfig.GraphQL.CostBudget,
		store:         p.Store,
		organizations: p.TenantLimits.Requests.Organizations.On(p.Managed.SharedTenancy()),
		metrics:       p.Metrics.GraphQL,
		tenancy:       p.Metrics.Tenancy,
		l:             p.Logger.Named("api.graphql.costbudget"),
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
	if !ok || authCtx == nil || authCtx.OrganizationID.IsNil() {
		return nil
	}

	requests := e.requests(authCtx, cost)
	decisions, err := e.store.Check(ctx, requests)
	if err != nil || len(decisions) != len(requests) {
		e.l.Warn("could not check the GraphQL cost budget; allowing the operation",
			zap.Error(err))
		return nil
	}

	denied := -1
	for i := range decisions {
		if decisions[i].Allowed {
			continue
		}
		if denied < 0 || decisions[i].RetryAfter > decisions[denied].RetryAfter {
			denied = i
		}
	}
	if denied < 0 {
		return nil
	}

	retryAfter := max(decisions[denied].RetryAfter, minCostBudgetRetryAfter).Round(time.Second)
	e.metrics.RecordRejection(rejectionReasonCostBudget)
	if denied > 0 {
		e.tenancy.RecordLimitRefusal("graphql_cost", platformmetrics.LimitScopeOrganization)
	}
	if status, found := gqlctx.ResponseStatusFrom(ctx); found {
		status.Override(http.StatusTooManyRequests, retryAfter)
	}
	e.l.Warn("GraphQL operation exceeded the cost budget",
		zap.String("operation", opCtx.OperationName),
		zap.Int("cost", cost),
		zap.Bool("organization", denied > 0),
		zap.Duration("retry_after", retryAfter),
		zap.String("request_id", gqlctx.RequestID(ctx)),
	)

	gqlErr := gqlerror.Errorf(
		"operation cost %d exceeds the remaining request budget; retry in %s",
		cost,
		retryAfter,
	)
	errcode.Set(gqlErr, CostBudgetErrorCode)

	return gqlErr
}

func (e *CostBudgetExtension) requests(
	authCtx *authctx.AuthContext,
	cost int,
) []kernelports.RateLimitRequest {
	principal := kernelports.RateLimitPolicy{
		Rate:   e.cfg.GetPointsPerMinute(),
		Period: costBudgetPeriod,
		Burst:  max(e.cfg.GetBurst(), querycost.MaxOperationCost),
	}
	requests := []kernelports.RateLimitRequest{{
		Key:    budgetKey(authCtx),
		Policy: principal,
		Cost:   min(cost, principal.Burst),
	}}
	if !e.organizations {
		return requests
	}
	organization := kernelports.RateLimitPolicy{
		Rate:   e.cfg.GetOrganizationPointsPerMinute(),
		Period: costBudgetPeriod,
		Burst:  max(e.cfg.GetOrganizationBurst(), querycost.MaxOperationCost),
	}
	return append(requests, kernelports.RateLimitRequest{
		Key:    organizationBudgetKey(authCtx),
		Policy: organization,
		Cost:   min(cost, organization.Burst),
	})
}

// budgetKey is whose bucket an operation is charged to: a person's own, and
func budgetKey(authCtx *authctx.AuthContext) string {
	principal := authCtx.UserID
	if authCtx.IsAgent() {
		principal = authCtx.PrincipalID
	}

	return "{" + authCtx.OrganizationID.String() + "}:gqlcost:" + principal.String()
}

func organizationBudgetKey(authCtx *authctx.AuthContext) string {
	return "{" + authCtx.OrganizationID.String() + "}:gqlcost"
}
