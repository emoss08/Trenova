package completionrouter

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const firstTokenDelay = 150 * time.Millisecond

// slowStartServer holds the headers back, then streams the given SSE frames,
// the way a model that is still thinking about its first word does.
func slowStartServer(t *testing.T, frames []string) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(firstTokenDelay)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flusher, ok := w.(http.Flusher)
		require.True(t, ok)
		for _, frame := range frames {
			_, _ = w.Write([]byte("data: " + frame + "\n\n"))
			flusher.Flush()
		}
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
		flusher.Flush()
	}))
	t.Cleanup(server.Close)

	return server
}

/*
How long a person waits for the first word is the number that says whether
the assistant feels fast, and nothing recorded it: an attempt's latency runs
to the end of the reply, so a fast model writing a long answer looked slow and
a slow one writing a word looked fast. The first piece of the reply — text or,
for a model that thinks aloud, its thinking — is stamped on the attempt.
*/
func TestStreamChat_RecordsTheTimeToTheFirstPieceOfTheReply(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"text":     `{"model":"m","choices":[{"index":0,"delta":{"content":"Friday."},"finish_reason":"stop"}]}`,
		"thinking": `{"model":"m","choices":[{"index":0,"delta":{"reasoning_content":"Checking."},"finish_reason":null}]}`,
	}
	for name, first := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			frames := []string{first}
			if name == "thinking" {
				frames = append(frames,
					`{"model":"m","choices":[{"index":0,"delta":{"content":"Friday."},"finish_reason":"stop"}]}`)
			}
			server := slowStartServer(t, frames)
			usage := &fakeUsage{}
			svc := newTestService(t, chatProvider("only", server.URL, 10))
			svc.usage = usage

			_, err := svc.StreamChat(t.Context(), chatRequest(pulid.Nil), func(string) {})
			require.NoError(t, err)

			rows := usage.recorded(t, 1)
			require.NotNil(t, rows[0].FirstTokenMs)
			assert.GreaterOrEqual(t, *rows[0].FirstTokenMs, firstTokenDelay.Milliseconds())
			assert.LessOrEqual(t, *rows[0].FirstTokenMs, rows[0].LatencyMs)
		})
	}
}

// A reply that is only a tool call writes nothing a person reads, so it has
// no first token to time.
func TestStreamChat_LeavesTheFirstTokenUnsetForAToolCallOnly(t *testing.T) {
	t.Parallel()

	server := slowStartServer(t, []string{
		`{"model":"m","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"lookup","arguments":"{}"}}]},"finish_reason":"tool_calls"}]}`,
	})
	usage := &fakeUsage{}
	svc := newTestService(t, chatProvider("only", server.URL, 10))
	svc.usage = usage

	_, err := svc.StreamChat(t.Context(), chatRequest(pulid.Nil), func(string) {})
	require.NoError(t, err)

	rows := usage.recorded(t, 1)
	assert.Nil(t, rows[0].FirstTokenMs)
}
