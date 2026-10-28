package completionrouter

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiproviderports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const advisedReply = `{"id":"msg_1","model":"claude-haiku-5-5","stop_reason":"end_turn",` +
	`"content":[{"type":"text","text":"done"}],"usage":{"input_tokens":10,"output_tokens":10}}`

type advisorCapture struct {
	mu      sync.Mutex
	advisor []string
	refuse  bool
}

func (c *advisorCapture) handler(w http.ResponseWriter, r *http.Request) {
	decoded := map[string]any{}
	_ = sonic.ConfigDefault.NewDecoder(r.Body).Decode(&decoded)
	model := ""
	if tools, ok := decoded["tools"].([]any); ok {
		for _, tool := range tools {
			if entry, isMap := tool.(map[string]any); isMap && entry["name"] == "advisor" {
				model, _ = entry["model"].(string)
			}
		}
	}
	c.mu.Lock()
	c.advisor = append(c.advisor, model)
	refuse := c.refuse && model != ""
	c.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	if refuse {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"type":"error","error":{"type":"invalid_request_error",` +
			`"message":"advisor model claude-haiku-5-5 must be at least as capable as the executor"}}`))
		return
	}
	_, _ = w.Write([]byte(advisedReply))
}

func (c *advisorCapture) declared() []string {
	c.mu.Lock()
	defer c.mu.Unlock()

	return append([]string(nil), c.advisor...)
}

type recordedRefusals struct {
	mu   sync.Mutex
	reqs []aiproviderports.RecordAdvisorRefusalRequest
}

func (r *recordedRefusals) RecordAdvisorRefusal(
	_ context.Context,
	req *aiproviderports.RecordAdvisorRefusalRequest,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.reqs = append(r.reqs, *req)

	return nil
}

func advisedProvider(t *testing.T, capture *advisorCapture) (*Service, *aiprovider.Provider) {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(capture.handler))
	t.Cleanup(server.Close)

	provider := chatProvider("haiku", server.URL, 10)
	provider.Kind = aiprovider.KindAnthropicMessages
	provider.Model = "claude-haiku-5-5"
	provider.AdvisorEnabled = true

	svc := newTestService(t, provider)
	key, err := svc.encryption.EncryptString("sk-ant-test")
	require.NoError(t, err)
	provider.APIKey = key

	return svc, provider
}

func TestCompleteChat_ProviderSettingDeclaresTheAdvisor(t *testing.T) {
	t.Parallel()

	capture := &advisorCapture{}
	svc, provider := advisedProvider(t, capture)

	_, err := svc.CompleteChat(t.Context(), chatRequest(pulid.Nil))
	require.NoError(t, err)
	assert.Equal(t, []string{aiprovider.DefaultAdvisorModel}, capture.declared(),
		"every chat call routed to the provider declares its advisor")

	provider.AdvisorModel = "claude-fable-5-1"
	_, err = svc.CompleteChat(t.Context(), chatRequest(pulid.Nil))
	require.NoError(t, err)
	assert.Equal(t, "claude-fable-5-1", capture.declared()[1])

	provider.AdvisorEnabled = false
	_, err = svc.CompleteChat(t.Context(), chatRequest(pulid.Nil))
	require.NoError(t, err)
	assert.Empty(t, capture.declared()[2], "a provider with the advisor off declares none")
}

func TestCompleteChat_TheBenchCellOverridesTheProviderSetting(t *testing.T) {
	t.Parallel()

	capture := &advisorCapture{}
	svc, provider := advisedProvider(t, capture)
	provider.AdvisorEnabled = false

	req := chatRequest(pulid.Nil)
	req.Advisor = llmports.DefaultAdvisor("claude-fable-5-1")
	_, err := svc.CompleteChat(t.Context(), req)
	require.NoError(t, err)
	assert.Equal(t, []string{"claude-fable-5-1"}, capture.declared())
}

func TestCompleteChat_RecordsTheAPIsRefusalAndAnswersWithoutTheAdvisor(t *testing.T) {
	t.Parallel()

	capture := &advisorCapture{refuse: true}
	svc, provider := advisedProvider(t, capture)
	provider.AdvisorModel = "claude-haiku-5-5"
	refusals := &recordedRefusals{}
	svc.advisors = refusals

	result, err := svc.CompleteChat(t.Context(), chatRequest(pulid.Nil))
	require.NoError(t, err, "the turn is answered without the advisor, not failed")
	assert.Equal(t, "done", result.Text)
	assert.Equal(t, []string{"claude-haiku-5-5", ""}, capture.declared())

	require.Len(t, refusals.reqs, 1)
	refusal := refusals.reqs[0]
	assert.Equal(t, provider.ID, refusal.ID)
	assert.Equal(t, "claude-haiku-5-5", refusal.Refusal.Advisor)
	assert.Equal(t, provider.Model, refusal.Refusal.Model)
	assert.Contains(t, refusal.Refusal.Message, "at least as capable")
	assert.NotZero(t, refusal.Refusal.RefusedAt)

	provider.AdvisorRefusal = &refusal.Refusal
	_, err = svc.CompleteChat(t.Context(), chatRequest(pulid.Nil))
	require.NoError(t, err)
	assert.Empty(t, capture.declared()[2],
		"once refused, the advisor is left off until the advisor or the model changes")
}

func TestCompleteChat_ARefusedBenchAdvisorFailsLoudly(t *testing.T) {
	t.Parallel()

	capture := &advisorCapture{refuse: true}
	svc, provider := advisedProvider(t, capture)
	provider.AdvisorEnabled = false
	refusals := &recordedRefusals{}
	svc.advisors = refusals

	req := chatRequest(pulid.Nil)
	req.Advisor = llmports.DefaultAdvisor("claude-haiku-5-5")
	_, err := svc.CompleteChat(t.Context(), req)
	require.Error(t, err, "a cell that asked for an advisor is not quietly run without one")
	assert.Empty(t, refusals.reqs)
}
