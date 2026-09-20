package completionrouter

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// dyingStreamServer sends some of an SSE reply and then drops the connection,
// which is what a rate-limited or overloaded endpoint actually does.
func dyingStreamServer(t *testing.T, chunks []string) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		flusher, ok := w.(http.Flusher)
		require.True(t, ok)

		for _, chunk := range chunks {
			payload := `{"choices":[{"delta":{"content":"` + chunk + `"}}]}`
			_, _ = w.Write([]byte("data: " + payload + "\n\n"))
			flusher.Flush()
		}

		// No [DONE], no close frame: the stream simply stops.
		hijacker, ok := w.(http.Hijacker)
		if !ok {
			return
		}
		conn, _, err := hijacker.Hijack()
		if err == nil {
			_ = conn.Close()
		}
	}))
}

/*
A reply that stops partway is still a reply.

Asked which drivers hold a current hazmat endorsement, a model listed both
correctly and was most of the way through explaining why they were
NonCompliant — "Sarah's medical card expires in" — when the stream died. The
reader watched a good answer appear and then be replaced by "the assistant
could not finish this reply". Everything that had arrived was discarded.

The words that arrived are the ones that answered the question. They are kept,
the result says it was cut off, and the turn ends rather than erroring.
*/
func TestStreamChat_KeepsWhatArrivedWhenTheStreamDies(t *testing.T) {
	t.Parallel()

	server := dyingStreamServer(t, []string{
		"Sarah Williams - endorsement X. ",
		"Jane Doe - endorsement H. ",
		"Both are NonCompliant because",
	})
	provider := chatProvider("flaky", server.URL, 10)
	service := newTestService(t, provider)

	var streamed strings.Builder
	result, err := service.StreamChat(t.Context(), chatRequest(pulid.Nil), func(delta string) {
		streamed.WriteString(delta)
	})

	require.NoError(t, err, "a reply that stopped partway is not an error the reader should see")
	require.NotNil(t, result)

	assert.True(t, result.Truncated)
	assert.Contains(t, result.Text, "Sarah Williams")
	assert.Contains(t, result.Text, "Jane Doe")
	assert.Equal(t, streamed.String(), result.Text,
		"what was saved and what the reader watched have to be the same text")
}

// A stream that dies before saying anything has nothing worth keeping, so the
// usual fallthrough to the next provider still applies.
func TestStreamChat_FallsThroughWhenNothingWasSaid(t *testing.T) {
	t.Parallel()

	dead := dyingStreamServer(t, nil)
	// A streaming server, not the plain-JSON one: the router streams whenever
	// the adapter can, so a fake that only answers the non-streaming shape
	// parses as an empty reply rather than as the fallthrough's success.
	good := completeStreamServer(t, "Answered by the second provider.")

	service := newTestService(t,
		chatProvider("dies-first", dead.URL, 10),
		chatProvider("answers", good.URL, 20),
	)

	result, err := service.StreamChat(t.Context(), chatRequest(pulid.Nil), func(string) {})
	require.NoError(t, err)

	assert.False(t, result.Truncated)
	assert.Contains(t, result.Text, "second provider")
}

// completeStreamServer sends a whole reply and terminates it properly, which is
// what the fallthrough needs to land on.
func completeStreamServer(t *testing.T, content string) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		flusher, ok := w.(http.Flusher)
		require.True(t, ok)

		payload := `{"choices":[{"delta":{"content":"` + content + `"}}]}`
		_, _ = w.Write([]byte("data: " + payload + "\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
		flusher.Flush()
	}))
}
