package benchcheck

import (
	"testing"

	"github.com/emoss08/trenova/internal/deskbench"
	"github.com/emoss08/trenova/internal/deskbench/benchrecord"
	"github.com/emoss08/trenova/internal/deskbench/benchresult"
	"github.com/emoss08/trenova/internal/deskbench/benchscenario"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantstreamports"
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

func TestGroundingCheck_FailsAReplyTheGuardCorrected(t *testing.T) {
	t.Parallel()

	clean := groundingCheck([]*benchrecord.TurnRecord{{Events: []benchrecord.Event{{
		Name: assistantstreamports.AssistantEventReplyRegrounded,
		Data: map[string]any{"action": "relink_artifacts"},
	}}}})
	assert.True(t, clean.Passed, "an artifact link correction has a check of its own")

	corrected := groundingCheck([]*benchrecord.TurnRecord{{Events: []benchrecord.Event{{
		Name: assistantstreamports.AssistantEventReplyRegrounded,
		Data: map[string]any{"action": "rewrite", "figures": []any{"52.3"}},
	}}}})
	assert.False(t, corrected.Passed)
	assert.Equal(t, "rewrite: 52.3", corrected.Detail)
}

func TestArtifactCorrections_CountOnlyWhatRestartedTheReply(t *testing.T) {
	t.Parallel()

	placed := benchrecord.Event{
		Name: assistantstreamports.AssistantEventReplyRegrounded,
		Data: map[string]any{"action": string(assistantstreamports.RegroundPlaceArtifacts)},
	}
	relinked := benchrecord.Event{
		Name: assistantstreamports.AssistantEventReplyRegrounded,
		Data: map[string]any{"action": string(assistantstreamports.RegroundRelinkArtifacts)},
	}

	assert.Empty(t, artifactCorrections([]benchrecord.Event{placed}),
		"a left-out table named at the end restarted nothing")
	assert.Equal(t, []string{"relink_artifacts", "place_artifacts"},
		artifactCorrections([]benchrecord.Event{relinked, placed}))
}

func TestNoDelegateCalls_GradesOnlyWhatADelegateCalled(t *testing.T) {
	t.Parallel()

	expect := &benchscenario.Expect{NoDelegateCalls: []string{"shipment_place_hold"}}
	byName := func(tools ...*benchrecord.ToolCallRecord) benchresult.CheckResult {
		evidence := &Evidence{Turns: []*benchrecord.TurnRecord{{Tools: tools}}}
		for _, check := range (&Checker{}).Evaluate(t.Context(), &deskbench.Session{}, expect, evidence) {
			if check.Name == "no delegate called shipment_place_hold" {
				return check
			}
		}
		t.Fatal("the check was not run")

		return benchresult.CheckResult{}
	}

	asked := byName(
		&benchrecord.ToolCallRecord{
			Name:     "shipment_place_hold",
			Verdict:  benchrecord.VerdictProposed,
			Finished: true,
		},
		&benchrecord.ToolCallRecord{Name: "shipment_get", DelegateCallID: "call_1", Finished: true},
	)
	assert.True(t, asked.Passed, "the conversation's own proposal is not the delegate's")

	delegated := byName(&benchrecord.ToolCallRecord{
		Name: "shipment_place_hold", DelegateCallID: "call_1", Verdict: "unregistered",
		Finished: true,
	})
	assert.False(t, delegated.Passed)
	assert.Equal(t, "shipment_place_hold (unregistered)", delegated.Detail)
}

func TestStepEvidence_CountsOnlyWhatWaitedForAPersonAsAProposal(t *testing.T) {
	t.Parallel()

	evidence := &Evidence{
		Turns: []*benchrecord.TurnRecord{{Proposals: []*benchrecord.ProposalRecord{
			{
				Tool:   "agent_raise_exception",
				Tier:   agent.TierAutoExecute,
				Status: agent.ProposalStatusExecuted,
			},
			{
				Tool:   "shipment_place_hold",
				Tier:   agent.TierPropose,
				Status: agent.ProposalStatusPending,
			},
			{Tool: "billing_hold_queue_item", Tier: agent.TierActWithApproval},
		}}},
	}

	proposals := evidence.proposals()
	names := make([]string, 0, len(proposals))
	for _, proposal := range proposals {
		names = append(names, proposal.Tool)
	}
	assert.Equal(t, []string{"shipment_place_hold", "billing_hold_queue_item"}, names)
}
