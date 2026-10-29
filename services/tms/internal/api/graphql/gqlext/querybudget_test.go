package gqlext

import (
	"context"
	"testing"

	gqlgen "github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/config/databaseconfig"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/querybudget"
	"github.com/stretchr/testify/assert"
	"github.com/vektah/gqlparser/v2/ast"
	"go.uber.org/zap"
)

func queryBudgetExtension() *QueryBudgetExtension {
	return NewQueryBudgetExtension(QueryBudgetParams{
		Budgets: querybudget.New(
			querybudget.Params{DatabaseConfig: &databaseconfig.Config{}, Logger: zap.NewNop()},
		),
	})
}

func budgetedResponse(ctx context.Context, ext *QueryBudgetExtension) bool {
	var budgeted bool
	ext.InterceptResponse(ctx, func(ctx context.Context) *gqlgen.Response {
		_, budgeted = keel.BudgetReportFrom(ctx)
		return &gqlgen.Response{}
	})

	return budgeted
}

func budgetOperationContext(ctx context.Context, operation ast.Operation) context.Context {
	return gqlgen.WithOperationContext(ctx, &gqlgen.OperationContext{
		OperationName: "ShipmentBoard",
		Operation:     &ast.OperationDefinition{Operation: operation},
	})
}

func TestQueryBudgetExtensionBudgetsQueriesAndMutations(t *testing.T) {
	t.Parallel()

	ext := queryBudgetExtension()
	assert.True(t, budgetedResponse(budgetOperationContext(t.Context(), ast.Query), ext))
	assert.True(t, budgetedResponse(budgetOperationContext(t.Context(), ast.Mutation), ext))
}

func TestQueryBudgetExtensionLeavesSubscriptionsAndBareContextsAlone(t *testing.T) {
	t.Parallel()

	ext := queryBudgetExtension()
	assert.False(t, budgetedResponse(budgetOperationContext(t.Context(), ast.Subscription), ext))
	assert.False(t, budgetedResponse(t.Context(), ext))
}

func TestExtensionsRunTheQueryBudgetWhenProvided(t *testing.T) {
	t.Parallel()

	ext := queryBudgetExtension()
	assert.Contains(t, Extensions(Params{QueryBudget: ext}), gqlgen.HandlerExtension(ext))
	assert.NotContains(
		t,
		Extensions(Params{}),
		gqlgen.HandlerExtension((*QueryBudgetExtension)(nil)),
	)
}
