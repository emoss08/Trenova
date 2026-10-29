package agentruntime

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/runtimetest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime/turnhistory"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
No two calls in a conversation share an id.

A provider that sends none gets call_0 synthesized in every completion, so the
same id named a different call on every turn: the artifacts keyed on it, the
transcript pairing results with calls, and the provider reading the history
all took one call for another. A provider's own unique ids are kept.
*/
func TestDistinctCallIDs_ReplacesAnIDTheConversationAlreadyUsed(t *testing.T) {
	t.Parallel()

	used := turnhistory.UsedCallIDs([]conversation.Message{{
		Role:      conversation.RoleAssistant,
		ToolCalls: []conversation.ToolCallRecord{{ID: "call_0", Name: "shipment_list"}},
	}})
	completion := &llmports.ChatCompletionResult{ToolCalls: []llmports.ToolCall{
		{ID: "call_0", Name: "shipment_get"},
		{ID: "", Name: "customer_get"},
		{ID: "toolu_unique", Name: "worker_get"},
		{ID: "toolu_unique", Name: "equipment_get_tractor"},
	}}

	turnhistory.DistinctCallIDs(completion, used, NewCallID)

	ids := make(map[string]struct{}, len(completion.ToolCalls))
	for _, call := range completion.ToolCalls {
		require.NotEmpty(t, call.ID)
		assert.NotEqual(t, "call_0", call.ID)
		ids[call.ID] = struct{}{}
	}
	assert.Len(t, ids, 4, "every call has its own id")
	assert.Equal(t, "toolu_unique", completion.ToolCalls[2].ID, "a provider's own id is kept")
}

/*
An id the adapter made up is replaced even when the conversation does not hold
it.

The conversation a turn replays is only its most recent messages, so a call_0
from before them was not among the ids checked, and the next call_0 overwrote
the artifact that older call had produced.
*/
func TestDistinctCallIDs_AlwaysReplacesASynthesizedID(t *testing.T) {
	t.Parallel()

	minted := 0
	current := &llmports.ChatCompletionResult{ToolCalls: []llmports.ToolCall{
		{ID: "call_0", SynthesizedID: true, Name: "shipment_get"},
		{ID: "call_1", SynthesizedID: true, Name: "customer_get"},
		{ID: "toolu_unique", Name: "worker_get"},
	}}
	turnhistory.DistinctCallIDs(current, map[string]struct{}{}, func() string {
		minted++
		return NewCallID()
	})

	assert.Equal(t, 2, minted)
	assert.NotEqual(t, "call_0", current.ToolCalls[0].ID)
	assert.NotEqual(t, "call_1", current.ToolCalls[1].ID)
	assert.False(t, current.ToolCalls[0].SynthesizedID, "a minted id is the loop's own")
	assert.Equal(t, "toolu_unique", current.ToolCalls[2].ID, "a provider's own id is kept")

	plain := &llmports.ChatCompletionResult{ToolCalls: []llmports.ToolCall{
		{ID: "toolu_1", Name: "worker_get"},
	}}
	turnhistory.DistinctCallIDs(plain, map[string]struct{}{}, func() string {
		t.Fatal("a provider's own unique id is never replaced")
		return ""
	})
	assert.Equal(t, "toolu_1", plain.ToolCalls[0].ID)
}

/*
Two turns of a provider that gives no ids both start at call 0. Each call is
still its own in the transcript, and in what its artifacts are keyed on.
*/
func TestRun_GivesSynthesizedCallIDsOfEachTurnTheirOwnID(t *testing.T) {
	t.Parallel()

	synthesized := func() *llmports.ChatCompletionResult {
		return &llmports.ChatCompletionResult{
			ToolCalls: []llmports.ToolCall{{
				ID:            "call_0",
				SynthesizedID: true,
				Name:          "worker_list",
				Arguments:     map[string]any{"query": "Maria"},
			}},
			ModelIdentifier: "test-model",
		}
	}
	tool := queryTool("worker_list", map[string]any{"results": []any{}}, nil)
	completion := &scriptedCompletion{Turns: []*llmports.ChatCompletionResult{
		synthesized(),
		synthesized(),
		runtimetest.TextTurn("Nobody by that name."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{Tools: []agenttoolports.AgentQueryTool{tool}},
		&stubActionRegistry{}, nil)

	var observed []string
	result, err := rt.Run(t.Context(), &agentruntimeports.RunRequest{
		Definition: runtimetest.Definition("worker_list"),
		Actor:      runtimetest.Actor(),
		Input:      "Who is Maria?",
		ToolObserver: func(observation agentruntimeports.ToolObservation) (*agentruntimeports.ShownArtifact, error) {
			observed = append(observed, observation.Call.ID)
			return nil, nil
		},
	})
	require.NoError(t, err)

	require.Len(t, observed, 2)
	assert.NotEqual(t, observed[0], observed[1], "each call is keyed on an id of its own")
	for _, id := range observed {
		assert.NotEqual(t, "call_0", id, "a made-up id never reaches what keys on it")
	}
	pairs := map[string]int{}
	for _, message := range result.Messages {
		if message.ToolCallID != "" {
			pairs[message.ToolCallID]++
		}
	}
	assert.Len(t, pairs, 2, "each result pairs with its own call")
}
