package shipmenteffects

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/shipmentcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentstate"
)

func DelayThresholdMinutes(control *shipmentcontrol.ShipmentControl) int16 {
	if control == nil || !control.AutoDelayShipments {
		return shipmentstate.DisabledDelayThresholdMinutes
	}
	if control.AutoDelayShipmentsThreshold == nil {
		return shipmentstate.ResolveDelayThresholdMinutes(0)
	}

	return shipmentstate.ResolveDelayThresholdMinutes(*control.AutoDelayShipmentsThreshold)
}
