package billingqueuecontract

import (
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
)

const BulkApprovalWorkflowName = "BillingQueueBulkApprovalWorkflow"

// UndoSignalName ends a run inside its undo window. Like the transfer's
// cancel signal it only shortens the wait: the run row's
// cancel_requested_at is the authority, read when the window closes.
const UndoSignalName = "billing-queue-approval-undo"

type ApprovalRunPayload struct {
	temporaltype.BasePayload

	RunID pulid.ID `json:"runId"`
	// AssignApprover: the person agreed to become the biller of the items
	// that have none.
	AssignApprover bool `json:"assignApprover,omitempty"`
}

func (p *ApprovalRunPayload) Tenant() pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  p.OrganizationID,
		BuID:   p.BusinessUnitID,
		UserID: p.UserID,
	}
}
