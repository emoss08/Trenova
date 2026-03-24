package shipmentstate

import "github.com/emoss08/trenova/internal/core/domain/shipment"

const DefaultDelayThresholdMinutes int16 = 30
const DisabledDelayThresholdMinutes int16 = -1

func ResolveDelayThresholdMinutes(delayThresholdMinutes int16) int16 {
	if delayThresholdMinutes <= 0 {
		return DefaultDelayThresholdMinutes
	}

	return delayThresholdMinutes
}

func IsStopOverdue(stop *shipment.Stop, currentTime int64, delayThresholdMinutes int16) bool {
	if stop == nil || stop.IsCompleted() || stop.IsCanceled() || stop.ActualDeparture != nil {
		return false
	}
	if delayThresholdMinutes == DisabledDelayThresholdMinutes {
		return false
	}

	scheduledDeadline := stop.EffectiveScheduledWindowEnd()
	if scheduledDeadline <= 0 {
		return false
	}

	thresholdSeconds := int64(ResolveDelayThresholdMinutes(delayThresholdMinutes)) * 60
	return currentTime > scheduledDeadline+thresholdSeconds
}
