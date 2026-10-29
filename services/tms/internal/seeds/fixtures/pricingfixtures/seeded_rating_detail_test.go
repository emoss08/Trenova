package pricingfixtures

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratequote"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSeededRatingDetail_MirrorsWhatTheCalculatorWritesForAFormulaFallback(t *testing.T) {
	t.Parallel()

	template := &formulatemplate.FormulaTemplate{ID: pulid.MustNew("ft_"), Name: "Flat Rate"}
	shippedAt := int64(1_789_333_941)
	freight := decimal.NewFromInt(3150)
	shp := &shipment.Shipment{
		FormulaTemplateID:   template.ID,
		FreightChargeAmount: decimal.NewNullDecimal(freight),
		TariffInputs:        SeededTariffInputs(template.Name, freight),
		ActualShipDate:      &shippedAt,
	}

	detail := SeededRatingDetail(shp, template, shippedAt+86_400)

	require.NotNil(t, detail)
	assert.Equal(t, template.ID.String(), detail.FormulaTemplateID)
	assert.Equal(t, "Flat Rate", detail.FormulaTemplateName)
	assert.Equal(t, shippedAt, detail.RatedAt)
	assert.Equal(t, string(ratequote.OutcomeFormulaFallback), detail.Source)
	assert.Equal(
		t,
		(&ratequote.RateQuote{Outcome: ratequote.OutcomeFormulaFallback}).Explanation(),
		detail.Explanation,
	)
	require.NotNil(t, detail.Bill)
	require.Len(t, detail.Bill.Lines, 1)
	assert.Equal(t, tarifftypes.LineCharge, detail.Bill.Lines[0].Kind)
	assert.Empty(t, detail.Bill.Lines[0].Code)
	assert.True(t, freight.Equal(detail.Bill.FreightTotal))
	assert.True(t, freight.Equal(detail.Bill.Total))
	assert.True(t, shp.TariffInputs.Equal(detail.Inputs))
	assert.Empty(t, detail.AgreementID)
	assert.Empty(t, detail.RateQuoteID)
}

func TestSeededRatingDetail_RatesAtNowWhenTheShipmentHasNotShipped(t *testing.T) {
	t.Parallel()

	template := &formulatemplate.FormulaTemplate{ID: pulid.MustNew("ft_"), Name: "Per Mile"}
	now := int64(1_800_000_000)

	unshipped := SeededRatingDetail(
		&shipment.Shipment{FormulaTemplateID: template.ID},
		template,
		now,
	)
	assert.Equal(t, now, unshipped.RatedAt)
	assert.True(t, unshipped.Bill.Total.IsZero())

	zeroDate := int64(0)
	blank := SeededRatingDetail(
		&shipment.Shipment{FormulaTemplateID: template.ID, ActualShipDate: &zeroDate},
		template,
		now,
	)
	assert.Equal(t, now, blank.RatedAt)
}

func TestSeededTariffInputs_TypeTheRateToItsTemplate(t *testing.T) {
	t.Parallel()

	perMile, err := rules.DecodeQuantity(
		SeededTariffInputs(formulatemplate.StandardPerMile, decimal.NewFromInt(2))["rate"],
	)
	require.NoError(t, err)
	assert.Equal(t, "$/mi", perMile.Unit)

	flat, err := rules.DecodeQuantity(
		SeededTariffInputs(formulatemplate.StandardFlatRate, decimal.NewFromInt(900))["rate"],
	)
	require.NoError(t, err)
	assert.Equal(t, rules.UnitMoney, flat.Unit)
}
