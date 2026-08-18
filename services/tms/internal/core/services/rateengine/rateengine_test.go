package rateengine

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/rateagreement"
	"github.com/emoss08/trenova/internal/core/domain/rategeo"
	"github.com/emoss08/trenova/internal/core/domain/ratequote"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
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
	svc        *Service
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
		RatingBasis:          rateagreement.RatingBasisPerMile,
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

	base, err := d.svc.buildContext(rateRequest(testShipment()))
	require.NoError(t, err)

	same, err := d.svc.buildContext(rateRequest(testShipment()))
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
