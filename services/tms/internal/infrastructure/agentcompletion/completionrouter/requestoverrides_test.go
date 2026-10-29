package completionrouter

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aireasoning"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func capturingChatServer(t *testing.T) (*httptest.Server, func() map[string]any) {
	t.Helper()

	var (
		mu   sync.Mutex
		body map[string]any
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		decoded := map[string]any{}
		_ = sonic.ConfigDefault.NewDecoder(r.Body).Decode(&decoded)
		mu.Lock()
		body = decoded
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"model":"test-model","choices":[{"finish_reason":"stop",` +
			`"message":{"role":"assistant","content":"done"}}],` +
			`"usage":{"prompt_tokens":3,"completion_tokens":1}}`))
	}))
	t.Cleanup(server.Close)

	return server, func() map[string]any {
		mu.Lock()
		defer mu.Unlock()
		return body
	}
}

func TestCompleteChat_CarriesTheRequestsReasoningAndToolChoice(t *testing.T) {
	t.Parallel()

	server, captured := capturingChatServer(t)
	provider := chatProvider("chat", server.URL, 10)
	provider.ReasoningEffort = aireasoning.ReasoningHigh

	req := chatRequest(provider.ID)
	req.Reasoning = aireasoning.ReasoningLow
	req.ToolChoiceNone = true
	req.Tools = []kernelports.ToolSpec{{
		Name:        "lookup_shipment",
		Description: "Look up a shipment",
		Parameters:  map[string]any{"type": "object", "properties": map[string]any{}},
	}}

	_, err := newTestService(t, provider).CompleteChat(t.Context(), req)
	require.NoError(t, err)

	body := captured()
	assert.Equal(t, "low", body["reasoning_effort"], "the request's effort outranks the provider's")
	assert.Equal(t, "none", body["tool_choice"])
}

func TestCompleteStructured_CarriesTheCacheReadTheProviderReported(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"model":"test-model","choices":[{"finish_reason":"stop",` +
			`"message":{"role":"assistant","content":"{\"answer\":\"ok\"}"}}],` +
			`"usage":{"prompt_tokens":1000,"completion_tokens":7,` +
			`"prompt_tokens_details":{"cached_tokens":800}}}`))
	}))
	t.Cleanup(server.Close)

	result, err := newTestService(t, openAIChatProvider("chat", server.URL, 10)).
		CompleteStructured(t.Context(), generalRequest())
	require.NoError(t, err)

	assert.Equal(t, 1000, result.InputTokens)
	assert.Equal(t, 800, result.CacheReadTokens)
}

func TestCompleteChat_AsksAgainWithoutToolChoiceWhenAGatewayRefusesIt(t *testing.T) {
	t.Parallel()

	var (
		mu     sync.Mutex
		bodies []map[string]any
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		decoded := map[string]any{}
		_ = sonic.ConfigDefault.NewDecoder(r.Body).Decode(&decoded)
		mu.Lock()
		bodies = append(bodies, decoded)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if _, ok := decoded["tool_choice"]; ok {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":{"message":"Invalid value for 'tool_choice': none"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"model":"test-model","choices":[{"finish_reason":"stop",` +
			`"message":{"role":"assistant","content":"done"}}],` +
			`"usage":{"prompt_tokens":3,"completion_tokens":1}}`))
	}))
	t.Cleanup(server.Close)

	provider := chatProvider("gateway", server.URL, 10)
	req := chatRequest(provider.ID)
	req.ToolChoiceNone = true
	req.Tools = []kernelports.ToolSpec{{
		Name:        "lookup_shipment",
		Description: "Look up a shipment",
		Parameters:  map[string]any{"type": "object", "properties": map[string]any{}},
	}}

	result, err := newTestService(t, provider).CompleteChat(t.Context(), req)
	require.NoError(t, err)
	assert.Equal(t, "done", result.Text)

	mu.Lock()
	defer mu.Unlock()
	require.Len(t, bodies, 2, "one refusal, then one call without tool_choice")
	assert.Equal(t, "none", bodies[0]["tool_choice"])
	assert.NotContains(t, bodies[1], "tool_choice")
	assert.NotEmpty(t, bodies[1]["tools"], "the tools stay declared")
}

func TestCompleteChat_DoesNotAskAgainForAnUnrelatedBadRequest(t *testing.T) {
	t.Parallel()

	var (
		mu    sync.Mutex
		count int
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		mu.Lock()
		count++
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"message":"messages must not be empty"}}`))
	}))
	t.Cleanup(server.Close)

	provider := chatProvider("gateway", server.URL, 10)
	req := chatRequest(provider.ID)
	req.ToolChoiceNone = true
	req.Tools = []kernelports.ToolSpec{{
		Name:        "lookup_shipment",
		Description: "Look up a shipment",
		Parameters:  map[string]any{"type": "object", "properties": map[string]any{}},
	}}

	_, err := newTestService(t, provider).CompleteChat(t.Context(), req)
	require.Error(t, err)

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, 1, count, "only a refusal that names tool_choice is asked again")
}
