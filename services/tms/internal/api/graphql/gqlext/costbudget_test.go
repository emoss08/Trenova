package gqlext

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/errcode"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/graphql/querycost"
	"github.com/emoss08/trenova/internal/infrastructure/config/managedconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/securityconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/tenantlimitsconfig"
	"github.com/emoss08/trenova/internal/infrastructure/observability/httpmetrics"
	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/internal/infrastructure/observability/platformmetrics"
	"github.com/emoss08/trenova/internal/infrastructure/ratelimit"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/gqlerror"
	"go.uber.org/zap"
)

type costClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *costClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *costClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

type costBudgetRig struct {
	ext      *CostBudgetExtension
	clock    *costClock
	registry *prometheus.Registry
}

func newCostBudgetRig(
	t *testing.T,
	cfg securityconfig.GraphQLCostBudgetConfig,
	managed *managedconfig.Config,
	store kernelports.RateLimitStore,
) *costBudgetRig {
	t.Helper()

	clock := &costClock{now: time.Date(2026, time.September, 5, 12, 0, 0, 0, time.UTC)}
	if store == nil {
		store = ratelimit.NewMemoryStore(ratelimit.MemoryStoreOptions{Now: clock.Now})
	}
	registry := prometheus.NewRegistry()
	ext := NewCostBudgetExtension(CostBudgetParams{
		SecurityConfig: &securityconfig.Config{
			GraphQL: securityconfig.GraphQLConfig{CostBudget: cfg},
		},
		Managed:      managed,
		TenantLimits: &tenantlimitsconfig.Config{},
		Store:        store,
		Logger:       zap.NewNop(),
		Metrics: &metrics.Registry{
			GraphQL: httpmetrics.NewGraphQL(nil, zap.NewNop(), false, httpmetrics.GraphQLOptions{}),
			Tenancy: platformmetrics.NewTenancy(registry, zap.NewNop(), true),
		},
	})

	return &costBudgetRig{ext: ext, clock: clock, registry: registry}
}

func newTestCostBudget(
	t *testing.T,
	cfg securityconfig.GraphQLCostBudgetConfig,
) *costBudgetRig {
	t.Helper()
	return newCostBudgetRig(t, cfg, &managedconfig.Config{}, nil)
}

func costBudgetContextIn(
	t *testing.T,
	org pulid.ID,
) (context.Context, *gqlctx.ResponseStatus) {
	t.Helper()

	status := gqlctx.NewResponseStatus()
	ctx := gqlctx.WithAuthContext(context.Background(), &authctx.AuthContext{
		UserID:         pulid.MustNew("usr_"),
		OrganizationID: org,
		BusinessUnitID: pulid.MustNew("bu_"),
	})
	ctx = gqlctx.WithResponseStatus(ctx, status)

	return ctx, status
}

func costBudgetContext(t *testing.T) (context.Context, *gqlctx.ResponseStatus) {
	t.Helper()
	return costBudgetContextIn(t, pulid.MustNew("org_"))
}

func costedOperationContext(cost int) *graphql.OperationContext {
	opCtx := &graphql.OperationContext{OperationName: "Costed"}
	opCtx.Stats.SetExtension(complexityStatsKey, &extension.ComplexityStats{
		Complexity:      cost,
		ComplexityLimit: querycost.MaxOperationCost,
	})

	return opCtx
}

type failingStore struct{ calls int }

func (s *failingStore) Check(
	context.Context,
	[]kernelports.RateLimitRequest,
) ([]kernelports.RateLimitDecision, error) {
	s.calls++
	return nil, errors.New("redis is down")
}

func TestCostBudgetExtension_Contract(t *testing.T) {
	t.Parallel()

	rig := newTestCostBudget(t, securityconfig.GraphQLCostBudgetConfig{Enabled: true})

	assert.Equal(t, costBudgetExtensionName, rig.ext.ExtensionName())
	assert.NoError(t, rig.ext.Validate(nil))
	err := gqlerror.Errorf("budget")
	errcode.Set(err, CostBudgetErrorCode)
	assert.Equal(t, errcode.KindProtocol, errcode.GetErrorKind(gqlerror.List{err}))
}

func TestCostBudgetExtension_DisabledSkipsAccounting(t *testing.T) {
	t.Parallel()

	store := &failingStore{}
	rig := newCostBudgetRig(
		t,
		securityconfig.GraphQLCostBudgetConfig{Enabled: false, Burst: 10},
		&managedconfig.Config{},
		store,
	)
	ctx, _ := costBudgetContext(t)

	for range 5 {
		assert.Nil(t, rig.ext.MutateOperationContext(ctx, costedOperationContext(100)))
	}
	assert.Zero(t, store.calls)
}

func TestCostBudgetExtension_AllowsWithinBurst(t *testing.T) {
	t.Parallel()

	rig := newTestCostBudget(t, securityconfig.GraphQLCostBudgetConfig{
		Enabled:         true,
		PointsPerMinute: 60,
		Burst:           querycost.MaxOperationCost,
	})
	ctx, status := costBudgetContext(t)

	half := costedOperationContext(querycost.MaxOperationCost / 2)
	assert.Nil(t, rig.ext.MutateOperationContext(ctx, half))
	assert.Nil(t, rig.ext.MutateOperationContext(ctx, half))

	_, overridden := status.Code()
	assert.False(t, overridden)
}

func TestCostBudgetExtension_RejectsWhenBudgetExhausted(t *testing.T) {
	t.Parallel()

	rig := newTestCostBudget(t, securityconfig.GraphQLCostBudgetConfig{
		Enabled:         true,
		PointsPerMinute: 60,
		Burst:           querycost.MaxOperationCost,
	})
	ctx, status := costBudgetContext(t)

	require.Nil(
		t,
		rig.ext.MutateOperationContext(ctx, costedOperationContext(querycost.MaxOperationCost)),
	)

	err := rig.ext.MutateOperationContext(ctx, costedOperationContext(1000))
	require.NotNil(t, err)
	assert.Equal(t, CostBudgetErrorCode, err.Extensions["code"])
	assert.Contains(t, err.Message, "exceeds the remaining request budget")

	code, overridden := status.Code()
	assert.True(t, overridden)
	assert.Equal(t, http.StatusTooManyRequests, code)
	assert.GreaterOrEqual(t, status.RetryAfter(), 999*time.Second)
}

func TestCostBudgetExtension_RefillsOverTime(t *testing.T) {
	t.Parallel()

	rig := newTestCostBudget(t, securityconfig.GraphQLCostBudgetConfig{
		Enabled:         true,
		PointsPerMinute: 6000,
		Burst:           querycost.MaxOperationCost,
	})
	ctx, _ := costBudgetContext(t)

	require.Nil(
		t,
		rig.ext.MutateOperationContext(ctx, costedOperationContext(querycost.MaxOperationCost)),
	)
	require.NotNil(t, rig.ext.MutateOperationContext(ctx, costedOperationContext(500)))

	rig.clock.Advance(5 * time.Second)
	assert.Nil(t, rig.ext.MutateOperationContext(ctx, costedOperationContext(500)))
}

func TestCostBudgetExtension_BudgetsAreScopedToPrincipal(t *testing.T) {
	t.Parallel()

	rig := newTestCostBudget(t, securityconfig.GraphQLCostBudgetConfig{
		Enabled:         true,
		PointsPerMinute: 60,
		Burst:           querycost.MaxOperationCost,
	})
	org := pulid.MustNew("org_")
	first, _ := costBudgetContextIn(t, org)
	second, _ := costBudgetContextIn(t, org)

	full := costedOperationContext(querycost.MaxOperationCost)
	require.Nil(t, rig.ext.MutateOperationContext(first, full))
	assert.NotNil(t, rig.ext.MutateOperationContext(first, costedOperationContext(1)))
	assert.Nil(t, rig.ext.MutateOperationContext(second, full),
		"off a shared server, a colleague's spending is not charged to anyone else")
}

func TestCostBudgetExtension_SkipsWithoutCostOrPrincipal(t *testing.T) {
	t.Parallel()

	store := &failingStore{}
	rig := newCostBudgetRig(
		t,
		securityconfig.GraphQLCostBudgetConfig{Enabled: true, Burst: 1},
		&managedconfig.Config{},
		store,
	)
	ctx, _ := costBudgetContext(t)

	assert.Nil(t, rig.ext.MutateOperationContext(ctx, &graphql.OperationContext{}))
	assert.Nil(
		t,
		rig.ext.MutateOperationContext(context.Background(), costedOperationContext(10)),
	)
	assert.Zero(t, store.calls)
}

func TestCostBudgetExtension_AllowsWhenTheStoreIsDown(t *testing.T) {
	t.Parallel()

	store := &failingStore{}
	rig := newCostBudgetRig(
		t,
		securityconfig.GraphQLCostBudgetConfig{Enabled: true},
		&managedconfig.Config{},
		store,
	)
	ctx, status := costBudgetContext(t)

	assert.Nil(t, rig.ext.MutateOperationContext(ctx, costedOperationContext(10)))
	assert.Equal(t, 1, store.calls)
	_, overridden := status.Code()
	assert.False(t, overridden)
}

func TestCostBudgetExtension_BurstNeverBelowSingleOperationLimit(t *testing.T) {
	t.Parallel()

	rig := newTestCostBudget(t, securityconfig.GraphQLCostBudgetConfig{Enabled: true, Burst: 1})
	ctx, _ := costBudgetContext(t)

	assert.Nil(
		t,
		rig.ext.MutateOperationContext(ctx, costedOperationContext(querycost.MaxOperationCost)),
	)
}

func TestCostBudgetExtension_ChargesTheOrganizationOnASharedServer(t *testing.T) {
	t.Parallel()

	rig := newCostBudgetRig(t, securityconfig.GraphQLCostBudgetConfig{
		Enabled:                     true,
		PointsPerMinute:             60,
		Burst:                       querycost.MaxOperationCost,
		OrganizationPointsPerMinute: 60,
		OrganizationBurst:           querycost.MaxOperationCost,
	}, &managedconfig.Config{Enabled: true, Tenancy: managedconfig.TenancyShared}, nil)
	busy := pulid.MustNew("org_")
	first, _ := costBudgetContextIn(t, busy)
	colleague, status := costBudgetContextIn(t, busy)
	elsewhere, _ := costBudgetContext(t)

	full := costedOperationContext(querycost.MaxOperationCost)
	require.Nil(t, rig.ext.MutateOperationContext(first, full))

	err := rig.ext.MutateOperationContext(colleague, full)
	require.NotNil(t, err, "a colleague with budget of their own waits for the organization's")
	assert.Equal(t, CostBudgetErrorCode, err.Extensions["code"])
	code, _ := status.Code()
	assert.Equal(t, http.StatusTooManyRequests, code)

	assert.Nil(t, rig.ext.MutateOperationContext(elsewhere, full),
		"another organization keeps its own budget")

	families, gatherErr := rig.registry.Gather()
	require.NoError(t, gatherErr)
	refused := 0.0
	for _, family := range families {
		if family.GetName() == "trenova_tenancy_limit_refusals_total" {
			for _, metric := range family.GetMetric() {
				refused += metric.GetCounter().GetValue()
			}
		}
	}
	assert.InDelta(t, 1, refused, 0)
}

func TestBudgetKey_ChargesEachUnattendedAgentItsOwnBucket(t *testing.T) {
	t.Parallel()

	org := pulid.MustNew("org_")
	first := &authctx.AuthContext{
		PrincipalType: authctx.PrincipalTypeAgent, PrincipalID: pulid.MustNew("agd_"),
		OrganizationID: org,
	}
	second := &authctx.AuthContext{
		PrincipalType: authctx.PrincipalTypeAgent, PrincipalID: pulid.MustNew("agd_"),
		OrganizationID: org,
	}
	person := &authctx.AuthContext{
		PrincipalType: authctx.PrincipalTypeUser, PrincipalID: pulid.MustNew("usr_"),
		UserID: pulid.MustNew("usr_"), OrganizationID: org,
	}

	assert.NotEqual(t, budgetKey(first), budgetKey(second))
	assert.Equal(t, "{"+org.String()+"}:gqlcost:"+person.UserID.String(), budgetKey(person))
}
