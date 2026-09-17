package completionrouter

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/encryptionservice"
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeRepo struct {
	repositories.AIProviderRepository

	providers []*aiprovider.Provider
	err       error
}

func (f *fakeRepo) ListForTask(
	_ context.Context,
	_ repositories.ListAIProvidersForTaskRequest,
) ([]*aiprovider.Provider, error) {
	return f.providers, f.err
}

func (f *fakeRepo) List(
	_ context.Context,
	_ *repositories.ListAIProviderRequest,
) (*pagination.ListResult[*aiprovider.Provider], error) {
	return nil, nil
}

func (f *fakeRepo) GetByID(
	_ context.Context,
	_ repositories.GetAIProviderByIDRequest,
) (*aiprovider.Provider, error) {
	return nil, nil
}

func (f *fakeRepo) Create(
	_ context.Context,
	e *aiprovider.Provider,
) (*aiprovider.Provider, error) {
	return e, nil
}

func (f *fakeRepo) Update(
	_ context.Context,
	e *aiprovider.Provider,
) (*aiprovider.Provider, error) {
	return e, nil
}

func (f *fakeRepo) Delete(_ context.Context, _ repositories.DeleteAIProviderRequest) error {
	return nil
}

func newTestService(t *testing.T, providers ...*aiprovider.Provider) *Service {
	t.Helper()

	return &Service{
		logger: zap.NewNop(),
		cfg: &config.DocumentIntelligenceConfig{
			EnableAI:     true,
			AIMaxRetries: 1,
		},
		repo:       &fakeRepo{providers: providers},
		encryption: encryptionservice.NewWithKeyManager(encryptionservice.NewLocalKeyManager("k")),
		adapters:   modeladapter.NewRegistry(),
		clients:    make(map[bool]*http.Client, 2),
	}
}

// chatServer stands in for any OpenAI-compatible endpoint. It records how many
// times it was called so fallback behaviour can be asserted.
func chatServer(t *testing.T, status int, content string) (*httptest.Server, *atomic.Int32) {
	t.Helper()

	var calls atomic.Int32
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			calls.Add(1)
			w.Header().Set("Content-Type", "application/json")
			if status != http.StatusOK {
				w.WriteHeader(status)
				_, _ = w.Write([]byte(`{"error":{"message":"upstream failure"}}`))
				return
			}
			payload := map[string]any{
				"model": "test-model",
				"choices": []map[string]any{
					{"finish_reason": "stop", "message": map[string]any{
						"role": "assistant", "content": content,
					}},
				},
				"usage": map[string]any{"prompt_tokens": 11, "completion_tokens": 7},
			}
			encoded, _ := sonic.Marshal(payload)
			_, _ = w.Write(encoded)
		}),
	)
	t.Cleanup(server.Close)

	return server, &calls
}

func openAIChatProvider(name, baseURL string, priority int) *aiprovider.Provider {
	return &aiprovider.Provider{
		ID:                   pulid.MustNew("aiprv_"),
		Name:                 name,
		Kind:                 aiprovider.KindOpenAIChat,
		BaseURL:              baseURL,
		Model:                "qwen3:32b",
		AllowPrivateNetwork:  true,
		StructuredOutputMode: aiprovider.StructuredOutputPrompted,
		MaxTokens:            2048,
		Priority:             priority,
		Tasks:                []aiprovider.Task{aiprovider.TaskGeneral},
		Enabled:              true,
	}
}

func generalRequest() *serviceports.StructuredCompletionRequest {
	return &serviceports.StructuredCompletionRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: pulid.MustNew("org_"),
			BuID:  pulid.MustNew("bu_"),
		},
		Task:   aiprovider.TaskGeneral,
		System: "You are a test.",
		Context: serviceports.DelimitedContext{
			Sections: []serviceports.ContextSection{
				{Title: "Input", Trusted: true, Content: "hello"},
			},
		},
	}
}

func TestCompleteStructured_RoutesToConfiguredProvider(t *testing.T) {
	t.Parallel()

	server, calls := chatServer(t, http.StatusOK, `{"answer":"ok"}`)
	svc := newTestService(t, openAIChatProvider("local", server.URL, 10))

	result, err := svc.CompleteStructured(t.Context(), generalRequest())
	require.NoError(t, err)

	assert.Equal(t, `{"answer":"ok"}`, result.Text)
	assert.Equal(t, "test-model", result.ModelIdentifier)
	assert.Equal(t, 11, result.InputTokens)
	assert.Equal(t, 7, result.OutputTokens)
	assert.Equal(t, aiprovider.KindOpenAIChat, result.ProviderKind)
	assert.Equal(t, int32(1), calls.Load())
}

func TestCompleteStructured_NoProviderConfigured(t *testing.T) {
	t.Parallel()

	svc := newTestService(t)

	_, err := svc.CompleteStructured(t.Context(), generalRequest())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "No AI provider is configured")
}

func TestCompleteStructured_SkipsProviderNotServingTask(t *testing.T) {
	t.Parallel()

	server, calls := chatServer(t, http.StatusOK, `{"answer":"ok"}`)
	provider := openAIChatProvider("extraction-only", server.URL, 10)
	provider.Tasks = []aiprovider.Task{aiprovider.TaskDocumentExtraction}

	svc := newTestService(t, provider)

	_, err := svc.CompleteStructured(t.Context(), generalRequest())
	require.Error(t, err)
	assert.Equal(t, int32(0), calls.Load(), "a provider not serving the task must not be called")
}

func TestCompleteStructured_FallsThroughToNextProvider(t *testing.T) {
	t.Parallel()

	failing, failingCalls := chatServer(t, http.StatusInternalServerError, "")
	healthy, healthyCalls := chatServer(t, http.StatusOK, `{"answer":"second"}`)

	svc := newTestService(t,
		openAIChatProvider("primary", failing.URL, 10),
		openAIChatProvider("secondary", healthy.URL, 20),
	)

	result, err := svc.CompleteStructured(t.Context(), generalRequest())
	require.NoError(t, err)

	assert.Equal(t, `{"answer":"second"}`, result.Text)
	assert.Positive(t, failingCalls.Load())
	assert.Equal(t, int32(1), healthyCalls.Load())
}

func TestCompleteStructured_FailsWhenEveryProviderFails(t *testing.T) {
	t.Parallel()

	first, _ := chatServer(t, http.StatusInternalServerError, "")
	second, _ := chatServer(t, http.StatusBadGateway, "")

	svc := newTestService(t,
		openAIChatProvider("primary", first.URL, 10),
		openAIChatProvider("secondary", second.URL, 20),
	)

	_, err := svc.CompleteStructured(t.Context(), generalRequest())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "every configured provider")
}

func TestCompleteStructured_EmptyContentIsAFailure(t *testing.T) {
	t.Parallel()

	empty, _ := chatServer(t, http.StatusOK, "")
	healthy, healthyCalls := chatServer(t, http.StatusOK, `{"answer":"second"}`)

	svc := newTestService(t,
		openAIChatProvider("primary", empty.URL, 10),
		openAIChatProvider("secondary", healthy.URL, 20),
	)

	result, err := svc.CompleteStructured(t.Context(), generalRequest())
	require.NoError(t, err)
	assert.Equal(t, `{"answer":"second"}`, result.Text)
	assert.Equal(t, int32(1), healthyCalls.Load())
}

func TestCompleteStructured_DisabledGlobally(t *testing.T) {
	t.Parallel()

	svc := newTestService(t)
	svc.cfg = &config.DocumentIntelligenceConfig{EnableAI: false}

	_, err := svc.CompleteStructured(t.Context(), generalRequest())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "disabled")
}

func TestCompleteStructured_FallsThroughOnUnparseableOutput(t *testing.T) {
	t.Parallel()

	garbage, _ := chatServer(t, http.StatusOK, "I cannot help with that.")
	good, goodCalls := chatServer(t, http.StatusOK,
		`{"proposals":[],"exceptions":[{"category":"Other","severity":"Low",`+
			`"attemptSummary":"none","blastRadius":1,`+
			`"evidence":[{"type":"doc","id":"d1"}]}]}`)

	weak := openAIChatProvider("weak-local", garbage.URL, 10)
	weak.Tasks = []aiprovider.Task{aiprovider.TaskBillingDiagnosis}
	weak.Trusted = true

	strong := openAIChatProvider("hosted", good.URL, 20)
	strong.Tasks = []aiprovider.Task{aiprovider.TaskBillingDiagnosis}
	strong.Trusted = true

	svc := newTestService(t, weak, strong)

	result, err := svc.CompleteStructured(t.Context(), &serviceports.StructuredCompletionRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: pulid.MustNew("org_"),
			BuID:  pulid.MustNew("bu_"),
		},
		Task:   aiprovider.TaskBillingDiagnosis,
		System: "Diagnose the blocker.",
		Context: serviceports.DelimitedContext{
			Sections: []serviceports.ContextSection{
				{Title: "Item", Trusted: true, Content: "blocked"},
			},
		},
		OutputSchema: exceptionsSchema(),
		SchemaName:   "exceptions",
	})
	require.NoError(t, err)

	assert.Contains(t, result.Text, `"category":"Other"`)
	assert.Equal(t, int32(1), goodCalls.Load())
}

func exceptionsSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"proposals":  map[string]any{"type": "array"},
			"exceptions": map[string]any{"type": "array"},
		},
		"required":             []string{"proposals", "exceptions"},
		"additionalProperties": false,
	}
}

func TestCompleteStructured_RefusesUntrustedProviderForLedgerTask(t *testing.T) {
	t.Parallel()

	server, calls := chatServer(t, http.StatusOK, `{"proposals":[],"exceptions":[]}`)
	provider := openAIChatProvider("untrusted-local", server.URL, 10)
	provider.Tasks = []aiprovider.Task{aiprovider.TaskBillingDiagnosis}
	provider.Trusted = false

	svc := newTestService(t, provider)

	_, err := svc.CompleteStructured(t.Context(), &serviceports.StructuredCompletionRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: pulid.MustNew("org_"),
			BuID:  pulid.MustNew("bu_"),
		},
		Task:   aiprovider.TaskBillingDiagnosis,
		System: "Diagnose the blocker.",
	})
	require.Error(t, err)
	assert.Equal(t, int32(0), calls.Load(), "an untrusted provider must not be called")
}
