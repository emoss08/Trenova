package formula_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/rating/app/formula"
	"github.com/emoss08/trenova/internal/modules/rating/app/formula/engine"
	"github.com/emoss08/trenova/internal/modules/rating/app/formula/resolver"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratematrixports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/formulatypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func setupServiceWithResolver(
	t *testing.T,
	res *resolver.Resolver,
	matrices ratematrixports.RateMatrixRepository,
) *formula.Service {
	t.Helper()

	registry := newTestRegistry(t)
	eng, err := engine.NewEngine(engine.Params{
		Registry: registry,
		Resolver: res,
		EnvBuilder: engine.NewEnvironmentBuilder(engine.EnvironmentBuilderParams{
			Registry: registry,
			Resolver: res,
		}),
	})
	require.NoError(t, err)

	return formula.NewService(formula.ServiceParams{
		Logger:         zap.NewNop(),
		Registry:       registry,
		Engine:         eng,
		Resolver:       res,
		VersionRepo:    &stubVersionRepo{},
		RateMatrixRepo: matrices,
	})
}


func TestService_Rate_APanickingResolverIsAValidationErrorNotACrash(t *testing.T) {
	t.Parallel()

	res := defaultResolver()
	svc := setupServiceWithResolver(t, res, &stubMatrixRepo{})
	res.RegisterComputed("computeHasHazmat", func(any) (any, error) {
		panic("reflect: call of reflect.Value.Field on zero Value")
	})

	_, err := svc.Rate(t.Context(), &formula.RateRequest{
		Template: activeTemplate(`hasHazmat ? 50 : 10`),
		Entity:   shippedOnMarchFirst(),
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsError(err), "a 422 the author can act on, not a 500")
	assert.Contains(t, err.Error(), "hasHazmat")
	assert.NotContains(t, err.Error(), "reflect")

	resp, err := svc.Rate(t.Context(), &formula.RateRequest{
		Template: activeTemplate(`totalStops + 10`),
		Entity:   shippedOnMarchFirst(),
	})
	require.NoError(t, err, "a formula that never reads the broken field still prices")
	assert.Equal(t, "10", resp.Amount.String())
}

func TestService_ErrorsNameNoGoTypes(t *testing.T) {
	t.Parallel()

	svc := setupService(t)

	cases := map[string]string{
		"a list result":                     `stops`,
		"a text result":                     `"85"`,
		"unresolved fields beside an error": `stops * 2`,
	}

	for name, expression := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := svc.EvaluateExpression(t.Context(), &formula.EvaluateExpressionRequest{
				Expression: expression,
				Entity:     &shipment.Shipment{},
				SchemaID:   "shipment",
			})
			require.Error(t, err)
			for _, internal := range []string{"*shipment.", "shipment.Shipment", "interface {}", "[]"} {
				assert.NotContains(t, err.Error(), internal)
			}
		})
	}
}

type stampedLookupRepo struct {
	stubMatrixRepo
}

func (*stampedLookupRepo) GetLookupStamp(
	_ context.Context,
	_ pagination.TenantInfo,
) (string, error) {
	return "3:7:1700000000", nil
}

func TestService_Rate_TheReceiptCarriesTheRateTableStamp(t *testing.T) {
	t.Parallel()

	repo := &stampedLookupRepo{stubMatrixRepo{data: freightClassMatrix()}}
	svc := setupServiceWithResolver(t, defaultResolver(), repo)

	template := activeTemplate(`lookup("class_rates", "85")`)
	template.OrganizationID = pulid.MustNew("org_")
	template.BusinessUnitID = pulid.MustNew("bu_")

	resp, err := svc.Rate(t.Context(), &formula.RateRequest{
		Template: template,
		Entity:   shippedOnMarchFirst(),
	})
	require.NoError(t, err)
	require.NotEmpty(t, resp.Receipt.Lookups)
	assert.Equal(t, "3:7:1700000000", resp.Receipt.RateTableStamp)

	template.Expression = `25`
	resp, err = svc.Rate(t.Context(), &formula.RateRequest{
		Template: template,
		Entity:   shippedOnMarchFirst(),
	})
	require.NoError(t, err)
	assert.Empty(t, resp.Receipt.RateTableStamp, "no table was read, so none is named")
}

func defaultResolver() *resolver.Resolver {
	res := resolver.NewResolver()
	resolver.RegisterDefaultComputed(res)
	return res
}

func TestService_Rate_BreakdownLinesRefuseTrueOrFalse(t *testing.T) {
	t.Parallel()

	svc := setupService(t)
	template := activeTemplate(`100`)
	template.BreakdownDefinitions = []*formulatypes.BreakdownDefinition{
		{Name: "flag", Label: "Flag", Expression: `totalStops > 0`},
	}

	resp, err := svc.Rate(t.Context(), &formula.RateRequest{
		Template: template,
		Entity:   shippedOnMarchFirst(),
	})
	require.NoError(t, err)
	require.Len(t, resp.Breakdown, 1)
	assert.Contains(t, resp.Breakdown[0].Error, "true/false")
}

func TestApplyChargePolicy_RefusesANegativeCharge(t *testing.T) {
	t.Parallel()

	_, _, _, err := formula.ApplyChargePolicy(
		formulatypes.ChargePolicy{},
		decimal.RequireFromString("-12.504"),
	)
	require.Error(t, err)
	assert.True(t, errortypes.IsError(err))
	assert.Contains(t, err.Error(), "-12.50")

	amount, guardrail, _, err := formula.ApplyChargePolicy(
		formulatypes.ChargePolicy{MinCharge: decimal.NewNullDecimal(decimal.NewFromInt(75))},
		decimal.RequireFromString("-12.504"),
	)
	require.NoError(t, err, "a minimum charge is the author's answer to a negative result")
	assert.Equal(t, "75", amount.String())
	assert.True(t, guardrail.Applied)

	amount, _, _, err = formula.ApplyChargePolicy(formulatypes.ChargePolicy{}, decimal.Zero)
	require.NoError(t, err, "nothing owed is not a negative charge")
	assert.True(t, amount.IsZero())
}
