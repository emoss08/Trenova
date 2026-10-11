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
	StatusApproved: {
		StatusInReview: {},
		StatusPosted:   {},
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
	StatusPosted:   {},
	StatusCanceled: {},
}

// voidReleaseTransitions is the one path that moves an item backwards: voiding
// the invoice it sat on. Rebill returns it to Approved for the next invoice;
// DoNotRebill cancels it. Nothing else may take a posted item out of Posted.
var voidReleaseTransitions = map[Status]map[Status]struct{}{
	StatusApproved: {
		StatusApproved: {},
		StatusCanceled: {},
	},
	StatusPosted: {
		StatusApproved: {},
		StatusCanceled: {},
	},
}

func IsAllowedVoidReleaseTransition(from, to Status) bool {
	next, ok := voidReleaseTransitions[from]
	if !ok {
		return false
	}

	_, allowed := next[to]
	return allowed
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
