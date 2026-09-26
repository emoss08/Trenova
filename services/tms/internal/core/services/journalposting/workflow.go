package journalposting

import (
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	StatusPending  = "Pending"
	StatusApproved = "Approved"
	StatusPosted   = "Posted"
)

type Workflow struct {
	EntryStatus      string
	BatchStatus      string
	PostedAt         *int64
	PostedByID       pulid.ID
	RequiresApproval bool
	IsApproved       bool
	ApprovedByID     pulid.ID
	ApprovedAt       *int64
}

func (w *Workflow) Posted() bool {
	return w.PostedAt != nil
}

func ResolveWorkflow(control *tenant.AccountingControl, userID pulid.ID, now int64) Workflow {
	if control == nil || control.JournalPostingMode != tenant.JournalPostingModeManual {
		return Workflow{
			EntryStatus:  StatusPosted,
			BatchStatus:  StatusPosted,
			PostedAt:     &now,
			PostedByID:   userID,
			IsApproved:   true,
			ApprovedByID: userID,
			ApprovedAt:   &now,
		}
	}
	if control.RequireManualJEApproval {
		return Workflow{
			EntryStatus:      StatusPending,
			BatchStatus:      StatusPending,
			RequiresApproval: true,
		}
	}
	return Workflow{
		EntryStatus:  StatusApproved,
		BatchStatus:  StatusApproved,
		IsApproved:   true,
		ApprovedByID: userID,
		ApprovedAt:   &now,
	}
}
