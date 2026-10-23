package rateengine

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratequote"
	"github.com/emoss08/trenova/internal/modules/rating/ports/tariffports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const lowerOfTemplateID = pulid.ID("ft_lower_of_stub")

const lowerOfSource = `tariff "Per mile or flat"

/// The rate per mile.
input rate: Money/mi where $0.50/mi..=$15/mi
/// The most a load pays.
input flatRate: Money = $1_000

best "Linehaul" {
  option "Per mile" {
    charge "Linehaul" (quantity: shipment.distance, rate: rate)
  }
  option "Flat" {
    charge "Flat rate" = flatRate
  }
  option "Volume" {
    require (shipment.weight ?? 0 lb) >= 50_000 lb else "Volume rates start at 50,000 lb"
    charge "Volume" = $0.01/lb * (shipment.weight ?? 0 lb)
  }
}
`

func lowerOfTemplate() *tariffports.Template {
	return &tariffports.Template{
		TemplateID:    lowerOfTemplateID,
		VersionNumber: 1,
		Name:          "Per mile or flat",
		Source:        lowerOfSource,
		Inputs: []tarifftypes.InputDecl{
			{
				Name:       "rate",
				Type:       "Money/mi",
				Descriptor: rules.RawJSON(`{"kind":"quantity","unit":"$/mi"}`),
				Required:   true,
			},
			{Name: "flatRate", Type: "Money", Descriptor: rules.RawJSON(`{"kind":"money"}`)},
		},
	}
}

func lowerOfRule(rate string) *rateagreement.RateAgreementRule {
	rule := perMileRule(rate, 100)
	rule.FormulaTemplateID = lowerOfTemplateID
	return rule
}

func TestARatingRecordsEveryChoiceOnTheResultAndTheQuote(t *testing.T) {
	t.Parallel()

	d := setup(t)
	d.templates.Add(lowerOfTemplate())
	expectRules(d, lowerOfRule("2.15"))

	result, err := d.svc.RateShipment(t.Context(), rateRequest(testShipment()))

	require.NoError(t, err)
	assert.Equal(t, ratequote.OutcomeRated, result.Outcome)
	assert.True(t, result.Amount.Equal(decimal.NewFromInt(1000)), "amount %s", result.Amount)

	require.Len(t, result.Choices, 1)
	choice := result.Choices[0]
	assert.Equal(t, "Linehaul", choice.Label)
	assert.Equal(t, tarifftypes.ChoiceBest, choice.Kind)
	assert.Equal(t, "USD", choice.Currency)
	require.NotNil(t, choice.Chosen)
	assert.Equal(t, "Flat", *choice.Chosen)
	require.Len(t, choice.Options, 3)
	assert.Equal(t, tarifftypes.OptionLost, choice.Options[0].Status)
	require.NotNil(t, choice.Options[0].Subtotal)
	assert.True(t, choice.Options[0].Subtotal.Equal(decimal.RequireFromString("1548")))
	assert.Equal(t, tarifftypes.OptionChosen, choice.Options[1].Status)
	assert.Equal(t, tarifftypes.OptionRefused, choice.Options[2].Status)
	assert.Equal(t, "Volume rates start at 50,000 lb", choice.Options[2].Refusal)

	assert.Equal(t, result.Choices, result.Quote.Trace.Choices)
}

func TestATariffWithoutChoicesRecordsNone(t *testing.T) {
	t.Parallel()

	d := setup(t)
	expectRules(d, perMileRule("2.15", 100))

	result, err := d.svc.RateShipment(t.Context(), rateRequest(testShipment()))

	require.NoError(t, err)
	assert.Empty(t, result.Choices)
	assert.Empty(t, result.Quote.Trace.Choices)
}

func TestAStoredValueOutsideItsConstraintStopsTheRatingNamingTheBounds(t *testing.T) {
	t.Parallel()

	d := setup(t)
	d.templates.Add(lowerOfTemplate())
	expectRules(d, lowerOfRule("235"))

	result, err := d.svc.RateShipment(t.Context(), rateRequest(testShipment()))

	require.NoError(t, err)
	assert.Equal(t, ratequote.OutcomeError, result.Outcome)
	assert.True(t, result.Amount.IsZero())
	assert.Equal(t,
		"`rate` must be from $0.50/mi to $15.00/mi; $235.00/mi is outside it",
		result.Quote.Trace.Error,
	)
}

func TestAShipmentsOwnValueOutsideItsConstraintRefusesTheRateOnItsTariffInputs(t *testing.T) {
	t.Parallel()

	d := setup(t)
	d.templates.Add(lowerOfTemplate())
	expectRules(d)

	entity := testShipment()
	entity.FormulaTemplateID = lowerOfTemplateID
	entity.TariffInputs = tarifftypes.Inputs{
		"rate": rules.QuantityValue(decimal.RequireFromString("0.10"), "$/mi"),
	}

	_, err := d.svc.RateShipment(t.Context(), rateRequest(entity))

	require.Error(t, err)
	var validation *errortypes.Error
	require.ErrorAs(t, err, &validation)
	assert.Equal(t, "tariffInputs", validation.Field)
	assert.Equal(t,
		"`rate` must be from $0.50/mi to $15.00/mi; $0.10/mi is outside it",
		errortypes.LocalizedText(t.Context(), err),
	)
}
