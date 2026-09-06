package ptojobs

import (
	"github.com/emoss08/trenova/internal/core/temporaljobs"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	PTOAccrualWorkflowName = "PTOAccrualWorkflow"
	ScheduleID             = "pto-accrual"
	accrualPageSize        = 100
)

type PTOAccrualWorkflowInput struct {
	OrganizationID pulid.ID `json:"organizationId,omitempty"`
	BusinessUnitID pulid.ID `json:"businessUnitId,omitempty"`
	WorkerID       pulid.ID `json:"workerId,omitempty"`
	AsOf           int64    `json:"asOf,omitempty"`
}

type AccrualTenantPageInput struct {
	temporaljobs.TenantWorkItem
	WorkerID pulid.ID `json:"workerId,omitempty"`
	AsOf     int64    `json:"asOf"`
	AfterID  pulid.ID `json:"afterId,omitempty"`
}

type AccrualTenantPageResult struct {
	WorkersProcessed int      `json:"workersProcessed"`
	WorkersSkipped   int      `json:"workersSkipped"`
	EntriesPosted    int      `json:"entriesPosted"`
	EntriesCapped    int      `json:"entriesCapped"`
	EntriesSkipped   int      `json:"entriesSkipped"`
	NextAfterID      pulid.ID `json:"nextAfterId,omitempty"`
}

type PTOAccrualResult struct {
	temporaljobs.TenantRunResult
	EntriesPosted  int   `json:"entriesPosted"`
	EntriesCapped  int   `json:"entriesCapped"`
	EntriesSkipped int   `json:"entriesSkipped"`
	CompletedAt    int64 `json:"completedAt"`
}
