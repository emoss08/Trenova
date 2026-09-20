package modeladapter

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// slowStreamServer writes the first delta, pauses, then finishes. The pause
// stands in for a model that thinks partway through an answer: the connection
// is healthy and bytes are still coming, just not yet.
func slowStreamServer(t *testing.T, pause time.Duration) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, ok := w.(http.Flusher)
		require.True(t, ok)

		_, _ = w.Write([]byte(sse([2]string{
			"",
			`{"model":"m","choices":[{"index":0,"delta":{"content":"**Dr"},"finish_reason":null}]}`,
		})))
		flusher.Flush()

		time.Sleep(pause)

		_, _ = w.Write([]byte(sse(
			[2]string{
				"",
				`{"model":"m","choices":[{"index":0,"delta":{"content":"iver list complete."},"finish_reason":"stop"}]}`,
			},
			[2]string{"", "[DONE]"},
		)))
		flusher.Flush()
	}))
	t.Cleanup(server.Close)

	return server
}

// A generation that takes longer than the blocking-call timeout is normal, not
// a failure: http.Client.Timeout covers reading the body, so a stream served by
// a client carrying one dies mid-sentence however healthy the connection is.
func TestStream_OutlivesTheBlockingRequestTimeout(t *testing.T) {
	t.Parallel()

	server := slowStreamServer(t, 300*time.Millisecond)

	call := &Call{
		Provider: &aiprovider.Provider{
			Kind:                aiprovider.KindOpenAIChat,
			BaseURL:             server.URL,
			Model:               "m",
			AllowPrivateNetwork: true,
		},
		Client:       &http.Client{Timeout: 50 * time.Millisecond},
		StreamClient: &http.Client{},
		Request:      &Request{Messages: []Message{{Role: "user", Content: "who?"}}},
	}

	resp, deltas := streamWith(t, NewOpenAIChatAdapter(), call)

	assert.Equal(t, "**Driver list complete.", resp.Text)
	assert.Equal(t, []string{"**Dr", "iver list complete."}, deltas)
}

// Removing the whole-request deadline must not leave a dead provider holding
// the turn open forever. Liveness moves to the gap between bytes: a stream that
// goes silent past its idle window ends, and says that is what happened.
func TestStream_EndsAStreamThatGoesSilent(t *testing.T) {
	t.Parallel()

	released := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, ok := w.(http.Flusher)
		require.True(t, ok)

		_, _ = w.Write([]byte(sse([2]string{
			"",
			`{"model":"m","choices":[{"index":0,"delta":{"content":"**Dr"},"finish_reason":null}]}`,
		})))
		flusher.Flush()

		select {
		case <-r.Context().Done():
		case <-released:
		}
	}))
	t.Cleanup(func() {
		close(released)
		server.Close()
	})

	call := &Call{
		Provider: &aiprovider.Provider{
			Kind:                aiprovider.KindOpenAIChat,
			BaseURL:             server.URL,
			Model:               "m",
			AllowPrivateNetwork: true,
		},
		StreamClient: &http.Client{},
		StreamIdle:   150 * time.Millisecond,
		Request:      &Request{Messages: []Message{{Role: "user", Content: "who?"}}},
	}

	streamer, ok := NewOpenAIChatAdapter().(Streamer)
	require.True(t, ok)

	sink, deltas := collectDeltas()
	started := time.Now()
	_, err := streamer.Stream(t.Context(), call, sink)

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrStreamStalled)
	assert.Less(t, time.Since(started), 5*time.Second, "the idle guard, not a test timeout, ended this")
	// What did arrive is still the reader's: the router keeps it and marks the
	// reply as cut off rather than discarding a partial answer.
	assert.Equal(t, []string{"**Dr"}, *deltas)
}
