package billingqueuejobs

import (
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingapprovalrun"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
)

const (

	// invalidationResource is the key the browser maps back to its approval
	// run query; it must match RESOURCE_QUERY_KEY_MAP on the client.
	invalidationResource = "billing-queue-approval-run"

	ErrTypeApprovalItem = "BILLING_QUEUE_APPROVAL_ITEM"
)

type ApproveItemPayload struct {
	temporaltype.BasePayload

	RunID          pulid.ID `json:"runId"`
	ItemID         pulid.ID `json:"itemId"`
	AssignApprover bool     `json:"assignApprover,omitempty"`
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

	RunID          pulid.ID                               `json:"runId"`
	Status         billingapprovalrun.ApprovalRunStatus   `json:"status"`
	FailureMessage string                                 `json:"failureMessage,omitempty"`
	LeftoverCode   billingapprovalrun.ApprovalFailureCode `json:"leftoverCode,omitempty"`
}

type ApproveItemResult struct {
	Status      billingapprovalrun.ApprovalItemStatus  `json:"status"`
	FailureCode billingapprovalrun.ApprovalFailureCode `json:"failureCode,omitempty"`
}

type ApprovalRunResult struct {
	RunID         pulid.ID                             `json:"runId"`
	Status        billingapprovalrun.ApprovalRunStatus `json:"status"`
	ApprovedCount int                                  `json:"approvedCount"`
	FailedCount   int                                  `json:"failedCount"`
	SkippedCount  int                                  `json:"skippedCount"`
}
