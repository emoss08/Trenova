package shipmentlifecycle

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/shipmentcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentstate"
)

func autoCancelThresholdDays(control *shipmentcontrol.ShipmentControl) int16 {
	if control.AutoCancelShipmentsThreshold == nil {
		return shipmentstate.ResolveAutoCancelThresholdDays(0)
	}

	return shipmentstate.ResolveAutoCancelThresholdDays(*control.AutoCancelShipmentsThreshold)
}
