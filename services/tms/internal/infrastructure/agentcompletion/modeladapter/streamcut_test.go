package modeladapter

import (
	"io"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A connection that drops mid-reply closes the stream without a finish reason
// or a closing marker. Read as a normal end, the half reply was saved as the
// whole answer; it is reported as a cut so it is kept and marked as one.
func TestOpenAIChatAdapter_StreamWithoutAnEndingIsCut(t *testing.T) {
	t.Parallel()

	server, _ := streamServer(t, "text/event-stream", sse(
		[2]string{"", `{"model":"m","choices":[{"index":0,"delta":{"content":"These are the drivers on"},"finish_reason":null}]}`},
	))

	sink, deltas := collectDeltas()
	streamer := NewOpenAIChatAdapter().(Streamer)
	_, err := streamer.Stream(t.Context(), callFor(
		aiprovider.KindOpenAIChat, server.URL, &Request{Messages: UserMessage("who?")},
	), sink)

	require.ErrorIs(t, err, io.ErrUnexpectedEOF)
	assert.Equal(t, []string{"These are the drivers on"}, *deltas)
}

func TestAnthropicAdapter_StreamWithoutAnEndingIsCut(t *testing.T) {
	t.Parallel()

	server, _ := streamServer(t, "text/event-stream", sse(
		[2]string{"message_start", `{"type":"message_start","message":{"model":"claude-x","usage":{"input_tokens":9}}}`},
		[2]string{"content_block_start", `{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`},
		[2]string{"content_block_delta", `{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"These are"}}`},
	))

	sink, _ := collectDeltas()
	streamer := NewAnthropicAdapter().(Streamer)
	_, err := streamer.Stream(t.Context(), callFor(
		aiprovider.KindAnthropicMessages, server.URL,
		&Request{System: "sys", Messages: UserMessage("who?")},
	), sink)

	require.ErrorIs(t, err, io.ErrUnexpectedEOF)
}
