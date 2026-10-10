package shipmentjobs

import (
	"github.com/emoss08/trenova/internal/core/temporaljobs"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
)

const AutoDelayShipmentsWorkflowName = "AutoDelayShipmentsWorkflow"
const AutoCancelShipmentsWorkflowName = "AutoCancelShipmentsWorkflow"

type BulkDuplicateShipmentsResult struct {
	ShipmentIDs      []pulid.ID `json:"shipmentIds"`
	DuplicatedCount  int        `json:"duplicatedCount"`
	CompletedAt      int64      `json:"completedAt"`
	SourceShipmentID pulid.ID   `json:"sourceShipmentId"`
}

type AutoDelayShipmentsPayload struct {
	temporaltype.BasePayload
}

type AutoDelayShipmentsResult struct {
	temporaljobs.TenantRunResult
	ShipmentIDs  []pulid.ID `json:"shipmentIds"`
	DelayedCount int        `json:"delayedCount"`
	CompletedAt  int64      `json:"completedAt"`
}

type AutoCancelShipmentsResult struct {
	temporaljobs.TenantRunResult
	ShipmentIDs   []pulid.ID `json:"shipmentIds"`
	CanceledCount int        `json:"canceledCount"`
	CompletedAt   int64      `json:"completedAt"`
}

type ListShipmentTenantsPayload struct {
	Limit int `json:"limit"`
}

type ListShipmentTenantsResult struct {
	Tenants []temporaljobs.TenantWorkItem `json:"tenants"`
}

type ShipmentTenantWorkPayload struct {
	temporaljobs.TenantWorkItem
}
