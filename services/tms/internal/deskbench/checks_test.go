package deskbench

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
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
		Name: agentports.AssistantEventReplyRegrounded,
		Data: map[string]any{"action": "relink_artifacts"},
	}}}})
	assert.True(t, clean.Passed, "an artifact link correction has a check of its own")

	corrected := groundingCheck([]*TurnRecord{{Events: []Event{{
		Name: agentports.AssistantEventReplyRegrounded,
		Data: map[string]any{"action": "rewrite", "figures": []any{"52.3"}},
	}}}})
	assert.False(t, corrected.Passed)
	assert.Equal(t, "rewrite: 52.3", corrected.Detail)
}

func TestArtifactCorrections_CountOnlyWhatRestartedTheReply(t *testing.T) {
	t.Parallel()

	placed := Event{
		Name: agentports.AssistantEventReplyRegrounded,
		Data: map[string]any{"action": string(agentports.RegroundPlaceArtifacts)},
	}
	relinked := Event{
		Name: agentports.AssistantEventReplyRegrounded,
		Data: map[string]any{"action": string(agentports.RegroundRelinkArtifacts)},
	}

	assert.Empty(t, artifactCorrections([]Event{placed}),
		"a left-out table named at the end restarted nothing")
	assert.Equal(t, []string{"relink_artifacts", "place_artifacts"},
		artifactCorrections([]Event{relinked, placed}))
}
