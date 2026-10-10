package shipmentcontract

import (
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
)

const BulkDuplicateShipmentsWorkflowName = "BulkDuplicateShipmentsWorkflow"

type BulkDuplicateShipmentsPayload struct {
	temporaltype.BasePayload
	ShipmentID    pulid.ID `json:"shipmentId"`
	Count         int      `json:"count"`
	OverrideDates bool     `json:"overrideDates"`
	FirstPickupAt *int64   `json:"firstPickupAt,omitempty"`
	RequestedBy   pulid.ID `json:"requestedBy"`
}
