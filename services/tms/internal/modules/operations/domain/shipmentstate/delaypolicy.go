package shipmentstate

import (
	"slices"

	"github.com/emoss08/trenova/internal/modules/identity/domain/shipmentcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
)

const DefaultDelayThresholdMinutes int16 = 30
const DisabledDelayThresholdMinutes int16 = -1

func ResolveDelayThresholdMinutes(delayThresholdMinutes int16) int16 {
	if delayThresholdMinutes <= 0 {
		return DefaultDelayThresholdMinutes
	}

	return delayThresholdMinutes
}

// ResolveControlDelayThreshold derives the delay threshold from the tenant's
// shipment control: delay automation is disabled when the control is missing
// or the AutoDelayShipments flag is off; otherwise the configured threshold
// (or the default) applies.
func ResolveControlDelayThreshold(control *shipmentcontrol.ShipmentControl) int16 {
	if control == nil || !control.AutoDelayShipments {
		return DisabledDelayThresholdMinutes
	}
	if control.AutoDelayShipmentsThreshold == nil {
		return ResolveDelayThresholdMinutes(0)
	}

	return ResolveDelayThresholdMinutes(*control.AutoDelayShipmentsThreshold)
}

func DelayedExcludedShipmentStatuses() []shipmentvocab.Status {
	return []shipmentvocab.Status{
		shipmentvocab.StatusDelayed,
		shipmentvocab.StatusCanceled,
		shipmentvocab.StatusReadyToInvoice,
		shipmentvocab.StatusCompleted,
		shipmentvocab.StatusInvoiced,
	}
}

func DelayedExcludedMoveStatuses() []shipmentvocab.MoveStatus {
	return []shipmentvocab.MoveStatus{
		shipmentvocab.MoveStatusCompleted,
		shipmentvocab.MoveStatusCanceled,
	}
}

func DelayedExcludedStopStatuses() []shipmentvocab.StopStatus {
	return []shipmentvocab.StopStatus{
		shipmentvocab.StopStatusCompleted,
		shipmentvocab.StopStatusCanceled,
	}
}

func IsDelayedEligibleShipmentStatus(status shipmentvocab.Status) bool {
	return !slices.Contains(DelayedExcludedShipmentStatuses(), status)
}

func IsDelayedEligibleMoveStatus(status shipmentvocab.MoveStatus) bool {
	return !slices.Contains(DelayedExcludedMoveStatuses(), status)
}

func IsDelayedEligibleStopStatus(status shipmentvocab.StopStatus) bool {
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
