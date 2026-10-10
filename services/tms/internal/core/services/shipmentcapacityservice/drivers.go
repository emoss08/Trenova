package shipmentcapacityservice

import (
	"cmp"
	"context"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/core/services/dispatchconsoleservice"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/shopspring/decimal"
)

const msPerHour = float64(3_600_000)

func (s *Service) driverCapacity(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*shipmentports.ShipmentCapacity, error) {
	caps, err := s.capabilities.Capabilities(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	board, err := s.console.GetBoard(ctx, &dispatchconsoleservice.GetBoardRequest{
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}

	units := driverUnits(board.Drivers, s.now().Unix(), caps.HOS)
	summary := &shipmentports.DriverCapacitySummary{Uncovered: countUncovered(board.Moves)}
	for _, unit := range units {
		if unit.Group == shipmentports.CapacityGroupReadyNow {
			summary.Ready++
		} else {
			summary.WithinTwoHours++
		}
	}
	summary.Short = max(0, summary.Uncovered-summary.Ready-summary.WithinTwoHours)

	return &shipmentports.ShipmentCapacity{
		Kind:    shipmentports.CapacityUnitDriver,
		Units:   units,
		Drivers: summary,
	}, nil
}

func countUncovered(moves []*dispatchconsoleservice.BoardMove) int {
	count := 0
	for _, move := range moves {
		if move != nil && !move.IsCovered {
			count++
		}
	}

	return count
}

func driverUnits(
	drivers []*dispatchconsoleservice.BoardDriver,
	now int64,
	hos bool,
) []*shipmentports.CapacityUnit {
	cutoff := now + int64(withinTwoHours.Seconds())
	units := make([]*shipmentports.CapacityUnit, 0, len(drivers))
	for _, driver := range drivers {
		if driver == nil || driver.BoardDriver == nil || !driver.AvailableForDispatch {
			continue
		}

		var group shipmentports.CapacityGroup
		switch {
		case driver.Availability == dispatchconsoleservice.AvailabilityOpen:
			group = shipmentports.CapacityGroupReadyNow
		case driver.Availability == dispatchconsoleservice.AvailabilityFinishing &&
			driver.ProjectedTimeAvailable <= cutoff:
			group = shipmentports.CapacityGroupWithinTwoHours
		default:
			continue
		}

		name := strings.TrimSpace(driver.FirstName + " " + driver.LastName)
		unit := &shipmentports.CapacityUnit{
			ID:        driver.WorkerID,
			Kind:      shipmentports.CapacityUnitDriver,
			Name:      name,
			Initials:  stringutils.Initials(name),
			Group:     group,
			City:      driverCity(driver),
			UnitLabel: driver.TractorCode,
			TractorID: driver.TractorID,
		}
		if group == shipmentports.CapacityGroupWithinTwoHours {
			freeAt := driver.ProjectedTimeAvailable
			unit.FreeAt = &freeAt
		}
		if hos && !driver.HOSIsStale && driver.HOSRecordedAt > 0 {
			remaining := driver.DriveRemainingMs
			hours := float64(remaining) / msPerHour
			unit.DriveRemainingMs = &remaining
			unit.Ring = &shipmentports.CapacityRing{
				Value: hours,
				Max:   hosMaxHours,
				Low:   hours < hosLowHours,
			}
		}
		units = append(units, unit)
	}

	slices.SortStableFunc(units, func(a, b *shipmentports.CapacityUnit) int {
		return cmp.Or(
			cmp.Compare(groupOrder(a.Group), groupOrder(b.Group)),
			cmp.Compare(freeAtOf(a), freeAtOf(b)),
			strings.Compare(a.Name, b.Name),
		)
	})

	return units
}

func groupOrder(group shipmentports.CapacityGroup) int {
	switch group {
	case shipmentports.CapacityGroupReadyNow, shipmentports.CapacityGroupTrucksPosted:
		return 0
	case shipmentports.CapacityGroupWithinTwoHours, shipmentports.CapacityGroupUsuallyAccept:
		return 1
	default:
		return 1
	}
}

func freeAtOf(unit *shipmentports.CapacityUnit) int64 {
	if unit.FreeAt == nil {
		return 0
	}

	return *unit.FreeAt
}

func driverCity(driver *dispatchconsoleservice.BoardDriver) string {
	if location := strings.TrimSpace(driver.FormattedLocation); location != "" {
		return location
	}

	return strings.TrimSpace(driver.City)
}

func (s *Service) driverMatches(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
	limit int,
) ([]*shipmentports.CapacityMatch, error) {
	ranked, err := s.console.GetDriverMoves(ctx, &dispatchconsoleservice.DriverMovesRequest{
		TenantInfo: tenantInfo,
		WorkerID:   workerID,
		Limit:      driverMatchScan,
	})
	if err != nil {
		return nil, err
	}

	matches := make([]*shipmentports.CapacityMatch, 0, limit)
	for _, candidate := range ranked {
		if len(matches) == limit {
			break
		}
		if candidate == nil || candidate.Move == nil || candidate.Score == nil {
			continue
		}
		if candidate.Move.IsCovered || candidate.Score.Blocked() {
			continue
		}
		match := matchFromMove(candidate.Move)
		match.DeadheadMiles = candidate.Score.DeadheadMiles
		fit := float64(candidate.Score.Score)
		match.FitPercent = &fit
		matches = append(matches, match)
	}

	return matches, nil
}

func matchFromMove(move *dispatchconsoleservice.BoardMove) *shipmentports.CapacityMatch {
	revenue := decimal.Zero
	if move.Revenue != nil {
		revenue = decimal.NewFromFloat(*move.Revenue)
	}

	return &shipmentports.CapacityMatch{
		ShipmentID:      move.ShipmentID,
		MoveID:          move.MoveID,
		ProNumber:       move.ProNumber,
		OriginCity:      move.OriginCity,
		DestinationCity: move.DestinationCity,
		PickupAt:        move.OriginWindowStart,
		Revenue:         revenue,
	}
}
