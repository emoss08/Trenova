package openaiadapter

import (
	"io"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladaptertest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A connection that drops mid-reply closes the stream without a finish reason
// or a closing marker. Read as a normal end, the half reply was saved as the
// whole answer; it is reported as a cut so it is kept and marked as one.
func TestOpenAIChatAdapter_StreamWithoutAnEndingIsCut(t *testing.T) {
	t.Parallel()

	server, _ := modeladaptertest.StreamServer(t, "text/event-stream", modeladaptertest.SSE(
		[2]string{
			"",
			`{"model":"m","choices":[{"index":0,"delta":{"content":"These are the drivers on"},"finish_reason":null}]}`,
		},
	))

	sink, deltas := modeladaptertest.CollectDeltas()
	streamer := NewOpenAIChatAdapter().(modeladapter.Streamer)
	_, err := streamer.Stream(t.Context(), modeladaptertest.CallFor(
		aiprovider.KindOpenAIChat,
		server.URL,
		&modeladapter.Request{Messages: modeladapter.UserMessage("who?")},
	), sink)

	require.ErrorIs(t, err, io.ErrUnexpectedEOF)
	assert.Equal(t, []string{"These are the drivers on"}, *deltas)
}
