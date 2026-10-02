package modeladapter

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// headerServer replies with a canned body and keeps the request's beta header
// and decoded body.
func headerServer(t *testing.T, contentType, reply string) (*httptest.Server, func() (string, map[string]any)) {
	t.Helper()

	var (
		mu   sync.Mutex
		beta string
		body map[string]any
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		beta = r.Header.Get("anthropic-beta")
		body = map[string]any{}
		_ = sonic.ConfigDefault.NewDecoder(r.Body).Decode(&body)
		mu.Unlock()
		w.Header().Set("Content-Type", contentType)
		_, _ = w.Write([]byte(reply))
	}))
	t.Cleanup(server.Close)

	return server, func() (string, map[string]any) {
		mu.Lock()
		defer mu.Unlock()
		return beta, body
	}
}

func toolLoopHistory() []Message {
	earlier := &ReasoningTrace{Text: "earlier turn", Signature: "sig_earlier"}
	current := &ReasoningTrace{Text: "this turn", Signature: "sig_current"}

	return []Message{
		{Role: RoleUser, Content: "Where is S1?"},
		{
			Role:      RoleAssistant,
			Reasoning: earlier,
			ToolCalls: []ToolCall{{ID: "call_1", Name: "get_shipment", Arguments: map[string]any{}}},
		},
		{Role: RoleTool, ToolCallID: "call_1", Content: "{}"},
		{Role: RoleAssistant, Content: "In Dallas.", Reasoning: earlier},
		{Role: RoleUser, Content: "And S2?"},
		{
			Role:      RoleAssistant,
			Reasoning: current,
			ToolCalls: []ToolCall{{ID: "call_2", Name: "get_shipment", Arguments: map[string]any{}}},
		},
		{Role: RoleTool, ToolCallID: "call_2", Content: "{}"},
	}
}

func signaturesSent(messages []anthropicMessage) []string {
	var signatures []string
	for _, message := range messages {
		for _, block := range message.Content {
			if block.Type == "thinking" {
				signatures = append(signatures, block.Signature)
			}
		}
	}

	return signatures
}

/*
A thinking block is bound to the conversation that produced it, and the
conversation before an earlier turn's block changes on every turn: the system
prompt carries the turn's page and memories, and older tool results are
shortened. Replaying that block was a 400 on the newest models. Only the
current turn's tool loop needs its thinking back; earlier turns' is dropped,
which is the edit the API allows.
*/
func TestToAnthropicMessages_ReplaysOnlyTheCurrentTurnsThinking(t *testing.T) {
	t.Parallel()

	sent := toAnthropicMessages(toolLoopHistory())

	assert.Equal(t, []string{"sig_current"}, signaturesSent(sent))
}

/*
Within a turn the conversation before a block holds still, but a tool the
model finds mid-turn changes the tool list a block is bound to. On a model
that checks, the request asks for a mismatched block to be dropped rather than
the request refused.
*/
func TestAnthropicAdapter_AsksABindingModelToDropAMismatchedBlock(t *testing.T) {
	t.Parallel()

	reply := sse(
		[2]string{"message_start", `{"type":"message_start","message":{"model":"m","usage":{"input_tokens":1}}}`},
		[2]string{"message_stop", `{"type":"message_stop"}`},
	)
	for _, tc := range []struct {
		model  string
		effort aiprovider.ReasoningEffort
		bound  bool
	}{
		{"claude-opus-5-5", aiprovider.ReasoningOff, true},
		{"claude-opus-5-5", aiprovider.ReasoningHigh, true},
		{"claude-sonnet-5-5", aiprovider.ReasoningNone, true},
		{"claude-fable-5-1", aiprovider.ReasoningOff, true},
		{"claude-opus-4-8", aiprovider.ReasoningHigh, false},
		{"claude-haiku-4-5", aiprovider.ReasoningHigh, false},
		{"claude-mythos-5-1", aiprovider.ReasoningOff, false},
	} {
		server, sent := headerServer(t, "text/event-stream", reply)
		call := reasoningCall(aiprovider.KindAnthropicMessages, server.URL, tc.effort,
			&Request{Messages: toolLoopHistory()})
		call.Provider.Model = tc.model
		_, _ = streamWith(t, NewAnthropicAdapter(), call)

		beta, body := sent()
		thinking, _ := body["thinking"].(map[string]any)
		if !tc.bound {
			assert.Empty(t, beta, tc.model)
			if thinking != nil {
				assert.NotContains(t, thinking, "block_binding", tc.model)
			}
			continue
		}
		assert.Equal(t, anthropicBindingBeta, beta, tc.model)
		require.NotNil(t, thinking, tc.model)
		assert.Equal(t, "adaptive", thinking["type"], tc.model)
		assert.Equal(t,
			map[string]any{"prefix_mismatch_behavior": "drop_block"},
			thinking["block_binding"], tc.model)
	}
}

// A dropped block is the API working around an edit, and a recurring one is a
// harness edit worth finding, so each drop is counted on the reply.
func TestAnthropicAdapter_CountsTheThinkingTheAPIDropped(t *testing.T) {
	t.Parallel()

	transformations := `"input_transformations":[` +
		`{"type":"thinking_dropped","path":"messages.1.content.0","reason":"prefix_binding_mismatch"},` +
		`{"type":"thinking_dropped","path":"messages.3.content.0","reason":"model_binding_mismatch"},` +
		`{"type":"thinking_mismatch_allowed","path":"messages.5.content.0","reason":"prefix_binding_mismatch"}]`

	stream, _ := headerServer(t, "text/event-stream", sse(
		[2]string{"message_start", `{"type":"message_start","message":{"model":"m","usage":{"input_tokens":1},` + transformations + `}}`},
		[2]string{"content_block_start", `{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`},
		[2]string{"content_block_delta", `{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hi"}}`},
		[2]string{"content_block_stop", `{"type":"content_block_stop","index":0}`},
		[2]string{"message_stop", `{"type":"message_stop"}`},
	))
	call := reasoningCall(aiprovider.KindAnthropicMessages, stream.URL, aiprovider.ReasoningOff,
		&Request{Messages: UserMessage("hi")})
	call.Provider.Model = "claude-opus-5-5"
	resp, _ := streamWith(t, NewAnthropicAdapter(), call)
	require.NotNil(t, resp)
	assert.Equal(t, 2, resp.ThinkingDropped)

	blocking, _ := headerServer(t, "application/json",
		`{"model":"m","stop_reason":"end_turn","content":[{"type":"text","text":"Hi"}],"usage":{"input_tokens":1,"output_tokens":1},`+transformations+`}`)
	call = reasoningCall(aiprovider.KindAnthropicMessages, blocking.URL, aiprovider.ReasoningOff,
		&Request{Messages: UserMessage("hi")})
	call.Provider.Model = "claude-opus-5-5"
	complete, err := NewAnthropicAdapter().Complete(t.Context(), call)
	require.NoError(t, err)
	assert.Equal(t, 2, complete.ThinkingDropped)
}
