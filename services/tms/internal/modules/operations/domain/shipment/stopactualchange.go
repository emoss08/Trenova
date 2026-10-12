package shipment

import (
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

type StopActualChange struct {
	StopID    pulid.ID
	Arrival   *int64
	Departure *int64
}

func (c StopActualChange) Empty() bool {
	return c.Arrival == nil && c.Departure == nil
}

func (m *ShipmentMove) ApplyStopActualChanges(changes []StopActualChange) ([]*Stop, error) {
	if m.Status == MoveStatusCanceled {
		return nil, errortypes.NewBusinessError("This load has been canceled")
	}

	stops := m.ActiveStopsBySequence()
	byID := make(map[pulid.ID]*Stop, len(stops))
	for _, stop := range stops {
		byID[stop.ID] = stop
	}

	changed := make(map[pulid.ID]struct{}, len(changes))
	touched := make([]*Stop, 0, len(changes))
	for _, change := range changes {
		if change.Empty() {
			continue
		}
		stop, ok := byID[change.StopID]
		if !ok {
			return nil, errortypes.NewNotFoundError("Stop not found on this load")
		}
		if change.Arrival != nil {
			arrival := *change.Arrival
			stop.ActualArrival = &arrival
		}
		if change.Departure != nil {
			departure := *change.Departure
			stop.ActualDeparture = &departure
		}
		if _, seen := changed[stop.ID]; !seen {
			changed[stop.ID] = struct{}{}
			touched = append(touched, stop)
		}
	}

	if err := validateStopTimeline(stops, changed); err != nil {
		return nil, err
	}

	for _, stop := range touched {
		switch {
		case stop.ActualDeparture != nil:
			stop.Status = StopStatusCompleted
		case stop.ActualArrival != nil:
			stop.Status = StopStatusInTransit
		}
	}

	return touched, nil
}

func validateStopTimeline(stops []*Stop, changed map[pulid.ID]struct{}) error {
	var previousDeparture *int64
	earlierOpen := false
	previousChanged := false
	for _, stop := range stops {
		_, isChanged := changed[stop.ID]
		check := isChanged || previousChanged

		if stop.ActualArrival != nil && check {
			if isChanged && earlierOpen {
				return errortypes.NewBusinessError("Complete the earlier stops on this load first")
			}
			if previousDeparture != nil && *previousDeparture > *stop.ActualArrival {
				return errortypes.NewBusinessError(
					"Event time predates the previous stop's departure",
				)
			}
		}
		if stop.ActualDeparture != nil {
			if check {
				if stop.ActualArrival == nil {
					return errortypes.NewBusinessError("Arrive at this stop before departing")
				}
				if *stop.ActualArrival > *stop.ActualDeparture {
					return errortypes.NewBusinessError(
						"Event time predates the arrival at this stop",
					)
				}
			}
			previousDeparture = stop.ActualDeparture
		} else {
			earlierOpen = true
		}
		previousChanged = isChanged
	}
	return nil
}
