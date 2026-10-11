package invoicerun

// allowedTransitions is the only path a run may take.
//
// Nothing leaves a terminal status. In particular a Failed run is never
// re-committed: its groups may already hold invoices, and re-running it would
// bill them twice. Building a fresh run from the same period is the way back.
var allowedTransitions = map[Status][]Status{
	StatusBuilding:   {StatusReady, StatusFailed, StatusCanceled},
	StatusReady:      {StatusCommitting, StatusCanceled},
	StatusCommitting: {StatusCommitted, StatusFailed},
	StatusCommitted:  {},
	StatusFailed:     {},
	StatusCanceled:   {},
}

// CanTransition reports whether a run may move from one status to another.
// A status is always allowed to stay where it is, so a retry that re-applies
// the same status is not an error.
func CanTransition(from, to Status) bool {
	if from == to {
		return true
	}

	for _, next := range allowedTransitions[from] {
		if next == to {
			return true
		}
	}

	return false
}
