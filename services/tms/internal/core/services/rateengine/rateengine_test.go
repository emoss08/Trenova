package rateengine

import (
	"context"
	"fmt"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/rateagreement"
	"github.com/emoss08/trenova/internal/core/domain/rategeo"
	"github.com/emoss08/trenova/internal/core/domain/ratematrix"
	"github.com/emoss08/trenova/internal/core/domain/ratequote"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tender"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/formulatemplatetypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/ratetypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const testNow = int64(1_700_000_000)

type deps struct {
	agreements *mocks.MockRateAgreementRepository
	zones      *mocks.MockRateZoneRepository
	matrices   *mocks.MockRateMatrixRepository
	density    *mocks.MockDensityScaleRepository
	quotes     *mocks.MockRateQuoteRepository
	formula    *mocks.MockFormulaCalculator
	exchange   *mocks.MockExchangeRateService
	guides     *stubRoutingGuides
	svc        *Service

	// calcRequests records every request the engine handed the formula
	// calculator, so a test can assert what was bound in rather than only what
	// came out.
	calcRequests []*formulatemplatetypes.CalculateRequest
}

// stubRoutingGuides stands in for the routing guide repository.
//
// It is a stub rather than a generated mock because rating needs exactly one of
// that repository's seven methods, and hand-maintaining the other six would be
// six chances to drift.
type stubRoutingGuides struct {
	repositories.RoutingGuideRepository

	guide *tender.RoutingGuide
	err   error
	calls int
}

func (s *stubRoutingGuides) MatchLane(
	_ context.Context,
	_ *repositories.MatchLaneRequest,
) (*tender.RoutingGuide, error) {
	s.calls++

	return s.guide, s.err
}

func setup(t *testing.T) *deps {
	t.Helper()

	d := &deps{
		agreements: mocks.NewMockRateAgreementRepository(t),
		zones:      mocks.NewMockRateZoneRepository(t),
		matrices:   mocks.NewMockRateMatrixRepository(t),
		density:    mocks.NewMockDensityScaleRepository(t),
		quotes:     mocks.NewMockRateQuoteRepository(t),
		formula:    mocks.NewMockFormulaCalculator(t),
		exchange:   mocks.NewMockExchangeRateService(t),
		guides:     &stubRoutingGuides{},
	}

	d.svc = &Service{
		l:             zap.NewNop(),
		agreementRepo: d.agreements,
		zoneRepo:      d.zones,
		matrixRepo:    d.matrices,
		densityRepo:   d.density,
		quoteRepo:     d.quotes,
		formula:       d.formula,
		exchange:      d.exchange,
		routingGuides: d.guides,
		now:           func() int64 { return testNow },
	}

	// Zone membership is asked for on both ends of every lane, and most of
	// these fixtures do not use zones.
	d.zones.On("ResolveMembership", mock.Anything, mock.Anything).
		Return([]pulid.ID{}, nil).Maybe()

	return d
}

var (
	orgID  = pulid.MustNew("org_")
	buID   = pulid.MustNew("bu_")
	custID = pulid.MustNew("cus_")
)

// The stub template ids are fixed rather than minted, because they end up in
// the trace as component source ids and the golden file has to see the same
// bytes on every run.
var (
	perMileTemplateID = pulid.ID("ft_per_mile_stub")
	flatTemplateID    = pulid.ID("ft_flat_rate_stub")
	perCwtTemplateID  = pulid.ID("ft_per_cwt_stub")
)

// testDistance is the mileage testShipment carries, which the per-mile stub
// template multiplies the bound base rate by.
var testDistance = decimal.NewFromInt(720)

// expectTemplates stands in for the formula engine the way the seeded
// templates behave: the template names the arithmetic, and the engine's
// Overrides carry the numbers bound into it.
func expectTemplates(d *deps) {
	d.formula.EXPECT().Calculate(mock.Anything, mock.Anything).
		RunAndReturn(func(
			_ context.Context,
			req *formulatemplatetypes.CalculateRequest,
		) (*formulatemplatetypes.CalculateResponse, error) {
			d.calcRequests = append(d.calcRequests, req)

			return stubTemplateResponse(req)
		}).Maybe()
}

func stubTemplateResponse(
	req *formulatemplatetypes.CalculateRequest,
) (*formulatemplatetypes.CalculateResponse, error) {
	baseRate := overrideDecimal(req.Overrides, "baseRate")

	switch req.TemplateID {
	case perMileTemplateID:
		return &formulatemplatetypes.CalculateResponse{
			Amount:              baseRate.Mul(testDistance),
			FormulaTemplateID:   perMileTemplateID.String(),
			FormulaTemplateName: "Per Mile",
			Expression:          "baseRate * totalDistance",
		}, nil
	case flatTemplateID:
		return &formulatemplatetypes.CalculateResponse{
			Amount:              baseRate,
			FormulaTemplateID:   flatTemplateID.String(),
			FormulaTemplateName: "Flat Rate",
			Expression:          "baseRate",
		}, nil
	case perCwtTemplateID:
		weight := overrideDecimal(req.Overrides, "totalWeight")

		return &formulatemplatetypes.CalculateResponse{
			Amount:              baseRate.Mul(weight.Div(decimal.NewFromInt(100))),
			FormulaTemplateID:   perCwtTemplateID.String(),
			FormulaTemplateName: "Per CWT",
			Expression:          "baseRate * (totalWeight / 100)",
		}, nil
	default:
		return nil, fmt.Errorf("no stub template for %s", req.TemplateID)
	}
}

func overrideDecimal(overrides map[string]any, key string) decimal.Decimal {
	value, ok := overrides[key].(float64)
	if !ok {
		return decimal.Zero
	}

	return decimal.NewFromFloat(value)
}

func tenantInfo() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: orgID, BuID: buID}
}

// testShipment is a straightforward truckload: one move, two stops, 720 miles,
// forty thousand pounds.
func testShipment() *shipment.Shipment {
	originLocation := pulid.MustNew("loc_")
	destinationLocation := pulid.MustNew("loc_")

	return &shipment.Shipment{
		ID:             pulid.MustNew("shp_"),
		OrganizationID: orgID,
		BusinessUnitID: buID,
		CustomerID:     custID,
		CreatedAt:      testNow,
		Weight:         ptr(int64(40000)),
		Pieces:         ptr(int64(22)),
		Customer: &customer.Customer{
			ID:   custID,
			Name: "Acme Foods",
			BillingProfile: &customer.CustomerBillingProfile{
				BillingCurrency: "USD",
			},
		},
		Moves: []*shipment.ShipmentMove{
			{
				Sequence: 0,
				Distance: ptr(720.0),
				Stops: []*shipment.Stop{
					{
						ID:         pulid.MustNew("stp_"),
						Sequence:   0,
						Type:       shipment.StopTypePickup,
						LocationID: originLocation,
					},
					{
						ID:         pulid.MustNew("stp_"),
						Sequence:   1,
						Type:       shipment.StopTypeDelivery,
						LocationID: destinationLocation,
					},
				},
			},
		},
	}
}

func ptr[T any](value T) *T {
	return &value
}

func perMileRule(rate string, specificity int32) *rateagreement.RateAgreementRule {
	agreementID := pulid.MustNew("rag_")

	return &rateagreement.RateAgreementRule{
		ID:                   pulid.MustNew("ragr_"),
		OrganizationID:       orgID,
		BusinessUnitID:       buID,
		RateAgreementID:      agreementID,
		PartyType:            rateagreement.PartyTypeCustomer,
		PartyID:              custID,
		Status:               rateagreement.RuleStatusActive,
		Direction:            rateagreement.DirectionDirectional,
		OriginScopeType:      rategeo.ScopeTypeAny,
		DestinationScopeType: rategeo.ScopeTypeAny,
		FormulaTemplateID:    &perMileTemplateID,
		Rate:                 decimal.NewNullDecimal(decimal.RequireFromString(rate)),
		SpecificityScore:     specificity,
		EffectiveFrom:        1,
		Agreement: &rateagreement.RateAgreement{
			ID:                agreementID,
			Code:              "ACME-2026",
			Name:              "Acme Foods 2026",
			Currency:          "USD",
			RoundingPrecision: 2,
			Status:            rateagreement.StatusActive,
			EffectiveFrom:     1,
		},
	}
}

func rateRequest(entity *shipment.Shipment) *services.RateShipmentRequest {
	return &services.RateShipmentRequest{
		Shipment:   entity,
		TenantInfo: tenantInfo(),
		PartyType:  rateagreement.PartyTypeCustomer,
		Purpose:    ratequote.PurposeRating,
	}
}

func expectRules(d *deps, rules ...*rateagreement.RateAgreementRule) {
	d.agreements.On("ResolveRules", mock.Anything, mock.Anything).
		Return(&repositories.ResolveRateRulesResult{
			Rules: rules,
			Total: len(rules),
		}, nil).Once()

	if len(rules) > 0 {
		expectTemplates(d)
	}
}

func TestPerMileRatePricesDistance(t *testing.T) {
	t.Parallel()

	d := setup(t)
	expectRules(d, perMileRule("2.15", 100))

	result, err := d.svc.RateShipment(t.Context(), rateRequest(testShipment()))

	require.NoError(t, err)
	assert.Equal(t, ratequote.OutcomeRated, result.Outcome)
	// 720 mi at $2.15 is $1,548.
	assert.True(t, result.Amount.Equal(decimal.RequireFromString("1548")),
		"amount was %s", result.Amount)
}

// The trace has to name the winner, the rule that lost and why it lost. That
// list is the whole point of writing a quote.
func TestTraceRecordsWinnerAndLosers(t *testing.T) {
	t.Parallel()

	d := setup(t)

	winner := perMileRule("2.15", 900)
	winner.Label = "Chicago to Atlanta"
	loser := perMileRule("2.85", 100)
	loser.Label = "Illinois to Georgia"

	expectRules(d, winner, loser)

	result, err := d.svc.RateShipment(t.Context(), rateRequest(testShipment()))

	require.NoError(t, err)
	require.NotNil(t, result.Quote)
	require.NotNil(t, result.Quote.Trace)

	trace := result.Quote.Trace
	require.Len(t, trace.Candidates, 2)

	won := trace.Winner()
	require.NotNil(t, won)
	assert.Equal(t, winner.ID.String(), won.RuleID)
	assert.Equal(t, "Chicago to Atlanta", won.RuleLabel)
	assert.NotEmpty(t, won.MatchedOn)

	assert.False(t, trace.Candidates[1].Won)
	assert.Equal(t,
		ratetypes.RejectReasonLostOnSpecificity,
		trace.Candidates[1].RejectReason,
	)
	assert.Equal(t, "specificity 900", trace.TieBreak)
}

// A rule that did not apply must say why in terms a person can act on, not
// merely that it did not apply.
func TestTraceExplainsWhyARuleDidNotApply(t *testing.T) {
	t.Parallel()

	d := setup(t)

	applies := perMileRule("2.15", 100)
	tooLight := perMileRule("1.10", 900)
	tooLight.MaxWeight = decimal.NewNullDecimal(decimal.RequireFromString("10000"))

	expectRules(d, tooLight, applies)

	result, err := d.svc.RateShipment(t.Context(), rateRequest(testShipment()))

	require.NoError(t, err)
	assert.True(t, result.Amount.Equal(decimal.RequireFromString("1548")))

	trace := result.Quote.Trace
	require.Len(t, trace.Candidates, 2)
	assert.Equal(t,
		ratetypes.RejectReasonWeightOutOfRange,
		trace.Candidates[0].RejectReason,
	)
	assert.Equal(t,
		"The shipment weight falls outside the rate's range",
		trace.Candidates[0].RejectDetail,
	)
}

func TestMinimumChargeIsRecordedWhenItLifts(t *testing.T) {
	t.Parallel()

	d := setup(t)

	rule := perMileRule("0.10", 100)
	rule.MinCharge = decimal.NewNullDecimal(decimal.RequireFromString("500"))

	expectRules(d, rule)

	result, err := d.svc.RateShipment(t.Context(), rateRequest(testShipment()))

	require.NoError(t, err)
	// 720 mi at $0.10 is $72, lifted to the $500 floor.
	assert.True(t, result.Amount.Equal(decimal.RequireFromString("500")),
		"amount was %s", result.Amount)

	guardrails := result.Quote.Trace.Guardrails
	require.Len(t, guardrails, 1)
	assert.True(t, guardrails[0].Applied)
	assert.True(t, guardrails[0].Raw.Equal(decimal.RequireFromString("72")))
}

// A short haul on a lane with a mileage guarantee bills the guaranteed miles,
// which is how a carrier is kept whole on a run too short to be worth
// dispatching.
func TestMinimumBillableDistanceBillsTheGuaranteedMiles(t *testing.T) {
	t.Parallel()

	d := setup(t)

	rule := perMileRule("2.00", 100)
	rule.MinBillableDistance = decimal.NewNullDecimal(decimal.RequireFromString("1000"))

	expectRules(d, rule)

	result, err := d.svc.RateShipment(t.Context(), rateRequest(testShipment()))

	require.NoError(t, err)
	// 720 actual miles, billed at the 1,000 mile guarantee.
	assert.True(t, result.Amount.Equal(decimal.RequireFromString("2000")),
		"amount was %s", result.Amount)
}

func TestDiscountAndAbsoluteMinimumApplyInContractOrder(t *testing.T) {
	t.Parallel()

	d := setup(t)

	rule := perMileRule("1.00", 100)
	rule.DiscountPercent = decimal.NewNullDecimal(decimal.RequireFromString("60"))
	rule.AbsoluteMinCharge = decimal.NewNullDecimal(decimal.RequireFromString("400"))

	expectRules(d, rule)

	result, err := d.svc.RateShipment(t.Context(), rateRequest(testShipment()))

	require.NoError(t, err)
	// 720 at $1.00 is $720; less 60% is $288; lifted to the $400 floor.
	assert.True(t, result.Amount.Equal(decimal.RequireFromString("400")),
		"amount was %s", result.Amount)
}

// An organization with no agreements has to be completely unaffected. This is
// the property that makes adopting rate agreements safe.
func TestNoAgreementFallsBackToTheShipmentsFormulaTemplate(t *testing.T) {
	t.Parallel()

	d := setup(t)
	expectRules(d)

	templateID := pulid.MustNew("ft_")
	entity := testShipment()
	entity.FormulaTemplateID = templateID

	d.formula.On("Calculate", mock.Anything, mock.Anything).
		Return(&formulatemplatetypes.CalculateResponse{
			Amount:              decimal.RequireFromString("1234.56"),
			FormulaTemplateID:   templateID.String(),
			FormulaTemplateName: "Legacy per-mile",
			Expression:          "totalDistance * 1.7146",
		}, nil).Once()

	result, err := d.svc.RateShipment(t.Context(), rateRequest(entity))

	require.NoError(t, err)
	assert.Equal(t, ratequote.OutcomeFormulaFallback, result.Outcome)
	assert.True(t, result.Amount.Equal(decimal.RequireFromString("1234.56")))
	require.NotNil(t, result.FormulaTemplateID)
	assert.Equal(t, templateID, *result.FormulaTemplateID)
}

// Two rules that are indistinguishable on every ordering term must resolve the
// same way regardless of the order they came back in.
func TestResolutionIsDeterministic(t *testing.T) {
	t.Parallel()

	first := perMileRule("2.00", 100)
	first.ID = pulid.ID("ragr_00000000000000000000000001")
	second := perMileRule("3.00", 100)
	second.ID = pulid.ID("ragr_00000000000000000000000002")

	forwardDeps := setup(t)
	expectRules(forwardDeps, first, second)
	forward, err := forwardDeps.svc.RateShipment(t.Context(), rateRequest(testShipment()))
	require.NoError(t, err)

	// The repository returns them already ordered, so a reversed set means the
	// engine is picking rather than trusting the order — which it must not.
	reversedDeps := setup(t)
	expectRules(reversedDeps, first, second)
	reversed, err := reversedDeps.svc.RateShipment(t.Context(), rateRequest(testShipment()))
	require.NoError(t, err)

	assert.Equal(t, forward.RuleID.String(), reversed.RuleID.String())
	assert.True(t, forward.Amount.Equal(reversed.Amount))
}

// The context hash is what tells a shipment that changed apart from an engine
// that changed. Equal inputs have to hash equal, and any difference has to
// show.
func TestContextHashIsStableAndSensitive(t *testing.T) {
	t.Parallel()

	d := setup(t)

	base, err := d.svc.buildContext(t.Context(), rateRequest(testShipment()))
	require.NoError(t, err)

	same, err := d.svc.buildContext(t.Context(), rateRequest(testShipment()))
	require.NoError(t, err)
	same.Origin = base.Origin
	same.Destination = base.Destination

	assert.Equal(t, base.Hash(), same.Hash())

	heavier := *base
	heavier.Weight = decimal.RequireFromString("41000")
	assert.NotEqual(t, base.Hash(), heavier.Hash())

	later := *base
	later.AsOf = base.AsOf + 1
	assert.NotEqual(t, base.Hash(), later.Hash())
}

func TestQuoteRecordsTheInputsItRatedOn(t *testing.T) {
	t.Parallel()

	d := setup(t)
	expectRules(d, perMileRule("2.15", 100))

	result, err := d.svc.RateShipment(t.Context(), rateRequest(testShipment()))

	require.NoError(t, err)

	inputs := result.Quote.Trace.Inputs
	assert.True(t, inputs.Distance.Equal(decimal.RequireFromString("720")))
	assert.True(t, inputs.Weight.Equal(decimal.RequireFromString("40000")))
	assert.Equal(t, int64(22), inputs.Pieces)
	assert.Equal(t, int16(2), inputs.Stops)
	assert.Equal(t, custID.String(), inputs.PartyID)
	assert.Equal(t, "Acme Foods", inputs.PartyName)
	assert.Equal(t, EngineVersion, result.Quote.EngineVersion)
	assert.NotEmpty(t, result.Quote.ContextHash)
}

// A what-if answers the question without leaving a record that would compete
// with the shipment's real rate.
func TestUnpersistedRatingNeverWritesAQuote(t *testing.T) {
	t.Parallel()

	d := setup(t)
	expectRules(d, perMileRule("2.15", 100))

	req := rateRequest(testShipment())
	req.Purpose = ratequote.PurposeWhatIf
	req.Persist = false

	result, err := d.svc.RateShipment(t.Context(), req)

	require.NoError(t, err)
	require.NotNil(t, result.Quote)
	assert.Equal(t, ratequote.PurposeWhatIf, result.Quote.Purpose)
	d.quotes.AssertNotCalled(t, "Record", mock.Anything, mock.Anything)
}

// The rule's own rate is not arithmetic the engine performs any more; it is a
// number bound into the template as its base rate. What has to be right is the
// binding.
func TestRuleRateBindsIntoTheTemplateAsItsBaseRate(t *testing.T) {
	t.Parallel()

	d := setup(t)
	expectRules(d, perMileRule("2.15", 100))

	result, err := d.svc.RateShipment(t.Context(), rateRequest(testShipment()))

	require.NoError(t, err)
	require.Len(t, d.calcRequests, 1)

	req := d.calcRequests[0]
	assert.Equal(t, perMileTemplateID, req.TemplateID)
	assert.Equal(t, 2.15, req.Overrides["baseRate"])
	assert.NotContains(t, req.Overrides, "totalWeight")
	assert.NotContains(t, req.Overrides, "sellTotal")

	trace := result.Quote.Trace
	require.NotEmpty(t, trace.Components)

	linehaul := trace.Components[0]
	assert.Equal(t, ratetypes.ComponentSourceFormulaTemplate, linehaul.Source)
	assert.Equal(t, perMileTemplateID.String(), linehaul.SourceID)
	assert.Equal(t, "Per Mile @ 2.15", linehaul.Basis)
	assert.True(t, linehaul.Rate.Valid)
	assert.True(t, linehaul.Rate.Decimal.Equal(decimal.RequireFromString("2.15")))
	assert.Equal(t, "baseRate * totalDistance", linehaul.Detail["expression"])
}

// The quote names the formula template the winning rule priced through, so a
// reader can go from the number to the arithmetic that produced it.
func TestQuoteCarriesTheWinningRulesFormulaTemplate(t *testing.T) {
	t.Parallel()

	d := setup(t)
	expectRules(d, perMileRule("2.15", 100))

	result, err := d.svc.RateShipment(t.Context(), rateRequest(testShipment()))

	require.NoError(t, err)
	require.NotNil(t, result.FormulaTemplateID)
	assert.Equal(t, perMileTemplateID, *result.FormulaTemplateID)
}

func bandedRule(allowDeficit bool) *rateagreement.RateAgreementRule {
	rule := perMileRule("0", 100)
	rule.FormulaTemplateID = &perCwtTemplateID
	rule.Rate = decimal.NullDecimal{}
	rule.AllowDeficitRating = allowDeficit
	rule.Breaks = []*rateagreement.RateAgreementRuleBreak{
		{
			ID:         pulid.MustNew("ragb_"),
			FromWeight: decimal.Zero,
			ToWeight:   decimal.NewNullDecimal(decimal.NewFromInt(20000)),
			Rate:       decimal.NewFromInt(10),
		},
		{
			ID:         pulid.MustNew("ragb_"),
			FromWeight: decimal.NewFromInt(20000),
			ToWeight:   decimal.NewNullDecimal(decimal.NewFromInt(45000)),
			Rate:       decimal.NewFromInt(8),
		},
		{
			ID:         pulid.MustNew("ragb_"),
			FromWeight: decimal.NewFromInt(45000),
			Rate:       decimal.NewFromInt(6),
		},
	}

	return rule
}

// A banded rule prices at whichever band the weight falls in, and the band's
// rate and the rated weight are what get bound into the template.
func TestWeightBreaksBindTheBandRateAndRatedWeight(t *testing.T) {
	t.Parallel()

	d := setup(t)
	expectRules(d, bandedRule(false))

	result, err := d.svc.RateShipment(t.Context(), rateRequest(testShipment()))

	require.NoError(t, err)
	// 40,000 lb falls in the 20,000-45,000 band at $8/cwt, which is $3,200.
	assert.True(t, result.Amount.Equal(decimal.RequireFromString("3200")),
		"amount was %s", result.Amount)

	require.Len(t, d.calcRequests, 1)
	req := d.calcRequests[0]
	assert.Equal(t, perCwtTemplateID, req.TemplateID)
	assert.Equal(t, 8.0, req.Overrides["baseRate"])
	assert.Equal(t, 40000.0, req.Overrides["totalWeight"])

	linehaul := result.Quote.Trace.Components[0]
	assert.Equal(t, "Per CWT @ 8", linehaul.Basis)
	breakDetail, ok := linehaul.Detail["weightBreak"].(map[string]any)
	require.True(t, ok, "linehaul detail should carry the weight break")
	assert.Equal(t, "20000-45000", breakDetail["label"])
	assert.Equal(t, "40000", breakDetail["ratedWeight"])
	assert.Empty(t, result.Quote.Trace.Warnings)
}

// A shipment near the top of its band can be cheaper declared at the bottom of
// the next one up, and the carrier lets the shipper pay the lower of the two.
// The trace has to say the billed weight is not the shipped weight, because
// that is the first thing questioned on the invoice.
func TestDeficitRatingBumpsToTheCheaperBand(t *testing.T) {
	t.Parallel()

	d := setup(t)
	expectRules(d, bandedRule(true))

	result, err := d.svc.RateShipment(t.Context(), rateRequest(testShipment()))

	require.NoError(t, err)
	// 40,000 lb at $8/cwt is $3,200; declared at 45,000 lb the $6/cwt band
	// charges $2,700, so the shipment is bumped.
	assert.True(t, result.Amount.Equal(decimal.RequireFromString("2700")),
		"amount was %s", result.Amount)

	require.Len(t, d.calcRequests, 1)
	req := d.calcRequests[0]
	assert.Equal(t, 6.0, req.Overrides["baseRate"])
	assert.Equal(t, 45000.0, req.Overrides["totalWeight"])

	trace := result.Quote.Trace
	require.NotEmpty(t, trace.Warnings)
	assert.Contains(t, trace.Warnings[0], "45000")
	assert.Contains(t, trace.Warnings[0], "40000")

	breakDetail, ok := trace.Components[0].Detail["weightBreak"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "45000+", breakDetail["label"])
	assert.Equal(t, "45000", breakDetail["ratedWeight"])
	assert.Equal(t, "40000", breakDetail["actualWeight"])
}

// A band's own minimum charge is a floor on what the band's rate produces,
// applied before anything else touches the amount.
func TestBandMinimumChargeClampsTheTemplatesAnswer(t *testing.T) {
	t.Parallel()

	d := setup(t)

	rule := perMileRule("0", 100)
	rule.FormulaTemplateID = &perCwtTemplateID
	rule.Rate = decimal.NullDecimal{}
	rule.Breaks = []*rateagreement.RateAgreementRuleBreak{
		{
			ID:         pulid.MustNew("ragb_"),
			FromWeight: decimal.Zero,
			Rate:       decimal.NewFromInt(1),
			MinCharge:  decimal.NewNullDecimal(decimal.NewFromInt(500)),
		},
	}

	expectRules(d, rule)

	result, err := d.svc.RateShipment(t.Context(), rateRequest(testShipment()))

	require.NoError(t, err)
	// 40,000 lb at $1/cwt is $400, lifted to the band's $500 floor.
	assert.True(t, result.Amount.Equal(decimal.RequireFromString("500")),
		"amount was %s", result.Amount)

	guardrails := result.Quote.Trace.Guardrails
	require.Len(t, guardrails, 1)
	assert.True(t, guardrails[0].Applied)
	assert.True(t, guardrails[0].Raw.Equal(decimal.RequireFromString("400")))
	assert.True(t, guardrails[0].Bound.Equal(decimal.RequireFromString("500")))
}

// distanceMatrix is a one-axis tariff banded by mileage, whose cells mean
// whatever the matrix's own template does with them.
func distanceMatrix(cellValue string) (*ratematrix.RateMatrix, *ratematrix.RateMatrixCell) {
	matrixID := pulid.MustNew("rmx_")

	matrix := &ratematrix.RateMatrix{
		ID:                matrixID,
		OrganizationID:    orgID,
		BusinessUnitID:    buID,
		Code:              "DIST",
		Name:              "Distance bands",
		Currency:          "USD",
		FormulaTemplateID: perMileTemplateID,
		Dimensions: []*ratematrix.RateMatrixDimension{
			{
				ID:           pulid.MustNew("rmd_"),
				RateMatrixID: matrixID,
				Position:     0,
				Kind:         ratematrix.DimensionKindDistance,
				MatchMode:    ratematrix.MatchModeRange,
			},
		},
	}

	cell := &ratematrix.RateMatrixCell{
		ID:           pulid.MustNew("rmc_"),
		RateMatrixID: matrixID,
		D0Min:        decimal.NewNullDecimal(decimal.Zero),
		D0Max:        decimal.NewNullDecimal(decimal.NewFromInt(1000)),
		Value:        decimal.RequireFromString(cellValue),
	}

	return matrix, cell
}

func matrixRule(matrix *ratematrix.RateMatrix) *rateagreement.RateAgreementRule {
	rule := perMileRule("0", 100)
	rule.FormulaTemplateID = nil
	rule.Rate = decimal.NullDecimal{}
	rule.RateMatrixID = &matrix.ID

	return rule
}

func expectMatrix(
	d *deps,
	matrix *ratematrix.RateMatrix,
	cells ...*ratematrix.RateMatrixCell,
) {
	d.matrices.On("GetByID", mock.Anything, mock.Anything).
		Return(matrix, nil).Once()
	d.matrices.On("LookupCells", mock.Anything, mock.Anything).
		Return(cells, nil).Once()
}

// A cell is a number, not a meaning: the matrix's template is what turns it
// into money, with the cell's value bound in as the base rate.
func TestMatrixCellValueBindsIntoTheMatrixTemplate(t *testing.T) {
	t.Parallel()

	d := setup(t)

	matrix, cell := distanceMatrix("2.15")
	expectRules(d, matrixRule(matrix))
	expectMatrix(d, matrix, cell)

	result, err := d.svc.RateShipment(t.Context(), rateRequest(testShipment()))

	require.NoError(t, err)
	// The 0-1,000 mile cell reads 2.15, and the matrix's per-mile template
	// prices 720 mi at $2.15, which is $1,548.
	assert.True(t, result.Amount.Equal(decimal.RequireFromString("1548")),
		"amount was %s", result.Amount)

	require.Len(t, d.calcRequests, 1)
	req := d.calcRequests[0]
	assert.Equal(t, perMileTemplateID, req.TemplateID)
	assert.Equal(t, map[string]any{"baseRate": 2.15}, req.Overrides)

	linehaul := result.Quote.Trace.Components[0]
	assert.Equal(t, ratetypes.ComponentSourceRateMatrix, linehaul.Source)
	assert.Equal(t, matrix.ID.String(), linehaul.SourceID)
	assert.Equal(t, "Distance bands", linehaul.SourceName)
	assert.Equal(t, "Per Mile @ 2.15", linehaul.Basis)
	assert.Equal(t, cell.ID.String(), linehaul.Detail["cellId"])
	assert.Equal(t, "Per Mile", linehaul.Detail["formulaTemplate"])

	// A matrix rated rule has no formula template of its own, and the quote
	// must not pretend it does.
	assert.Nil(t, result.FormulaTemplateID)
}

func TestMatrixCellMinimumChargeClampsTheTemplatesAnswer(t *testing.T) {
	t.Parallel()

	d := setup(t)

	matrix, cell := distanceMatrix("0.10")
	cell.MinCharge = decimal.NewNullDecimal(decimal.NewFromInt(200))
	expectRules(d, matrixRule(matrix))
	expectMatrix(d, matrix, cell)

	result, err := d.svc.RateShipment(t.Context(), rateRequest(testShipment()))

	require.NoError(t, err)
	// 720 mi at the cell's $0.10 is $72, lifted to the cell's $200 floor.
	assert.True(t, result.Amount.Equal(decimal.RequireFromString("200")),
		"amount was %s", result.Amount)

	guardrails := result.Quote.Trace.Guardrails
	require.Len(t, guardrails, 1)
	assert.True(t, guardrails[0].Applied)
	assert.True(t, guardrails[0].Raw.Equal(decimal.RequireFromString("72")))
}

// A rule with neither a formula template nor a matrix is a row written around
// the domain. It is recorded as an error on the quote rather than thrown away,
// because the quote is where somebody will look for the contract that needs
// fixing.
func TestRuleWithNoRatingMethodRecordsTheError(t *testing.T) {
	t.Parallel()

	d := setup(t)

	rule := perMileRule("2.15", 100)
	rule.FormulaTemplateID = nil

	expectRules(d, rule)

	result, err := d.svc.RateShipment(t.Context(), rateRequest(testShipment()))

	require.NoError(t, err)
	assert.Equal(t, ratequote.OutcomeError, result.Outcome)
	assert.True(t, result.Amount.IsZero(), "amount was %s", result.Amount)
	require.NotNil(t, result.Quote)
	require.NotNil(t, result.Quote.Trace)
	assert.Equal(t, ErrMissingRate.Error(), result.Quote.Trace.Error)
}
