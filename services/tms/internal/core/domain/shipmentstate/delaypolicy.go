package shipmentstate

import (
	"slices"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
)

const DefaultDelayThresholdMinutes int16 = 30
const DisabledDelayThresholdMinutes int16 = -1

func ResolveDelayThresholdMinutes(delayThresholdMinutes int16) int16 {
	if delayThresholdMinutes <= 0 {
		return DefaultDelayThresholdMinutes
	}

	return delayThresholdMinutes
}

func DelayedExcludedShipmentStatuses() []shipment.Status {
	return []shipment.Status{
		shipment.StatusDelayed,
		shipment.StatusCanceled,
		shipment.StatusReadyToInvoice,
		shipment.StatusCompleted,
		shipment.StatusInvoiced,
	}
}

func DelayedExcludedMoveStatuses() []shipment.MoveStatus {
	return []shipment.MoveStatus{
		shipment.MoveStatusCompleted,
		shipment.MoveStatusCanceled,
	}
}

func DelayedExcludedStopStatuses() []shipment.StopStatus {
	return []shipment.StopStatus{
		shipment.StopStatusCompleted,
		shipment.StopStatusCanceled,
	}
}

func IsDelayedEligibleShipmentStatus(status shipment.Status) bool {
	return !slices.Contains(DelayedExcludedShipmentStatuses(), status)
}

func IsDelayedEligibleMoveStatus(status shipment.MoveStatus) bool {
	return !slices.Contains(DelayedExcludedMoveStatuses(), status)
}

func IsDelayedEligibleStopStatus(status shipment.StopStatus) bool {
	return !slices.Contains(DelayedExcludedStopStatuses(), status)
}

func IsStopOverdue(stop *shipment.Stop, currentTime int64, delayThresholdMinutes int16) bool {
	if stop == nil || stop.IsCompleted() || stop.IsCanceled() || stop.ActualDeparture != nil {
		return false
	}
	if delayThresholdMinutes == DisabledDelayThresholdMinutes {
		return false
	}

	scheduledDeadline := stop.EffectiveScheduledCutoff()
	if scheduledDeadline <= 0 {
		return false
	}

	thresholdSeconds := int64(ResolveDelayThresholdMinutes(delayThresholdMinutes)) * 60
	return currentTime > scheduledDeadline+thresholdSeconds
}
