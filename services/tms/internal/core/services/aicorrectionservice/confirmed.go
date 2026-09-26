package aicorrectionservice

import (
	"cmp"
	"slices"
	"strconv"

	"github.com/emoss08/trenova/internal/core/domain/aicorrection"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/shared/timeutils"
)

func readConfirmation(shp *shipment.Shipment) *aicorrection.Snapshot {
	snapshot := &aicorrection.Snapshot{
		Fields: map[string]string{},
		Stops:  orderedStops(shp),
	}

	setField(snapshot.Fields, aicorrection.FieldReference, shp.BOL)
	if shp.FreightChargeAmount.Valid && !shp.FreightChargeAmount.Decimal.IsZero() {
		setField(snapshot.Fields, aicorrection.FieldRate, shp.FreightChargeAmount.Decimal.StringFixed(2))
	}
	if shp.Weight != nil && *shp.Weight > 0 {
		setField(snapshot.Fields, aicorrection.FieldWeight, strconv.FormatInt(*shp.Weight, 10))
	}
	if shp.Pieces != nil && *shp.Pieces > 0 {
		setField(snapshot.Fields, aicorrection.FieldPieces, strconv.FormatInt(*shp.Pieces, 10))
	}
	for _, sc := range shp.Commodities {
		if sc != nil && sc.Commodity != nil && sc.Commodity.Name != "" {
			setField(snapshot.Fields, aicorrection.FieldCommodity, sc.Commodity.Name)
			break
		}
	}

	if first := snapshot.FirstStop(aicorrection.RolePickup); first != nil {
		setField(snapshot.Fields, aicorrection.FieldShipper, first.Name)
		setField(snapshot.Fields, aicorrection.FieldPickupWindow, first.Date)
	}
	if last := snapshot.LastStop(aicorrection.RoleDelivery); last != nil {
		setField(snapshot.Fields, aicorrection.FieldConsignee, last.Name)
		setField(snapshot.Fields, aicorrection.FieldDeliveryWindow, last.Date)
	}

	return snapshot
}

func orderedStops(shp *shipment.Shipment) []aicorrection.StopSnapshot {
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
	result := make([]aicorrection.StopSnapshot, 0, len(moves)*2)
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
			if perRole[role] >= aicorrection.MaxStopsPerRole {
				continue
			}
			result = append(result, confirmedStopFrom(stop, role, perRole[role]))
			perRole[role]++
		}
	}

	return result
}

func confirmedStopFrom(stop *shipment.Stop, role string, sequence int) aicorrection.StopSnapshot {
	snapshot := aicorrection.StopSnapshot{
		Role:                 role,
		Sequence:             sequence,
		AppointmentRequired:  stop.ScheduleType == shipment.StopScheduleTypeAppointment,
		ScheduledWindowStart: stop.ScheduledWindowStart,
		ScheduledWindowEnd:   stop.ScheduledWindowEnd,
	}

	if l := stop.Location; l != nil {
		snapshot.Name = aicorrection.BoundedText(l.Name)
		snapshot.AddressLine1 = aicorrection.BoundedText(l.AddressLine1)
		snapshot.AddressLine2 = aicorrection.BoundedText(l.AddressLine2)
		snapshot.City = aicorrection.BoundedText(l.City)
		snapshot.PostalCode = aicorrection.BoundedText(l.PostalCode)
		if l.State != nil {
			snapshot.State = l.State.Abbreviation
		}
		snapshot.Timezone = l.Timezone
		if snapshot.Location().String() != l.Timezone {
			snapshot.Timezone = ""
		}
	}
	if snapshot.AddressLine1 == "" {
		snapshot.AddressLine1 = aicorrection.BoundedText(stop.AddressLine)
	}
	if stop.ScheduledWindowStart > 0 {
		snapshot.Date = timeutils.FormatCalendarDate(stop.ScheduledWindowStart, snapshot.Location())
	}

	return snapshot
}

func stopRole(stopType shipment.StopType) string {
	if slices.Contains(shipment.DeliveryStopTypes(), stopType) {
		return aicorrection.RoleDelivery
	}

	return aicorrection.RolePickup
}

func setField(fields map[string]string, key, value string) {
	if bounded := aicorrection.BoundedText(value); bounded != "" {
		fields[key] = bounded
	}
}
