package agentruntime

import (
	"strings"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func requiredSchema(required []string, fields ...string) map[string]any {
	properties := make(map[string]any, len(fields))
	for _, field := range fields {
		properties[field] = map[string]any{"type": "string"}
	}

	return map[string]any{
		"type":                 "object",
		"properties":           properties,
		"required":             required,
		"additionalProperties": false,
	}
}

/*
A record asked for by "id" reaches the tool under the name it takes.

The Dispatch desk called get_shipment with {"id": "shp_..."} and was told the
shipmentId it had in fact supplied was missing.
*/
func TestAliasedArguments_ReadsABareIDAsTheOneMissingID(t *testing.T) {
	t.Parallel()

	schema := requiredSchema([]string{"shipmentId"}, "shipmentId")
	sent := map[string]any{"id": "shp_1"}

	got := aliasedArguments(schema, sent)

	assert.Equal(t, map[string]any{"shipmentId": "shp_1"}, got)
	assert.Equal(t, map[string]any{"id": "shp_1"}, sent, "the model's own call is not rewritten")
}

func TestAliasedArguments_ReadsTheSameNameSpelledAnotherWay(t *testing.T) {
	t.Parallel()

	schema := requiredSchema([]string{"shipmentId"}, "shipmentId", "includeStops")

	assert.Equal(t, map[string]any{"shipmentId": "shp_1"},
		aliasedArguments(schema, map[string]any{"shipment_id": "shp_1"}))
	assert.Equal(t, map[string]any{"shipmentId": "shp_1"},
		aliasedArguments(schema, map[string]any{"ShipmentID": "shp_1"}))
}

// Anything less than plain is left for the tool to refuse, with the name it
// wanted: a bare id when two ids are missing could mean either.
func TestAliasedArguments_LeavesAnAmbiguousIDAlone(t *testing.T) {
	t.Parallel()

	schema := requiredSchema([]string{"moveId", "workerId"}, "moveId", "workerId")
	sent := map[string]any{"id": "mv_1"}

	assert.Equal(t, sent, aliasedArguments(schema, sent))
}

// Arguments already complete are passed through untouched.
func TestAliasedArguments_LeavesACompleteCallAlone(t *testing.T) {
	t.Parallel()

	schema := requiredSchema([]string{"shipmentId"}, "shipmentId")
	sent := map[string]any{"shipmentId": "shp_1", "id": "shp_2"}

	assert.Equal(t, sent, aliasedArguments(schema, sent))
}

/*
A record reaches the model without what it carries for the database.

The get_shipment the Dispatch desk read repeated the tenant, a version and two
audit stamps on every commodity, stop, location and charge. The record's own
version stays, because a write that checks it reads it from there.
*/
func TestEncodeToolResult_TrimsWhatARecordCarriesForTheDatabase(t *testing.T) {
	t.Parallel()

	record := map[string]any{
		"id":             "shp_1",
		"organizationId": "org_1",
		"businessUnitId": "bu_1",
		"version":        3,
		"moves": []any{map[string]any{
			"id":             "sm_1",
			"organizationId": "org_1",
			"version":        0,
			"createdAt":      transcriptNow,
			"stops": []any{map[string]any{
				"id":             "stp_1",
				"businessUnitId": "bu_1",
				"updatedAt":      transcriptNow,
				"type":           "Pickup",
			}},
		}},
	}

	encoded, err := encodeToolResult(record, transcriptNow, "UTC")
	require.NoError(t, err)

	assert.NotContains(t, encoded, "organizationId")
	assert.NotContains(t, encoded, "businessUnitId")
	assert.NotContains(t, encoded, "createdAt")
	assert.NotContains(t, encoded, "updatedAt")
	assert.Equal(t, 1, strings.Count(encoded, `"version"`), "only the record's own version is kept")
	assert.Contains(t, encoded, `"type":"Pickup"`)
}

/*
An appointment window is read with its hour.

The window arrived as a raw epoch, and the model told the dispatcher the stop
had no window set. A window, an arrival and a departure keep the time of day;
a plain date field stays a date.
*/
func TestEncodeToolResult_RendersAnAppointmentWindowWithItsHour(t *testing.T) {
	t.Parallel()

	start := transcriptNow + 2*86400 + 14*3600
	encoded, err := encodeToolResult(map[string]any{
		"scheduledWindowStart": start,
		"actualArrival":        start,
		"deliveryDate":         start,
	}, transcriptNow, "UTC")
	require.NoError(t, err)

	var document map[string]string
	require.NoError(t, sonic.UnmarshalString(encoded, &document))
	assert.Equal(t, "2026-09-21 14:00 UTC (in 2 days)", document["scheduledWindowStart"])
	assert.Equal(t, "2026-09-21 14:00 UTC (in 2 days)", document["actualArrival"])
	assert.Equal(t, "2026-09-21 (in 2 days)", document["deliveryDate"])
}

/*
No two calls in a conversation share an id.

A provider that sends none gets call_0 synthesized in every completion, so the
same id named a different call on every turn: the artifacts keyed on it, the
transcript pairing results with calls, and the provider reading the history
all took one call for another. A provider's own unique ids are kept.
*/
func TestDistinctCallIDs_ReplacesAnIDTheConversationAlreadyUsed(t *testing.T) {
	t.Parallel()

	used := usedCallIDs([]conversation.Message{{
		Role:      conversation.RoleAssistant,
		ToolCalls: []conversation.ToolCallRecord{{ID: "call_0", Name: "search_shipments"}},
	}})
	completion := &serviceports.ChatCompletionResult{ToolCalls: []serviceports.ToolCall{
		{ID: "call_0", Name: "get_shipment"},
		{ID: "", Name: "get_customer"},
		{ID: "toolu_unique", Name: "get_worker"},
		{ID: "toolu_unique", Name: "get_tractor"},
	}}

	distinctCallIDs(completion, used, callIDMinter{
		mint:             NewCallID,
		renewSynthesized: func() bool { return true },
	})

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
the artifact that older call had produced. An execution that started before
the change keeps the ids it was given, so its history still replays.
*/
func TestDistinctCallIDs_AlwaysReplacesASynthesizedID(t *testing.T) {
	t.Parallel()

	completion := func() *serviceports.ChatCompletionResult {
		return &serviceports.ChatCompletionResult{ToolCalls: []serviceports.ToolCall{
			{ID: "ollama_call_0_get_shipment", SynthesizedID: true, Name: "get_shipment"},
			{ID: "ollama_call_1_get_customer", SynthesizedID: true, Name: "get_customer"},
			{ID: "toolu_unique", Name: "get_worker"},
		}}
	}

	asked := 0
	minted := 0
	current := completion()
	distinctCallIDs(current, map[string]struct{}{}, callIDMinter{
		mint: func() string {
			minted++
			return NewCallID()
		},
		renewSynthesized: func() bool {
			asked++
			return true
		},
	})

	assert.Equal(t, 1, asked, "the change is asked about once per completion")
	assert.Equal(t, 2, minted)
	assert.NotEqual(t, "ollama_call_0_get_shipment", current.ToolCalls[0].ID)
	assert.NotEqual(t, "ollama_call_1_get_customer", current.ToolCalls[1].ID)
	assert.False(t, current.ToolCalls[0].SynthesizedID, "a minted id is the loop's own")
	assert.Equal(t, "toolu_unique", current.ToolCalls[2].ID, "a provider's own id is kept")

	before := completion()
	distinctCallIDs(before, map[string]struct{}{}, callIDMinter{
		mint: func() string {
			t.Fatal("an execution from before the change mints nothing new here")
			return ""
		},
		renewSynthesized: func() bool { return false },
	})
	assert.Equal(t, "ollama_call_0_get_shipment", before.ToolCalls[0].ID)

	plain := &serviceports.ChatCompletionResult{ToolCalls: []serviceports.ToolCall{
		{ID: "toolu_1", Name: "get_worker"},
	}}
	distinctCallIDs(plain, map[string]struct{}{}, callIDMinter{
		mint: NewCallID,
		renewSynthesized: func() bool {
			t.Fatal("a completion with no synthesized id records no change marker")
			return false
		},
	})
	assert.Equal(t, "toolu_1", plain.ToolCalls[0].ID)
}

/*
Two turns of a provider that gives no ids both start at call 0. Each call is
still its own in the transcript, and in what its artifacts are keyed on.
*/
func TestRun_GivesSynthesizedCallIDsOfEachTurnTheirOwnID(t *testing.T) {
	t.Parallel()

	synthesized := func() *serviceports.ChatCompletionResult {
		return &serviceports.ChatCompletionResult{
			ToolCalls: []serviceports.ToolCall{{
				ID:            "ollama_call_0_search_worker",
				SynthesizedID: true,
				Name:          "search_worker",
				Arguments:     map[string]any{"query": "Maria"},
			}},
			ModelIdentifier: "test-model",
		}
	}
	tool := queryTool("search_worker", map[string]any{"results": []any{}}, nil)
	completion := &scriptedCompletion{Turns: []*serviceports.ChatCompletionResult{
		synthesized(),
		synthesized(),
		textTurn("Nobody by that name."),
	}}
	rt := newRuntime(completion, &stubQueryRegistry{Tools: []serviceports.AgentQueryTool{tool}},
		&stubActionRegistry{}, nil)

	var observed []string
	result, err := rt.Run(t.Context(), &serviceports.RunRequest{
		Definition: testDefinition("search_worker"),
		Actor:      testActor(),
		Input:      "Who is Maria?",
		ToolObserver: func(observation serviceports.ToolObservation) (*serviceports.ShownArtifact, error) {
			observed = append(observed, observation.Call.ID)
			return nil, nil
		},
	})
	require.NoError(t, err)

	require.Len(t, observed, 2)
	assert.NotEqual(t, observed[0], observed[1], "each call is keyed on an id of its own")
	for _, id := range observed {
		assert.NotContains(t, id, "ollama_call_", "a made-up id never reaches what keys on it")
	}
	pairs := map[string]int{}
	for _, message := range result.Messages {
		if message.ToolCallID != "" {
			pairs[message.ToolCallID]++
		}
	}
	assert.Len(t, pairs, 2, "each result pairs with its own call")
}
