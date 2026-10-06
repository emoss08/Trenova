package shipmentcapacityservice

import (
	"cmp"
	"context"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/carrier"
	"github.com/emoss08/trenova/internal/core/domain/carriercapacity"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/dispatchconsoleservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/geoutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/shopspring/decimal"
)

const (
	fitWindow      = 40.0
	fitOrigin      = 30.0
	fitDestination = 10.0
	fitAcceptance  = 0.2
	fitHistoryOnly = 0.6
	percent        = 100.0
)

type carrierFacts struct {
	postings    map[pulid.ID][]*carriercapacity.Posting
	performance map[pulid.ID]*repositories.CarrierPerformance
	carriers    map[pulid.ID]*carrier.Carrier
}

func (s *Service) loadCarrierFacts(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	carrierIDs []pulid.ID,
) (*carrierFacts, error) {
	now := s.now()
	postings, err := s.postings.ListOpen(ctx, &repositories.ListOpenCarrierCapacityRequest{
		TenantInfo: tenantInfo,
		OpenAt:     now.Unix(),
		Through:    now.Add(carrierWindow).Unix(),
		CarrierIDs: carrierIDs,
	})
	if err != nil {
		return nil, err
	}
	performance, err := s.performance.ListCarrierPerformance(
		ctx,
		&repositories.ListCarrierPerformanceRequest{
			TenantInfo: tenantInfo,
			Since:      now.Add(-performanceLookback).Unix(),
			CarrierIDs: carrierIDs,
		},
	)
	if err != nil {
		return nil, err
	}

	facts := &carrierFacts{
		postings:    make(map[pulid.ID][]*carriercapacity.Posting, len(postings)),
		performance: make(map[pulid.ID]*repositories.CarrierPerformance, len(performance)),
		carriers:    make(map[pulid.ID]*carrier.Carrier, len(postings)),
	}
	for _, posting := range postings {
		facts.postings[posting.CarrierID] = append(facts.postings[posting.CarrierID], posting)
		if posting.Carrier != nil {
			facts.carriers[posting.CarrierID] = posting.Carrier
		}
	}
	for _, perf := range performance {
		facts.performance[perf.CarrierID] = perf
	}

	missing := make([]pulid.ID, 0)
	for id := range facts.performance {
		if _, ok := facts.carriers[id]; !ok {
			missing = append(missing, id)
		}
	}
	if len(missing) > 0 {
		found, lookupErr := s.carriers.GetByIDs(ctx, repositories.GetCarriersByIDsRequest{
			TenantInfo: tenantInfo,
			CarrierIDs: missing,
		})
		if lookupErr != nil {
			return nil, lookupErr
		}
		for _, entity := range found {
			facts.carriers[entity.ID] = entity
		}
	}

	return facts, nil
}

func (s *Service) carrierCapacity(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*services.ShipmentCapacity, error) {
	facts, err := s.loadCarrierFacts(ctx, tenantInfo, nil)
	if err != nil {
		return nil, err
	}
	board, err := s.console.GetBoard(ctx, &dispatchconsoleservice.GetBoardRequest{
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}

	summary := &services.CarrierCapacitySummary{
		Posting:        len(facts.postings),
		AvgRatePerMile: weightedRatePerMile(facts.performance),
	}
	for _, move := range board.Moves {
		if move == nil || move.IsCovered {
			continue
		}
		if move.LiveTender != nil {
			summary.AwaitingAcceptance++
		} else {
			summary.Untendered++
		}
	}

	return &services.ShipmentCapacity{
		Kind:     services.CapacityUnitCarrier,
		Units:    carrierUnits(facts),
		Carriers: summary,
	}, nil
}

func carrierUnits(facts *carrierFacts) []*services.CapacityUnit {
	units := make([]*services.CapacityUnit, 0, len(facts.carriers))
	for id, entity := range facts.carriers {
		perf := facts.performance[id]
		acceptance, answered := perf.AcceptancePercent()
		postings := facts.postings[id]

		var group services.CapacityGroup
		switch {
		case len(postings) > 0:
			group = services.CapacityGroupTrucksPosted
		case answered && perf.OffersAnswered >= usuallyAcceptMinAnswers &&
			acceptance >= usuallyAcceptPercent:
			group = services.CapacityGroupUsuallyAccept
		default:
			continue
		}

		unit := &services.CapacityUnit{
			ID:          id,
			Kind:        services.CapacityUnitCarrier,
			Name:        entity.Name,
			Initials:    stringutils.Initials(entity.Name),
			Group:       group,
			UnitLabel:   entity.MCNumber,
			RatePerMile: carrierRatePerMile(postings, perf),
		}
		if trucks := truckCount(postings); trucks > 0 {
			unit.BadgeCount = &trucks
			unit.City = postingOrigin(postings[0])
		}
		if answered {
			value := acceptance
			unit.AcceptancePercent = &value
			unit.Ring = &services.CapacityRing{
				Value: acceptance,
				Max:   percent,
				Low:   acceptance < lowAcceptancePercent,
			}
		}
		units = append(units, unit)
	}

	slices.SortStableFunc(units, func(a, b *services.CapacityUnit) int {
		return cmp.Or(
			cmp.Compare(groupOrder(a.Group), groupOrder(b.Group)),
			-cmp.Compare(acceptanceOf(a), acceptanceOf(b)),
			strings.Compare(a.Name, b.Name),
		)
	})

	return units
}

func acceptanceOf(unit *services.CapacityUnit) float64 {
	if unit.AcceptancePercent == nil {
		return -1
	}

	return *unit.AcceptancePercent
}

func truckCount(postings []*carriercapacity.Posting) int {
	total := 0
	for _, posting := range postings {
		total += posting.TruckCount
	}

	return total
}

func postingOrigin(posting *carriercapacity.Posting) string {
	switch {
	case posting.OriginLocation != nil:
		return posting.OriginLocation.City
	case posting.OriginState != nil:
		return posting.OriginState.Abbreviation
	default:
		return ""
	}
}

func carrierRatePerMile(
	postings []*carriercapacity.Posting,
	perf *repositories.CarrierPerformance,
) decimal.NullDecimal {
	best := decimal.NullDecimal{}
	for _, posting := range postings {
		if posting.RateMethod != carriercapacity.RateMethodPerMile || !posting.Rate.Valid {
			continue
		}
		if !best.Valid || posting.Rate.Decimal.LessThan(best.Decimal) {
			best = posting.Rate
		}
	}
	if best.Valid || perf == nil {
		return best
	}

	return perf.AvgRatePerMile
}

func weightedRatePerMile(
	performance map[pulid.ID]*repositories.CarrierPerformance,
) decimal.NullDecimal {
	total := decimal.Zero
	loads := 0
	for _, perf := range performance {
		if !perf.AvgRatePerMile.Valid || perf.PerMileLoads == 0 {
			continue
		}
		total = total.Add(
			perf.AvgRatePerMile.Decimal.Mul(decimal.NewFromInt(int64(perf.PerMileLoads))),
		)
		loads += perf.PerMileLoads
	}
	if loads == 0 {
		return decimal.NullDecimal{}
	}

	return decimal.NewNullDecimal(total.Div(decimal.NewFromInt(int64(loads))).Round(2))
}

type laneFit struct {
	posting *carriercapacity.Posting
	score   float64
}

func bestPostingFor(
	move *dispatchconsoleservice.BoardMove,
	postings []*carriercapacity.Posting,
) laneFit {
	best := laneFit{}
	for _, posting := range postings {
		if move.OriginWindowStart < posting.AvailableFrom ||
			move.OriginWindowStart > posting.AvailableTo {
			continue
		}
		score := fitWindow
		if originMatches(posting, move) {
			score += fitOrigin
		}
		if posting.DestinationState != nil &&
			strings.EqualFold(posting.DestinationState.Abbreviation, move.DestinationState) {
			score += fitDestination
		}
		if score > best.score {
			best = laneFit{posting: posting, score: score}
		}
	}

	return best
}

func originMatches(posting *carriercapacity.Posting, move *dispatchconsoleservice.BoardMove) bool {
	if posting.OriginLocation != nil {
		if posting.OriginLocationID != nil && *posting.OriginLocationID == move.OriginLocationID {
			return true
		}
		if posting.OriginLocation.Latitude == nil || posting.OriginLocation.Longitude == nil ||
			move.OriginLatitude == nil || move.OriginLongitude == nil {
			return false
		}
		radius := defaultPickupRadius
		if posting.OriginRadiusMiles != nil {
			radius = float64(*posting.OriginRadiusMiles)
		}

		return geoutils.HaversineMiles(
			*posting.OriginLocation.Latitude, *posting.OriginLocation.Longitude,
			*move.OriginLatitude, *move.OriginLongitude,
		) <= radius
	}

	return posting.OriginState != nil &&
		strings.EqualFold(posting.OriginState.Abbreviation, move.OriginState)
}

func quoteFor(
	posting *carriercapacity.Posting,
	perf *repositories.CarrierPerformance,
	distance *float64,
) decimal.NullDecimal {
	miles := decimal.Zero
	if distance != nil && *distance > 0 {
		miles = decimal.NewFromFloat(*distance)
	}
	if posting != nil && posting.Rate.Valid {
		if posting.RateMethod == carriercapacity.RateMethodFlat {
			return posting.Rate
		}
		if miles.IsPositive() {
			return decimal.NewNullDecimal(posting.Rate.Decimal.Mul(miles).Round(2))
		}
	}
	if perf != nil && perf.AvgRatePerMile.Valid && miles.IsPositive() {
		return decimal.NewNullDecimal(perf.AvgRatePerMile.Decimal.Mul(miles).Round(2))
	}

	return decimal.NullDecimal{}
}

func marginPercent(revenue decimal.Decimal, quote decimal.NullDecimal) *float64 {
	if !quote.Valid || !revenue.IsPositive() {
		return nil
	}
	value, _ := revenue.Sub(quote.Decimal).Div(revenue).Mul(decimal.NewFromInt(100)).
		Round(1).Float64()

	return &value
}

func (s *Service) carrierMatches(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	carrierID pulid.ID,
	limit int,
) ([]*services.CapacityMatch, error) {
	facts, err := s.loadCarrierFacts(ctx, tenantInfo, []pulid.ID{carrierID})
	if err != nil {
		return nil, err
	}
	board, err := s.console.GetBoard(ctx, &dispatchconsoleservice.GetBoardRequest{
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}

	return rankCarrierMatches(
		board.Moves,
		facts.postings[carrierID],
		facts.performance[carrierID],
		limit,
	), nil
}

func rankCarrierMatches(
	moves []*dispatchconsoleservice.BoardMove,
	postings []*carriercapacity.Posting,
	perf *repositories.CarrierPerformance,
	limit int,
) []*services.CapacityMatch {
	acceptance, answered := perf.AcceptancePercent()

	type scored struct {
		match *services.CapacityMatch
		fit   float64
	}
	ranked := make([]scored, 0, len(moves))
	for _, move := range moves {
		if move == nil || move.IsCovered || move.LiveTender != nil {
			continue
		}
		lane := bestPostingFor(move, postings)
		fit := lane.score
		switch {
		case lane.posting != nil && answered:
			fit += acceptance * fitAcceptance
		case lane.posting == nil && answered:
			fit = acceptance * fitHistoryOnly
		case lane.posting == nil:
			continue
		}

		match := matchFromMove(move)
		match.Quote = quoteFor(lane.posting, perf, move.Distance)
		match.MarginPercent = marginPercent(match.Revenue, match.Quote)
		rounded := min(percent, fit)
		match.FitPercent = &rounded
		ranked = append(ranked, scored{match: match, fit: rounded})
	}

	slices.SortStableFunc(ranked, func(a, b scored) int {
		return cmp.Or(
			-cmp.Compare(a.fit, b.fit),
			cmp.Compare(a.match.PickupAt, b.match.PickupAt),
		)
	})

	out := make([]*services.CapacityMatch, 0, min(limit, len(ranked)))
	for _, entry := range ranked[:min(limit, len(ranked))] {
		out = append(out, entry.match)
	}

	return out
}
