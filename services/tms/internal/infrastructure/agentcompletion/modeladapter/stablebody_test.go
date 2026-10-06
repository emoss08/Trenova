package modeladapter

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
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

// wideRequest carries tool schemas and replayed tool-call arguments with
// enough keys that an unordered encoding would differ between calls.
func wideRequest() *Request {
	properties := map[string]any{}
	arguments := map[string]any{}
	for i := range 24 {
		key := "field_" + strconv.Itoa(i)
		properties[key] = map[string]any{"type": "string", "description": "field " + key}
		arguments[key] = "value " + key
	}

	return &Request{
		System: "be brief",
		Messages: []Message{
			{Role: RoleUser, Content: "look these up"},
			{
				Role: RoleAssistant,
				ToolCalls: []ToolCall{
					{ID: "call_1", Name: "lookup_wide", Arguments: arguments},
				},
			},
			{Role: RoleTool, ToolCallID: "call_1", Content: "{}"},
			{Role: RoleUser, Content: "and again"},
		},
		Tools: []ToolSpec{{
			Name:        "lookup_wide",
			Description: "Look up many fields",
			Parameters:  map[string]any{"type": "object", "properties": properties},
		}},
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
		adapter     Adapter
		contentType string
		reply       string
	}{
		{
			kind:        aiprovider.KindAnthropicMessages,
			adapter:     NewAnthropicAdapter(),
			contentType: "text/event-stream",
			reply: sse(
				[2]string{"message_start", `{"type":"message_start","message":{"model":"m","usage":{"input_tokens":1}}}`},
				[2]string{"message_stop", `{"type":"message_stop"}`},
			),
		},
		{
			kind:        aiprovider.KindOpenAIChat,
			adapter:     NewOpenAIChatAdapter(),
			contentType: "text/event-stream",
			reply: sse(
				[2]string{"", `{"model":"m","choices":[{"index":0,"delta":{"content":"ok"},"finish_reason":"stop"}]}`},
				[2]string{"", "[DONE]"},
			),
		},
		{
			kind:        aiprovider.KindOpenAIResponses,
			adapter:     NewOpenAIResponsesAdapter(),
			contentType: "text/event-stream",
			reply: sse(
				[2]string{"response.completed", `{"type":"response.completed","response":{"model":"m","status":"completed","output":[]}}`},
			),
		},
		{
			kind:        aiprovider.KindOllama,
			adapter:     NewOllamaAdapter(),
			contentType: "application/x-ndjson",
			reply:       `{"model":"m","message":{"role":"assistant","content":"ok"},"done":true,"done_reason":"stop"}` + "\n",
		},
	}

	for _, tc := range cases {
		t.Run(string(tc.kind), func(t *testing.T) {
			t.Parallel()

			server, bodies := rawServer(t, tc.contentType, tc.reply)
			for range 20 {
				_, _ = streamWith(t, tc.adapter, callFor(tc.kind, server.URL, wideRequest()))
			}

			sent := bodies()
			require.Len(t, sent, 20)
			for i := 1; i < len(sent); i++ {
				require.Equal(t, string(sent[0]), string(sent[i]), "request %d differed", i)
			}
		})
	}
}
