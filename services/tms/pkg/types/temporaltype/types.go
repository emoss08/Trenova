package temporaltype

import (
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	ShipmentTaskQueue     = "shipment-queue"
	NotificationTaskQueue = "notification-queue"
	SystemTaskQueue       = "system-queue"
)

const (
	CancelShipmentsScheduleID    = "cancel-shipments-schedule"
	DeleteAuditEntriesScheduleID = "delete-audit-entries-schedule"
)

type BasePayload struct {
	OrganizationID pulid.ID       `json:"organizationId"`
	BusinessUnitID pulid.ID       `json:"businessUnitId"`
	UserID         pulid.ID       `json:"userId,omitempty"`
	Timestamp      int64          `json:"timestamp"`
	Metadata       map[string]any `json:"metadata,omitempty"`
}

type WorkflowDefinition struct {
	Name        string
	Fn          any
	TaskQueue   string
	Description string
}
