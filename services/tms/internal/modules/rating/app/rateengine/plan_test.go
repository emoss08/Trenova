package rateengine

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratequote"
	"github.com/emoss08/trenova/internal/modules/rating/ratingtest/tarifftest"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func baseOnly(t *testing.T, standardFlat bool) *deps {
	t.Helper()

	d := setup(t)
	flat := flatTemplate()
	flat.Standard = standardFlat
	d.templates = tarifftest.NewTemplates(tarifftest.PerMile(), flat, accessorialTemplate())
	d.svc.resolver = d.templates
	d.svc.features = plantest.BaseOnly()

	return d
}

func flatShipment() *shipment.Shipment {
	entity := testShipment()
	entity.FormulaTemplateID = flatTemplateID
	entity.TariffInputs = tarifftypes.Inputs{
		"rate": rules.MoneyValue(decimal.RequireFromString("850")),
	}
	return entity
}

func TestAStandardTemplateRatesOnABasePlan(t *testing.T) {
	t.Parallel()

	d := baseOnly(t, true)
	expectRules(d)

	result, err := d.svc.RateShipment(t.Context(), rateRequest(flatShipment()))

	require.NoError(t, err)
	assert.Equal(t, ratequote.OutcomeFormulaFallback, result.Outcome)
	assert.True(t, result.Amount.Equal(decimal.RequireFromString("850")),
		"amount was %s", result.Amount)
}

func TestTheOrganizationsOwnTemplateDoesNotRateWithoutRating(t *testing.T) {
	t.Parallel()

	d := baseOnly(t, false)
	expectRules(d)

	result, err := d.svc.RateShipment(t.Context(), rateRequest(flatShipment()))

	require.Error(t, err)
	assert.Nil(t, result)
	assert.True(t, errortypes.IsPlanRestrictionError(err), "a plan restriction, got %v", err)
	assert.Zero(t, d.evaluator.Calls(), "the tariff is never evaluated")
}

func TestAContractLaneWithTheOrganizationsOwnTemplateIsAnErrorWithoutRating(t *testing.T) {
	t.Parallel()

	d := baseOnly(t, false)
	expectRules(d, perMileRule("2.15", 100))

	result, err := d.svc.RateShipment(t.Context(), rateRequest(testShipment()))

	require.NoError(t, err, "the contract's quote records the refusal instead of failing")
	assert.Equal(t, ratequote.OutcomeError, result.Outcome)
	assert.Zero(t, d.evaluator.Calls())
}

func TestTheOrganizationsOwnTemplateRatesWithRating(t *testing.T) {
	t.Parallel()

	d := baseOnly(t, false)
	d.svc.features = plantest.Without(platformcatalog.FeatureDetention)
	expectRules(d)

	result, err := d.svc.RateShipment(t.Context(), rateRequest(flatShipment()))

	require.NoError(t, err)
	assert.Equal(t, ratequote.OutcomeFormulaFallback, result.Outcome)
}
