package settlementshared

import (
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/shared/pulid"
)

// PostingWorkflow captures the journal entry/batch workflow state a
// settlement posting starts in, derived from the accounting control's posting
// mode.
type PostingWorkflow struct {
	EntryStatus      string
	BatchStatus      string
	PostedAt         *int64
	PostedByID       pulid.ID
	RequiresApproval bool
	IsApproved       bool
	ApprovedByID     pulid.ID
	ApprovedAt       *int64
}

// ResolvePostingWorkflow mirrors the customer payment workflow shape: journals
// post immediately unless the control demands manual posting, which parks them
// as Pending (or Approved when no manual approval is required).
func ResolvePostingWorkflow(
	control *tenant.AccountingControl,
	userID pulid.ID,
	now int64,
) PostingWorkflow {
	workflow := PostingWorkflow{
		EntryStatus:      "Posted",
		BatchStatus:      "Posted",
		PostedAt:         &now,
		PostedByID:       userID,
		RequiresApproval: false,
		IsApproved:       true,
		ApprovedByID:     userID,
		ApprovedAt:       &now,
	}
	if control != nil && control.JournalPostingMode == tenant.JournalPostingModeManual {
		workflow.EntryStatus = "Pending"
		workflow.BatchStatus = "Pending"
		workflow.PostedAt = nil
		workflow.PostedByID = pulid.Nil
		workflow.RequiresApproval = control.RequireManualJEApproval
		workflow.IsApproved = !control.RequireManualJEApproval
		if !workflow.RequiresApproval {
			workflow.EntryStatus = "Approved"
			workflow.BatchStatus = "Approved"
		} else {
			workflow.ApprovedByID = pulid.Nil
			workflow.ApprovedAt = nil
		}
	}
	return workflow
}
