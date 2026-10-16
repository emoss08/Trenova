package distancecalculationservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/places/domain/location"
	"github.com/emoss08/trenova/internal/modules/places/ports/locationports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
)

func (s *Service) loadStopLocations(ctx context.Context, entity *shipment.Shipment) error {
	stops := stopsAwaitingLocation(entity.Moves)
	if len(stops) > 0 {
		ids := make([]pulid.ID, 0, len(stops))
		for _, stop := range stops {
			ids = append(ids, stop.LocationID)
		}

		locations, err := s.locationRepo.GetByIDs(ctx, locationports.GetLocationsByIDsRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			},
			LocationIDs: sliceutils.Dedupe(ids),
		})
		if err != nil {
			return fmt.Errorf("load stop locations: %w", err)
		}

		byID := make(map[pulid.ID]*location.Location, len(locations))
		for _, loc := range locations {
			if loc != nil {
				byID[loc.ID] = loc
			}
		}
		for _, stop := range stops {
			stop.Location = byID[stop.LocationID]
		}
	}

	return s.loadLocationStates(ctx, entity.Moves)
}

func (s *Service) loadLocationStates(ctx context.Context, moves []*shipment.ShipmentMove) error {
	locations := locationsAwaitingState(moves)
	if len(locations) == 0 {
		return nil
	}

	ids := make([]pulid.ID, 0, len(locations))
	for _, loc := range locations {
		ids = append(ids, loc.StateID)
	}

	states, err := s.stateRepo.GetByIDs(ctx, sliceutils.Dedupe(ids))
	if err != nil {
		return fmt.Errorf("load stop location states: %w", err)
	}

	byID := make(map[pulid.ID]*usstate.UsState, len(states))
	for _, state := range states {
		if state != nil {
			byID[state.ID] = state
		}
	}
	for _, loc := range locations {
		loc.State = byID[loc.StateID]
	}

	return nil
}

func stopsAwaitingLocation(moves []*shipment.ShipmentMove) []*shipment.Stop {
	stops := make([]*shipment.Stop, 0, len(moves)*2)
	for _, move := range moves {
		if move == nil {
			continue
		}
		for _, stop := range move.Stops {
			if stop == nil || stop.LocationID.IsNil() {
				continue
			}
			if stop.Location == nil || stop.Location.ID != stop.LocationID {
				stops = append(stops, stop)
			}
		}
	}
	return stops
}

func locationsAwaitingState(moves []*shipment.ShipmentMove) []*location.Location {
	locations := make([]*location.Location, 0, len(moves)*2)
	for _, move := range moves {
		if move == nil {
			continue
		}
		for _, stop := range move.Stops {
			if stop == nil || stop.Location == nil || stop.Location.StateID.IsNil() {
				continue
			}
			if stop.Location.State == nil || stop.Location.State.ID != stop.Location.StateID {
				locations = append(locations, stop.Location)
			}
		}
	}
	return locations
}
