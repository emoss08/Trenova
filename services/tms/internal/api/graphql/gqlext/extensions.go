package gqlext

import (
	"context"
	"runtime/debug"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/graphql/querycost"
	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
	"go.uber.org/fx"
	"go.uber.org/zap"
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
	Logger        *zap.Logger
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

// Recover turns a resolver's panic into the error the client sees, and logs it with its stack:
// the client is told only that the request failed, so the log is the one place the cause shows.
func Recover(logger *zap.Logger) graphql.RecoverFunc {
	l := logger.Named("graphql.recover")
	return func(ctx context.Context, err any) error {
		l.Error("a GraphQL resolver panicked",
			zap.Any("panic", err),
			zap.String("request_id", gqlctx.RequestID(ctx)),
			zap.ByteString("stack", debug.Stack()),
		)
		return recoverFunc(ctx, err)
	}
}
