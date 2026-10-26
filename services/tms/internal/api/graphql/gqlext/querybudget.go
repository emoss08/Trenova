package gqlext

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/querybudget"
	"github.com/vektah/gqlparser/v2/ast"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	queryBudgetExtensionName = "TrenovaQueryBudget"
	maxLoggedOperationName   = 128
)

type QueryBudgetParams struct {
	fx.In

	Budgets *querybudget.Budgets
}

type QueryBudgetExtension struct {
	budgets *querybudget.Budgets
}

var _ interface {
	graphql.HandlerExtension
	graphql.ResponseInterceptor
} = (*QueryBudgetExtension)(nil)

func NewQueryBudgetExtension(p QueryBudgetParams) *QueryBudgetExtension {
	return &QueryBudgetExtension{budgets: p.Budgets}
}

func (*QueryBudgetExtension) ExtensionName() string {
	return queryBudgetExtensionName
}

func (*QueryBudgetExtension) Validate(graphql.ExecutableSchema) error {
	return nil
}

func (e *QueryBudgetExtension) InterceptResponse(
	ctx context.Context,
	next graphql.ResponseHandler,
) *graphql.Response {
	if !graphql.HasOperationContext(ctx) {
		return next(ctx)
	}

	opCtx := graphql.GetOperationContext(ctx)
	if opCtx.Operation != nil && opCtx.Operation.Operation == ast.Subscription {
		return next(ctx)
	}

	name := opCtx.OperationName
	if len(name) > maxLoggedOperationName {
		name = name[:maxLoggedOperationName]
	}

	return next(e.budgets.Start(ctx,
		zap.String("graphql_operation", name),
		zap.String("graphql_operation_type", operationType(opCtx)),
	))
}
