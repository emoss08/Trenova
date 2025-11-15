package temporaltype

import (
	"github.com/emoss08/trenova/pkg/pulid"
)

const (
	ShipmentTaskQueue     = "shipment-queue"
	NotificationTaskQueue = "notification-queue"
	AILogTaskQueue        = "ailog-queue"
	SystemTaskQueue       = "system-queue"
	EmailTaskQueue        = "email-queue"
	AuditTaskQueue        = "audit-queue"
	ReportTaskQueue       = "report-queue"
	WorkflowTaskQueue     = "workflow-queue"
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

func (b *BasePayload) GetOrganizationID() pulid.ID {
	return b.OrganizationID
}

func (b *BasePayload) GetBusinessUnitID() pulid.ID {
	return b.BusinessUnitID
}

type WorkflowDefinition struct {
	Name        string
	Fn          any
	TaskQueue   string
	Description string
}
