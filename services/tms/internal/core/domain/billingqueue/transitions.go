package billingqueue

var statusTransitions = map[Status]map[Status]struct{}{
	StatusReadyForReview: {
		StatusInReview: {},
		StatusOnHold:   {},
		StatusCanceled: {},
	},
	StatusInReview: {
		StatusApproved:      {},
		StatusSentBackToOps: {},
		StatusException:     {},
		StatusOnHold:        {},
		StatusCanceled:      {},
	},
	StatusOnHold: {
		StatusReadyForReview: {},
		StatusInReview:       {},
		StatusCanceled:       {},
	},
	StatusSentBackToOps: {
		StatusReadyForReview: {},
		StatusCanceled:       {},
	},
	StatusException: {
		StatusReadyForReview: {},
		StatusInReview:       {},
		StatusCanceled:       {},
	},
	StatusApproved: {
		StatusInReview: {},
	},
	StatusCanceled: {},
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
