package invoice

// statusTransitions is the invoice lifecycle. A posted invoice is voided only
// through a full-reversal adjustment, which is what keeps the ledger honest: the
// adjustment writes the reversing entries and then marks the invoice.
var statusTransitions = map[Status]map[Status]struct{}{
	StatusDraft: {
		StatusPosted: {},
		StatusVoided: {},
	},
	StatusPosted: {
		StatusVoided: {},
	},
	StatusVoided: {},
}

func IsAllowedTransition(from, to Status) bool {
	if from == to {
		return true
	}

	next, ok := statusTransitions[from]
	if !ok {
		return false
	}

	_, allowed := next[to]
	return allowed
}

func IsTerminalStatus(status Status) bool {
	return len(statusTransitions[status]) == 0
}
