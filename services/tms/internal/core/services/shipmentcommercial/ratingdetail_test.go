package shipmentcommercial

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/ratequote"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/ratetypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// A contract prices from its own tables, so there are no formula variables to
// record — but every reader of the detail, the GraphQL schema included, treats
// the map as always present. A nil one fails the response, not just the field.
func TestRecalculate_ContractRatingLeavesResolvedVariablesEmptyNotNil(t *testing.T) {
	t.Parallel()

	entity := validShipment()

	calculator := New(Params{Logger: zap.NewNop(), RateEngine: StubRateEngine(t, 1000)})
	calculator.now = func() int64 { return ratedAt }

	require.NoError(
		t,
		calculator.Recalculate(t.Context(), entity, &tenant.ShipmentControl{}, pulid.MustNew("usr_")),
	)

	require.NotNil(t, entity.RatingDetail)
	assert.NotNil(t, entity.RatingDetail.ResolvedVariables)
	assert.Empty(t, entity.RatingDetail.ResolvedVariables)
}

// When a formula did produce the linehaul, the detail has to name it: the
// billing panel shows the template and its expression, and the trace is the
// only place they exist by the time the quote comes back.
func TestRecalculate_FormulaPricedShipmentNamesItsTemplate(t *testing.T) {
	t.Parallel()

	entity := validShipment()
	linehaul := decimal.NewFromInt(1500)

	trace := &ratetypes.Trace{
		Totals: ratetypes.Totals{Linehaul: linehaul, Total: linehaul},
	}
	trace.AddComponent(&ratetypes.Component{
		Kind:       ratetypes.ComponentKindLinehaul,
		Label:      "Linehaul",
		Amount:     linehaul,
		Source:     ratetypes.ComponentSourceFormulaTemplate,
		SourceID:   "ft_01JT",
		SourceName: "Mileage Base",
		Detail: map[string]any{
			"expression":    "distance * ratePerMile",
			"versionNumber": int64(4),
		},
	})

	engine := mocks.NewMockRateEngine(t)
	engine.EXPECT().
		RateShipment(mock.Anything, mock.AnythingOfType("*services.RateShipmentRequest")).
		Return(&services.RatedShipment{
			Amount:   linehaul,
			Currency: "USD",
			Outcome:  ratequote.OutcomeFormulaFallback,
			Quote: &ratequote.RateQuote{
				Outcome:        ratequote.OutcomeFormulaFallback,
				LinehaulAmount: linehaul,
				TotalAmount:    linehaul,
				RatedAt:        ratedAt,
				Trace:          trace,
			},
		}, nil).
		Once()

	calculator := New(Params{Logger: zap.NewNop(), RateEngine: engine})
	calculator.now = func() int64 { return ratedAt }

	require.NoError(
		t,
		calculator.Recalculate(t.Context(), entity, &tenant.ShipmentControl{}, pulid.MustNew("usr_")),
	)

	detail := entity.RatingDetail
	require.NotNil(t, detail)
	assert.Equal(t, "ft_01JT", detail.FormulaTemplateID)
	assert.Equal(t, "Mileage Base", detail.FormulaTemplateName)
	assert.Equal(t, "distance * ratePerMile", detail.Expression)
	assert.Equal(t, int64(4), detail.VersionNumber)
	assert.NotNil(t, detail.ResolvedVariables)
}

// A rate that a minimum charge took over is the first thing a customer
// questions, so the shipment has to say so — and say which of the bounds it
// was, since the last one applied is what produced the number they were quoted.
func TestRecalculate_ClampedRateRecordsTheGuardrailThatDecidedIt(t *testing.T) {
	t.Parallel()

	entity := validShipment()
	clamped := decimal.NewFromInt(500)

	trace := &ratetypes.Trace{
		Totals: ratetypes.Totals{Linehaul: clamped, Total: clamped},
		Guardrails: []ratetypes.Guardrail{
			{
				Kind:    ratetypes.ComponentKindMinimumCharge,
				Applied: true,
				Bound:   decimal.NewFromInt(400),
				Raw:     decimal.NewFromInt(320),
				Result:  decimal.NewFromInt(400),
			},
			{
				Kind:    ratetypes.ComponentKindAbsoluteMinCharge,
				Applied: true,
				Bound:   clamped,
				Raw:     decimal.NewFromInt(400),
				Result:  clamped,
			},
		},
	}

	engine := mocks.NewMockRateEngine(t)
	engine.EXPECT().
		RateShipment(mock.Anything, mock.AnythingOfType("*services.RateShipmentRequest")).
		Return(&services.RatedShipment{
			Amount:   clamped,
			Currency: "USD",
			Outcome:  ratequote.OutcomeRated,
			Quote: &ratequote.RateQuote{
				Outcome:        ratequote.OutcomeRated,
				LinehaulAmount: clamped,
				TotalAmount:    clamped,
				RatedAt:        ratedAt,
				Trace:          trace,
			},
		}, nil).
		Once()

	calculator := New(Params{Logger: zap.NewNop(), RateEngine: engine})
	calculator.now = func() int64 { return ratedAt }

	require.NoError(
		t,
		calculator.Recalculate(t.Context(), entity, &tenant.ShipmentControl{}, pulid.MustNew("usr_")),
	)

	require.NotNil(t, entity.RatingDetail)
	guardrail := entity.RatingDetail.Guardrail
	require.NotNil(t, guardrail)
	assert.True(t, guardrail.Applied)
	assert.Equal(t, "min", guardrail.Bound)
	assert.InDelta(t, 400.0, guardrail.RawResult, 0.001)
	require.NotNil(t, guardrail.MinCharge)
	assert.InDelta(t, 500.0, *guardrail.MinCharge, 0.001)
	assert.Nil(t, guardrail.MaxCharge)
}
