package formula_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/rating/app/formula"
	"github.com/emoss08/trenova/internal/modules/rating/app/formula/contextvariablecache"
	"github.com/emoss08/trenova/internal/modules/rating/app/formula/environment"
	"github.com/emoss08/trenova/internal/modules/rating/app/formula/resolver"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/internal/modules/reference/domain/servicetype"
	"github.com/emoss08/trenova/pkg/formulatemplatetypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
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
	return formula.NewService(formula.ServiceParams{
		Logger:   zap.NewNop(),
		Registry: registry,
		Environment: environment.NewBuilder(environment.Params{
			Registry: registry,
			Resolver: res,
		}),
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

func TestPreview_WithoutAShipmentSeesTheTenantsProvidedValuesExactly(t *testing.T) {
	t.Parallel()

	fuel := &stubContextProvider{variables: map[string]any{"fuelPrice": 3.85}}
	svc := setupServiceWithProviders(t, fuel)

	preview, err := svc.Preview(t.Context(), &formula.PreviewRequest{
		SchemaID:   "shipment",
		Expression: "coalesce(fuelPrice, 0) * customRate",
		Variables:  map[string]any{"customRate": 2.0},
		TenantInfo: testTenant(),
	})
	require.NoError(t, err)
	require.True(t, preview.Valid(), "%+v", preview.Diagnostics)
	assert.Equal(t, decimal.RequireFromString("7.7"), preview.Evaluated.Amount)
	assert.Equal(t, decimal.RequireFromString("3.85"), preview.Variables["fuelPrice"])
	assert.Equal(t, decimal.RequireFromString("2"), preview.Variables["customRate"])
}

func TestEvaluatePredicates_SeeTheTenantsProvidedValues(t *testing.T) {
	t.Parallel()

	fuel := &stubContextProvider{variables: map[string]any{"fuelPrice": 4.5}}
	svc := setupServiceWithProviders(t, fuel)

	outcomes, err := svc.EvaluatePredicates(
		t.Context(),
		&formulatemplateports.EvaluatePredicatesRequest{
			Expressions: []string{"fuelPrice > 4", "fuelPrice > 5"},
			SchemaID:    "shipment",
			Entity:      &shipment.Shipment{ServiceType: &servicetype.ServiceType{Code: "STD"}},
			TenantInfo:  testTenant(),
		},
	)
	require.NoError(t, err)
	require.Len(t, outcomes, 2)
	require.NoError(t, outcomes[0].Err)
	assert.True(t, outcomes[0].Holds)
	assert.False(t, outcomes[1].Holds)
	assert.EqualValues(t, 1, fuel.calls.Load(), "one batch builds one environment")
}
