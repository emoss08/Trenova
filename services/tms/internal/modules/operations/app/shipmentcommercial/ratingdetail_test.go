package shipmentcommercial

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmentcommercial/shipmentcommercialtest"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratequote"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratequoteports"
	"github.com/emoss08/trenova/internal/testutil/mocks/ratequoteportsmock"
	"github.com/emoss08/trenova/pkg/ratetypes"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func rateOnce(
	t *testing.T,
	rated *ratequoteports.RatedShipment,
) *ratequoteportsmock.MockRateEngine {
	t.Helper()

	engine := ratequoteportsmock.NewMockRateEngine(t)
	engine.EXPECT().
		RateShipment(mock.Anything, mock.AnythingOfType("*ratequoteports.RateShipmentRequest")).
		Return(rated, nil).
		Once()

	return engine
}

func recalculateWith(
	t *testing.T,
	engine ratequoteports.RateEngine,
	entity *shipment.Shipment,
) {
	t.Helper()

	calculator := New(Params{Logger: zap.NewNop(), RateEngine: engine})
	calculator.now = func() int64 { return shipmentcommercialtest.RatedAt }

	require.NoError(t, calculator.Recalculate(
		t.Context(),
		entity,
		&tenant.ShipmentControl{},
		pulid.MustNew("usr_"),
	))
}

func TestRecalculate_TheDetailRecordsTheTariffsBillAndInputs(t *testing.T) {
	t.Parallel()

	entity := validShipment()
	freight := decimal.NewFromInt(1500)
	templateID := pulid.MustNew("ft_")
	inputs, err := tarifftypes.InputsFromLiterals(map[string]string{"rate": "$2.50/mi"})
	require.NoError(t, err)
	statements := rules.RawJSON(`[{"line":4,"value":{"$q":"600","u":"mi"}}]`)

	recalculateWith(t, rateOnce(t, &ratequoteports.RatedShipment{
		Amount:              freight,
		Total:               freight,
		Currency:            "USD",
		Outcome:             ratequote.OutcomeFormulaFallback,
		FormulaTemplateID:   &templateID,
		FormulaTemplateName: "Per Mile",
		TemplateVersion:     4,
		TariffInputs:        inputs,
		Bill:                shipmentcommercialtest.FreightBill(freight),
		Quote: &ratequote.RateQuote{
			Outcome:        ratequote.OutcomeFormulaFallback,
			LinehaulAmount: freight,
			TotalAmount:    freight,
			RatedAt:        shipmentcommercialtest.RatedAt,
			Trace:          &ratetypes.Trace{Statements: statements},
		},
	}), entity)

	detail := entity.RatingDetail
	require.NotNil(t, detail)
	assert.Equal(t, templateID.String(), detail.FormulaTemplateID)
	assert.Equal(t, "Per Mile", detail.FormulaTemplateName)
	assert.Equal(t, int64(4), detail.VersionNumber)
	assert.Equal(t, string(ratequote.OutcomeFormulaFallback), detail.Source)
	assert.Equal(t, shipmentcommercialtest.RatedAt, detail.RatedAt)
	assert.True(t, inputs.Equal(detail.Inputs))
	require.NotNil(t, detail.Bill)
	assert.True(t, detail.Bill.FreightTotal.Equal(freight))
	assert.JSONEq(t, string(statements), string(detail.Trace))
	assert.Empty(t, detail.Refusal)
}

func TestRecalculate_ARefusalIsRecordedInTheTariffsWords(t *testing.T) {
	t.Parallel()

	entity := validShipment()

	recalculateWith(t, rateOnce(t, &ratequoteports.RatedShipment{
		Currency: "USD",
		Outcome:  ratequote.OutcomeNoRateFound,
		Refusal:  "Hazmat loads are quoted by hand",
		Quote: &ratequote.RateQuote{
			Outcome: ratequote.OutcomeNoRateFound,
			RatedAt: shipmentcommercialtest.RatedAt,
		},
	}), entity)

	detail := entity.RatingDetail
	require.NotNil(t, detail)
	assert.Equal(t, "Hazmat loads are quoted by hand", detail.Refusal)
	assert.Nil(t, detail.Bill)
	assert.Empty(t, entity.FreightLines)
	assert.True(t, entity.FreightChargeAmount.Decimal.IsZero())
}

func TestRecalculate_AHandRateIsRecordedAsManual(t *testing.T) {
	t.Parallel()

	entity := validShipment()
	entity.FormulaTemplateID = pulid.Nil
	entity.FreightChargeAmount = decimal.NewNullDecimal(decimal.NewFromInt(900))

	recalculateWith(t, ratequoteportsmock.NewMockRateEngine(t), entity)

	detail := entity.RatingDetail
	require.NotNil(t, detail)
	assert.Equal(t, shipment.RatingSourceManual, detail.Source)
	assert.True(t, detail.RatedByHand())
	require.Len(t, entity.FreightLines, 1)
	assert.True(t, entity.FreightLines[0].Amount.Equal(decimal.NewFromInt(900)))
}
