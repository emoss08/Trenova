package graphql

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/errcode"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/graphql/querycost"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/gqlerror"
	"go.uber.org/zap"
)

func newTestCostBudget(t *testing.T, cfg config.GraphQLCostBudgetConfig) *CostBudgetExtension {
	t.Helper()

	ext := newCostBudgetExtension(
		cfg,
		metrics.NewGraphQL(nil, zap.NewNop(), false, metrics.GraphQLOptions{}),
		zap.NewNop(),
	)
	base := time.Date(2026, time.September, 5, 12, 0, 0, 0, time.UTC)
	ext.now = func() time.Time { return base }

	return ext
}

func costBudgetContext(t *testing.T) (context.Context, *gqlctx.ResponseStatus) {
	t.Helper()

	status := gqlctx.NewResponseStatus()
	ctx := gqlctx.WithAuthContext(context.Background(), &authctx.AuthContext{
		UserID:         pulid.MustNew("usr_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	})
	ctx = gqlctx.WithResponseStatus(ctx, status)

	return ctx, status
}

func costedOperationContext(cost int) *graphql.OperationContext {
	opCtx := &graphql.OperationContext{OperationName: "Costed"}
	opCtx.Stats.SetExtension(complexityStatsKey, &extension.ComplexityStats{
		Complexity:      cost,
		ComplexityLimit: querycost.MaxOperationCost,
	})

	return opCtx
}

func TestCostBudgetExtension_Contract(t *testing.T) {
	t.Parallel()

	ext := newTestCostBudget(t, config.GraphQLCostBudgetConfig{Enabled: true})

	assert.Equal(t, costBudgetExtensionName, ext.ExtensionName())
	assert.NoError(t, ext.Validate(nil))
	err := gqlerror.Errorf("budget")
	errcode.Set(err, CostBudgetErrorCode)
	assert.Equal(t, errcode.KindProtocol, errcode.GetErrorKind(gqlerror.List{err}))
}

func TestCostBudgetExtension_DisabledSkipsAccounting(t *testing.T) {
	t.Parallel()

	ext := newTestCostBudget(t, config.GraphQLCostBudgetConfig{Enabled: false, Burst: 10})
	ctx, _ := costBudgetContext(t)

	for range 5 {
		assert.Nil(t, ext.MutateOperationContext(ctx, costedOperationContext(100)))
	}
	assert.Empty(t, ext.budgets)
}

func TestCostBudgetExtension_AllowsWithinBurst(t *testing.T) {
	t.Parallel()

	ext := newTestCostBudget(t, config.GraphQLCostBudgetConfig{
		Enabled:         true,
		PointsPerMinute: 60,
		Burst:           querycost.MaxOperationCost,
	})
	ctx, status := costBudgetContext(t)

	assert.Nil(
		t,
		ext.MutateOperationContext(ctx, costedOperationContext(querycost.MaxOperationCost/2)),
	)
	assert.Nil(
		t,
		ext.MutateOperationContext(ctx, costedOperationContext(querycost.MaxOperationCost/2)),
	)

	_, overridden := status.Code()
	assert.False(t, overridden)
}

func TestCostBudgetExtension_RejectsWhenBudgetExhausted(t *testing.T) {
	t.Parallel()

	ext := newTestCostBudget(t, config.GraphQLCostBudgetConfig{
		Enabled:         true,
		PointsPerMinute: 60,
		Burst:           querycost.MaxOperationCost,
	})
	ctx, status := costBudgetContext(t)

	require.Nil(
		t,
		ext.MutateOperationContext(ctx, costedOperationContext(querycost.MaxOperationCost)),
	)

	err := ext.MutateOperationContext(ctx, costedOperationContext(1000))
	require.NotNil(t, err)
	assert.Equal(t, CostBudgetErrorCode, err.Extensions["code"])
	assert.Contains(t, err.Message, "exceeds the remaining request budget")

	code, overridden := status.Code()
	assert.True(t, overridden)
	assert.Equal(t, http.StatusTooManyRequests, code)
	assert.Equal(t, 1000*time.Second, status.RetryAfter())
}

func TestCostBudgetExtension_RefillsOverTime(t *testing.T) {
	t.Parallel()

	ext := newTestCostBudget(t, config.GraphQLCostBudgetConfig{
		Enabled:         true,
		PointsPerMinute: 6000,
		Burst:           querycost.MaxOperationCost,
	})
	ctx, _ := costBudgetContext(t)
	start := ext.now()

	require.Nil(
		t,
		ext.MutateOperationContext(ctx, costedOperationContext(querycost.MaxOperationCost)),
	)
	require.NotNil(t, ext.MutateOperationContext(ctx, costedOperationContext(500)))

	ext.now = func() time.Time { return start.Add(5 * time.Second) }
	assert.Nil(t, ext.MutateOperationContext(ctx, costedOperationContext(500)))
}

func TestCostBudgetExtension_BudgetsAreScopedToPrincipal(t *testing.T) {
	t.Parallel()

	ext := newTestCostBudget(t, config.GraphQLCostBudgetConfig{
		Enabled:         true,
		PointsPerMinute: 60,
		Burst:           querycost.MaxOperationCost,
	})
	first, _ := costBudgetContext(t)
	second, _ := costBudgetContext(t)

	require.Nil(
		t,
		ext.MutateOperationContext(first, costedOperationContext(querycost.MaxOperationCost)),
	)
	assert.NotNil(t, ext.MutateOperationContext(first, costedOperationContext(1)))
	assert.Nil(
		t,
		ext.MutateOperationContext(second, costedOperationContext(querycost.MaxOperationCost)),
	)
	assert.Len(t, ext.budgets, 2)
}

func TestCostBudgetExtension_SkipsWithoutCostOrPrincipal(t *testing.T) {
	t.Parallel()

	ext := newTestCostBudget(t, config.GraphQLCostBudgetConfig{Enabled: true, Burst: 1})
	ctx, _ := costBudgetContext(t)

	assert.Nil(t, ext.MutateOperationContext(ctx, &graphql.OperationContext{}))
	assert.Nil(t, ext.MutateOperationContext(context.Background(), costedOperationContext(10)))
	assert.Empty(t, ext.budgets)
}

func TestCostBudgetExtension_BurstNeverBelowSingleOperationLimit(t *testing.T) {
	t.Parallel()

	ext := newTestCostBudget(t, config.GraphQLCostBudgetConfig{Enabled: true, Burst: 1})
	ctx, _ := costBudgetContext(t)

	assert.Nil(
		t,
		ext.MutateOperationContext(ctx, costedOperationContext(querycost.MaxOperationCost)),
	)
}

func TestCostBudgetExtension_CleanupDropsIdlePrincipals(t *testing.T) {
	t.Parallel()

	ext := newTestCostBudget(t, config.GraphQLCostBudgetConfig{
		Enabled:         true,
		CleanupInterval: time.Minute,
	})
	first, _ := costBudgetContext(t)
	second, _ := costBudgetContext(t)
	start := ext.now()

	require.Nil(t, ext.MutateOperationContext(first, costedOperationContext(10)))
	assert.Len(t, ext.budgets, 1)

	ext.now = func() time.Time { return start.Add(2 * time.Minute) }
	require.Nil(t, ext.MutateOperationContext(second, costedOperationContext(10)))
	assert.Len(t, ext.budgets, 1)
}
