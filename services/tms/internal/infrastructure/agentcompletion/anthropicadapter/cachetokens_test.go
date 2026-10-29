package anthropicadapter

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladaptertest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/stretchr/testify/assert"
)

func TestAnthropicAdapter_ReadsPromptCacheTokens(t *testing.T) {
	t.Parallel()

	server, _ := modeladaptertest.StreamServer(
		t,
		"application/json",
		`{"model":"model-a","stop_reason":"end_turn",`+
			`"content":[{"type":"text","text":"Done."}],`+
			`"usage":{"input_tokens":40,"output_tokens":7,"cache_read_input_tokens":1800,"cache_creation_input_tokens":120}}`,
	)

	resp := modeladaptertest.CompleteWith(t, NewAnthropicAdapter(), modeladaptertest.CallFor(
		aiprovider.KindAnthropicMessages,
		server.URL,
		&modeladapter.Request{Messages: modeladapter.UserMessage("hi")},
	))

	assert.Equal(t, 40, resp.InputTokens)
	assert.Equal(t, 7, resp.OutputTokens)
	assert.Equal(t, 1800, resp.CacheReadTokens)
	assert.Equal(t, 120, resp.CacheWriteTokens)
	assert.True(t, aiprovider.KindAnthropicMessages.CachesOutsideInput())
}

func TestAnthropicAdapter_StreamReadsPromptCacheTokens(t *testing.T) {
	t.Parallel()

	server, _ := modeladaptertest.StreamServer(t, "text/event-stream", modeladaptertest.SSE(
		[2]string{
			"message_start",
			`{"type":"message_start","message":{"model":"model-a","usage":{"input_tokens":12,"output_tokens":1,"cache_read_input_tokens":2048,"cache_creation_input_tokens":64}}}`,
		},
		[2]string{
			"content_block_start",
			`{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`,
		},
		[2]string{
			"content_block_delta",
			`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Done."}}`,
		},
		[2]string{
			"message_delta",
			`{"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":9}}`,
		},
		[2]string{"message_stop", `{"type":"message_stop"}`},
	))

	resp, _ := modeladaptertest.StreamWith(t, NewAnthropicAdapter(), modeladaptertest.CallFor(
		aiprovider.KindAnthropicMessages,
		server.URL,
		&modeladapter.Request{Messages: modeladapter.UserMessage("hi")},
	))

	assert.Equal(t, 12, resp.InputTokens)
	assert.Equal(t, 9, resp.OutputTokens)
	assert.Equal(t, 2048, resp.CacheReadTokens)
	assert.Equal(t, 64, resp.CacheWriteTokens)
}
