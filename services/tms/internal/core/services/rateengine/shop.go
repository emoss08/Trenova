package rateengine

import (
	"cmp"
	"context"
	"fmt"
	"slices"

	"github.com/emoss08/trenova/internal/core/domain/rateagreement"
	"github.com/emoss08/trenova/internal/core/domain/ratequote"
	"github.com/emoss08/trenova/internal/core/domain/tender"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/ratetablecache"
	"github.com/emoss08/trenova/pkg/ratetypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

// defaultShopLimit bounds what comes back by default.
//
// A routing guide can carry forty carriers, and nobody reads forty priced
// options. The ones past the tenth are there for completeness rather than for
// a decision, and the caller can ask for more.
const defaultShopLimit = 10

// Shop prices this shipment against several carriers and ranks them.
//
// Each carrier is rated through the same path a real buy side rating takes, so
// every option carries its own quote and its own trace. That costs one
// resolution per carrier rather than one for all of them, and it is worth it:
// the alternative is a second pricing path that could disagree with the first,
// and an option somebody acted on has to be reproducible from its own record.
//
// Nothing here can fail the run. A carrier with no contract, a lane no guide
// covers, a rule that will not price — each becomes an option that says so, or
// a warning on the result. Somebody shopping a load needs to see what was
// considered, including what could not be used.
func (s *Service) Shop(
	ctx context.Context,
	req *services.ShopRequest,
) (*services.ShopResult, error) {
	if req == nil || req.Shipment == nil {
		return nil, ErrNoParty
	}

	strategy := req.Strategy
	if !strategy.IsValid() {
		strategy = services.ShopStrategyLeastCost
	}

	result := &services.ShopResult{
		Strategy:  strategy,
		SellTotal: req.SellTotal,
		Options:   []*services.ShopOption{},
	}

	candidates := s.shopCandidates(ctx, req, result)
	if len(candidates) == 0 {
		result.Warnings = append(result.Warnings,
			"No carriers to shop: this lane has no routing guide and no carriers were named")

		return result, nil
	}

	// Every candidate reads the same tenant's rate tables, so the run pays for
	// them once rather than once per carrier.
	ctx = ratetablecache.With(ctx)

	for _, candidate := range candidates {
		result.Options = append(result.Options, s.priceCandidate(ctx, req, candidate))
	}

	rankOptions(result.Options, strategy)

	limit := req.Limit
	if limit <= 0 {
		limit = defaultShopLimit
	}

	if len(result.Options) > limit {
		result.Warnings = append(result.Warnings, fmt.Sprintf(
			"%d carriers were priced; showing the top %d",
			len(result.Options), limit,
		))
		result.Options = result.Options[:limit]
	}

	return result, nil
}

// shopCandidate is one carrier to price, and whatever the routing guide had to
// say about it.
type shopCandidate struct {
	carrierID pulid.ID
	name      string
	guideRank int16
	offerTTL  int32
}

// shopCandidates decides who is being shopped.
//
// An explicit shortlist wins outright — that is how a what-if is asked, and
// consulting the guide anyway would answer a different question. Otherwise the
// shipment's routing guide supplies them, because that is where the
// organization already wrote down which carriers may haul which lane.
func (s *Service) shopCandidates(
	ctx context.Context,
	req *services.ShopRequest,
	result *services.ShopResult,
) []shopCandidate {
	if len(req.CarrierIDs) > 0 {
		candidates := make([]shopCandidate, 0, len(req.CarrierIDs))
		for _, carrierID := range req.CarrierIDs {
			candidates = append(candidates, shopCandidate{carrierID: carrierID})
		}

		return candidates
	}

	if s.routingGuides == nil {
		return nil
	}

	guide, err := s.routingGuides.MatchLane(ctx, matchLaneRequest(req))
	if err != nil {
		s.l.Warn("failed to match a routing guide while shopping",
			zap.String("shipmentId", req.Shipment.ID.String()),
			zap.Error(err),
		)

		result.Warnings = append(result.Warnings,
			"The routing guide for this lane could not be read")

		return nil
	}

	if guide == nil || len(guide.Entries) == 0 {
		return nil
	}

	guideID := guide.ID
	result.RoutingGuideID = &guideID

	candidates := make([]shopCandidate, 0, len(guide.Entries))
	for _, entry := range guide.Entries {
		if entry == nil {
			continue
		}

		candidates = append(candidates, shopCandidate{
			carrierID: entry.CarrierID,
			name:      carrierNameOf(entry),
			guideRank: entry.Rank,
			offerTTL:  entry.OfferTTLSeconds,
		})
	}

	return candidates
}

func carrierNameOf(entry *tender.RoutingGuideEntry) string {
	if entry.Carrier == nil {
		return ""
	}

	return entry.Carrier.Name
}

func matchLaneRequest(req *services.ShopRequest) *repositories.MatchLaneRequest {
	origin := req.Shipment.ShipperStop()
	destination := req.Shipment.DeliveryStop()

	match := &repositories.MatchLaneRequest{TenantInfo: req.TenantInfo}

	if origin != nil && origin.Location != nil {
		match.OriginLocationID = origin.Location.ID
		match.OriginCity = origin.Location.City
		if origin.Location.State != nil {
			match.OriginState = origin.Location.State.Abbreviation
		}
	}

	if destination != nil && destination.Location != nil {
		match.DestinationLocationID = destination.Location.ID
		match.DestinationCity = destination.Location.City
		if destination.Location.State != nil {
			match.DestinationState = destination.Location.State.Abbreviation
		}
	}

	return match
}

// priceCandidate rates one carrier and measures what would be left.
//
// A rating that errors outright becomes an unpriced option carrying the reason
// rather than aborting the run: one carrier's broken contract must not stop the
// other nine from being compared.
func (s *Service) priceCandidate(
	ctx context.Context,
	req *services.ShopRequest,
	candidate shopCandidate,
) *services.ShopOption {
	option := &services.ShopOption{
		CarrierID:       candidate.carrierID,
		CarrierName:     candidate.name,
		GuideRank:       candidate.guideRank,
		OfferTTLSeconds: candidate.offerTTL,
	}

	rated, err := s.RateShipment(ctx, &services.RateShipmentRequest{
		Shipment:       req.Shipment,
		TenantInfo:     req.TenantInfo,
		PartyType:      rateagreement.PartyTypeCarrier,
		PartyID:        candidate.carrierID,
		AsOf:           req.AsOf,
		BillingControl: req.BillingControl,
		SellTotal:      req.SellTotal,
		Purpose:        ratequote.PurposeShopping,
		Persist:        req.Persist,
		UserID:         req.UserID,
	})
	if err != nil {
		s.l.Warn("failed to price a carrier while shopping",
			zap.String("carrierId", candidate.carrierID.String()),
			zap.Error(err),
		)

		option.Outcome = ratequote.OutcomeError
		option.Note = err.Error()

		return option
	}

	option.Outcome = rated.Outcome
	option.Cost = rated.Amount
	option.Currency = rated.Currency
	option.Quote = rated.Quote
	option.AgreementID = rated.AgreementID
	option.RuleID = rated.RuleID

	if !option.Priced() {
		option.Note = "No carrier agreement covers this lane"

		return option
	}

	option.Margin = ratetypes.EvaluateMargin(&ratetypes.MarginInputs{
		Sell:      sellTotalOf(req),
		Buy:       rated.Amount,
		Floor:     marginFloorOf(rated),
		MaxPayPct: maxPayPercentOf(rated),
	})

	if option.Margin.Breached() {
		option.Note = option.Margin.Explanation
	}

	return option
}

func sellTotalOf(req *services.ShopRequest) decimal.Decimal {
	if !req.SellTotal.Valid {
		return decimal.Zero
	}

	return req.SellTotal.Decimal
}

// marginFloorOf reads the least margin the winning contract agreed to accept.
//
// It comes from the agreement that priced this option rather than from a
// setting, because the floor is a negotiated term: a carrier signed at a
// thinner margin than another one is not a breach of anything.
func marginFloorOf(rated *services.RatedShipment) decimal.NullDecimal {
	agreement := agreementOf(rated)
	if agreement == nil {
		return decimal.NullDecimal{}
	}

	return agreement.MarginFloorPercent
}

func maxPayPercentOf(rated *services.RatedShipment) decimal.NullDecimal {
	agreement := agreementOf(rated)
	if agreement == nil {
		return decimal.NullDecimal{}
	}

	return agreement.MaxPayPercentOfSell
}

func agreementOf(rated *services.RatedShipment) *rateagreement.RateAgreement {
	if rated == nil || rated.Quote == nil {
		return nil
	}

	return rated.Quote.Agreement
}

// rankOptions puts the options in the order the caller asked for, and stamps
// each with where it landed.
//
// Unpriced options always sort last, whatever the strategy. Least cost would
// otherwise put a carrier nobody can hire at the top, because a cost of nothing
// is the smallest number there is.
//
// Every comparison ends on the carrier id. Two carriers quoting the same number
// is ordinary, and without a final tiebreak the same load would shop two ways
// on two runs and nobody could reproduce which carrier was offered it.
func rankOptions(options []*services.ShopOption, strategy services.ShopStrategy) {
	slices.SortStableFunc(options, func(a, b *services.ShopOption) int {
		if a.Priced() != b.Priced() {
			if a.Priced() {
				return -1
			}

			return 1
		}

		if a.Priced() {
			if by := compareByStrategy(a, b, strategy); by != 0 {
				return by
			}
		}

		return cmp.Compare(a.CarrierID.String(), b.CarrierID.String())
	})

	for index, option := range options {
		option.Rank = index + 1
	}
}

func compareByStrategy(a, b *services.ShopOption, strategy services.ShopStrategy) int {
	switch strategy {
	case services.ShopStrategyBestMargin:
		// Reversed: the most margin comes first.
		return b.Margin.Amount.Cmp(a.Margin.Amount)

	case services.ShopStrategyGuideRank:
		return compareGuideRank(a, b)

	case services.ShopStrategyFastestAccept:
		return compareOfferTTL(a, b)

	case services.ShopStrategyLeastCost:
		return a.Cost.Cmp(b.Cost)

	default:
		return a.Cost.Cmp(b.Cost)
	}
}

// compareGuideRank orders by the guide's own ranking, with carriers that came
// from no guide behind every carrier that did.
func compareGuideRank(a, b *services.ShopOption) int {
	if (a.GuideRank == 0) != (b.GuideRank == 0) {
		if a.GuideRank == 0 {
			return 1
		}

		return -1
	}

	return cmp.Compare(a.GuideRank, b.GuideRank)
}

// compareOfferTTL orders by the shortest offer window, with carriers whose
// window is unknown behind every carrier whose is not.
func compareOfferTTL(a, b *services.ShopOption) int {
	if (a.OfferTTLSeconds == 0) != (b.OfferTTLSeconds == 0) {
		if a.OfferTTLSeconds == 0 {
			return 1
		}

		return -1
	}

	return cmp.Compare(a.OfferTTLSeconds, b.OfferTTLSeconds)
}
