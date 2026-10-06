package shipmentcapacityservice

import (
	"cmp"
	"context"
	"slices"

	"github.com/emoss08/trenova/internal/core/domain/carriercapacity"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/dispatchconsoleservice"
	"github.com/emoss08/trenova/internal/core/services/ratequoteservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/shopspring/decimal"
)

func (s *Service) loadShipment(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentID pulid.ID,
) (*shipment.Shipment, error) {
	return s.shipments.GetByID(ctx, &repositories.GetShipmentByIDRequest{
		ID:              shipmentID,
		TenantInfo:      tenantInfo,
		ShipmentOptions: repositories.ShipmentOptions{ExpandShipmentDetails: true},
	})
}

func firstUncoveredMove(sp *shipment.Shipment) *shipment.ShipmentMove {
	moves := slices.Clone(sp.Moves)
	slices.SortStableFunc(moves, func(a, b *shipment.ShipmentMove) int {
		return cmp.Compare(a.Sequence, b.Sequence)
	})
	for _, move := range moves {
		if move != nil && move.CoverageType == shipment.MoveCoverageTypeUnassigned {
			return move
		}
	}

	return nil
}

func (s *Service) CoverageSuggestions(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentID pulid.ID,
) (*services.ShipmentCoverageSuggestions, error) {
	out := &services.ShipmentCoverageSuggestions{
		Drivers:  []*services.DriverCoverageSuggestion{},
		Carriers: []*services.CarrierCoverageSuggestion{},
	}

	sp, err := s.loadShipment(ctx, tenantInfo, shipmentID)
	if err != nil {
		return nil, err
	}
	move := firstUncoveredMove(sp)
	if move == nil {
		return out, nil
	}

	out.Drivers, err = s.driverSuggestions(ctx, tenantInfo, move.ID)
	if err != nil {
		return nil, err
	}
	out.Carriers, err = s.carrierSuggestions(ctx, tenantInfo, sp, move)
	if err != nil {
		return nil, err
	}

	return out, nil
}

func (s *Service) driverSuggestions(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	moveID pulid.ID,
) ([]*services.DriverCoverageSuggestion, error) {
	ranked, err := s.console.GetMoveCandidates(ctx, &dispatchconsoleservice.MoveCandidatesRequest{
		TenantInfo: tenantInfo,
		MoveID:     moveID,
		Limit:      coverageSuggestions,
	})
	if err != nil {
		return nil, err
	}

	out := make([]*services.DriverCoverageSuggestion, 0, len(ranked))
	for _, candidate := range ranked {
		if candidate == nil || candidate.Blocked() {
			continue
		}
		remaining := candidate.DriveRemainingMs
		fit := float64(candidate.Score)
		out = append(out, &services.DriverCoverageSuggestion{
			WorkerID:         candidate.WorkerID,
			TractorID:        candidate.TractorID,
			MoveID:           moveID,
			Name:             candidate.WorkerName,
			Initials:         stringutils.Initials(candidate.WorkerName),
			DistanceMiles:    candidate.DeadheadMiles,
			DriveRemainingMs: &remaining,
			FitPercent:       &fit,
		})
	}

	return out, nil
}

func (s *Service) carrierSuggestions(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	sp *shipment.Shipment,
	move *shipment.ShipmentMove,
) ([]*services.CarrierCoverageSuggestion, error) {
	shopped, err := s.shopper.Shop(ctx, &ratequoteservice.ShopRequest{
		TenantInfo: tenantInfo,
		ShipmentID: sp.ID,
		Limit:      coverageSuggestions,
	})
	if err != nil {
		if errortypes.IsBusinessError(err) || errortypes.IsNotFoundError(err) {
			return []*services.CarrierCoverageSuggestion{}, nil
		}
		return nil, err
	}

	priced := make([]*services.ShopOption, 0, len(shopped.Options))
	ids := make([]pulid.ID, 0, len(shopped.Options))
	for _, option := range shopped.Options {
		if option != nil && option.Priced() {
			priced = append(priced, option)
			ids = append(ids, option.CarrierID)
		}
	}
	if len(priced) == 0 {
		return []*services.CarrierCoverageSuggestion{}, nil
	}

	facts, err := s.loadCarrierFacts(ctx, tenantInfo, ids)
	if err != nil {
		return nil, err
	}
	if err = s.fillCarriers(ctx, tenantInfo, facts, ids); err != nil {
		return nil, err
	}

	pickup := int64(0)
	if stop := sp.ShipperStop(); stop != nil {
		pickup = stop.ScheduledWindowStart
	}
	out := make([]*services.CarrierCoverageSuggestion, 0, len(priced))
	for _, option := range priced {
		suggestion := &services.CarrierCoverageSuggestion{
			CarrierID:   option.CarrierID,
			MoveID:      move.ID,
			Name:        option.CarrierName,
			Initials:    stringutils.Initials(option.CarrierName),
			Quote:       option.Cost,
			RatePerMile: perMile(option.Cost, move.Distance),
			Posted:      postedAt(facts.postings[option.CarrierID], pickup),
		}
		if entity := facts.carriers[option.CarrierID]; entity != nil {
			suggestion.MCNumber = entity.MCNumber
		}
		if acceptance, ok := facts.performance[option.CarrierID].AcceptancePercent(); ok {
			suggestion.AcceptancePercent = &acceptance
		}
		out = append(out, suggestion)
	}

	return out, nil
}

func (s *Service) fillCarriers(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	facts *carrierFacts,
	ids []pulid.ID,
) error {
	missing := make([]pulid.ID, 0, len(ids))
	for _, id := range ids {
		if _, ok := facts.carriers[id]; !ok {
			missing = append(missing, id)
		}
	}
	if len(missing) == 0 {
		return nil
	}

	found, err := s.carriers.GetByIDs(ctx, repositories.GetCarriersByIDsRequest{
		TenantInfo: tenantInfo,
		CarrierIDs: missing,
	})
	if err != nil {
		return err
	}
	for _, entity := range found {
		facts.carriers[entity.ID] = entity
	}

	return nil
}

func perMile(cost decimal.Decimal, distance *float64) decimal.Decimal {
	if distance == nil || *distance <= 0 {
		return decimal.Zero
	}

	return cost.Div(decimal.NewFromFloat(*distance)).Round(2)
}

func postedAt(postings []*carriercapacity.Posting, at int64) bool {
	if at == 0 {
		return len(postings) > 0
	}
	for _, posting := range postings {
		if at >= posting.AvailableFrom && at <= posting.AvailableTo {
			return true
		}
	}

	return false
}
