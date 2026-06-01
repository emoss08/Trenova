package servicefailureservice

import (
	"github.com/emoss08/trenova/internal/core/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
)

type shouldEvaluateStopParams struct {
	source      *shipment.Shipment
	stop        *shipment.Stop
	shipperStop *shipment.Stop
	policy      dispatchcontrol.ServiceIncidentType
}

func shouldEvaluateStop(params shouldEvaluateStopParams) bool {
	if params.stop.CountLateOverride != nil {
		return *params.stop.CountLateOverride
	}

	switch params.policy {
	case dispatchcontrol.ServiceIncidentTypePickup:
		return params.stop.IsOriginStop()
	case dispatchcontrol.ServiceIncidentTypeDelivery:
		return params.stop.IsDestinationStop()
	case dispatchcontrol.ServiceIncidentTypePickupDelivery:
		return params.stop.IsOriginStop() || params.stop.IsDestinationStop()
	case dispatchcontrol.ServiceIncidentTypeAllExceptShipper:
		if params.shipperStop != nil && params.stop.ID == params.shipperStop.ID {
			return false
		}
		return params.stop.IsOriginStop() || params.stop.IsDestinationStop()
	case dispatchcontrol.ServiceIncidentTypeNever:
		return false
	default:
		return false
	}
}
