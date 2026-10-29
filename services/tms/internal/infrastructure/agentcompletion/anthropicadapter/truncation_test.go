package anthropicadapter

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladaptertest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/stretchr/testify/assert"
)

func TestAnthropicAdapter_StreamReportsAMaxTokensStopAsTruncated(t *testing.T) {
	t.Parallel()

	server, _ := modeladaptertest.StreamServer(t, "text/event-stream", modeladaptertest.SSE(
		[2]string{
			"message_start",
			`{"type":"message_start","message":{"model":"claude-x","usage":{"input_tokens":9}}}`,
		},
		[2]string{
			"content_block_start",
			`{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`,
		},
		[2]string{
			"content_block_delta",
			`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"The drivers with"}}`,
		},
		[2]string{"content_block_stop", `{"type":"content_block_stop","index":0}`},
		[2]string{
			"message_delta",
			`{"type":"message_delta","delta":{"stop_reason":"max_tokens"},"usage":{"output_tokens":12}}`,
		},
		[2]string{"message_stop", `{"type":"message_stop"}`},
	))

	resp, _ := modeladaptertest.StreamWith(t, NewAnthropicAdapter(), modeladaptertest.CallFor(
		aiprovider.KindAnthropicMessages, server.URL,
		&modeladapter.Request{System: "sys", Messages: modeladapter.UserMessage("who?")},
	))

	assert.True(t, resp.Truncated)
	assert.False(t, resp.Refused)
}
