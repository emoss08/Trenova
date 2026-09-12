package invoicerun

// Status is where a run sits between being built and having issued its invoices.
//
// Committing exists so the commit is idempotent under retry: a worker that dies
// mid-commit leaves the run in Committing, and the next attempt resumes at the
// first group that has no invoice rather than starting over and double-billing.
type Status string

const (
	StatusBuilding   = Status("Building")
	StatusReady      = Status("Ready")
	StatusCommitting = Status("Committing")
	StatusCommitted  = Status("Committed")
	StatusFailed     = Status("Failed")
	StatusCanceled   = Status("Canceled")
)

func (s Status) IsValid() bool {
	switch s {
	case StatusBuilding, StatusReady, StatusCommitting, StatusCommitted,
		StatusFailed, StatusCanceled:
		return true
	default:
		return false
	}
}

// IsTerminal reports whether the run can no longer change.
func (s Status) IsTerminal() bool {
	return s == StatusCommitted || s == StatusFailed || s == StatusCanceled
}

// IsEditable reports whether an operator may still adjust group membership.
func (s Status) IsEditable() bool {
	return s == StatusReady
}

// Source records whether an operator asked for the run or the schedule did.
type Source string

const (
	SourceManual    = Source("Manual")
	SourceScheduled = Source("Scheduled")
)

func (s Source) IsValid() bool {
	switch s {
	case SourceManual, SourceScheduled:
		return true
	default:
		return false
	}
}

// GroupStatus is the outcome of one proposed invoice.
//
// Skipped is not a failure: a group below the customer's minimum, or one whose
// shipments were billed elsewhere between preview and commit, releases its items
// back to the next period rather than failing the whole run.
type GroupStatus string

const (
	GroupStatusPending   = GroupStatus("Pending")
	GroupStatusCommitted = GroupStatus("Committed")
	GroupStatusSkipped   = GroupStatus("Skipped")
	GroupStatusFailed    = GroupStatus("Failed")
)

func (s GroupStatus) IsValid() bool {
	switch s {
	case GroupStatusPending, GroupStatusCommitted, GroupStatusSkipped, GroupStatusFailed:
		return true
	default:
		return false
	}
}
