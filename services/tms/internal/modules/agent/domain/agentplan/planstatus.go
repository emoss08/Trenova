package agentplan

type PlanStatus string

const (
	PlanStatusPending = PlanStatus("Pending")
	// PlanStatusApproving is a plan approved from the person's own
	// conversation that commits a few seconds later unless they undo it.
	PlanStatusApproving = PlanStatus("Approving")
	PlanStatusApproved  = PlanStatus("Approved")
	PlanStatusCompleted = PlanStatus("Completed")
	PlanStatusFailed    = PlanStatus("Failed")
	PlanStatusRejected  = PlanStatus("Rejected")
	PlanStatusExpired   = PlanStatus("Expired")
)

func (s PlanStatus) IsValid() bool {
	switch s {
	case PlanStatusPending,
		PlanStatusApproving,
		PlanStatusApproved,
		PlanStatusCompleted,
		PlanStatusFailed,
		PlanStatusRejected,
		PlanStatusExpired:
		return true
	default:
		return false
	}
}

// Decidable reports whether a plan is still waiting on a person.
func (s PlanStatus) Decidable() bool { return s == PlanStatusPending }

// Expirable is ProposalStatus.Expirable for a plan.
func (s PlanStatus) Expirable() bool { return s == PlanStatusPending }
