package adapterregistry

import (
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/anthropicadapter"
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladaptertest"
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/openaiadapter"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/stretchr/testify/require"
)

// rawServer keeps every request body exactly as it arrived.
func rawServer(t *testing.T, contentType, reply string) (*httptest.Server, func() [][]byte) {
	t.Helper()

	var (
		mu     sync.Mutex
		bodies [][]byte
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		bodies = append(bodies, body)
		mu.Unlock()
		w.Header().Set("Content-Type", contentType)
		_, _ = w.Write([]byte(reply))
	}))
	t.Cleanup(server.Close)

	return server, func() [][]byte {
		mu.Lock()
		defer mu.Unlock()
		return bodies
	}
}

/*
A provider caches the start of a prompt only when the bytes are the same, and
the tools and the replayed tool calls lead every request. They are maps, and
an encoder that walks a map in Go's random order sent different bytes on every
call, so no provider could ever reuse a cached prefix.
*/
func TestRequests_AreTheSameBytesEveryTime(t *testing.T) {
	t.Parallel()

	cases := []struct {
		kind        aiprovider.Kind
		adapter     modeladapter.Adapter
		contentType string
		reply       string
	}{
		{
			kind:        aiprovider.KindAnthropicMessages,
			adapter:     anthropicadapter.NewAnthropicAdapter(),
			contentType: "text/event-stream",
			reply: modeladaptertest.SSE(
				[2]string{
					"message_start",
					`{"type":"message_start","message":{"model":"m","usage":{"input_tokens":1}}}`,
				},
				[2]string{"message_stop", `{"type":"message_stop"}`},
			),
		},
		{
			kind:        aiprovider.KindOpenAIChat,
			adapter:     openaiadapter.NewOpenAIChatAdapter(),
			contentType: "text/event-stream",
			reply: modeladaptertest.SSE(
				[2]string{
					"",
					`{"model":"m","choices":[{"index":0,"delta":{"content":"ok"},"finish_reason":"stop"}]}`,
				},
				[2]string{"", "[DONE]"},
			),
		},
		{
			kind:        aiprovider.KindOpenAIResponses,
			adapter:     openaiadapter.NewOpenAIResponsesAdapter(),
			contentType: "text/event-stream",
			reply: modeladaptertest.SSE(
				[2]string{
					"response.completed",
					`{"type":"response.completed","response":{"model":"m","status":"completed","output":[]}}`,
				},
			),
		},
	}

	for _, tc := range cases {
		t.Run(string(tc.kind), func(t *testing.T) {
			t.Parallel()

			server, bodies := rawServer(t, tc.contentType, tc.reply)
			for range 20 {
				_, _ = modeladaptertest.StreamWith(
					t,
					tc.adapter,
					modeladaptertest.CallFor(tc.kind, server.URL, modeladaptertest.WideRequest()),
				)
			}

			sent := bodies()
			require.Len(t, sent, 20)
			for i := 1; i < len(sent); i++ {
				require.Equal(t, string(sent[0]), string(sent[i]), "request %d differed", i)
			}
		})
	}
}
