package completionrouter

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
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
	provider.ReasoningEffort = aiprovider.ReasoningHigh

	req := chatRequest(provider.ID)
	req.Reasoning = aiprovider.ReasoningLow
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
