package driversettlement

var statusTransitions = map[Status]map[Status]struct{}{
	StatusDraft: {
		StatusPendingApproval: {},
		StatusVoided:          {},
	},
	StatusPendingApproval: {
		StatusDraft:    {},
		StatusApproved: {},
		StatusVoided:   {},
	},
	StatusApproved: {
		StatusPendingApproval: {},
		StatusPosted:          {},
		StatusVoided:          {},
	},
	StatusPosted: {
		StatusPaid:   {},
		StatusVoided: {},
	},
	StatusPaid:   {},
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
