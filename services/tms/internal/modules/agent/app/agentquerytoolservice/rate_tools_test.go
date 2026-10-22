package agentquerytoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratequote"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratequoteports"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/pkg/ratetypes"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
A rate explanation that does not match the invoice is worse than none.

The engine writes a trace on every rating, and this reads the trace of the
quote that actually governs the shipment rather than re-rating it. Contracts
get amended, fuel indexes move, stops get edited — a fresh rating months later
answers a question nobody asked, and it would answer it with a different
number than the one being disputed.
*/

type stubQuoteReader struct {
	quote *ratequote.RateQuote
	saw   *ratequoteports.GetShipmentRateQuoteRequest
}

func (s *stubQuoteReader) GetAppliedForShipment(
	_ context.Context,
	req *ratequoteports.GetShipmentRateQuoteRequest,
) (*ratequote.RateQuote, error) {
	s.saw = req

	return s.quote, nil
}

func explainTool(quote *ratequote.RateQuote) (*explainRateTool, *stubQuoteReader) {
	reader := &stubQuoteReader{quote: quote}

	return &explainRateTool{quotes: reader}, reader
}

func usd(raw string) decimal.Decimal {
	return decimal.RequireFromString(raw)
}

func ratedQuote(trace *ratetypes.Trace) *ratequote.RateQuote {
	return &ratequote.RateQuote{Trace: trace}
}

func fullTrace() *ratetypes.Trace {
	return &ratetypes.Trace{
		EngineVersion: "v3",
		TieBreak:      "specificity",
		Candidates: []ratetypes.Candidate{
			{
				AgreementCode: "ACME-2026",
				AgreementName: "Acme master",
				RuleLabel:     "Reno to Boise, dry van",
				LaneKey:       "NV:RNO>ID:BOI",
				Won:           true,
				MatchedOn:     []string{"origin zone", "destination zone", "equipment"},
			},
			{
				AgreementCode: "ACME-2025",
				RuleLabel:     "Nevada outbound",
				RejectReason:  ratetypes.RejectReason("expired"),
				RejectDetail:  "effective through 2025-12-31",
			},
		},
		Components: []ratetypes.Component{
			{
				Label:        "Linehaul",
				Basis:        "1,240.0 mi @ $2.15/mi",
				Amount:       usd("2666.00"),
				RunningTotal: usd("2666.00"),
			},
			{
				Label:        "Fuel surcharge",
				Basis:        "1,240.0 mi @ $0.14/mi",
				Amount:       usd("173.60"),
				RunningTotal: usd("2839.60"),
			},
		},
		TariffInputs: tarifftypes.Inputs{"rate": rules.QuantityValue(decimal.RequireFromString("2.15"), "$/mi")},
		Skipped:      []ratetypes.Skip{{Code: "FSC", Source: "FuelSurcharge"}},
		Totals: ratetypes.Totals{
			Linehaul: usd("2666.00"),
			Fuel:     usd("173.60"),
			Total:    usd("2800.00"),
		},
		Warnings: []string{"fuel index is 6 days stale"},
	}
}

func explain(t *testing.T, tool *explainRateTool, params map[string]any) rateExplanation {
	t.Helper()
	result, err := tool.Query(t.Context(), testParams(params))
	require.NoError(t, err)
	explanation, ok := result.(rateExplanation)
	require.True(t, ok)

	return explanation
}

func TestExplainRate_NamesWhatPricedTheShipmentAndWhy(t *testing.T) {
	t.Parallel()

	tool, _ := explainTool(ratedQuote(fullTrace()))
	explanation := explain(t, tool, map[string]any{"shipmentId": pulid.MustNew("shp_").String()})

	require.NotNil(t, explanation.Winner)
	assert.Equal(t, "ACME-2026", explanation.Winner.AgreementCode)
	assert.Equal(t, "Reno to Boise, dry van", explanation.Winner.RuleLabel)
	// The tie-break is the first question a disputed rate raises: why this
	// rule and not the other one.
	assert.Equal(t, "specificity", explanation.TieBreak)
}

// "No rate applied" and "a rate applied but not the one you expected" are
// different problems, and the reason a rule was passed over is what tells
// them apart.
func TestExplainRate_SaysWhyTheOtherRulesDidNotApply(t *testing.T) {
	t.Parallel()

	tool, _ := explainTool(ratedQuote(fullTrace()))
	explanation := explain(t, tool, map[string]any{"shipmentId": pulid.MustNew("shp_").String()})

	require.Len(t, explanation.Rejected, 1)
	assert.Equal(t, "ACME-2025", explanation.Rejected[0].AgreementCode)
	assert.Equal(t, "expired", explanation.Rejected[0].Reason)
	assert.Contains(t, explanation.Rejected[0].Detail, "2025-12-31")
}

// The basis is what goes on a dispute letter, so it is passed through as the
// engine wrote it rather than rebuilt from the numbers.
func TestExplainRate_KeepsEachChargesArithmetic(t *testing.T) {
	t.Parallel()

	tool, _ := explainTool(ratedQuote(fullTrace()))
	explanation := explain(t, tool, map[string]any{"shipmentId": pulid.MustNew("shp_").String()})

	require.Len(t, explanation.Components, 2)
	assert.Equal(t, "1,240.0 mi @ $2.15/mi", explanation.Components[0].Basis)
	assert.True(t, explanation.Components[1].RunningTotal.Equal(usd("2839.60")))
}

// The inputs the tariff priced with and the charges other engines left to it
// are what a dispute asks about once the winning rule is known.
func TestExplainRate_ReportsTheInputsAndTheSkippedCharges(t *testing.T) {
	t.Parallel()

	tool, _ := explainTool(ratedQuote(fullTrace()))
	explanation := explain(t, tool, map[string]any{"shipmentId": pulid.MustNew("shp_").String()})

	assert.Equal(t, map[string]string{"rate": "$2.15/mi"}, explanation.Inputs)
	require.Len(t, explanation.Skipped, 1)
	assert.Equal(t, "FSC", explanation.Skipped[0].Code)
	assert.Equal(t, "FuelSurcharge", explanation.Skipped[0].Source)
}

func TestExplainRate_CarriesTheWarningsAndTheTotals(t *testing.T) {
	t.Parallel()

	tool, _ := explainTool(ratedQuote(fullTrace()))
	explanation := explain(t, tool, map[string]any{"shipmentId": pulid.MustNew("shp_").String()})

	assert.Equal(t, []string{"fuel index is 6 days stale"}, explanation.Warnings)
	assert.True(t, explanation.Totals.Total.Equal(usd("2800.00")))
	assert.Equal(t, "v3", explanation.EngineVersion)
}

// An unrated shipment has no price to explain, and reporting its zero totals
// as the answer would be reporting a price nobody set.
func TestExplainRate_SaysThereIsNothingToExplainRatherThanQuotingZero(t *testing.T) {
	t.Parallel()

	tool, _ := explainTool(nil)
	explanation := explain(t, tool, map[string]any{"shipmentId": pulid.MustNew("shp_").String()})

	assert.NotEmpty(t, explanation.Note)
	assert.Nil(t, explanation.Winner)
	assert.Empty(t, explanation.Components)
}

func TestExplainRate_ReadsTheCarrierSideWhenAsked(t *testing.T) {
	t.Parallel()

	tool, reader := explainTool(ratedQuote(fullTrace()))
	explain(t, tool, map[string]any{
		"shipmentId": pulid.MustNew("shp_").String(),
		"side":       "Carrier",
	})

	require.NotNil(t, reader.saw)
	assert.Equal(t, "Carrier", string(reader.saw.PartyType))
}

// Nothing matched is a different answer from nothing was tried, and the lane
// keys are what separates them.
func TestExplainRate_CarriesTheLaneKeysWhenNothingMatched(t *testing.T) {
	t.Parallel()

	tool, _ := explainTool(ratedQuote(&ratetypes.Trace{
		LaneKeysTried: []string{"NV:RNO>ID:BOI", "NV>ID", "*"},
		Error:         "no rule matched",
	}))
	explanation := explain(t, tool, map[string]any{"shipmentId": pulid.MustNew("shp_").String()})

	assert.Equal(t, []string{"NV:RNO>ID:BOI", "NV>ID", "*"}, explanation.LaneKeysTried)
	assert.Equal(t, "no rule matched", explanation.Error)
	assert.Nil(t, explanation.Winner)
}

func TestExplainRate_RefusesSomethingThatIsNotAShipmentID(t *testing.T) {
	t.Parallel()

	tool, _ := explainTool(ratedQuote(fullTrace()))

	_, err := tool.Query(t.Context(), testParams(map[string]any{"shipmentId": "the reno load"}))

	require.Error(t, err)
}

type stubShipmentRating struct{ record *shipment.Shipment }

func (s *stubShipmentRating) GetByID(
	_ context.Context,
	_ *shipmentports.GetShipmentByIDRequest,
) (*shipment.Shipment, error) {
	return s.record, nil
}

/*
A shipment no agreement priced is explained from the rating it carries.

SEED-PAY-007 was a spot load priced by the Flat Rate tariff at $3,050. With no
quote trace the tool said there was no rating on record, and the person had to
ask three times before a delegate read it off the shipment.
*/
func TestExplainRate_ExplainsAShipmentPricedFromItsFormulaTemplate(t *testing.T) {
	t.Parallel()

	record := &shipment.Shipment{
		FreightChargeAmount: decimal.NewNullDecimal(usd("3050")),
		TotalChargeAmount:   decimal.NewNullDecimal(usd("3050")),
		RatingDetail: &shipment.RatingDetail{
			FormulaTemplateName: "Flat Rate",
			Source:              "FormulaFallback",
			Explanation:         "No agreement covered this lane; priced from a formula template",
			Inputs:              tarifftypes.Inputs{"rate": rules.MoneyValue(usd("3050"))},
			Bill: &tarifftypes.Bill{
				Currency: "USD",
				Lines: []tarifftypes.Line{{
					Kind: tarifftypes.LineCharge, Label: "Linehaul", Freight: true, Amount: usd("3050"),
				}},
				FreightTotal: usd("3050"),
				Total:        usd("3050"),
			},
		},
	}
	tool := &explainRateTool{quotes: &stubQuoteReader{}, shipments: &stubShipmentRating{record: record}}
	explanation := explain(t, tool, map[string]any{"shipmentId": pulid.MustNew("shp_").String()})

	require.NotNil(t, explanation.PricedBy)
	assert.Equal(t, "Flat Rate", explanation.PricedBy.Method)
	assert.Equal(t, map[string]string{"rate": "$3050"}, explanation.PricedBy.Inputs)
	require.Len(t, explanation.Components, 1)
	assert.Equal(t, "Linehaul", explanation.Components[0].Label)
	assert.True(t, explanation.Components[0].Amount.Equal(usd("3050")))
	assert.True(t, explanation.Totals.Total.Equal(usd("3050")))
	assert.Contains(t, explanation.Note, "No rate agreement priced this shipment")
}

func TestExplainRate_StillSaysNothingWhenTheShipmentCarriesNoRating(t *testing.T) {
	t.Parallel()

	tool := &explainRateTool{quotes: &stubQuoteReader{}, shipments: &stubShipmentRating{record: &shipment.Shipment{}}}
	explanation := explain(t, tool, map[string]any{"shipmentId": pulid.MustNew("shp_").String()})

	assert.Nil(t, explanation.PricedBy)
	assert.Contains(t, explanation.Note, "no rating on record")
}

/*
The trace prices what the agreement priced. A fuel surcharge program's charge
sits on the shipment beside it, and reading only the trace reported the
lifecycle load's $222.56 fuel surcharge as unpriced against a $1,121.36 total.
*/
func TestExplainRate_ListsTheChargesTheAgreementDidNotPrice(t *testing.T) {
	t.Parallel()

	quote := ratedQuote(fullTrace())
	quote.ID = pulid.MustNew("rq_")
	program := pulid.MustNew("fsp_")
	record := &shipment.Shipment{
		TotalChargeAmount: decimal.NewNullDecimal(usd("1121.36")),
		AdditionalCharges: []*shipment.AdditionalCharge{
			{
				Amount:                 usd("222.56"),
				FuelSurchargeProgramID: &program,
				AccessorialCharge:      &accessorialcharge.AccessorialCharge{Code: "FUEL", Description: "Fuel Surcharge"},
			},
			{Amount: usd("75"), RateQuoteID: &quote.ID},
		},
	}
	tool := &explainRateTool{quotes: &stubQuoteReader{quote: quote}, shipments: &stubShipmentRating{record: record}}
	explanation := explain(t, tool, map[string]any{"shipmentId": pulid.MustNew("shp_").String()})

	require.Len(t, explanation.OnShipment, 1, "a charge the quote produced is already in its totals")
	assert.Equal(t, "FUEL", explanation.OnShipment[0].Code)
	assert.Equal(t, "fuel surcharge program", explanation.OnShipment[0].Source)
	require.NotNil(t, explanation.ShipmentTotal)
	assert.True(t, explanation.ShipmentTotal.Equal(usd("1121.36")))
	assert.NotEmpty(t, explanation.ChargesNote)
}
