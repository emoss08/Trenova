package agentapprovalports

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// ApprovalUndoWindow is how long an approval made from a person's own
// conversation waits before it goes through. Until then they can undo it,
// and nothing has been written.
const ApprovalUndoWindow = 5 * time.Second

// CommitApprovalRequest names an approval whose undo window has closed: the
// decisions of one proposal or batch by the workflow that holds them, or a
// plan with what its decider sent. The commit runs as the person who approved.
type CommitApprovalRequest struct {
	OrganizationID  pulid.ID `json:"organizationId"`
	BusinessUnitID  pulid.ID `json:"businessUnitId"`
	DecidedByUserID pulid.ID `json:"decidedByUserId"`
	WorkflowID      string   `json:"workflowId"`
	PlanID          pulid.ID `json:"planId,omitempty"`
	ReasonCode      string   `json:"reasonCode,omitempty"`
	Note            string   `json:"note,omitempty"`
	PreviewDigest   string   `json:"previewDigest,omitempty"`
	// CommitsAt is when the window closes, in Unix seconds.
	CommitsAt int64 `json:"commitsAt"`
}

// TenantInfo is the approver's scope.
func (r *CommitApprovalRequest) TenantInfo() pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  r.OrganizationID,
		BuID:   r.BusinessUnitID,
		UserID: r.DecidedByUserID,
	}
}

// SettleApprovalRequest is a person acting on their own approval while it is
// in its undo window: on a proposal (and the batch it was approved with) or
// on a plan. Exactly one of the two is set.
type SettleApprovalRequest struct {
	ProposalID pulid.ID
	PlanID     pulid.ID
	TenantInfo pagination.TenantInfo
}

// ApprovalCommitter carries out approvals once their undo window closes, and
// takes them back while it is open. Undo after the commit is a conflict.
type ApprovalCommitter interface {
	CommitApproval(ctx context.Context, req *CommitApprovalRequest) error
	UndoOwnApproval(
		ctx context.Context,
		req *SettleApprovalRequest,
		actor *permissionports.RequestActor,
	) error
	CommitOwnApprovalNow(
		ctx context.Context,
		req *SettleApprovalRequest,
		actor *permissionports.RequestActor,
	) error
}
