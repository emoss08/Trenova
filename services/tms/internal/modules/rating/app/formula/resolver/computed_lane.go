package resolver

import (
	"time"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
)

func registerLaneComputed(r *Resolver) {
	r.RegisterComputed("computeOriginCity", computeOriginCity)
	r.RegisterComputed("computeOriginState", computeOriginState)
	r.RegisterComputed("computeOriginZip", computeOriginZip)
	r.RegisterComputed("computeDestinationCity", computeDestinationCity)
	r.RegisterComputed("computeDestinationState", computeDestinationState)
	r.RegisterComputed("computeDestinationZip", computeDestinationZip)
	r.RegisterComputed("computePickupDayOfWeek", computePickupDayOfWeek)
	r.RegisterComputed("computePickupHour", computePickupHour)
	r.RegisterComputed("computePickupMonth", computePickupMonth)
	r.RegisterComputed("computeIsWeekendPickup", computeIsWeekendPickup)
	r.RegisterComputed("computePickupDate", computePickupDate)
	r.RegisterComputed("computeDeliveryDate", computeDeliveryDate)
}

func originStopOf(entity any) (*shipment.Stop, error) {
	entry, err := shipmentOf(entity)
	if err != nil {
		return nil, err
	}
	return entry.ShipperStop(), nil
}

func destinationStopOf(entity any) (*shipment.Stop, error) {
	entry, err := shipmentOf(entity)
	if err != nil {
		return nil, err
	}
	return entry.DeliveryStop(), nil
}

func stopCity(stop *shipment.Stop) string {
	if stop == nil || stop.Location == nil {
		return ""
	}
	return stop.Location.City
}

func stopZip(stop *shipment.Stop) string {
	if stop == nil || stop.Location == nil {
		return ""
	}
	return stop.Location.PostalCode
}

func stopTimezone(stop *shipment.Stop) string {
	if stop == nil || stop.Location == nil {
		return ""
	}
	return stop.Location.Timezone
}

func stopState(stop *shipment.Stop) string {
	if stop == nil || stop.Location == nil || stop.Location.State == nil {
		return ""
	}
	return stop.Location.State.Abbreviation
}

func laneText(
	pick func(any) (*shipment.Stop, error),
	read func(*shipment.Stop) string,
) ComputedFunc {
	return func(entity any) (any, error) {
		stop, err := pick(entity)
		if err != nil {
			return "", err
		}
		return read(stop), nil
	}
}

var (
	computeOriginCity       = laneText(originStopOf, stopCity)
	computeOriginState      = laneText(originStopOf, stopState)
	computeOriginZip        = laneText(originStopOf, stopZip)
	computeDestinationCity  = laneText(destinationStopOf, stopCity)
	computeDestinationState = laneText(destinationStopOf, stopState)
	computeDestinationZip   = laneText(destinationStopOf, stopZip)
)

func stopWindowStart(stop *shipment.Stop) (int64, bool) {
	if stop == nil {
		return 0, false
	}
	if stop.ActualArrival != nil {
		return *stop.ActualArrival, true
	}
	if stop.ScheduledWindowStart > 0 {
		return stop.ScheduledWindowStart, true
	}
	return 0, false
}

func stopWindowEnd(stop *shipment.Stop) (int64, bool) {
	if stop == nil {
		return 0, false
	}
	switch {
	case stop.ActualDeparture != nil:
		return *stop.ActualDeparture, true
	case stop.ActualArrival != nil:
		return *stop.ActualArrival, true
	case stop.ScheduledWindowEnd != nil && *stop.ScheduledWindowEnd > 0:
		return *stop.ScheduledWindowEnd, true
	case stop.ScheduledWindowStart > 0:
		return stop.ScheduledWindowStart, true
	default:
		return 0, false
	}
}

func stopLocalTime(stop *shipment.Stop, timestamp int64) time.Time {
	instant := time.Unix(timestamp, 0).UTC()

	zoneName := stopTimezone(stop)
	if zoneName == "" {
		return instant
	}

	zone, err := time.LoadLocation(zoneName)
	if err != nil {
		return instant
	}

	return instant.In(zone)
}

func pickupTime(entity any) (time.Time, bool, error) {
	stop, err := originStopOf(entity)
	if err != nil {
		return time.Time{}, false, err
	}

	timestamp, ok := stopWindowStart(stop)
	if !ok {
		return time.Time{}, false, nil
	}

	return stopLocalTime(stop, timestamp), true, nil
}

func deliveryTime(entity any) (time.Time, bool, error) {
	stop, err := destinationStopOf(entity)
	if err != nil {
		return time.Time{}, false, err
	}

	timestamp, ok := stopWindowEnd(stop)
	if !ok {
		return time.Time{}, false, nil
	}

	return stopLocalTime(stop, timestamp), true, nil
}

func computePickupDate(entity any) (any, error) {
	pickup, ok, err := pickupTime(entity)
	if err != nil || !ok {
		return nil, err
	}
	return pickup, nil
}

func computeDeliveryDate(entity any) (any, error) {
	delivery, ok, err := deliveryTime(entity)
	if err != nil || !ok {
		return nil, err
	}
	return delivery, nil
}

func pickupPart(read func(time.Time) any, zero any) ComputedFunc {
	return func(entity any) (any, error) {
		pickup, ok, err := pickupTime(entity)
		if err != nil {
			return zero, err
		}
		if !ok {
			return zero, nil
		}
		return read(pickup), nil
	}
}

var (
	computePickupDayOfWeek = pickupPart(
		func(t time.Time) any { return int64(t.Weekday()) },
		int64(0),
	)
	computePickupHour      = pickupPart(func(t time.Time) any { return int64(t.Hour()) }, int64(0))
	computePickupMonth     = pickupPart(func(t time.Time) any { return int64(t.Month()) }, int64(0))
	computeIsWeekendPickup = pickupPart(func(t time.Time) any {
		return t.Weekday() == time.Saturday || t.Weekday() == time.Sunday
	}, false)
)
