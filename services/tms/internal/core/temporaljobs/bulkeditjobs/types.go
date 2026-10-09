package bulkeditjobs

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/bulkedit"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	BulkEditWorkflowName = "BulkEditWorkflow"
	BatchSize            = 50
	WorkflowIDPrefix     = "bulk-edit-"
	ErrTypeNotRunnable   = "BULK_EDIT_NOT_RUNNABLE"
)

type Runner interface {
	Prepare(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		id pulid.ID,
	) (*bulkedit.BulkEdit, error)
	ProcessBatch(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		id pulid.ID,
		limit int,
	) (int, error)
	Finalize(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		id pulid.ID,
		failure string,
	) (*bulkedit.BulkEdit, error)
}

type EditPayload struct {
	temporaltype.BasePayload

	EditID pulid.ID `json:"editId"`
}

type PreparedEdit struct {
	TotalCount int `json:"totalCount"`
}

type BatchPayload struct {
	temporaltype.BasePayload

	EditID pulid.ID `json:"editId"`
	Limit  int      `json:"limit"`
}

type FinalizePayload struct {
	temporaltype.BasePayload

	EditID  pulid.ID `json:"editId"`
	Failure string   `json:"failure,omitempty"`
}

type EditResult struct {
	Status         bulkedit.Status `json:"status"`
	ChangedCount   int             `json:"changedCount"`
	FailedCount    int             `json:"failedCount"`
	ProcessedCount int             `json:"processedCount"`
}

func tenantOf(payload temporaltype.BasePayload) pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  payload.OrganizationID,
		BuID:   payload.BusinessUnitID,
		UserID: payload.UserID,
	}
}
