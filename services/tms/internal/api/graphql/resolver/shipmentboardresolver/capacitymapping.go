package shipmentboardresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqldecimal"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentcapacityports"
)

func capacityToModel(c *shipmentcapacityports.ShipmentCapacity) *gqlmodel.ShipmentCapacity {
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
			AvgRatePerMile:     gqldecimal.NullDecimalStringPtr(c.Carriers.AvgRatePerMile),
		}
	}

	return out
}

func capacityUnitToModel(u *shipmentcapacityports.CapacityUnit) *gqlmodel.CapacityUnit {
	out := &gqlmodel.CapacityUnit{
		ID:                u.ID.String(),
		Kind:              gqlmodel.CapacityUnitKind(u.Kind),
		Name:              u.Name,
		Initials:          u.Initials,
		Group:             gqlmodel.CapacityGroup(u.Group),
		FreeAt:            gqlscalar.IntPtr(u.FreeAt),
		City:              gqlscalar.EmptyToNil(u.City),
		UnitLabel:         gqlscalar.EmptyToNil(u.UnitLabel),
		BadgeCount:        u.BadgeCount,
		RatePerMile:       gqldecimal.NullDecimalStringPtr(u.RatePerMile),
		AcceptancePercent: u.AcceptancePercent,
		DriveRemainingMs:  gqlscalar.IntPtr(u.DriveRemainingMs),
		TractorID:         gqlid.IDPtr(u.TractorID),
	}
	if u.Ring != nil {
		out.Ring = &gqlmodel.CapacityRing{Value: u.Ring.Value, Max: u.Ring.Max, Low: u.Ring.Low}
	}

	return out
}

func capacityMatchesToModel(
	matches []*shipmentcapacityports.CapacityMatch,
) []*gqlmodel.CapacityMatch {
	out := make([]*gqlmodel.CapacityMatch, 0, len(matches))
	for _, match := range matches {
		pickup := match.PickupAt
		out = append(out, &gqlmodel.CapacityMatch{
			ShipmentID:      match.ShipmentID.String(),
			MoveID:          match.MoveID.String(),
			ProNumber:       gqlscalar.EmptyToNil(match.ProNumber),
			OriginCity:      match.OriginCity,
			DestinationCity: match.DestinationCity,
			PickupAt:        gqlscalar.IntPtr(&pickup),
			Revenue:         match.Revenue.String(),
			DeadheadMiles:   match.DeadheadMiles,
			Quote:           gqldecimal.NullDecimalStringPtr(match.Quote),
			MarginPercent:   match.MarginPercent,
			FitPercent:      match.FitPercent,
		})
	}

	return out
}

func coverageSuggestionsToModel(
	s *shipmentcapacityports.ShipmentCoverageSuggestions,
) *gqlmodel.ShipmentCoverageSuggestions {
	drivers := make([]*gqlmodel.DriverCoverageSuggestion, 0, len(s.Drivers))
	for _, driver := range s.Drivers {
		drivers = append(drivers, &gqlmodel.DriverCoverageSuggestion{
			WorkerID:         driver.WorkerID.String(),
			TractorID:        gqlid.IDPtr(driver.TractorID),
			MoveID:           driver.MoveID.String(),
			Name:             driver.Name,
			Initials:         driver.Initials,
			UnitLabel:        gqlscalar.EmptyToNil(driver.UnitLabel),
			DistanceMiles:    driver.DistanceMiles,
			DriveRemainingMs: gqlscalar.IntPtr(driver.DriveRemainingMs),
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
			McNumber:          gqlscalar.EmptyToNil(carrier.MCNumber),
			Quote:             carrier.Quote.String(),
			RatePerMile:       carrier.RatePerMile.String(),
			AcceptancePercent: carrier.AcceptancePercent,
			Posted:            carrier.Posted,
		})
	}

	return &gqlmodel.ShipmentCoverageSuggestions{Drivers: drivers, Carriers: carriers}
}

func tenderResultToModel(
	r *shipmentcapacityports.TenderShipmentsResult,
) *gqlmodel.TenderShipmentsResult {
	tendered := make([]*gqlmodel.TenderShipmentSuccess, 0, len(r.Tendered))
	for _, success := range r.Tendered {
		tendered = append(tendered, &gqlmodel.TenderShipmentSuccess{
			ShipmentID:  success.ShipmentID.String(),
			TenderID:    success.TenderID.String(),
			CarrierID:   gqlid.IDPtr(success.CarrierID),
			CarrierName: gqlscalar.EmptyToNil(success.CarrierName),
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
