package openaiadapter

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladaptertest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A max-tokens stop was invisible in every adapter: the reply ended mid-sentence
// with nothing saying so, and when the cut fell inside a tool call's JSON the
// tool ran on whatever half-object survived. Each protocol names the stop
// differently; each is surfaced the same way.
func TestOpenAIChatAdapter_StreamReportsAMaxTokensStopAsTruncated(t *testing.T) {
	t.Parallel()

	server, _ := modeladaptertest.StreamServer(t, "text/event-stream", modeladaptertest.SSE(
		[2]string{
			"",
			`{"model":"m","choices":[{"index":0,"delta":{"content":"The drivers with"},"finish_reason":null}]}`,
		},
		[2]string{"", `{"model":"m","choices":[{"index":0,"delta":{},"finish_reason":"length"}]}`},
		[2]string{"", "[DONE]"},
	))

	resp, _ := modeladaptertest.StreamWith(t, NewOpenAIChatAdapter(), modeladaptertest.CallFor(
		aiprovider.KindOpenAIChat,
		server.URL,
		&modeladapter.Request{Messages: modeladapter.UserMessage("who?")},
	))

	assert.True(t, resp.Truncated)
	assert.Equal(t, "The drivers with", resp.Text)
}

// The cut can land inside a tool call. The half-object must not be handed to
// the tool as an argument map — a list tool given {} runs unfiltered and the
// model reports the first page as the filtered answer.
func TestOpenAIChatAdapter_StreamMarksToolArgumentsCutByTheOutputLimit(t *testing.T) {
	t.Parallel()

	server, _ := modeladaptertest.StreamServer(t, "text/event-stream", modeladaptertest.SSE(
		[2]string{
			"",
			`{"model":"m","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"list_shipments","arguments":"{\"filters\":[{\"field\":\"sta"}}]},"finish_reason":null}]}`,
		},
		[2]string{"", `{"model":"m","choices":[{"index":0,"delta":{},"finish_reason":"length"}]}`},
		[2]string{"", "[DONE]"},
	))

	resp, _ := modeladaptertest.StreamWith(t, NewOpenAIChatAdapter(), modeladaptertest.CallFor(
		aiprovider.KindOpenAIChat,
		server.URL,
		&modeladapter.Request{
			Messages: modeladapter.UserMessage("unbilled?"),
			Tools:    modeladaptertest.LookupTool(),
		},
	))

	require.Len(t, resp.ToolCalls, 1)
	assert.NotEmpty(t, resp.ToolCalls[0].ArgumentsError, "cut JSON is named, not silently emptied")
	assert.Empty(t, resp.ToolCalls[0].Arguments)
	assert.True(t, resp.Truncated)
}

// Some providers reuse index 0 for every call and tell them apart by id alone.
// Keyed on index, two calls merged into one buffer and both were mangled.
func TestOpenAIChatAdapter_StreamSeparatesCallsThatShareAnIndex(t *testing.T) {
	t.Parallel()

	server, _ := modeladaptertest.StreamServer(t, "text/event-stream", modeladaptertest.SSE(
		[2]string{
			"",
			`{"model":"m","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_a","type":"function","function":{"name":"get_worker","arguments":"{\"id\":\"w1\"}"}}]},"finish_reason":null}]}`,
		},
		[2]string{
			"",
			`{"model":"m","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_b","type":"function","function":{"name":"get_worker","arguments":"{\"id\":\"w2\"}"}}]},"finish_reason":null}]}`,
		},
		[2]string{
			"",
			`{"model":"m","choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}`,
		},
		[2]string{"", "[DONE]"},
	))

	resp, _ := modeladaptertest.StreamWith(t, NewOpenAIChatAdapter(), modeladaptertest.CallFor(
		aiprovider.KindOpenAIChat,
		server.URL,
		&modeladapter.Request{
			Messages: modeladapter.UserMessage("both?"),
			Tools:    modeladaptertest.LookupTool(),
		},
	))

	require.Len(t, resp.ToolCalls, 2)
	assert.Equal(t, "w1", resp.ToolCalls[0].Arguments["id"])
	assert.Equal(t, "w2", resp.ToolCalls[1].Arguments["id"])
	assert.False(t, resp.Truncated)
}

// The ordinary OpenAI shape still assembles: one call whose fragments carry the
// id only on the first fragment and the index on all of them.
func TestOpenAIChatAdapter_StreamStillAssemblesFragmentsByIndex(t *testing.T) {
	t.Parallel()

	server, _ := modeladaptertest.StreamServer(t, "text/event-stream", modeladaptertest.SSE(
		[2]string{
			"",
			`{"model":"m","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"get_worker","arguments":"{\"id\":"}}]},"finish_reason":null}]}`,
		},
		[2]string{
			"",
			`{"model":"m","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"\"w1\"}"}}]},"finish_reason":null}]}`,
		},
		[2]string{"", "[DONE]"},
	))

	resp, _ := modeladaptertest.StreamWith(t, NewOpenAIChatAdapter(), modeladaptertest.CallFor(
		aiprovider.KindOpenAIChat,
		server.URL,
		&modeladapter.Request{
			Messages: modeladapter.UserMessage("who?"),
			Tools:    modeladaptertest.LookupTool(),
		},
	))

	require.Len(t, resp.ToolCalls, 1)
	assert.Equal(t, "call_1", resp.ToolCalls[0].ID)
	assert.Equal(t, "w1", resp.ToolCalls[0].Arguments["id"])
	assert.Empty(t, resp.ToolCalls[0].ArgumentsError)
}

func TestOpenAIResponsesAdapter_StreamReportsAnIncompleteResponseAsTruncated(t *testing.T) {
	t.Parallel()

	server, _ := modeladaptertest.StreamServer(t, "text/event-stream", modeladaptertest.SSE(
		[2]string{"response.created", `{"type":"response.created","response":{"model":"gpt-x"}}`},
		[2]string{
			"response.output_text.delta",
			`{"type":"response.output_text.delta","delta":"The drivers with"}`,
		},
		[2]string{
			"response.incomplete",
			`{"type":"response.incomplete","response":{"model":"gpt-x","status":"incomplete","incomplete_details":{"reason":"max_output_tokens"},"output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"The drivers with"}]}],"usage":{"input_tokens":3,"output_tokens":8}}}`,
		},
	))

	resp, _ := modeladaptertest.StreamWith(t, NewOpenAIResponsesAdapter(), modeladaptertest.CallFor(
		aiprovider.KindOpenAIResponses,
		server.URL,
		&modeladapter.Request{Messages: modeladapter.UserMessage("who?")},
	))

	assert.True(t, resp.Truncated)
	assert.Equal(t, "The drivers with", resp.Text)
}
