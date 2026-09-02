package formula_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/emoss08/trenova/internal/core/services/formula"
	"github.com/emoss08/trenova/internal/core/services/formula/contextvariablecache"
	"github.com/emoss08/trenova/internal/core/services/formula/engine"
	"github.com/emoss08/trenova/internal/core/services/formula/resolver"
	"github.com/emoss08/trenova/pkg/formulatemplatetypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type stubContextProvider struct {
	variables map[string]any
	err       error
	calls     atomic.Int32
}

func (s *stubContextProvider) ContextVariables(
	_ context.Context,
	_ pagination.TenantInfo,
) (map[string]any, error) {
	s.calls.Add(1)
	return s.variables, s.err
}

func setupServiceWithProviders(
	t *testing.T,
	providers ...formulatemplatetypes.ContextVariableProvider,
) *formula.Service {
	t.Helper()

	registry := newTestRegistry(t)
	res := resolver.NewResolver()
	envBuilder := engine.NewEnvironmentBuilder(engine.EnvironmentBuilderParams{
		Registry: registry,
		Resolver: res,
	})
	eng, err := engine.NewEngine(engine.Params{
		Registry:   registry,
		Resolver:   res,
		EnvBuilder: envBuilder,
	})
	require.NoError(t, err)

	return formula.NewService(formula.ServiceParams{
		Logger:           zap.NewNop(),
		Registry:         registry,
		Engine:           eng,
		Resolver:         res,
		VersionRepo:      &stubVersionRepo{},
		RateMatrixRepo:   &stubMatrixRepo{},
		ContextProviders: providers,
	})
}

func testTenant() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
}

func TestContextVariables_MergesProvidersAndSurvivesOneFailing(t *testing.T) {
	t.Parallel()

	fuel := &stubContextProvider{variables: map[string]any{"fuelPrice": 3.85}}
	broken := &stubContextProvider{err: errors.New("feed down")}
	fx := &stubContextProvider{variables: map[string]any{"usdCadRate": 1.36}}
	svc := setupServiceWithProviders(t, fuel, broken, fx)

	variables := svc.ContextVariables(t.Context(), testTenant())

	assert.InDelta(t, 3.85, variables["fuelPrice"], 0.0001)
	assert.InDelta(t, 1.36, variables["usdCadRate"], 0.0001)
	assert.Len(t, variables, 2, "a failing provider costs its own variables, nothing else")
}

func TestContextVariables_MemoizedPerContextAndTenant(t *testing.T) {
	t.Parallel()

	fuel := &stubContextProvider{variables: map[string]any{"fuelPrice": 3.85}}
	svc := setupServiceWithProviders(t, fuel)
	tenant := testTenant()

	ctx := contextvariablecache.With(t.Context())
	svc.ContextVariables(ctx, tenant)
	svc.ContextVariables(ctx, tenant)
	assert.EqualValues(t, 1, fuel.calls.Load(), "one batch asks each feed once")

	svc.ContextVariables(ctx, testTenant())
	assert.EqualValues(t, 2, fuel.calls.Load(), "another tenant is another question")

	svc.ContextVariables(t.Context(), tenant)
	assert.EqualValues(t, 3, fuel.calls.Load(), "without a memo every call asks")
}

func TestBuildValidationEnvironmentForTenant_IncludesProvidedVariables(t *testing.T) {
	t.Parallel()

	fuel := &stubContextProvider{variables: map[string]any{"fuelPrice": 3.85}}
	svc := setupServiceWithProviders(t, fuel)

	env, err := svc.BuildValidationEnvironmentForTenant(
		t.Context(),
		testTenant(),
		"shipment",
		map[string]any{"customRate": 2.0},
	)
	require.NoError(t, err)
	assert.InDelta(t, 3.85, env["fuelPrice"], 0.0001)
	assert.InDelta(t, 2.0, env["customRate"], 0.0001)
}
