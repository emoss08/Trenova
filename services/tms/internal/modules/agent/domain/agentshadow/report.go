// Package agentshadow measures an agent in shadow against the people doing
// the same work: each write it recorded instead of offering is set beside
// what a person did to the same record afterwards.
package agentshadow

import "strings"

const (
	DefaultDays = 30
	MaxDays     = 90
	// ResponseWindowSeconds is how long after a recorded proposal a person's
	// change to the same record still counts as their answer to the same need.
	ResponseWindowSeconds int64 = 3 * 24 * 60 * 60
	// MaxProposals bounds how much history one report reads.
	MaxProposals = 2000
)

// Proposal is one write an agent recorded while in shadow.
type Proposal struct {
	TargetID  string
	CreatedAt int64
	// Fields are the top-level fields the write would have changed. Empty
	// when its tool could not say.
	Fields []string
	Failed bool
}

// PersonChange is a person's change to a record, from the audit trail.
type PersonChange struct {
	ResourceID string
	Timestamp  int64
	Fields     []string
}

// Report is what an agent's shadow period says about letting it go live.
type Report struct {
	Days int
	// Recorded is every write the agent recorded in the period.
	Recorded int
	// Matched is a recorded write a person then made too: they changed the
	// same record, and the same fields where the write named them.
	Matched int
	// WouldReject is a recorded write whose record a person changed some
	// other way: they handled it, and not as the agent would have.
	WouldReject int
	// WouldFail is a recorded write that failed when it was simulated.
	WouldFail int
	// Unanswered is a recorded write whose record nobody touched afterwards.
	Unanswered int
}

// MatchRate is the share of the writes people answered that matched what
// they did, between 0 and 1. Nil until people have answered one.
func (r *Report) MatchRate() *float64 {
	answered := r.Matched + r.WouldReject
	if answered == 0 {
		return nil
	}
	rate := float64(r.Matched) / float64(answered)
	return &rate
}

// ClampDays keeps a requested period within what a report reads.
func ClampDays(days int) int {
	switch {
	case days <= 0:
		return DefaultDays
	case days > MaxDays:
		return MaxDays
	default:
		return days
	}
}

// Build sets each recorded write beside the first change a person made to
// its record within the response window.
func Build(days int, proposals []Proposal, changes []PersonChange) *Report {
	byRecord := make(map[string][]PersonChange, len(changes))
	for _, change := range changes {
		byRecord[change.ResourceID] = append(byRecord[change.ResourceID], change)
	}

	report := &Report{Days: days, Recorded: len(proposals)}
	for idx := range proposals {
		proposal := &proposals[idx]
		if proposal.Failed {
			report.WouldFail++
			continue
		}
		answer, ok := firstAnswer(proposal, byRecord[proposal.TargetID])
		switch {
		case !ok:
			report.Unanswered++
		case sameFields(proposal.Fields, answer.Fields):
			report.Matched++
		default:
			report.WouldReject++
		}
	}

	return report
}

func firstAnswer(proposal *Proposal, changes []PersonChange) (PersonChange, bool) {
	if proposal.TargetID == "" {
		return PersonChange{}, false
	}
	var first PersonChange
	found := false
	for _, change := range changes {
		if change.Timestamp < proposal.CreatedAt ||
			change.Timestamp > proposal.CreatedAt+ResponseWindowSeconds {
			continue
		}
		if !found || change.Timestamp < first.Timestamp {
			first = change
			found = true
		}
	}
	return first, found
}

// sameFields reports whether a person's change covers every field the write
// named. A write that named none matches any change to its record.
func sameFields(proposed, changed []string) bool {
	if len(proposed) == 0 {
		return true
	}
	seen := make(map[string]struct{}, len(changed))
	for _, field := range changed {
		seen[TopLevel(field)] = struct{}{}
	}
	for _, field := range proposed {
		if _, ok := seen[TopLevel(field)]; !ok {
			return false
		}
	}
	return true
}

// TopLevel is the first segment of a field path: "moves" for
// "moves.stops[1].locationId", "bol" for "bol".
func TopLevel(path string) string {
	if idx := strings.IndexAny(path, ".["); idx >= 0 {
		return path[:idx]
	}
	return path
}
