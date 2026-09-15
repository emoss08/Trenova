package development

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/domain/ratequote"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSeededRatingDetail_MirrorsWhatTheCalculatorWritesForAFormulaFallback(t *testing.T) {
	t.Parallel()

	template := &formulatemplate.FormulaTemplate{
		ID:         pulid.MustNew("ft_"),
		Name:       "Flat Rate",
		Expression: "baseRate",
	}
	shippedAt := int64(1_789_333_941)
	shp := &shipment.Shipment{
		FormulaTemplateID:   template.ID,
		FreightChargeAmount: decimal.NewNullDecimal(decimal.NewFromFloat(3150)),
		ActualShipDate:      &shippedAt,
	}

	detail := seededRatingDetail(shp, template, shippedAt+86_400)

	require.NotNil(t, detail)
	assert.Equal(t, template.ID.String(), detail.FormulaTemplateID)
	assert.Equal(t, "Flat Rate", detail.FormulaTemplateName)
	assert.Equal(t, "baseRate", detail.Expression)
	assert.InDelta(t, 3150, detail.Result, 0)
	assert.Equal(t, shippedAt, detail.RatedAt)
	assert.Equal(t, string(ratequote.OutcomeFormulaFallback), detail.Source)
	assert.Equal(
		t,
		(&ratequote.RateQuote{Outcome: ratequote.OutcomeFormulaFallback}).Explanation(),
		detail.Explanation,
	)
	require.Len(t, detail.Breakdown, 1)
	assert.Equal(t, "Linehaul", detail.Breakdown[0].Name)
	assert.Equal(t, "Linehaul", detail.Breakdown[0].Label)
	assert.InDelta(t, 3150, detail.Breakdown[0].Amount, 0)
	require.NotNil(t, detail.ResolvedVariables)
	assert.Empty(t, detail.ResolvedVariables)
	assert.Nil(t, detail.Guardrail)
	assert.Nil(t, detail.Receipt)
	assert.Empty(t, detail.AgreementID)
	assert.Empty(t, detail.RateQuoteID)
}

func TestSeededRatingDetail_RatesAtNowWhenTheShipmentHasNotShipped(t *testing.T) {
	t.Parallel()

	template := &formulatemplate.FormulaTemplate{ID: pulid.MustNew("ft_"), Name: "Per Mile"}
	now := int64(1_800_000_000)

	unshipped := seededRatingDetail(&shipment.Shipment{FormulaTemplateID: template.ID}, template, now)
	assert.Equal(t, now, unshipped.RatedAt)
	assert.InDelta(t, 0, unshipped.Result, 0)

	zeroDate := int64(0)
	blank := seededRatingDetail(
		&shipment.Shipment{FormulaTemplateID: template.ID, ActualShipDate: &zeroDate},
		template,
		now,
	)
	assert.Equal(t, now, blank.RatedAt)
}
