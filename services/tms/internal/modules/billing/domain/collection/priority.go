package collection

// PromiseStatus is how a promise to pay is going.
type PromiseStatus string

const (
	// PromiseOpen is a promise whose date has not passed and is not yet paid.
	PromiseOpen = PromiseStatus("Open")
	// PromiseKept is a promise paid in full since it was made.
	PromiseKept = PromiseStatus("Kept")
	// PromiseBroken is a promise whose date has passed short of the amount.
	PromiseBroken = PromiseStatus("Broken")
)

const secondsPerDay = 24 * 60 * 60

// PromiseStatusOf judges a promise from what the customer has paid since it
// was made. promisedOn is the start of the promised day; the promise has the
// whole of that day to be kept.
func PromiseStatusOf(amountMinor, paidSinceMinor, promisedOn, now int64) PromiseStatus {
	switch {
	case paidSinceMinor >= amountMinor:
		return PromiseKept
	case now >= promisedOn+secondsPerDay:
		return PromiseBroken
	default:
		return PromiseOpen
	}
}

// FollowUpIsDue reports whether an account's follow-up has come round. The
// follow-up is the start of its day in the scheduler's time zone, so it is due
// from the moment that day begins there.
func FollowUpIsDue(nextFollowUpOn *int64, now int64) bool {
	return nextFollowUpOn != nil && *nextFollowUpOn <= now
}

// Priority orders the worklist: who to call first.
type Priority string

const (
	PriorityCritical = Priority("Critical")
	PriorityHigh     = Priority("High")
	PriorityNormal   = Priority("Normal")
	PriorityLow      = Priority("Low")
)

// PriorityFacts is what the worklist knows about an account when ranking it.
type PriorityFacts struct {
	OldestDaysPastDue int
	OverdueMinor      int64
	Promise           PromiseStatus
	HasPromise        bool
	FollowUpDue       bool
	FollowUpScheduled bool
}

// PriorityOf ranks an account. A broken promise or anything sixty days late
// comes first; a kept or open promise, or a follow-up not yet due, waits.
func PriorityOf(f PriorityFacts) Priority {
	switch {
	case f.HasPromise && f.Promise == PromiseBroken, f.OldestDaysPastDue >= 60:
		return PriorityCritical
	case f.HasPromise && f.Promise == PromiseOpen && !f.FollowUpDue:
		return PriorityLow
	case f.FollowUpScheduled && !f.FollowUpDue:
		return PriorityLow
	case f.OldestDaysPastDue >= 30, f.FollowUpDue:
		return PriorityHigh
	case f.OverdueMinor > 0:
		return PriorityNormal
	default:
		return PriorityLow
	}
}

// Rank is the sort weight of a priority, lowest first.
func (p Priority) Rank() int {
	switch p {
	case PriorityCritical:
		return 0
	case PriorityHigh:
		return 1
	case PriorityNormal:
		return 2
	default:
		return 3
	}
}
