package resolver

import "time"

// Lane and temporal variables never error: a shipment mid-entry may not have
// its stop locations loaded or any stops at all, and a formula referencing
// origin.state on it should price with an empty value rather than fail the
// save. Timestamps are interpreted in UTC because stops carry no timezone.

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
}

func firstStopOf(entity any) any {
	moves, err := getFieldSlice(entity, "Moves")
	if err != nil {
		return nil
	}

	for _, move := range moves {
		stops, stopsErr := getFieldSlice(move, "Stops")
		if stopsErr != nil || len(stops) == 0 {
			continue
		}
		return stops[0]
	}

	return nil
}

func lastStopOf(entity any) any {
	moves, err := getFieldSlice(entity, "Moves")
	if err != nil {
		return nil
	}

	for i := len(moves) - 1; i >= 0; i-- {
		stops, stopsErr := getFieldSlice(moves[i], "Stops")
		if stopsErr != nil || len(stops) == 0 {
			continue
		}
		return stops[len(stops)-1]
	}

	return nil
}

func stopLocationString(stop any, fieldName string) string {
	if stop == nil {
		return ""
	}

	location, err := getFieldValue(stop, "Location")
	if err != nil || location == nil || isNilInterface(location) {
		return ""
	}

	value, err := getFieldString(location, fieldName)
	if err != nil {
		return ""
	}

	return value
}

func stopLocationState(stop any) string {
	if stop == nil {
		return ""
	}

	location, err := getFieldValue(stop, "Location")
	if err != nil || location == nil || isNilInterface(location) {
		return ""
	}

	state, err := getFieldValue(location, "State")
	if err != nil || state == nil || isNilInterface(state) {
		return ""
	}

	abbreviation, err := getFieldString(state, "Abbreviation")
	if err != nil {
		return ""
	}

	return abbreviation
}

func computeOriginCity(entity any) (any, error) {
	return stopLocationString(firstStopOf(entity), "City"), nil
}

func computeOriginState(entity any) (any, error) {
	return stopLocationState(firstStopOf(entity)), nil
}

func computeOriginZip(entity any) (any, error) {
	return stopLocationString(firstStopOf(entity), "PostalCode"), nil
}

func computeDestinationCity(entity any) (any, error) {
	return stopLocationString(lastStopOf(entity), "City"), nil
}

func computeDestinationState(entity any) (any, error) {
	return stopLocationState(lastStopOf(entity)), nil
}

func computeDestinationZip(entity any) (any, error) {
	return stopLocationString(lastStopOf(entity), "PostalCode"), nil
}

func pickupTime(entity any) (time.Time, bool) {
	stop := firstStopOf(entity)
	if stop == nil {
		return time.Time{}, false
	}

	timestamp, ok := stopWindowStart(stop)
	if !ok {
		return time.Time{}, false
	}

	return time.Unix(timestamp, 0).UTC(), true
}

func computePickupDayOfWeek(entity any) (any, error) {
	pickup, ok := pickupTime(entity)
	if !ok {
		return 0, nil
	}

	return int(pickup.Weekday()), nil
}

func computePickupHour(entity any) (any, error) {
	pickup, ok := pickupTime(entity)
	if !ok {
		return 0, nil
	}

	return pickup.Hour(), nil
}

func computePickupMonth(entity any) (any, error) {
	pickup, ok := pickupTime(entity)
	if !ok {
		return 0, nil
	}

	return int(pickup.Month()), nil
}

func computeIsWeekendPickup(entity any) (any, error) {
	pickup, ok := pickupTime(entity)
	if !ok {
		return false, nil
	}

	weekday := pickup.Weekday()
	return weekday == time.Saturday || weekday == time.Sunday, nil
}

func getFieldString(entity any, fieldName string) (string, error) {
	value, err := getFieldValue(entity, fieldName)
	if err != nil {
		return "", err
	}

	switch typed := value.(type) {
	case string:
		return typed, nil
	case *string:
		if typed == nil {
			return "", nil
		}
		return *typed, nil
	default:
		return "", nil
	}
}
