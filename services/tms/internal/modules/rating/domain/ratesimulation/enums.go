package ratesimulation

// Status is where a simulation is in its life.
//
// A simulation is a long-running job — a year of shipments re-rated one at a
// time — so it has to be able to say "still working" rather than only "done".
type Status string

const (
	StatusPending   = Status("Pending")
	StatusRunning   = Status("Running")
	StatusCompleted = Status("Completed")
	StatusFailed    = Status("Failed")
	StatusCanceled  = Status("Canceled")
)

func (s Status) String() string { return string(s) }

func (s Status) IsValid() bool {
	switch s {
	case StatusPending, StatusRunning, StatusCompleted, StatusFailed, StatusCanceled:
		return true
	default:
		return false
	}
}

// IsTerminal reports whether a simulation has finished, whatever the outcome.
func (s Status) IsTerminal() bool {
	return s == StatusCompleted || s == StatusFailed || s == StatusCanceled
}

// RuleOutcome is what happened to one of the simulated agreement's rules across
// a whole run.
//
// The two failure modes are quiet and different. A rule that never matched
// anything was written for traffic that does not exist. A rule that matched but
// always lost was written under a lane somebody else already covers more
// narrowly. Both look identical from the revenue total, and both are the reason
// somebody's carefully written tariff does nothing.
type RuleOutcome string

const (
	RuleOutcomeWon        = RuleOutcome("Won")
	RuleOutcomeLost       = RuleOutcome("Lost")
	RuleOutcomeNeverFired = RuleOutcome("NeverFired")
)

func (o RuleOutcome) String() string { return string(o) }

func (o RuleOutcome) IsValid() bool {
	switch o {
	case RuleOutcomeWon, RuleOutcomeLost, RuleOutcomeNeverFired:
		return true
	default:
		return false
	}
}
