package anthropicadapter

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladaptertest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnthropicAdapter_StreamsTextBlocksAndToolUseInput(t *testing.T) {
	t.Parallel()

	server, captured := modeladaptertest.StreamServer(t, "text/event-stream", modeladaptertest.SSE(
		[2]string{
			"message_start",
			`{"type":"message_start","message":{"model":"claude-x","usage":{"input_tokens":9,"output_tokens":1}}}`,
		},
		[2]string{
			"content_block_start",
			`{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`,
		},
		[2]string{
			"content_block_delta",
			`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Looking"}}`,
		},
		[2]string{
			"content_block_delta",
			`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":" that up."}}`,
		},
		[2]string{"content_block_stop", `{"type":"content_block_stop","index":0}`},
		[2]string{
			"content_block_start",
			`{"type":"content_block_start","index":1,"content_block":{"type":"tool_use","id":"toolu_1","name":"lookup_shipment","input":{}}}`,
		},
		[2]string{
			"content_block_delta",
			`{"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":"{\"number\":"}}`,
		},
		[2]string{
			"content_block_delta",
			`{"type":"content_block_delta","index":1,"delta":{"type":"input_json_delta","partial_json":"\"S12345\"}"}}`,
		},
		[2]string{"content_block_stop", `{"type":"content_block_stop","index":1}`},
		[2]string{
			"message_delta",
			`{"type":"message_delta","delta":{"stop_reason":"tool_use"},"usage":{"output_tokens":12}}`,
		},
		[2]string{"message_stop", `{"type":"message_stop"}`},
	))

	resp, deltas := modeladaptertest.StreamWith(t, NewAnthropicAdapter(), modeladaptertest.CallFor(
		aiprovider.KindAnthropicMessages,
		server.URL,
		&modeladapter.Request{
			System:   "sys",
			Messages: modeladapter.UserMessage("Where is S12345?"),
			Tools:    modeladaptertest.LookupTool(),
		},
	))

	assert.Equal(t, []string{"Looking", " that up."}, deltas)
	assert.Equal(t, "Looking that up.", resp.Text)
	require.Len(t, resp.ToolCalls, 1)
	assert.Equal(t, "toolu_1", resp.ToolCalls[0].ID)
	assert.Equal(t, "S12345", resp.ToolCalls[0].Arguments["number"])
	assert.Equal(t, "claude-x", resp.ModelIdentifier)
	assert.Equal(t, 9, resp.InputTokens)
	assert.Equal(t, 12, resp.OutputTokens)
	assert.Equal(t, true, (*captured)["stream"])
}

func TestAnthropicAdapter_StreamErrorEventFails(t *testing.T) {
	t.Parallel()

	server, _ := modeladaptertest.StreamServer(t, "text/event-stream", modeladaptertest.SSE(
		[2]string{
			"message_start",
			`{"type":"message_start","message":{"model":"claude-x","usage":{"input_tokens":1}}}`,
		},
		[2]string{
			"error",
			`{"type":"error","error":{"type":"overloaded_error","message":"Overloaded"}}`,
		},
	))

	streamer := NewAnthropicAdapter().(modeladapter.Streamer)
	_, err := streamer.Stream(t.Context(), modeladaptertest.CallFor(
		aiprovider.KindAnthropicMessages,
		server.URL,
		&modeladapter.Request{Messages: modeladapter.UserMessage("hi")},
	), func(string) {})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Overloaded")
	assert.True(t, modeladapter.IsRetryable(err), "an overloaded stream is worth retrying")
}
