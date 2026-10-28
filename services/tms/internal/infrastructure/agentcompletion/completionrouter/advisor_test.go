package completionrouter

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace/aitracetest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
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
	req.Advisor = llmports.DefaultAdvisor("")

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
	assert.True(
		t,
		result.CostUSD.Equal(decimal.RequireFromString("6.006")),
		result.CostUSD.String(),
		"Haiku's own tokens at its price plus the advisor's at Opus 5.5's list price",
	)
}

func TestCompleteChat_PricesEachExecutorIterationAndRecordsTheAdvisor(t *testing.T) {
	t.Parallel()
	aitracetest.Install()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"msg_2","model":"claude-haiku-5-5","stop_reason":"end_turn",` +
			`"content":[{"type":"text","text":"Checking. "},` +
			`{"type":"server_tool_use","id":"srvtoolu_1","name":"advisor","input":{}},` +
			`{"type":"advisor_tool_result","tool_use_id":"srvtoolu_1","content":` +
			`{"type":"advisor_result","text":"Look at the stops.","stop_reason":"end_turn"}},` +
			`{"type":"text","text":"Done."}],` +
			`"usage":{"input_tokens":120000,"output_tokens":2000,"iterations":[` +
			`{"type":"message","input_tokens":60000,"output_tokens":1000,"model":"claude-haiku-5-5"},` +
			`{"type":"advisor_message","input_tokens":10000,"output_tokens":1000,` +
			`"cache_read_input_tokens":500,"model":"claude-opus-5-5"},` +
			`{"type":"message","input_tokens":60000,"output_tokens":1000,"model":"claude-haiku-5-5"}]}}`))
	}))
	t.Cleanup(server.Close)

	input, output := decimal.RequireFromString("0.10"), decimal.RequireFromString("0.50")
	provider := chatProvider("haiku", server.URL, 10)
	provider.Kind = aiprovider.KindAnthropicMessages
	provider.Model = "claude-haiku-5-5"
	provider.InputCostPerMillion = &input
	provider.OutputCostPerMillion = &output

	usage := &fakeUsage{}
	svc := newTestService(t, provider)
	svc.usage = usage
	key, err := svc.encryption.EncryptString("sk-ant-test")
	require.NoError(t, err)
	provider.APIKey = key

	attribution := turnAttribution()
	req := chatRequest(pulid.Nil)
	req.Attribution = attribution
	req.Advisor = llmports.DefaultAdvisor("")

	result, err := svc.CompleteChat(t.Context(), req)
	require.NoError(t, err)

	assert.Equal(t, "Checking. Done.", result.Text)
	assert.Equal(t, 120000, result.InputTokens, "the summed usage stays the call's token count")
	require.NotNil(t, result.CostUSD)
	assert.True(t, result.CostUSD.Equal(decimal.RequireFromString("0.0731")),
		result.CostUSD.String(),
		"two 60K-prompt iterations at Haiku 5.5's short rates ($0.013), not the summed 120K "+
			"at its long ones, plus the advisor at Opus 5.5's list price ($0.0601)")

	rows := usage.recorded(t, 1)
	require.Len(t, rows, 1)
	row := rows[0]
	require.NotNil(t, row.CostUSD)
	assert.True(t, row.CostUSD.Equal(*result.CostUSD), row.CostUSD.String())
	assert.Equal(t, 120000, row.InputTokens)
	assert.Equal(t, "claude-opus-5-5", row.AdvisorModel)
	assert.Equal(t, 1, row.AdvisorCalls)
	assert.Equal(t, 10000, row.AdvisorInputTokens)
	assert.Equal(t, 1000, row.AdvisorOutputTokens)
	assert.Equal(t, 500, row.AdvisorCacheReadTokens)
	assert.Zero(t, row.AdvisorCacheWriteTokens)
	require.NotNil(t, row.AdvisorCostUSD)
	assert.True(t, row.AdvisorCostUSD.Equal(decimal.RequireFromString("0.0601")),
		row.AdvisorCostUSD.String())

	span := aitracetest.One(t, aitrace.ForAttribution(&attribution).TraceID,
		"chat claude-haiku-5-5")
	for _, attr := range []attribute.KeyValue{
		aitrace.AIAdvisorModel.String("claude-opus-5-5"),
		aitrace.AIAdvisorCalls.Int(1),
		aitrace.AIAdvisorInputTokens.Int(10000),
		aitrace.AIAdvisorOutputTokens.Int(1000),
		aitrace.AIAdvisorCacheReadTokens.Int(500),
		aitrace.AIAdvisorCacheWriteTokens.Int(0),
		aitrace.AIAdvisorCostUSD.Float64(0.0601),
	} {
		assert.Contains(t, span.Attributes, attr)
	}
}
