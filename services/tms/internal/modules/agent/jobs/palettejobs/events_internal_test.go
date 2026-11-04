package palettejobs

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpagecontext"
	"github.com/emoss08/trenova/internal/modules/agent/domain/palettequestion"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func callFor(id, name string, arguments map[string]any) llmports.ToolCall {
	return llmports.ToolCall{ID: id, Name: name, Arguments: arguments}
}

// A refused read is kept with why it was refused and the read it went to, so
// "an agent read needs the tool call's read scope" can be found afterwards
// rather than only in the person's stream.
func TestToolCall_KeepsARefusedReadWithItsReasonAndRead(t *testing.T) {
	t.Parallel()

	started := time.Now().Add(-250 * time.Millisecond).UnixMilli()
	events := &streamEvents{model: "claude-haiku-5-5", started: map[string]int64{"c1": started}}

	kept := events.toolCall(
		callFor("c1", "get_record", map[string]any{"id": "cus_1"}),
		llmports.NovaToolResult{
			Content: "customer_get failed: an agent read needs the tool call's read scope",
			Failed:  true,
			Read:    "customer_get",
			Params:  map[string]any{"customerId": "cus_1"},
		},
	)

	assert.Equal(t, "c1", kept.CallID)
	assert.Equal(t, 1, kept.Sequence)
	assert.Equal(t, "claude-haiku-5-5", kept.Model)
	assert.Equal(t, "get_record", kept.Tool)
	assert.Equal(t, map[string]any{"id": "cus_1"}, kept.Arguments)
	assert.Equal(t, "customer_get", kept.Read)
	assert.Equal(t, map[string]any{"customerId": "cus_1"}, kept.Params)
	assert.Equal(t, palettequestion.ToolCallFailed, kept.Outcome)
	assert.Equal(
		t,
		"customer_get failed: an agent read needs the tool call's read scope",
		kept.Reason,
	)
	assert.Equal(t, started, kept.StartedAtMs)
	assert.GreaterOrEqual(t, kept.DurationMs, int64(250))
}

func TestToolCall_TellsARefusalFromAFailure(t *testing.T) {
	t.Parallel()

	events := &streamEvents{}

	refused := events.toolCall(callFor("c1", "list_records", map[string]any{"kind": "invoice"}),
		llmports.NovaToolResult{
			Content: "The person asking cannot read invoice records.",
			Failed:  true,
			Refused: true,
			Read:    "invoice_list",
		})

	assert.Equal(t, palettequestion.ToolCallRefused, refused.Outcome)
	assert.Equal(t, "The person asking cannot read invoice records.", refused.Reason)
}

// A read that ran keeps what it returned only in a few words and how many
// records it named, never the result itself.
func TestToolCall_KeepsARunReadsSummaryNotItsResult(t *testing.T) {
	t.Parallel()

	events := &streamEvents{}

	kept := events.toolCall(callFor("c1", "search_records", map[string]any{"query": "Gordon"}),
		llmports.NovaToolResult{
			Content: "Result from search_records:\n[{\"id\":\"cus_1\"}]",
			Summary: "2 records",
			Records: []agentpagecontext.EntityRef{
				{Type: "customer", ID: "cus_1"},
				{Type: "customer", ID: "cus_2"},
			},
		})

	assert.Equal(t, palettequestion.ToolCallRan, kept.Outcome)
	assert.Empty(t, kept.Reason)
	assert.Equal(t, "2 records", kept.Summary)
	assert.Equal(t, 2, kept.Records)
	assert.Empty(t, kept.Read, "a search goes to no Desk read")
	assert.Equal(t, map[string]any{}, kept.Params)
}

func TestToolCall_NumbersAQuestionsReadsInOrder(t *testing.T) {
	t.Parallel()

	events := &streamEvents{}

	first := events.toolCall(callFor("c1", "get_record", nil), llmports.NovaToolResult{})
	second := events.toolCall(callFor("c2", "list_records", nil), llmports.NovaToolResult{})

	require.Equal(t, 1, first.Sequence)
	assert.Equal(t, 2, second.Sequence)
	assert.Equal(t, map[string]any{}, first.Arguments, "the column holds an object, never null")
}
