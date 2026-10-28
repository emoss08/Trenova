package completionrouter

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace/aitracetest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func diagnosedServer(t *testing.T, diagnostics string) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(
			[]byte(`{"id":"msg_01REPLY","model":"claude-opus-5-5","stop_reason":"end_turn",` +
				`"content":[{"type":"text","text":"Answered."}],` +
				`"usage":{"input_tokens":10,"output_tokens":2,"cache_creation_input_tokens":40000},` +
				`"diagnostics":` + diagnostics + `}`),
		)
	}))
	t.Cleanup(server.Close)

	return server
}

/*
When the operator turns cache diagnostics on, what the provider said about why
a turn's prompt cache missed reaches the log and the attempt's span, and the
reply's id comes back so the turn can name it on its next call.
*/
func TestCompleteChat_ReportsWhyThePromptCacheMissed(t *testing.T) {
	t.Parallel()
	aitracetest.Install()

	server := diagnosedServer(t,
		`{"cache_miss_reason":{"type":"messages_changed","cache_missed_input_tokens":38000}}`)
	provider := chatProvider("claude", server.URL, 10)
	provider.Kind = aiprovider.KindAnthropicMessages
	provider.Model = "claude-opus-5-5"
	svc := newTestService(t, provider)
	svc.ai.CacheDiagnostics = true
	core, logs := observer.New(zapcore.DebugLevel)
	svc.logger = zap.New(core)
	key, err := svc.encryption.EncryptString("sk-ant-test")
	require.NoError(t, err)
	provider.APIKey = key
	attribution := turnAttribution()
	req := chatRequest(pulid.Nil)
	req.Attribution = attribution

	result, err := svc.CompleteChat(t.Context(), req)

	require.NoError(t, err)
	assert.Equal(t, "msg_01REPLY", result.ProviderMessageID)
	anchor := aitrace.ForAttribution(&attribution)
	answered := aitracetest.One(t, anchor.TraceID, "chat claude-opus-5-5")
	assert.Contains(t, answered.Attributes, aitrace.AICacheMissReason.String("messages_changed"))
	assert.Contains(t, answered.Attributes, aitrace.AICacheMissedTokens.Int64(38000))

	logged := logs.FilterMessage("the prompt stopped matching the call before it").All()
	require.Len(t, logged, 1)
	assert.Equal(t, zapcore.InfoLevel, logged[0].Level)
	fields := logged[0].ContextMap()
	assert.Equal(t, "messages_changed", fields["reason"])
	assert.Equal(t, int64(38000), fields["missedTokens"])
	assert.Equal(t, int64(40000), fields["cacheWriteTokens"])
	assert.Equal(t, attribution.ThreadID.String(), fields["thread"])
}

// A call with nothing to compare against is not the harness's fault, so it
// is logged for debugging only; a call whose prompt matched says nothing.
func TestCompleteChat_KeepsAnUncomparedCallOutOfTheOperatorsLog(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		diagnostics string
		debug       int
	}{
		{diagnostics: `{"cache_miss_reason":{"type":"previous_message_not_found"}}`, debug: 1},
		{diagnostics: `{"cache_miss_reason":null}`, debug: 1},
		{diagnostics: `null`},
	} {
		server := diagnosedServer(t, tc.diagnostics)
		provider := chatProvider("claude", server.URL, 10)
		provider.Kind = aiprovider.KindAnthropicMessages
		provider.Model = "claude-opus-5-5"
		svc := newTestService(t, provider)
		svc.ai.CacheDiagnostics = true
		core, logs := observer.New(zapcore.DebugLevel)
		svc.logger = zap.New(core)
		key, err := svc.encryption.EncryptString("sk-ant-test")
		require.NoError(t, err)
		provider.APIKey = key

		_, err = svc.CompleteChat(t.Context(), chatRequest(pulid.Nil))

		require.NoError(t, err, tc.diagnostics)
		assert.Empty(t, logs.FilterMessage("the prompt stopped matching the call before it").All(),
			tc.diagnostics)
		assert.Len(
			t,
			logs.FilterMessage("the prompt was not compared with the call before it").All(),
			tc.debug,
			tc.diagnostics,
		)
	}
}

// toClaudeAPI sends requests for api.anthropic.com to a test server.
type toClaudeAPI struct{ target *url.URL }

func (r toClaudeAPI) RoundTrip(req *http.Request) (*http.Response, error) {
	routed := req.Clone(req.Context())
	if req.URL.Host == "api.anthropic.com" {
		routed.URL.Scheme = r.target.Scheme
		routed.URL.Host = r.target.Host
		routed.Host = r.target.Host
	}

	return http.DefaultTransport.RoundTrip(routed)
}

/*
A provider calling the Claude API itself is asked, on each turn's call, to
compare it with the reply the conversation last had from it, which the turn
hands over on that reply's message. With the setting off nothing is asked.
*/
func TestCompleteChat_AsksTheClaudeAPIAboutTheCallBeforeWhenTurnedOn(t *testing.T) {
	t.Parallel()

	for _, enabled := range []bool{true, false} {
		var (
			mu     sync.Mutex
			bodies []map[string]any
		)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body := map[string]any{}
			_ = sonic.ConfigDefault.NewDecoder(r.Body).Decode(&body)
			mu.Lock()
			bodies = append(bodies, body)
			mu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(
				[]byte(`{"id":"msg_01REPLY","model":"claude-opus-5-5","stop_reason":"end_turn",` +
					`"content":[{"type":"text","text":"Answered."}],"usage":{"input_tokens":1,"output_tokens":1}}`),
			)
		}))
		t.Cleanup(server.Close)
		target, err := url.Parse(server.URL)
		require.NoError(t, err)

		provider := chatProvider("claude", "", 10)
		provider.Kind = aiprovider.KindAnthropicMessages
		provider.Model = "claude-opus-5-5"
		svc := newTestService(t, provider)
		svc.ai.CacheDiagnostics = enabled
		client := &http.Client{Timeout: 5 * time.Second, Transport: toClaudeAPI{target: target}}
		svc.clients = map[bool]*http.Client{true: client, false: client}
		svc.streamClients = map[bool]*http.Client{true: client, false: client}
		key, err := svc.encryption.EncryptString("sk-ant-test")
		require.NoError(t, err)
		provider.APIKey = key
		req := chatRequest(pulid.Nil)
		req.Messages = []llmports.Message{
			{Role: llmports.RoleUser, Content: "Where is S1?"},
			{
				Role:          llmports.RoleAssistant,
				Content:       "In Dallas.",
				ReplyID:       "msg_01BEFORE",
				ReplyProvider: provider.ID,
			},
			{Role: llmports.RoleUser, Content: "And S2?"},
		}

		_, err = svc.CompleteChat(t.Context(), req)
		require.NoError(t, err, "enabled=%v", enabled)

		mu.Lock()
		require.Len(t, bodies, 1, "enabled=%v", enabled)
		diagnostics, asked := bodies[0]["diagnostics"].(map[string]any)
		mu.Unlock()
		assert.Equal(t, enabled, asked, "enabled=%v", enabled)
		if enabled {
			assert.Equal(t, "msg_01BEFORE", diagnostics["previous_message_id"])
		}
	}
}
