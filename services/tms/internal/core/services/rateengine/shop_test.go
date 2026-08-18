package rateengine

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/carrier"
	"github.com/emoss08/trenova/internal/core/domain/rateagreement"
	"github.com/emoss08/trenova/internal/core/domain/ratequote"
	"github.com/emoss08/trenova/internal/core/domain/tender"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var (
	cheapCarrier  = pulid.MustNew("car_")
	pricyCarrier  = pulid.MustNew("car_")
	silentCarrier = pulid.MustNew("car_")
)

// buyRule prices a carrier's lane at a flat amount, which keeps the arithmetic
// out of the way of what these tests are about: the order.
func buyRule(carrierID pulid.ID, amount string) *rateagreement.RateAgreementRule {
	rule := perMileRule("1", 100)
	rule.PartyType = rateagreement.PartyTypeCarrier
	rule.PartyID = carrierID
	rule.FormulaTemplateID = &flatTemplateID
	rule.Rate = decimal.NewNullDecimal(decimal.RequireFromString(amount))
	rule.Agreement.PartyType = rateagreement.PartyTypeCarrier

	return rule
}

// expectBuyRules answers each carrier's resolution with the rule written for
// it, so one shopping run exercises several contracts.
func expectBuyRules(d *deps, byCarrier map[pulid.ID]*rateagreement.RateAgreementRule) {
	d.agreements.EXPECT().ResolveRules(mock.Anything, mock.Anything).
		RunAndReturn(func(
			_ context.Context,
			req *repositories.ResolveRateRulesRequest,
		) (*repositories.ResolveRateRulesResult, error) {
			for _, partyID := range req.PartyIDs {
				if rule, ok := byCarrier[partyID]; ok {
					return &repositories.ResolveRateRulesResult{
						Rules: []*rateagreement.RateAgreementRule{rule},
						Total: 1,
					}, nil
				}
			}

			return &repositories.ResolveRateRulesResult{}, nil
		})

	expectTemplates(d)
}

func expectGuide(d *deps, entries ...*tender.RoutingGuideEntry) *tender.RoutingGuide {
	guide := &tender.RoutingGuide{
		ID:      pulid.MustNew("rg_"),
		Entries: entries,
	}

	d.guides.guide = guide

	return guide
}

func guideEntry(carrierID pulid.ID, rank int16, ttl int32) *tender.RoutingGuideEntry {
	return &tender.RoutingGuideEntry{
		ID:              pulid.MustNew("rge_"),
		CarrierID:       carrierID,
		Rank:            rank,
		OfferTTLSeconds: ttl,
		Carrier:         &carrier.Carrier{ID: carrierID, Name: "Carrier " + carrierID.String()},
	}
}

func shopRequest(strategy services.ShopStrategy) *services.ShopRequest {
	return &services.ShopRequest{
		Shipment:   testShipment(),
		TenantInfo: tenantInfo(),
		Strategy:   strategy,
		SellTotal:  decimal.NewNullDecimal(decimal.RequireFromString("2000")),
	}
}

// The routing guide is where an organization already wrote down which carriers
// may haul which lane. Shopping reads it rather than asking for the shortlist
// again, so a guide edited for tendering is the same one shopping honours.
func TestShop_TakesItsCandidatesFromTheRoutingGuide(t *testing.T) {
	t.Parallel()

	d := setup(t)
	guide := expectGuide(d,
		guideEntry(cheapCarrier, 1, 900),
		guideEntry(pricyCarrier, 2, 1800),
	)
	expectBuyRules(d, map[pulid.ID]*rateagreement.RateAgreementRule{
		cheapCarrier: buyRule(cheapCarrier, "1400"),
		pricyCarrier: buyRule(pricyCarrier, "1700"),
	})

	result, err := d.svc.Shop(t.Context(), shopRequest(services.ShopStrategyLeastCost))

	require.NoError(t, err)
	require.Len(t, result.Options, 2)
	require.NotNil(t, result.RoutingGuideID)
	assert.Equal(t, guide.ID, *result.RoutingGuideID)
}

func TestShop_LeastCostPutsTheCheapestFirst(t *testing.T) {
	t.Parallel()

	d := setup(t)
	expectGuide(d,
		guideEntry(pricyCarrier, 1, 1800),
		guideEntry(cheapCarrier, 2, 900),
	)
	expectBuyRules(d, map[pulid.ID]*rateagreement.RateAgreementRule{
		cheapCarrier: buyRule(cheapCarrier, "1400"),
		pricyCarrier: buyRule(pricyCarrier, "1700"),
	})

	result, err := d.svc.Shop(t.Context(), shopRequest(services.ShopStrategyLeastCost))

	require.NoError(t, err)
	require.Len(t, result.Options, 2)
	assert.Equal(t, cheapCarrier, result.Options[0].CarrierID)
	assert.Equal(t, 1, result.Options[0].Rank)
	assert.Equal(t, 2, result.Options[1].Rank)
}

// A committed primary carrier is offered the load first whatever the spot
// market says, which is the entire point of having written the guide.
func TestShop_GuideRankKeepsTheGuidesOwnOrder(t *testing.T) {
	t.Parallel()

	d := setup(t)
	expectGuide(d,
		guideEntry(pricyCarrier, 1, 1800),
		guideEntry(cheapCarrier, 2, 900),
	)
	expectBuyRules(d, map[pulid.ID]*rateagreement.RateAgreementRule{
		cheapCarrier: buyRule(cheapCarrier, "1400"),
		pricyCarrier: buyRule(pricyCarrier, "1700"),
	})

	result, err := d.svc.Shop(t.Context(), shopRequest(services.ShopStrategyGuideRank))

	require.NoError(t, err)
	assert.Equal(t, pricyCarrier, result.Options[0].CarrierID)
	assert.Equal(t, int16(1), result.Options[0].GuideRank)
}

func TestShop_FastestAcceptPutsTheShortestOfferFirst(t *testing.T) {
	t.Parallel()

	d := setup(t)
	expectGuide(d,
		guideEntry(pricyCarrier, 1, 1800),
		guideEntry(cheapCarrier, 2, 900),
	)
	expectBuyRules(d, map[pulid.ID]*rateagreement.RateAgreementRule{
		cheapCarrier: buyRule(cheapCarrier, "1400"),
		pricyCarrier: buyRule(pricyCarrier, "1700"),
	})

	result, err := d.svc.Shop(t.Context(), shopRequest(services.ShopStrategyFastestAccept))

	require.NoError(t, err)
	assert.Equal(t, cheapCarrier, result.Options[0].CarrierID)
	assert.Equal(t, int32(900), result.Options[0].OfferTTLSeconds)
}

// Margin is what the organization keeps, and it is the number a broker is
// actually managing.
func TestShop_ReportsMarginAgainstTheSellPrice(t *testing.T) {
	t.Parallel()

	d := setup(t)
	expectGuide(d, guideEntry(cheapCarrier, 1, 900))
	expectBuyRules(d, map[pulid.ID]*rateagreement.RateAgreementRule{
		cheapCarrier: buyRule(cheapCarrier, "1400"),
	})

	result, err := d.svc.Shop(t.Context(), shopRequest(services.ShopStrategyBestMargin))

	require.NoError(t, err)
	require.Len(t, result.Options, 1)
	// 2000 sold, 1400 bought, 600 left, which is 30%.
	assert.True(t, result.Options[0].Margin.Amount.Equal(decimal.NewFromInt(600)),
		"margin was %s", result.Options[0].Margin.Amount)
	assert.True(t, result.Options[0].Margin.Percent.Equal(decimal.NewFromInt(30)),
		"percent was %s", result.Options[0].Margin.Percent)
}

// A carrier agreement written as a share of what the customer pays needs the
// sell total in scope, so the buy side binds it into the template alongside the
// rule's own rate.
func TestShop_BindsTheSellTotalIntoTheCarriersTemplate(t *testing.T) {
	t.Parallel()

	d := setup(t)
	expectGuide(d, guideEntry(cheapCarrier, 1, 900))
	expectBuyRules(d, map[pulid.ID]*rateagreement.RateAgreementRule{
		cheapCarrier: buyRule(cheapCarrier, "1400"),
	})

	_, err := d.svc.Shop(t.Context(), shopRequest(services.ShopStrategyLeastCost))

	require.NoError(t, err)
	require.Len(t, d.calcRequests, 1)

	req := d.calcRequests[0]
	assert.Equal(t, flatTemplateID, req.TemplateID)
	assert.Equal(t, 1400.0, req.Overrides["baseRate"])
	assert.Equal(t, 2000.0, req.Overrides["sellTotal"])
}

// A carrier no contract covers is not an error and must not take the shopping
// run down with it. It is listed, unpriced, and ranked last — somebody needs to
// see that it was considered and why it could not be used.
func TestShop_CarrierWithNoContractIsListedLastRatherThanDropped(t *testing.T) {
	t.Parallel()

	d := setup(t)
	expectGuide(d,
		guideEntry(silentCarrier, 1, 600),
		guideEntry(cheapCarrier, 2, 900),
	)
	expectBuyRules(d, map[pulid.ID]*rateagreement.RateAgreementRule{
		cheapCarrier: buyRule(cheapCarrier, "1400"),
	})

	result, err := d.svc.Shop(t.Context(), shopRequest(services.ShopStrategyLeastCost))

	require.NoError(t, err)
	require.Len(t, result.Options, 2)
	assert.Equal(t, cheapCarrier, result.Options[0].CarrierID)
	assert.Equal(t, silentCarrier, result.Options[1].CarrierID)
	assert.False(t, result.Options[1].Priced())
	assert.NotEmpty(t, result.Options[1].Note)
}

// An unpriced carrier must not be ranked first just because zero is the
// smallest number. Least cost means the cheapest real price.
func TestShop_LeastCostDoesNotTreatUnpricedAsFree(t *testing.T) {
	t.Parallel()

	d := setup(t)
	expectGuide(d, guideEntry(silentCarrier, 1, 600), guideEntry(pricyCarrier, 2, 1800))
	expectBuyRules(d, map[pulid.ID]*rateagreement.RateAgreementRule{
		pricyCarrier: buyRule(pricyCarrier, "1700"),
	})

	result, err := d.svc.Shop(t.Context(), shopRequest(services.ShopStrategyLeastCost))

	require.NoError(t, err)
	assert.Equal(t, pricyCarrier, result.Options[0].CarrierID)
}

// An explicit shortlist is how a what-if is asked: "what would these three
// charge", regardless of what the guide says.
func TestShop_ExplicitShortlistSkipsTheGuideEntirely(t *testing.T) {
	t.Parallel()

	d := setup(t)
	expectBuyRules(d, map[pulid.ID]*rateagreement.RateAgreementRule{
		cheapCarrier: buyRule(cheapCarrier, "1400"),
	})

	req := shopRequest(services.ShopStrategyLeastCost)
	req.CarrierIDs = []pulid.ID{cheapCarrier}

	result, err := d.svc.Shop(t.Context(), req)

	require.NoError(t, err)
	require.Len(t, result.Options, 1)
	assert.Nil(t, result.RoutingGuideID)
	assert.Zero(t, d.guides.calls, "an explicit shortlist must not consult the guide")
}

// Shopping with nothing to shop is an ordinary outcome — a lane with no guide
// and no shortlist — and the caller is told so rather than handed an error.
func TestShop_NoCandidatesReturnsAnEmptyResultWithAWarning(t *testing.T) {
	t.Parallel()

	d := setup(t)

	result, err := d.svc.Shop(t.Context(), shopRequest(services.ShopStrategyLeastCost))

	require.NoError(t, err)
	assert.Empty(t, result.Options)
	assert.NotEmpty(t, result.Warnings)
}

// Two carriers quoting the same number have to come back in the same order
// however they arrived, or the same load shops two ways and nobody can
// reproduce which carrier was offered it.
//
// The guide is walked in both orders on purpose. A sort that is merely stable
// would echo whichever order it was handed and pass a test that only ran one
// way, so this is the arrangement that actually needs the tiebreak.
func TestShop_TiesAreBrokenTheSameWayWhateverOrderTheGuideIsIn(t *testing.T) {
	t.Parallel()

	shopWith := func(entries ...*tender.RoutingGuideEntry) string {
		d := setup(t)
		expectGuide(d, entries...)
		expectBuyRules(d, map[pulid.ID]*rateagreement.RateAgreementRule{
			cheapCarrier: buyRule(cheapCarrier, "1500"),
			pricyCarrier: buyRule(pricyCarrier, "1500"),
		})

		result, err := d.svc.Shop(t.Context(), shopRequest(services.ShopStrategyLeastCost))
		require.NoError(t, err)
		require.Len(t, result.Options, 2)

		return result.Options[0].CarrierID.String()
	}

	forwards := shopWith(guideEntry(cheapCarrier, 1, 900), guideEntry(pricyCarrier, 2, 900))
	backwards := shopWith(guideEntry(pricyCarrier, 2, 900), guideEntry(cheapCarrier, 1, 900))

	assert.Equal(t, forwards, backwards)
}

// A shipment being shopped is not yet assigned, so nothing about it should
// change. Persisting quotes is the caller's explicit choice.
func TestShop_DoesNotPersistQuotesUnlessAsked(t *testing.T) {
	t.Parallel()

	d := setup(t)
	expectGuide(d, guideEntry(cheapCarrier, 1, 900))
	expectBuyRules(d, map[pulid.ID]*rateagreement.RateAgreementRule{
		cheapCarrier: buyRule(cheapCarrier, "1400"),
	})

	_, err := d.svc.Shop(t.Context(), shopRequest(services.ShopStrategyLeastCost))

	require.NoError(t, err)
	d.quotes.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
	d.quotes.AssertNotCalled(t, "Save", mock.Anything, mock.Anything)
}

// Every option carries its own trace, because "why was this carrier offered the
// load" is asked about the one that was picked and the ones that were not.
func TestShop_EveryOptionCarriesItsOwnQuoteAndTrace(t *testing.T) {
	t.Parallel()

	d := setup(t)
	expectGuide(d, guideEntry(cheapCarrier, 1, 900), guideEntry(pricyCarrier, 2, 1800))
	expectBuyRules(d, map[pulid.ID]*rateagreement.RateAgreementRule{
		cheapCarrier: buyRule(cheapCarrier, "1400"),
		pricyCarrier: buyRule(pricyCarrier, "1700"),
	})

	result, err := d.svc.Shop(t.Context(), shopRequest(services.ShopStrategyLeastCost))

	require.NoError(t, err)
	for _, option := range result.Options {
		require.NotNil(t, option.Quote, "carrier %s", option.CarrierID)
		require.NotNil(t, option.Quote.Trace, "carrier %s", option.CarrierID)
		assert.Equal(t, ratequote.PurposeShopping, option.Quote.Purpose)
	}
}

// Limit is what keeps a guide with forty carriers on it from returning forty
// priced options nobody will read.
func TestShop_LimitCapsTheOptionsReturned(t *testing.T) {
	t.Parallel()

	d := setup(t)
	expectGuide(d,
		guideEntry(cheapCarrier, 1, 900),
		guideEntry(pricyCarrier, 2, 1800),
		guideEntry(silentCarrier, 3, 600),
	)
	expectBuyRules(d, map[pulid.ID]*rateagreement.RateAgreementRule{
		cheapCarrier: buyRule(cheapCarrier, "1400"),
		pricyCarrier: buyRule(pricyCarrier, "1700"),
	})

	req := shopRequest(services.ShopStrategyLeastCost)
	req.Limit = 1

	result, err := d.svc.Shop(t.Context(), req)

	require.NoError(t, err)
	require.Len(t, result.Options, 1)
	assert.Equal(t, cheapCarrier, result.Options[0].CarrierID)
}

// A carrier the contract prices below the margin floor is still shown. Hiding
// it would leave somebody wondering why the carrier they expected is missing;
// showing it flagged is what lets them decide to override.
func TestShop_OptionBelowTheMarginFloorIsShownAndFlagged(t *testing.T) {
	t.Parallel()

	d := setup(t)
	expectGuide(d, guideEntry(pricyCarrier, 1, 900))

	rule := buyRule(pricyCarrier, "1900")
	rule.Agreement.MarginFloorPercent = decimal.NewNullDecimal(decimal.NewFromInt(15))
	expectBuyRules(d, map[pulid.ID]*rateagreement.RateAgreementRule{pricyCarrier: rule})

	result, err := d.svc.Shop(t.Context(), shopRequest(services.ShopStrategyBestMargin))

	require.NoError(t, err)
	require.Len(t, result.Options, 1)
	assert.True(t, result.Options[0].Margin.BelowFloor)
	assert.NotEmpty(t, result.Options[0].Margin.Explanation)
}

// A strategy nobody set is least cost, which is the answer somebody shopping
// without saying what they meant is asking for.
func TestShop_UnsetStrategyDefaultsToLeastCost(t *testing.T) {
	t.Parallel()

	d := setup(t)
	expectGuide(d, guideEntry(pricyCarrier, 1, 900), guideEntry(cheapCarrier, 2, 1800))
	expectBuyRules(d, map[pulid.ID]*rateagreement.RateAgreementRule{
		cheapCarrier: buyRule(cheapCarrier, "1400"),
		pricyCarrier: buyRule(pricyCarrier, "1700"),
	})

	req := shopRequest("")

	result, err := d.svc.Shop(t.Context(), req)

	require.NoError(t, err)
	assert.Equal(t, services.ShopStrategyLeastCost, result.Strategy)
	assert.Equal(t, cheapCarrier, result.Options[0].CarrierID)
}
