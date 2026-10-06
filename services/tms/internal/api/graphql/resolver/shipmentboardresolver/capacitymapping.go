package shipmentboardresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/base"
	"github.com/emoss08/trenova/internal/core/ports/services"
)

func capacityToModel(c *services.ShipmentCapacity) *gqlmodel.ShipmentCapacity {
	units := make([]*gqlmodel.CapacityUnit, 0, len(c.Units))
	for _, unit := range c.Units {
		units = append(units, capacityUnitToModel(unit))
	}

	out := &gqlmodel.ShipmentCapacity{
		Kind:  gqlmodel.CapacityUnitKind(c.Kind),
		Units: units,
	}
	if c.Drivers != nil {
		out.Drivers = &gqlmodel.DriverCapacitySummary{
			Ready:          c.Drivers.Ready,
			WithinTwoHours: c.Drivers.WithinTwoHours,
			Short:          c.Drivers.Short,
			Uncovered:      c.Drivers.Uncovered,
		}
	}
	if c.Carriers != nil {
		out.Carriers = &gqlmodel.CarrierCapacitySummary{
			Posting:            c.Carriers.Posting,
			Untendered:         c.Carriers.Untendered,
			AwaitingAcceptance: c.Carriers.AwaitingAcceptance,
			AvgRatePerMile:     base.NullDecimalStringPtr(c.Carriers.AvgRatePerMile),
		}
	}

	return out
}

func capacityUnitToModel(u *services.CapacityUnit) *gqlmodel.CapacityUnit {
	out := &gqlmodel.CapacityUnit{
		ID:                u.ID.String(),
		Kind:              gqlmodel.CapacityUnitKind(u.Kind),
		Name:              u.Name,
		Initials:          u.Initials,
		Group:             gqlmodel.CapacityGroup(u.Group),
		FreeAt:            base.IntPtr(u.FreeAt),
		City:              base.EmptyToNil(u.City),
		UnitLabel:         base.EmptyToNil(u.UnitLabel),
		BadgeCount:        u.BadgeCount,
		RatePerMile:       base.NullDecimalStringPtr(u.RatePerMile),
		AcceptancePercent: u.AcceptancePercent,
		DriveRemainingMs:  base.IntPtr(u.DriveRemainingMs),
		TractorID:         base.IDPtr(u.TractorID),
	}
	if u.Ring != nil {
		out.Ring = &gqlmodel.CapacityRing{Value: u.Ring.Value, Max: u.Ring.Max, Low: u.Ring.Low}
	}

	return out
}

func capacityMatchesToModel(matches []*services.CapacityMatch) []*gqlmodel.CapacityMatch {
	out := make([]*gqlmodel.CapacityMatch, 0, len(matches))
	for _, match := range matches {
		pickup := match.PickupAt
		out = append(out, &gqlmodel.CapacityMatch{
			ShipmentID:      match.ShipmentID.String(),
			MoveID:          match.MoveID.String(),
			ProNumber:       base.EmptyToNil(match.ProNumber),
			OriginCity:      match.OriginCity,
			DestinationCity: match.DestinationCity,
			PickupAt:        base.IntPtr(&pickup),
			Revenue:         match.Revenue.String(),
			DeadheadMiles:   match.DeadheadMiles,
			Quote:           base.NullDecimalStringPtr(match.Quote),
			MarginPercent:   match.MarginPercent,
			FitPercent:      match.FitPercent,
		})
	}

	return out
}

func coverageSuggestionsToModel(
	s *services.ShipmentCoverageSuggestions,
) *gqlmodel.ShipmentCoverageSuggestions {
	drivers := make([]*gqlmodel.DriverCoverageSuggestion, 0, len(s.Drivers))
	for _, driver := range s.Drivers {
		drivers = append(drivers, &gqlmodel.DriverCoverageSuggestion{
			WorkerID:         driver.WorkerID.String(),
			TractorID:        base.IDPtr(driver.TractorID),
			MoveID:           driver.MoveID.String(),
			Name:             driver.Name,
			Initials:         driver.Initials,
			UnitLabel:        base.EmptyToNil(driver.UnitLabel),
			DistanceMiles:    driver.DistanceMiles,
			DriveRemainingMs: base.IntPtr(driver.DriveRemainingMs),
			FitPercent:       driver.FitPercent,
		})
	}

	carriers := make([]*gqlmodel.CarrierCoverageSuggestion, 0, len(s.Carriers))
	for _, carrier := range s.Carriers {
		carriers = append(carriers, &gqlmodel.CarrierCoverageSuggestion{
			CarrierID:         carrier.CarrierID.String(),
			MoveID:            carrier.MoveID.String(),
			Name:              carrier.Name,
			Initials:          carrier.Initials,
			McNumber:          base.EmptyToNil(carrier.MCNumber),
			Quote:             carrier.Quote.String(),
			RatePerMile:       carrier.RatePerMile.String(),
			AcceptancePercent: carrier.AcceptancePercent,
			Posted:            carrier.Posted,
		})
	}

	return &gqlmodel.ShipmentCoverageSuggestions{Drivers: drivers, Carriers: carriers}
}

func tenderResultToModel(r *services.TenderShipmentsResult) *gqlmodel.TenderShipmentsResult {
	tendered := make([]*gqlmodel.TenderShipmentSuccess, 0, len(r.Tendered))
	for _, success := range r.Tendered {
		tendered = append(tendered, &gqlmodel.TenderShipmentSuccess{
			ShipmentID:  success.ShipmentID.String(),
			TenderID:    success.TenderID.String(),
			CarrierID:   base.IDPtr(success.CarrierID),
			CarrierName: base.EmptyToNil(success.CarrierName),
		})
	}

	failed := make([]*gqlmodel.TenderShipmentFailure, 0, len(r.Failed))
	for _, failure := range r.Failed {
		failed = append(failed, &gqlmodel.TenderShipmentFailure{
			ShipmentID: failure.ShipmentID.String(),
			Message:    failure.Message,
		})
	}

	return &gqlmodel.TenderShipmentsResult{Tendered: tendered, Failed: failed}
}
