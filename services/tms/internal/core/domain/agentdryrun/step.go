// Package agentdryrun names what each tool call of a dry run would have come
// to had the draft agent been saved and asked the same thing for real.
package agentdryrun

import (
	"slices"

	"github.com/emoss08/trenova/internal/core/domain/agent"
)

// Outcome is what one call would have done.
type Outcome string

const (
	// OutcomeRuns is a read that ran against live data.
	OutcomeRuns = Outcome("Runs")
	// OutcomeAskFirst is a write that would wait for a person's approval.
	OutcomeAskFirst = Outcome("AskFirst")
	// OutcomePropose is a write that would be offered as a proposal.
	OutcomePropose = Outcome("Propose")
	// OutcomeRecorded is a write a shadow agent would record, not offer,
	// whatever its tier.
	OutcomeRecorded = Outcome("Recorded")
	// OutcomeSimulated is a write that would run on its own, previewed here.
	OutcomeSimulated = Outcome("Simulated")
	// OutcomeNotHeld is a tool the model reached for that the draft does not
	// hold.
	OutcomeNotHeld = Outcome("NotHeld")
	// OutcomeFailed is a call that did not go through: bad arguments, a
	// permission the person lacks, a spent budget.
	OutcomeFailed = Outcome("Failed")
)

// Verdicts the runtime gives a call, as its trace records them.
const (
	verdictDenied  = "denied"
	verdictInvalid = "invalid"
)

// Call is one tool call of the dry run as the runtime reported it.
type Call struct {
	CallID  string
	Tool    string
	Verdict string
	Failed  bool
	Summary string
}

// Action is the write a call became, when it was one.
type Action struct {
	Tier      agent.AutonomyTier
	Simulated bool
}

// Step is one call with what it would have come to.
type Step struct {
	CallID  string  `json:"callId"`
	Tool    string  `json:"tool"`
	Outcome Outcome `json:"outcome"`
	Summary string  `json:"summary,omitempty"`
}

// Draft is what of the agent decides an outcome.
type Draft struct {
	Held   []string
	Shadow bool
}

// Classify names each call's outcome, in the order the calls were made.
func Classify(draft Draft, calls []Call, actions map[string]Action) []Step {
	steps := make([]Step, 0, len(calls))
	for idx := range calls {
		call := &calls[idx]
		action, isWrite := actions[call.CallID]
		steps = append(steps, Step{
			CallID:  call.CallID,
			Tool:    call.Tool,
			Outcome: outcomeOf(draft, call, action, isWrite),
			Summary: call.Summary,
		})
	}
	return steps
}

func outcomeOf(draft Draft, call *Call, action Action, isWrite bool) Outcome {
	switch {
	case isWrite && draft.Shadow:
		return OutcomeRecorded
	case isWrite && action.Simulated:
		return OutcomeSimulated
	case isWrite && action.Tier == agent.TierActWithApproval:
		return OutcomeAskFirst
	case isWrite:
		return OutcomePropose
	case (call.Verdict == verdictDenied || call.Verdict == verdictInvalid) &&
		!slices.Contains(draft.Held, call.Tool):
		return OutcomeNotHeld
	case call.Failed:
		return OutcomeFailed
	default:
		return OutcomeRuns
	}
}
