package common

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
)

const ParamShipmentID = "shipmentId"

const ParamOwnerID = "ownerId"

const KindShipment = "shipment"

func ShipmentIDProperty(what string) map[string]any {
	return agenttoolschema.RecordID(permission.ResourceShipment, what,
		"get_shipment, search_shipments or the page you are on")
}

func ShipmentResult(action string, entity *shipment.Shipment) *agent.ToolExecutionResult {
	result := &agent.ToolExecutionResult{Action: action, Kind: KindShipment}
	if entity == nil {
		return result
	}
	result.Name = entity.ProNumber
	result.IDs = map[string]string{ParamShipmentID: entity.ID.String()}
	result.Record = RecordOf(ShipmentRecordEntity, entity.ID)

	return result
}
