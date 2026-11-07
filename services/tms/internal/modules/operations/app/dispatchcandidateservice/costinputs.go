package dispatchcandidateservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/operations/domain/demandforecast"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"

	"github.com/emoss08/trenova/internal/modules/operations/app/dispatchcost"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchmarket"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchboardports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchmarketports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchplanports"
	"github.com/emoss08/trenova/shared/pulid"
)

func (s *Service) costInputs(
	ctx context.Context,
	req *SnapshotRequest,
) (dispatchcost.Model, EndMarketValuer, error) {
	economics := req.Control.ResolvedEconomics()
	var endMarket EndMarketValuer = noEndMarket{}
	if req.Control == nil {
		return dispatchcost.NewModel(&economics, nil), endMarket, nil
	}

	var multipliers dispatchcost.Multipliers
	if req.Control.LearnDispatchStyle {
		learned, err := s.style.Multipliers(ctx, req.TenantInfo)
		if err != nil {
			return dispatchcost.Model{}, nil, err
		}
		multipliers = learned
	}

	if req.Control.PlanForEndMarket && len(req.Destinations) > 0 {
		valuer, err := s.market.Valuer(ctx, &dispatchmarketports.ValuerRequest{
			TenantInfo: req.TenantInfo,
			Points:     req.Destinations,
		})
		if err != nil {
			return dispatchcost.Model{}, nil, err
		}
		if valuer.Len() > 0 {
			endMarket = valuer
		}
	}

	return dispatchcost.NewModel(&economics, multipliers), endMarket, nil
}

func DestinationsOf(moves []*dispatchboardports.BoardMove) []dispatchmarket.Point {
	points := make([]dispatchmarket.Point, 0, len(moves))
	for _, move := range moves {
		if move.DestinationLatitude == nil || move.DestinationLongitude == nil {
			continue
		}
		points = append(points, dispatchmarket.Point{
			Latitude:  *move.DestinationLatitude,
			Longitude: *move.DestinationLongitude,
		})
	}

	return points
}

func (s *FleetSnapshot) DriverStart(workerID pulid.ID) (dispatchmarket.Point, int64, bool) {
	if s.profiles == nil {
		s.Prepare()
	}
	profile := s.profiles[workerID]
	if profile == nil || profile.originSource == dispatchplanports.OriginUnknown {
		return dispatchmarket.Point{}, 0, false
	}

	return dispatchmarket.Point{
		Latitude:  profile.origin.lat,
		Longitude: profile.origin.lon,
	}, max(profile.availableAt, s.Now), true
}

func (s *FleetSnapshot) Supply() []demandforecast.Driver {
	supply := make([]demandforecast.Driver, 0, len(s.Drivers))
	for _, driver := range s.Drivers {
		start, freeAt, ok := s.DriverStart(driver.WorkerID)
		if !ok {
			continue
		}
		cell, ok := dispatchmarket.CellOf(start)
		if !ok {
			continue
		}
		supply = append(supply, demandforecast.Driver{
			WorkerID: driver.WorkerID,
			Cell:     cell,
			FreeAt:   freeAt,
		})
	}

	return supply
}

func SolvingOf(moves []*dispatchboardports.BoardMove) []demandforecast.Booked {
	solving := make([]demandforecast.Booked, 0, len(moves))
	for _, move := range moves {
		if move.OriginLatitude == nil || move.OriginLongitude == nil ||
			move.MoveStatus != shipmentvocab.MoveStatusNew {
			continue
		}
		cell, ok := dispatchmarket.CellOf(dispatchmarket.Point{
			Latitude:  *move.OriginLatitude,
			Longitude: *move.OriginLongitude,
		})
		if !ok {
			continue
		}
		solving = append(solving, demandforecast.Booked{
			Cell:      cell,
			SlotStart: move.OriginWindowStart,
			Total:     1,
			Open:      1,
		})
	}

	return solving
}
