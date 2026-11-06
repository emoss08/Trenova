package shipmentvalidator

import (
	"github.com/emoss08/trenova/internal/modules/operations/app/shipmenttimeline"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

type storedStop struct {
	stop   *shipment.Stop
	nextID pulid.ID
}

func validateMoveChronology(original, entity *shipment.Shipment) *errortypes.MultiError {
	if entity == nil {
		return nil
	}

	stored := storedStops(original)
	multiErr := errortypes.NewMultiError()
	for moveIndex, move := range entity.Moves {
		if move == nil {
			continue
		}

		for stopIndex := 0; stopIndex < len(move.Stops)-1; stopIndex++ {
			current, next := move.Stops[stopIndex], move.Stops[stopIndex+1]
			if current == nil || next == nil {
				continue
			}

			before, kept := stored[current.ID]
			if kept && before.nextID == next.ID {
				after := stored[next.ID]
				if !scheduleChanged(before.stop, current) && !scheduleChanged(after.stop, next) {
					continue
				}
			}

			checkStopPair(multiErr, moveIndex, stopIndex, current, next)
		}
	}

	if !multiErr.HasErrors() {
		return nil
	}

	return multiErr
}

func checkStopPair(
	multiErr *errortypes.MultiError,
	moveIndex, stopIndex int,
	current, next *shipment.Stop,
) {
	if current.EffectiveScheduledWindowEnd() >= next.ScheduledWindowStart {
		multiErr.Add(
			shipmenttimeline.StopFieldPath(moveIndex, stopIndex, "scheduledWindowEnd"),
			errortypes.ErrInvalid,
			"Scheduled window end must be before the next stop's scheduled window start",
		)
	}

	if current.ActualDeparture != nil &&
		next.ActualArrival != nil &&
		*current.ActualDeparture >= *next.ActualArrival {
		multiErr.Add(
			shipmenttimeline.StopFieldPath(moveIndex, stopIndex, "actualDeparture"),
			errortypes.ErrInvalid,
			"Actual departure must be before the next stop's actual arrival",
		)
	}
}

func storedStops(original *shipment.Shipment) map[pulid.ID]storedStop {
	stored := make(map[pulid.ID]storedStop)
	if original == nil {
		return stored
	}

	for _, move := range original.Moves {
		if move == nil {
			continue
		}

		for stopIndex, stop := range move.Stops {
			if stop == nil || stop.ID.IsNil() {
				continue
			}

			entry := storedStop{stop: stop}
			if stopIndex+1 < len(move.Stops) && move.Stops[stopIndex+1] != nil {
				entry.nextID = move.Stops[stopIndex+1].ID
			}
			stored[stop.ID] = entry
		}
	}

	return stored
}

func scheduleChanged(before, after *shipment.Stop) bool {
	if before == nil || after == nil {
		return true
	}

	return before.ScheduledWindowStart != after.ScheduledWindowStart ||
		!timestampsEqual(before.ScheduledWindowEnd, after.ScheduledWindowEnd) ||
		!timestampsEqual(before.ActualArrival, after.ActualArrival) ||
		!timestampsEqual(before.ActualDeparture, after.ActualDeparture)
}

func timestampsEqual(left, right *int64) bool {
	switch {
	case left == nil && right == nil:
		return true
	case left == nil || right == nil:
		return false
	default:
		return *left == *right
	}
}

func withMoveChronology(
	base *errortypes.MultiError,
	original, entity *shipment.Shipment,
) *errortypes.MultiError {
	chronology := validateMoveChronology(original, entity)
	if chronology == nil {
		return base
	}
	if base == nil {
		return chronology
	}
	for _, err := range chronology.Errors {
		base.AddError(err)
	}

	return base
}
