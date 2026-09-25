package aicorrectionservice

import (
	"cmp"
	"slices"
	"strconv"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/aicorrection"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/shared/timeutils"
)

const (
	confirmedReference = "referenceNumber"
	confirmedRate      = "rate"
	confirmedWeight    = "weight"
	confirmedPieces    = "pieceCount"
	confirmedCommodity = "commodity"
	confirmedShipper   = "shipper"
	confirmedConsignee = "consignee"
	confirmedPickup    = "pickupWindow"
	confirmedDelivery  = "deliveryWindow"
)

type confirmedStop struct {
	snapshot aicorrection.StopSnapshot
	location *time.Location
}

type confirmation struct {
	snapshot *aicorrection.Snapshot
	stops    []confirmedStop
}

func readConfirmation(shp *shipment.Shipment) *confirmation {
	c := &confirmation{
		snapshot: &aicorrection.Snapshot{Fields: map[string]string{}, Stops: []aicorrection.StopSnapshot{}},
	}

	setField(c.snapshot.Fields, confirmedReference, shp.BOL)
	if shp.FreightChargeAmount.Valid && !shp.FreightChargeAmount.Decimal.IsZero() {
		setField(c.snapshot.Fields, confirmedRate, shp.FreightChargeAmount.Decimal.StringFixed(2))
	}
	if shp.Weight != nil && *shp.Weight > 0 {
		setField(c.snapshot.Fields, confirmedWeight, strconv.FormatInt(*shp.Weight, 10))
	}
	if shp.Pieces != nil && *shp.Pieces > 0 {
		setField(c.snapshot.Fields, confirmedPieces, strconv.FormatInt(*shp.Pieces, 10))
	}
	for _, sc := range shp.Commodities {
		if sc != nil && sc.Commodity != nil && sc.Commodity.Name != "" {
			setField(c.snapshot.Fields, confirmedCommodity, sc.Commodity.Name)
			break
		}
	}

	c.stops = orderedStops(shp)
	for i := range c.stops {
		c.snapshot.Stops = append(c.snapshot.Stops, c.stops[i].snapshot)
	}

	if first := firstStop(c.stops, rolePickup); first != nil {
		setField(c.snapshot.Fields, confirmedShipper, first.snapshot.Name)
		setField(c.snapshot.Fields, confirmedPickup, first.snapshot.Date)
	}
	if last := lastStop(c.stops, roleDelivery); last != nil {
		setField(c.snapshot.Fields, confirmedConsignee, last.snapshot.Name)
		setField(c.snapshot.Fields, confirmedDelivery, last.snapshot.Date)
	}

	return c
}

func orderedStops(shp *shipment.Shipment) []confirmedStop {
	moves := make([]*shipment.ShipmentMove, 0, len(shp.Moves))
	for _, move := range shp.Moves {
		if move != nil {
			moves = append(moves, move)
		}
	}
	slices.SortStableFunc(moves, func(a, b *shipment.ShipmentMove) int {
		return cmp.Compare(a.Sequence, b.Sequence)
	})

	perRole := map[string]int{}
	result := make([]confirmedStop, 0, len(moves)*2)
	for _, move := range moves {
		stops := make([]*shipment.Stop, 0, len(move.Stops))
		for _, stop := range move.Stops {
			if stop != nil {
				stops = append(stops, stop)
			}
		}
		slices.SortStableFunc(stops, func(a, b *shipment.Stop) int {
			return cmp.Compare(a.Sequence, b.Sequence)
		})

		for _, stop := range stops {
			role := stopRole(stop.Type)
			if perRole[role] >= maxStopsPerRole {
				continue
			}
			result = append(result, confirmedStopFrom(stop, role, perRole[role]))
			perRole[role]++
		}
	}

	return result
}

func confirmedStopFrom(stop *shipment.Stop, role string, sequence int) confirmedStop {
	loc := time.UTC
	snapshot := aicorrection.StopSnapshot{
		Role:                 role,
		Sequence:             sequence,
		AppointmentRequired:  stop.ScheduleType == shipment.StopScheduleTypeAppointment,
		ScheduledWindowStart: stop.ScheduledWindowStart,
		ScheduledWindowEnd:   stop.ScheduledWindowEnd,
	}

	if l := stop.Location; l != nil {
		snapshot.Name = boundedText(l.Name)
		snapshot.AddressLine1 = boundedText(l.AddressLine1)
		snapshot.AddressLine2 = boundedText(l.AddressLine2)
		snapshot.City = boundedText(l.City)
		snapshot.PostalCode = boundedText(l.PostalCode)
		if l.State != nil {
			snapshot.State = l.State.Abbreviation
		}
		if l.Timezone != "" {
			if parsed, err := time.LoadLocation(l.Timezone); err == nil {
				loc = parsed
				snapshot.Timezone = l.Timezone
			}
		}
	}
	if snapshot.AddressLine1 == "" {
		snapshot.AddressLine1 = boundedText(stop.AddressLine)
	}
	if stop.ScheduledWindowStart > 0 {
		snapshot.Date = timeutils.FormatCalendarDate(stop.ScheduledWindowStart, loc)
	}

	return confirmedStop{snapshot: snapshot, location: loc}
}

func stopRole(stopType shipment.StopType) string {
	if slices.Contains(shipment.DeliveryStopTypes(), stopType) {
		return roleDelivery
	}

	return rolePickup
}

func firstStop(stops []confirmedStop, role string) *confirmedStop {
	for i := range stops {
		if stops[i].snapshot.Role == role {
			return &stops[i]
		}
	}

	return nil
}

func lastStop(stops []confirmedStop, role string) *confirmedStop {
	for i := len(stops) - 1; i >= 0; i-- {
		if stops[i].snapshot.Role == role {
			return &stops[i]
		}
	}

	return nil
}

func setField(fields map[string]string, key, value string) {
	if bounded := boundedText(value); bounded != "" {
		fields[key] = bounded
	}
}
