package agentdryrun

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/stretchr/testify/assert"
)

func TestClassifyNamesEveryOutcome(t *testing.T) {
	t.Parallel()

	calls := []Call{
		{CallID: "1", Tool: "get_invoice", Verdict: "ran"},
		{CallID: "2", Tool: "update_invoice", Verdict: "proposed"},
		{CallID: "3", Tool: "hold_invoice", Verdict: "proposed"},
		{CallID: "4", Tool: "note_invoice", Verdict: "simulated"},
		{CallID: "5", Tool: "cancel_shipment", Verdict: "denied", Failed: true},
		{CallID: "6", Tool: "get_shipment", Verdict: "denied", Failed: true},
		{CallID: "7", Tool: "get_invoice", Verdict: "invalid", Failed: true},
	}
	actions := map[string]Action{
		"2": {Tier: agent.TierActWithApproval},
		"3": {Tier: agent.TierPropose},
		"4": {Tier: agent.TierAutoExecute, Simulated: true},
	}
	draft := Draft{Held: []string{"get_invoice", "update_invoice", "hold_invoice", "note_invoice", "get_shipment"}}

	steps := Classify(draft, calls, actions)

	outcomes := make([]Outcome, 0, len(steps))
	for _, step := range steps {
		outcomes = append(outcomes, step.Outcome)
	}
	assert.Equal(t, []Outcome{
		OutcomeRuns, OutcomeAskFirst, OutcomePropose, OutcomeSimulated,
		OutcomeNotHeld, OutcomeFailed, OutcomeFailed,
	}, outcomes)
}

func TestClassifyRecordsAShadowAgentsWrites(t *testing.T) {
	t.Parallel()

	steps := Classify(
		Draft{Shadow: true},
		[]Call{{CallID: "1", Tool: "note_invoice"}, {CallID: "2", Tool: "update_invoice"}},
		map[string]Action{
			"1": {Tier: agent.TierAutoExecute, Simulated: true},
			"2": {Tier: agent.TierPropose},
		},
	)

	assert.Equal(t, OutcomeRecorded, steps[0].Outcome)
	assert.Equal(t, OutcomeRecorded, steps[1].Outcome)
}
