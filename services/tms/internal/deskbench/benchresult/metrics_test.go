package benchresult

import (
	"testing"

	"github.com/emoss08/trenova/internal/deskbench/benchrecord"
	"github.com/emoss08/trenova/internal/deskbench/benchscenario"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScoreSelection(t *testing.T) {
	t.Parallel()

	scenario := &benchscenario.Scenario{
		ExpectedTools:  []string{"shipment_get|shipment_list", "shipment_add_comment"},
		ForbiddenTools: []string{"shipment_cancel"},
	}
	turns := []*benchrecord.TurnRecord{toolTurnRecord(
		"find_tools", "customer_list", "shipment_list", "shipment_cancel", "agent_list_memories",
	)}

	selection := scoreSelection(scenario, turns)
	require.NotNil(t, selection)
	assert.InDelta(t, 0.5, selection.Recall, 1e-9)
	require.NotNil(t, selection.Precision)
	assert.InDelta(t, 1.0/3, *selection.Precision, 1e-9, "plumbing tools are not scored")
	assert.Equal(t, 2, selection.FirstCorrect)
	assert.Equal(t, 1, selection.ForbiddenCalls)
	assert.Equal(t, 3, selection.ScoredCalls)

	assert.Nil(t, scoreSelection(&benchscenario.Scenario{}, turns), "no labels, no score")
}

func TestScoreRetrieval_AgainstTheToolsThePassingTrialUsed(t *testing.T) {
	t.Parallel()

	turn := toolTurnRecord(
		"find_tools",
		"shipment_get",
		"equipment_list_trailers",
		"shipment_add_comment",
		"customer_get",
	)
	turn.ModelCalls = []*benchrecord.ModelCall{
		{Chat: &llmports.ChatCompletionRequest{Tools: []kernelports.ToolSpec{
			{Name: "shipment_get"}, {Name: "shipment_add_comment"}, {Name: "customer_get"},
			{Name: "find_tools"},
		}}},
	}
	turn.Disclosures = []benchrecord.Disclosure{
		{
			Kind:   benchrecord.DisclosurePreselected,
			Ranked: []string{"shipment_get", "shipment_add_comment"},
		},
		{
			Kind:   benchrecord.DisclosureFound,
			Need:   "trailers",
			Ranked: []string{"equipment_list_trailers"},
		},
		{Kind: benchrecord.DisclosureFound, Need: "drivers", Ranked: []string{"worker_list"}},
	}

	retrieval := scoreRetrieval([]*benchrecord.TurnRecord{turn})
	require.NotNil(t, retrieval)
	require.NotNil(t, retrieval.PreselectionRecall)
	assert.InDelta(t, 3.0/4, *retrieval.PreselectionRecall, 1e-9,
		"a tool carried over or loaded beside the ranked ones was in the opening set")
	require.NotNil(t, retrieval.FindRecall)
	assert.InDelta(t, 0.5, *retrieval.FindRecall, 1e-9, "each answer is scored on its own")
	require.NotNil(t, retrieval.FindUnionRecall)
	assert.InDelta(t, 1.0, *retrieval.FindUnionRecall, 1e-9)
	assert.Equal(t, 2, retrieval.FindAnswers)
	assert.Equal(t, 1, retrieval.FindHits)
}

func toolTurnRecord(names ...string) *benchrecord.TurnRecord {
	turn := &benchrecord.TurnRecord{}
	for _, name := range names {
		turn.Tools = append(
			turn.Tools,
			&benchrecord.ToolCallRecord{Name: name, Verdict: benchrecord.VerdictRan},
		)
	}

	return turn
}
