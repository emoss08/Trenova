package deskbench

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/stretchr/testify/assert"
)

func TestReplyStates_ReadsASmallCountInWords(t *testing.T) {
	t.Parallel()

	tests := []struct {
		reply string
		value string
		want  bool
	}{
		{reply: "saved with four document profiles", value: "4", want: true},
		{reply: "saved with 4 document profiles", value: "4", want: true},
		{reply: "Four profiles were saved", value: "4", want: true},
		{reply: "saved with fourteen profiles", value: "4", want: false},
		{reply: "nothing was saved for anyone", value: "1", want: false},
		{reply: "saved forty profiles", value: "40", want: false},
		{reply: "saved 2.5 profiles", value: "2.5", want: true},
		{reply: "No delivered shipments are waiting", value: "0", want: true},
		{reply: "none are waiting", value: "0", want: true},
		{reply: "0 shipments are waiting", value: "0", want: true},
		{reply: "nineteen shipments are waiting", value: "0", want: false},
		{reply: "know of three loads", value: "0", want: false},
		{reply: "there are no invoices past due", value: "0", want: true},
		{reply: "No loads are ready", value: "0", want: true},
		{reply: "nothing is ready to bill", value: "0", want: true},
		{reply: "zero loads are late", value: "0", want: true},
		{reply: "no problem, there are 3", value: "0", want: false},
		{reply: "No worries: three loads are late", value: "0", want: false},
		{reply: "no, three are late", value: "0", want: false},
		{reply: "the answer is no.", value: "0", want: false},
		{reply: "they are no longer late, three loads moved", value: "0", want: false},
		{reply: "know of 3 loads", value: "0", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.reply, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, replyStates(tt.reply, tt.value))
		})
	}
}

func TestRunUnits_RunsAnExclusiveGroupInOrderForEachProviderAndRepeat(t *testing.T) {
	t.Parallel()

	guide := &Scenario{Name: "edi-guide-new-partner", Exclusive: "acme-retail-edi"}
	revised := &Scenario{Name: "edi-guide-revised", Exclusive: "Acme-Retail-EDI "}
	alone := &Scenario{Name: "edi-pasted-x12"}
	planned := make([]plannedCase, 0, 12)
	for scenarioOrder, scenario := range []*Scenario{guide, revised, alone} {
		for providerOrder, provider := range []string{"Anthropic", "OpenAI"} {
			for rep := 1; rep <= 2; rep++ {
				planned = append(planned, plannedCase{
					scenario: scenario, provider: provider, repeat: rep,
					scenarioOrder: scenarioOrder, providerOrder: providerOrder,
				})
			}
		}
	}

	units := runUnits(planned)

	assert.Len(t, units, 5, "one unit for the group, one for each case of the other scenario")
	labels := make([]string, 0, len(units[0]))
	for _, idx := range units[0] {
		labels = append(labels, planned[idx].label())
	}
	assert.Equal(t, []string{
		"edi-guide-new-partner @ Anthropic #1", "edi-guide-revised @ Anthropic #1",
		"edi-guide-new-partner @ Anthropic #2", "edi-guide-revised @ Anthropic #2",
		"edi-guide-new-partner @ OpenAI #1", "edi-guide-revised @ OpenAI #1",
		"edi-guide-new-partner @ OpenAI #2", "edi-guide-revised @ OpenAI #2",
	}, labels, "each revised case follows the new-partner case that saved the partner")
	for _, unit := range units[1:] {
		assert.Len(t, unit, 1)
	}
}

func TestGroundingCheck_FailsAReplyTheGuardCorrected(t *testing.T) {
	t.Parallel()

	clean := groundingCheck([]*TurnRecord{{Events: []Event{{
		Name: assistantports.AssistantEventReplyRegrounded,
		Data: map[string]any{"action": "relink_artifacts"},
	}}}})
	assert.True(t, clean.Passed, "an artifact link correction has a check of its own")

	corrected := groundingCheck([]*TurnRecord{{Events: []Event{{
		Name: assistantports.AssistantEventReplyRegrounded,
		Data: map[string]any{"action": "rewrite", "figures": []any{"52.3"}},
	}}}})
	assert.False(t, corrected.Passed)
	assert.Equal(t, "rewrite: 52.3", corrected.Detail)
}

func TestArtifactCorrections_CountOnlyWhatRestartedTheReply(t *testing.T) {
	t.Parallel()

	placed := Event{
		Name: assistantports.AssistantEventReplyRegrounded,
		Data: map[string]any{"action": string(assistantports.RegroundPlaceArtifacts)},
	}
	relinked := Event{
		Name: assistantports.AssistantEventReplyRegrounded,
		Data: map[string]any{"action": string(assistantports.RegroundRelinkArtifacts)},
	}

	assert.Empty(t, artifactCorrections([]Event{placed}),
		"a left-out table named at the end restarted nothing")
	assert.Equal(t, []string{"relink_artifacts", "place_artifacts"},
		artifactCorrections([]Event{relinked, placed}))
}

func TestNoDelegateCalls_GradesOnlyWhatADelegateCalled(t *testing.T) {
	t.Parallel()

	expect := &Expect{NoDelegateCalls: []string{"shipment_place_hold"}}
	byName := func(tools ...*ToolCallRecord) CheckResult {
		evidence := &stepEvidence{turns: []*TurnRecord{{Tools: tools}}}
		for _, check := range (&Bench{}).evaluate(t.Context(), &Session{}, expect, evidence) {
			if check.Name == "no delegate called shipment_place_hold" {
				return check
			}
		}
		t.Fatal("the check was not run")

		return CheckResult{}
	}

	asked := byName(
		&ToolCallRecord{Name: "shipment_place_hold", Verdict: verdictProposed, Finished: true},
		&ToolCallRecord{Name: "shipment_get", DelegateCallID: "call_1", Finished: true},
	)
	assert.True(t, asked.Passed, "the conversation's own proposal is not the delegate's")

	delegated := byName(&ToolCallRecord{
		Name: "shipment_place_hold", DelegateCallID: "call_1", Verdict: "unregistered",
		Finished: true,
	})
	assert.False(t, delegated.Passed)
	assert.Equal(t, "shipment_place_hold (unregistered)", delegated.Detail)
}

func TestStepEvidence_CountsOnlyWhatWaitedForAPersonAsAProposal(t *testing.T) {
	t.Parallel()

	evidence := &stepEvidence{turns: []*TurnRecord{{Proposals: []*ProposalRecord{
		{
			Tool:   "agent_raise_exception",
			Tier:   agent.TierAutoExecute,
			Status: agent.ProposalStatusExecuted,
		},
		{Tool: "shipment_place_hold", Tier: agent.TierPropose, Status: agent.ProposalStatusPending},
		{Tool: "billing_hold_queue_item", Tier: agent.TierActWithApproval},
	}}}}

	proposals := evidence.proposals()
	names := make([]string, 0, len(proposals))
	for _, proposal := range proposals {
		names = append(names, proposal.Tool)
	}
	assert.Equal(t, []string{"shipment_place_hold", "billing_hold_queue_item"}, names)
}
