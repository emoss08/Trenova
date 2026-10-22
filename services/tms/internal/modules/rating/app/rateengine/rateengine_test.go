package rateengine

import (
	"github.com/emoss08/trenova/shared/i18n"
	"context"
	"sync"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/internal/modules/operations/ports/tenderports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/places/domain/rategeo"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratequote"
	"github.com/emoss08/trenova/internal/modules/rating/ports/rateagreementports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratequoteports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/tariffports"
	"github.com/emoss08/trenova/internal/modules/rating/ratingtest/tarifftest"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/modules/reference/ports/exchangerateports"
	"github.com/emoss08/trenova/internal/testutil/mocks/accessorialchargeportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/exchangerateportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/fuelsurchargeportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/rateagreementportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/ratequoteportsmock"
	"github.com/emoss08/trenova/pkg/pagination"
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

const testNow = int64(1_700_000_000)

type deps struct {
	agreements   *rateagreementportsmock.MockRateAgreementRepository
	quotes       *ratequoteportsmock.MockRateQuoteRepository
	fuel         *fuelsurchargeportsmock.MockFuelSurchargeResolver
	accessorials *accessorialchargeportsmock.MockAccessorialChargeRepository
	exchange     *exchangerateportsmock.MockExchangeRateService
	guides       *stubRoutingGuides
	templates    *tarifftest.Templates
	evaluator    *tarifftest.Evaluator
	svc          *Service

	known sync.Map
}

type stubRoutingGuides struct {
	tenderports.RoutingGuideRepository

	guide *tender.RoutingGuide
	err   error
	calls int
}

func (s *stubRoutingGuides) MatchLane(
	_ context.Context,
	_ *tenderports.MatchLaneRequest,
) (*tender.RoutingGuide, error) {
	s.calls++

	return s.guide, s.err
}

var (
	orgID  = pulid.MustNew("org_")
	buID   = pulid.MustNew("bu_")
	custID = pulid.MustNew("cus_")

	perMileTemplateID     = pulid.ID("ft_per_mile_stub")
	flatTemplateID        = pulid.ID("ft_flat_rate_stub")
	accessorialTemplateID = pulid.ID("ft_accessorial_stub")
	fuelChargeID          = pulid.ID("acc_fuel_stub")
	stopChargeID          = pulid.ID("acc_stop_stub")
)

const flatSource = `tariff "Flat Rate"

/// The amount the shipment pays.
input rate: Money
/// The least the shipment pays.
input minimumCharge: Money = $0

charge "Flat Rate" = rate

minimum minimumCharge
`

const accessorialSource = `tariff "Linehaul with fuel and stops"

/// The rate per mile.
input rate: Money/mi

charge "Linehaul" (quantity: shipment.distance, rate: rate)
charge "Fuel surcharge" as FSC = $100
charge "Extra stop" as STOP = $50
`

func flatTemplate() *tariffports.Template {
	return &tariffports.Template{
		TemplateID:    flatTemplateID,
		VersionNumber: 2,
		Name:          "Flat Rate",
		Source:        flatSource,
		Inputs: []tarifftypes.InputDecl{
			{Name: "rate", Descriptor: rules.RawJSON(`{"kind":"money"}`), Required: true},
			{Name: "minimumCharge", Descriptor: rules.RawJSON(`{"kind":"money"}`)},
		},
	}
}

func accessorialTemplate() *tariffports.Template {
	return &tariffports.Template{
		TemplateID:    accessorialTemplateID,
		VersionNumber: 1,
		Name:          "Linehaul with fuel and stops",
		Source:        accessorialSource,
		Inputs: []tarifftypes.InputDecl{
			{
				Name:       "rate",
				Descriptor: rules.RawJSON(`{"kind":"quantity","unit":"$/mi"}`),
				Required:   true,
			},
		},
	}
}

func setup(t *testing.T) *deps {
	t.Helper()

	d := &deps{
		agreements:   rateagreementportsmock.NewMockRateAgreementRepository(t),
		quotes:       ratequoteportsmock.NewMockRateQuoteRepository(t),
		fuel:         fuelsurchargeportsmock.NewMockFuelSurchargeResolver(t),
		accessorials: accessorialchargeportsmock.NewMockAccessorialChargeRepository(t),
		exchange:     exchangerateportsmock.NewMockExchangeRateService(t),
		guides:       &stubRoutingGuides{},
		templates: tarifftest.NewTemplates(
			tarifftest.PerMile(),
			flatTemplate(),
			accessorialTemplate(),
		),
		evaluator: &tarifftest.Evaluator{},
	}

	d.svc = &Service{
		l:             zap.NewNop(),
		agreementRepo: d.agreements,
		quoteRepo:     d.quotes,
		resolver:      d.templates,
		rater:         tarifftest.NewRater(&tarifftest.StaticContext{}, d.evaluator),
		records:       tarifftest.Records{},
		fuel:          d.fuel,
		accessorials:  d.accessorials,
		exchange:      d.exchange,
		routingGuides: d.guides,
		now:           func() int64 { return testNow },
	}

	d.agreements.EXPECT().GetByID(mock.Anything, mock.Anything).
		RunAndReturn(func(
			_ context.Context,
			req *rateagreementports.GetRateAgreementByIDRequest,
		) (*rateagreement.RateAgreement, error) {
			if known, ok := d.known.Load(req.RateAgreementID); ok {
				return known.(*rateagreement.RateAgreement), nil //nolint:errcheck // always an agreement
			}
			return &rateagreement.RateAgreement{
				ID:            req.RateAgreementID,
				Code:          "ACME-2026",
				Name:          "Acme Foods 2026",
				Currency:      "USD",
				Status:        rateagreement.StatusActive,
				EffectiveFrom: 1,
				Inputs:        tarifftypes.Inputs{},
			}, nil
		}).Maybe()

	return d
}

func tenantInfo() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: orgID, BuID: buID}
}

func testShipment() *shipment.Shipment {
	return &shipment.Shipment{
		ID:             pulid.MustNew("shp_"),
		OrganizationID: orgID,
		BusinessUnitID: buID,
		CustomerID:     custID,
		Status:         shipment.StatusNew,
		CreatedAt:      testNow,
		Weight:         new(int64(40000)),
		Pieces:         new(int64(22)),
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
				Loaded:   true,
				Distance: new(720.0),
				Stops: []*shipment.Stop{
					{
						ID:         pulid.MustNew("stp_"),
						Sequence:   0,
						Type:       shipment.StopTypePickup,
						Status:     shipment.StopStatusNew,
						LocationID: pulid.MustNew("loc_"),
					},
					{
						ID:         pulid.MustNew("stp_"),
						Sequence:   1,
						Type:       shipment.StopTypeDelivery,
						Status:     shipment.StopStatusNew,
						LocationID: pulid.MustNew("loc_"),
					},
				},
			},
		},
	}
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
		FormulaTemplateID:    perMileTemplateID,
		Inputs: tarifftypes.Inputs{
			"rate": rules.QuantityValue(decimal.RequireFromString(rate), "$/mi"),
		},
		SpecificityScore: specificity,
		EffectiveFrom:    1,
		Agreement: &rateagreement.RateAgreement{
			ID:            agreementID,
			Code:          "ACME-2026",
			Name:          "Acme Foods 2026",
			Currency:      "USD",
			Status:        rateagreement.StatusActive,
			EffectiveFrom: 1,
		},
	}
}

func rateRequest(entity *shipment.Shipment) *ratequoteports.RateShipmentRequest {
	return &ratequoteports.RateShipmentRequest{
		Shipment:   entity,
		TenantInfo: tenantInfo(),
		PartyType:  rateagreement.PartyTypeCustomer,
		Purpose:    ratequote.PurposeRating,
	}
}

func expectRules(d *deps, found ...*rateagreement.RateAgreementRule) {
	for _, rule := range found {
		if rule.Agreement != nil {
			d.known.LoadOrStore(rule.RateAgreementID, rule.Agreement)
		}
	}
	d.agreements.EXPECT().ResolveRules(mock.Anything, mock.Anything).
		Return(&rateagreementports.ResolveRateRulesResult{
			Rules: found,
			Total: len(found),
		}, nil).Once()
}

func expectAgreement(d *deps, agreement *rateagreement.RateAgreement) {
	d.known.Store(agreement.ID, agreement)
}

func TestPerMileRulePricesDistanceThroughTheTariff(t *testing.T) {
	t.Parallel()

	d := setup(t)
	expectRules(d, perMileRule("2.15", 100))

	result, err := d.svc.RateShipment(t.Context(), rateRequest(testShipment()))

	require.NoError(t, err)
	assert.Equal(t, ratequote.OutcomeRated, result.Outcome)
	assert.True(t, result.Amount.Equal(decimal.RequireFromString("1548")),
		"amount was %s", result.Amount)
	assert.True(t, result.Total.Equal(result.Amount))
	require.NotNil(t, result.Bill)
	require.Len(t, result.Bill.FreightLines(), 1)
	assert.EqualValues(t, 1, d.evaluator.Calls(), "one module call rates the shipment")
	require.NotNil(t, result.FormulaTemplateID)
	assert.Equal(t, perMileTemplateID, *result.FormulaTemplateID)
	assert.EqualValues(t, 3, result.TemplateVersion)
	assert.Equal(t, "Per Mile", result.FormulaTemplateName)
}

func TestRuleInputsOverrideTheAgreementsAndUndeclaredOnesAreDropped(t *testing.T) {
	t.Parallel()

	d := setup(t)
	rule := perMileRule("0.10", 100)
	expectAgreement(d, &rateagreement.RateAgreement{
		ID:       rule.RateAgreementID,
		Currency: "USD",
		Inputs: tarifftypes.Inputs{
			"minimumCharge": rules.MoneyValue(decimal.NewFromInt(500)),
			"rate":          rules.QuantityValue(decimal.NewFromInt(9), "$/mi"),
			"unused":        rules.NumberValue(decimal.NewFromInt(1)),
		},
	})
	expectRules(d, rule)

	result, err := d.svc.RateShipment(t.Context(), rateRequest(testShipment()))

	require.NoError(t, err)
	assert.True(t, result.Total.Equal(decimal.NewFromInt(500)), "total was %s", result.Total)
	assert.Contains(t, result.TariffInputs, "minimumCharge")
	assert.NotContains(t, result.TariffInputs, "unused")
	rate, err := rules.DecodeQuantity(result.TariffInputs["rate"])
	require.NoError(t, err)
	assert.True(t, rate.Value.Equal(decimal.RequireFromString("0.10")), "the rule's rate wins")

	kinds := make([]tarifftypes.LineKind, 0, len(result.Bill.Lines))
	for _, line := range result.Bill.Lines {
		kinds = append(kinds, line.Kind)
	}
	assert.Contains(t, kinds, tarifftypes.LineMinimum)
	assert.Contains(t, result.Quote.Trace.Components[len(result.Quote.Trace.Components)-1].Kind,
		ratetypes.ComponentKindMinimumCharge)
}

func TestConditionsAreEvaluatedInTheSameModuleCall(t *testing.T) {
	t.Parallel()

	d := setup(t)
	rule := perMileRule("2.00", 100)
	holds := &rateagreement.RateAgreementAccessorial{
		ID:                  pulid.MustNew("raga_"),
		AccessorialChargeID: stopChargeID,
		Method:              accessorialcharge.MethodFlat,
		Amount:              decimal.NewFromInt(75),
		AutoApply:           true,
		Condition:           "condition \"Two stops\"\n\nshipment.stops.count() >= 2\n",
	}
	fails := &rateagreement.RateAgreementAccessorial{
		ID:                  pulid.MustNew("raga_"),
		AccessorialChargeID: stopChargeID,
		Method:              accessorialcharge.MethodFlat,
		Amount:              decimal.NewFromInt(75),
		AutoApply:           true,
		Condition:           "condition \"Five stops\"\n\nshipment.stops.count() >= 5\n",
	}
	expectAgreement(d, &rateagreement.RateAgreement{
		ID:           rule.RateAgreementID,
		Currency:     "USD",
		Accessorials: []*rateagreement.RateAgreementAccessorial{holds, fails},
	})
	expectRules(d, rule)

	result, err := d.svc.RateShipment(t.Context(), rateRequest(testShipment()))

	require.NoError(t, err)
	assert.EqualValues(t, 1, d.evaluator.Calls())
	require.Len(t, d.evaluator.Requests()[0].Programs, 3, "the tariff and both conditions")
	assert.True(t, result.Conditions[holds.ID])
	assert.False(t, result.Conditions[fails.ID])
	require.NotNil(t, result.Agreement)
}

func TestCodedLinesSplitIntoFuelAndAccessorialTotals(t *testing.T) {
	t.Parallel()

	d := setup(t)
	rule := perMileRule("2.00", 100)
	rule.FormulaTemplateID = accessorialTemplateID
	expectRules(d, rule)
	d.fuel.EXPECT().FuelChargeIDs(mock.Anything, mock.Anything).
		Return([]pulid.ID{fuelChargeID}, nil).Once()
	d.accessorials.EXPECT().ListCatalog(mock.Anything, mock.Anything).
		Return([]*accessorialcharge.AccessorialCharge{
			{ID: fuelChargeID, Code: "FSC"},
			{ID: stopChargeID, Code: "STOP"},
		}, nil).Once()

	result, err := d.svc.RateShipment(t.Context(), rateRequest(testShipment()))

	require.NoError(t, err)
	totals := result.Quote.Trace.Totals
	assert.True(t, totals.Linehaul.Equal(decimal.NewFromInt(1440)), "linehaul %s", totals.Linehaul)
	assert.True(t, totals.Fuel.Equal(decimal.NewFromInt(100)), "fuel %s", totals.Fuel)
	assert.True(t, totals.Accessorial.Equal(decimal.NewFromInt(50)), "other %s", totals.Accessorial)
	assert.True(t, result.Total.Equal(decimal.NewFromInt(1590)))
	assert.True(t, result.Amount.Equal(decimal.NewFromInt(1440)))
}

func TestARefusalIsRecordedAsAnErrorWithTheAuthorsText(t *testing.T) {
	t.Parallel()

	d := setup(t)
	rule := perMileRule("2.00", 100)
	rule.Inputs = tarifftypes.Inputs{}
	expectRules(d, rule)

	result, err := d.svc.RateShipment(t.Context(), rateRequest(testShipment()))

	require.NoError(t, err)
	assert.Equal(t, ratequote.OutcomeError, result.Outcome)
	assert.True(t, result.Amount.IsZero())
	assert.NotEmpty(t, result.Quote.Trace.Error)
}

func TestARefusalTheModuleWroteIsInTheReadersLanguage(t *testing.T) {
	t.Parallel()

	d := setup(t)
	rule := perMileRule("2.00", 100)
	rule.Inputs = tarifftypes.Inputs{}
	expectRules(d, rule)

	ctx := i18n.WithLocale(t.Context(), i18n.ES)
	result, err := d.svc.RateShipment(ctx, rateRequest(testShipment()))

	require.NoError(t, err)
	assert.Equal(t, "La entrada rate no tiene valor ni valor predeterminado", result.Quote.Trace.Error)
}

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
	trace := result.Quote.Trace
	require.Len(t, trace.Candidates, 2)

	won := trace.Winner()
	require.NotNil(t, won)
	assert.Equal(t, winner.ID.String(), won.RuleID)
	assert.Equal(t, "Chicago to Atlanta", won.RuleLabel)
	assert.NotEmpty(t, won.MatchedOn)

	assert.False(t, trace.Candidates[1].Won)
	assert.Equal(t, ratetypes.RejectReasonLostOnSpecificity, trace.Candidates[1].RejectReason)
	assert.Equal(t, "specificity 900", trace.TieBreak)
	assert.Equal(t, perMileTemplateID.String(), trace.TemplateID)
	assert.NotEmpty(t, trace.Statements)
	require.NotNil(t, trace.Bill)
}

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
	assert.Equal(t, ratetypes.RejectReasonWeightOutOfRange, trace.Candidates[0].RejectReason)
	assert.Equal(t,
		"The shipment weight falls outside the rate's range",
		trace.Candidates[0].RejectDetail,
	)
}

func TestNoAgreementFallsBackToTheShipmentsOwnTariffAndInputs(t *testing.T) {
	t.Parallel()

	d := setup(t)
	expectRules(d)

	entity := testShipment()
	entity.FormulaTemplateID = flatTemplateID
	entity.TariffInputs = tarifftypes.Inputs{
		"rate": rules.MoneyValue(decimal.RequireFromString("1234.56")),
	}

	result, err := d.svc.RateShipment(t.Context(), rateRequest(entity))

	require.NoError(t, err)
	assert.Equal(t, ratequote.OutcomeFormulaFallback, result.Outcome)
	assert.True(t, result.Amount.Equal(decimal.RequireFromString("1234.56")))
	require.NotNil(t, result.FormulaTemplateID)
	assert.Equal(t, flatTemplateID, *result.FormulaTemplateID)
	require.NotNil(t, result.Quote.FormulaTemplateVersionNumber)
	assert.EqualValues(t, 2, *result.Quote.FormulaTemplateVersionNumber)
}

func TestTheBillingFallbackTemplateRunsWithItsDefaultsOnly(t *testing.T) {
	t.Parallel()

	d := setup(t)
	expectRules(d)

	fallback := flatTemplateID
	entity := testShipment()
	entity.FormulaTemplateID = pulid.Nil
	entity.TariffInputs = tarifftypes.Inputs{"rate": rules.MoneyValue(decimal.NewFromInt(999))}

	req := rateRequest(entity)
	req.BillingControl = &tenant.BillingControl{FallbackFormulaTemplateID: &fallback}

	result, err := d.svc.RateShipment(t.Context(), req)

	require.Error(t, err, "the flat template's rate has no default, so it refuses")
	assert.Nil(t, result)
}

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

	reversedDeps := setup(t)
	expectRules(reversedDeps, first, second)
	reversed, err := reversedDeps.svc.RateShipment(t.Context(), rateRequest(testShipment()))
	require.NoError(t, err)

	assert.Equal(t, forward.RuleID.String(), reversed.RuleID.String())
	assert.True(t, forward.Amount.Equal(reversed.Amount))
}

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
	assert.Contains(t, result.Quote.Trace.TariffInputs, "rate")
}

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

func TestAForeignCurrencyRuleIsPricedThroughTheExchangeRate(t *testing.T) {
	t.Parallel()

	d := setup(t)
	rule := perMileRule("2.00", 100)
	rule.Currency = "CAD"
	expectRules(d, rule)
	d.exchange.EXPECT().
		Convert(mock.Anything, mock.Anything, "CAD", "USD", mock.Anything, mock.Anything).
		RunAndReturn(exchangeAt("0.75")).Once()

	result, err := d.svc.RateShipment(t.Context(), rateRequest(testShipment()))

	require.NoError(t, err)
	require.Len(t, d.evaluator.Requests(), 1)
	exchange := d.evaluator.Requests()[0].Programs[0].Exchange
	require.NotNil(t, exchange)
	assert.Equal(t, "USD", exchange.Currency)
	assert.True(t, exchange.Rate.Equal(decimal.RequireFromString("0.75")))
	require.NotNil(t, result.Quote.Trace.FX)
}

func TestCarrierRatingTellsTheTariffItIsTheBuySide(t *testing.T) {
	t.Parallel()

	d := setup(t)
	rule := perMileRule("2.00", 100)
	rule.PartyType = rateagreement.PartyTypeCarrier
	expectRules(d, rule)

	req := rateRequest(testShipment())
	req.PartyType = rateagreement.PartyTypeCarrier
	req.PartyID = custID
	req.SellTotal = decimal.NewNullDecimal(decimal.NewFromInt(2000))

	_, err := d.svc.RateShipment(t.Context(), req)

	require.NoError(t, err)
	record := string(d.evaluator.Requests()[0].Record)
	assert.Contains(t, record, `"mode":"Buy"`)
	assert.Contains(t, record, `"sellTotal":{"$q":"2000","u":"$"}`)
}

func exchangeAt(rate string) func(
	context.Context,
	pagination.TenantInfo,
	string,
	string,
	decimal.Decimal,
	time.Time,
) (*exchangerateports.RateConversionResult, error) {
	return func(
		_ context.Context,
		_ pagination.TenantInfo,
		from, to string,
		amount decimal.Decimal,
		_ time.Time,
	) (*exchangerateports.RateConversionResult, error) {
		value := decimal.RequireFromString(rate)
		return &exchangerateports.RateConversionResult{
			FromCurrency: from,
			ToCurrency:   to,
			Rate:         value,
			Amount:       amount,
			Converted:    amount.Mul(value),
		}, nil
	}
}
