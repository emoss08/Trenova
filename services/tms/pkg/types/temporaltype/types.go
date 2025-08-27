package temporaltype

import (
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	TaskQueueShipmentWorker = "shipment-worker"
)

type BasePayload struct {
	OrganizationID pulid.ID       `json:"organizationId"`
	BusinessUnitID pulid.ID       `json:"businessUnitId"`
	UserID         pulid.ID       `json:"userId,omitempty"`
	Timestamp      int64          `json:"timestamp"`
	Metadata       map[string]any `json:"metadata,omitempty"`
}
