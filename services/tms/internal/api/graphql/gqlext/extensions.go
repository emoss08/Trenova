package gqlext

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/emoss08/trenova/internal/api/graphql/querycost"
	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
	"go.uber.org/fx"
)

// Params are the extensions every executor of Trenova's schema runs: the
// HTTP server and the agent read path take the same list, in the same
// order, so a limit or a guard added here reaches both.
type Params struct {
	fx.In

	Observability *ObservabilityExtension
	CostBudget    *CostBudgetExtension
	ReadOnly      *ReadOnlyExtension
	FieldGuard    *FieldGuardExtension
	QueryBudget   *QueryBudgetExtension
}

func Extensions(p Params) []graphql.HandlerExtension {
	return []graphql.HandlerExtension{
		operationDepthLimit{max: querycost.MaxOperationDepth},
		extension.FixedComplexityLimit(querycost.MaxOperationCost),
		p.CostBudget,
		p.ReadOnly,
		p.FieldGuard,
		p.Observability,
		p.QueryBudget,
	}
}

func NewCostLimitedSchema(es graphql.ExecutableSchema) graphql.ExecutableSchema {
	return newCostLimitedSchema(es)
}

func ErrorPresenter(cfg *appconfig.Config) graphql.ErrorPresenterFunc {
	return newErrorPresenter(cfg)
}

func Recover(ctx context.Context, err any) error {
	return recoverFunc(ctx, err)
}
