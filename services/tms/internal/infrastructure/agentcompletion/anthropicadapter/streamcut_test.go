package anthropicadapter

import (
	"io"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladaptertest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/stretchr/testify/require"
)

func TestAnthropicAdapter_StreamWithoutAnEndingIsCut(t *testing.T) {
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
			`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"These are"}}`,
		},
	))

	sink, _ := modeladaptertest.CollectDeltas()
	streamer := NewAnthropicAdapter().(modeladapter.Streamer)
	_, err := streamer.Stream(t.Context(), modeladaptertest.CallFor(
		aiprovider.KindAnthropicMessages, server.URL,
		&modeladapter.Request{System: "sys", Messages: modeladapter.UserMessage("who?")},
	), sink)

	require.ErrorIs(t, err, io.ErrUnexpectedEOF)
}
