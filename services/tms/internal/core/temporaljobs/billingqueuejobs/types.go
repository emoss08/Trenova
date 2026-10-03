package billingqueuejobs

import (
	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	BulkApprovalWorkflowName = "BillingQueueBulkApprovalWorkflow"

	// UndoSignalName ends a run inside its undo window. Like the transfer's
	// cancel signal it only shortens the wait: the run row's
	// cancel_requested_at is the authority, read when the window closes.
	UndoSignalName = "billing-queue-approval-undo"

	// invalidationResource is the key the browser maps back to its approval
	// run query; it must match RESOURCE_QUERY_KEY_MAP on the client.
	invalidationResource = "billing-queue-approval-run"

	ErrTypeApprovalItem = "BILLING_QUEUE_APPROVAL_ITEM"
)

type ApprovalRunPayload struct {
	temporaltype.BasePayload

	RunID pulid.ID `json:"runId"`
}

func (p *ApprovalRunPayload) tenant() pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  p.OrganizationID,
		BuID:   p.BusinessUnitID,
		UserID: p.UserID,
	}
}

type ApproveItemPayload struct {
	temporaltype.BasePayload

	RunID  pulid.ID `json:"runId"`
	ItemID pulid.ID `json:"itemId"`
}

func (p *ApproveItemPayload) tenant() pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  p.OrganizationID,
		BuID:   p.BusinessUnitID,
		UserID: p.UserID,
	}
}

type RecordItemFailurePayload struct {
	temporaltype.BasePayload

	RunID   pulid.ID `json:"runId"`
	ItemID  pulid.ID `json:"itemId"`
	Message string   `json:"message"`
}

type FinalizeApprovalRunPayload struct {
	temporaltype.BasePayload

	RunID          pulid.ID                         `json:"runId"`
	Status         billingqueue.ApprovalRunStatus   `json:"status"`
	FailureMessage string                           `json:"failureMessage,omitempty"`
	LeftoverCode   billingqueue.ApprovalFailureCode `json:"leftoverCode,omitempty"`
}

type ApproveItemResult struct {
	Status      billingqueue.ApprovalItemStatus  `json:"status"`
	FailureCode billingqueue.ApprovalFailureCode `json:"failureCode,omitempty"`
}

type ApprovalRunResult struct {
	RunID         pulid.ID                       `json:"runId"`
	Status        billingqueue.ApprovalRunStatus `json:"status"`
	ApprovedCount int                            `json:"approvedCount"`
	FailedCount   int                            `json:"failedCount"`
	SkippedCount  int                            `json:"skippedCount"`
}
