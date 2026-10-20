package completionrouter

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompleteChat_CarriesTheAdvisorAndChargesItsTokens(t *testing.T) {
	t.Parallel()

	var (
		mu     sync.Mutex
		body   map[string]any
		header string
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		decoded := map[string]any{}
		_ = sonic.ConfigDefault.NewDecoder(r.Body).Decode(&decoded)
		mu.Lock()
		body = decoded
		header = r.Header.Get("Anthropic-Beta")
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"msg_1","model":"claude-haiku-5-5","stop_reason":"end_turn",` +
			`"content":[{"type":"server_tool_use","id":"srvtoolu_1","name":"advisor","input":{}},` +
			`{"type":"advisor_tool_result","tool_use_id":"srvtoolu_1","content":` +
			`{"type":"advisor_redacted_result","encrypted_content":"enc","stop_reason":"end_turn"}},` +
			`{"type":"text","text":"done"}],` +
			`"usage":{"input_tokens":10000,"output_tokens":10000,"iterations":[` +
			`{"type":"advisor_message","input_tokens":1000000,"output_tokens":100000,` +
			`"model":"claude-opus-5-5"}]}}`))
	}))
	t.Cleanup(server.Close)

	input, output := decimal.RequireFromString("0.10"), decimal.RequireFromString("0.50")
	provider := chatProvider("haiku", server.URL, 10)
	provider.Kind = aiprovider.KindAnthropicMessages
	provider.Model = "claude-haiku-5-5"
	provider.InputCostPerMillion = &input
	provider.OutputCostPerMillion = &output

	svc := newTestService(t, provider)
	key, err := svc.encryption.EncryptString("sk-ant-test")
	require.NoError(t, err)
	provider.APIKey = key

	req := chatRequest(provider.ID)
	req.Advisor = agentports.DefaultAdvisor("")

	result, err := svc.CompleteChat(t.Context(), req)
	require.NoError(t, err)

	mu.Lock()
	defer mu.Unlock()
	assert.Contains(t, header, "advisor-tool-2026-03-01")
	tools, ok := body["tools"].([]any)
	require.True(t, ok)
	require.Len(t, tools, 1)
	assert.Equal(t, "claude-opus-5-5", tools[0].(map[string]any)["model"])

	assert.Equal(t, "done", result.Text)
	require.NotNil(t, result.Advisor)
	assert.Equal(t, 1, result.Advisor.Calls)
	require.NotNil(t, result.Reasoning)
	assert.Equal(t, "claude-opus-5-5", result.Reasoning.AdvisorModel())
	require.NotNil(t, result.CostUSD)
	assert.True(t, result.CostUSD.Equal(decimal.RequireFromString("6.006")), result.CostUSD.String(),
		"Haiku's own tokens at its price plus the advisor's at Opus 5.5's list price")
}
