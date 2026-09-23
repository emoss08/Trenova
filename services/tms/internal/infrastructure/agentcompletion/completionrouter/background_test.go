package completionrouter

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/encryptionservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// responsesServer stands in for an OpenAI Responses endpoint. POST starts a run
// and GET reports on it, so one server serves both halves of the background
// contract and records what it was asked.
type responsesServer struct {
	URL        string
	Submits    *atomic.Int32
	Polls      *atomic.Int32
	Background *atomic.Bool
	PolledPath *atomic.Value
}

func newResponsesServer(t *testing.T, poll map[string]any) *responsesServer {
	t.Helper()

	rec := &responsesServer{
		Submits:    &atomic.Int32{},
		Polls:      &atomic.Int32{},
		Background: &atomic.Bool{},
		PolledPath: &atomic.Value{},
	}
	rec.PolledPath.Store("")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method == http.MethodGet {
			rec.Polls.Add(1)
			rec.PolledPath.Store(r.URL.Path)
			encoded, _ := sonic.Marshal(poll)
			_, _ = w.Write(encoded)

			return
		}

		rec.Submits.Add(1)
		var body map[string]any
		_ = sonic.ConfigDefault.NewDecoder(r.Body).Decode(&body)
		background, _ := body["background"].(bool)
		rec.Background.Store(background)

		encoded, _ := sonic.Marshal(map[string]any{
			"id": "resp_abc123", "status": "queued", "model": "gpt-test",
		})
		_, _ = w.Write(encoded)
	}))
	t.Cleanup(server.Close)

	rec.URL = server.URL

	return rec
}

func responsesProvider(t *testing.T, name, baseURL string, priority int) *aiprovider.Provider {
	t.Helper()

	// This protocol is hosted-only, so the provider must carry a credential the
	// router can decrypt; an unencrypted fixture would fail before any request.
	encrypted, err := encryptionservice.
		NewWithKeyManager(encryptionservice.NewLocalKeyManager("k")).
		EncryptString("sk-test")
	require.NoError(t, err)

	return &aiprovider.Provider{
		APIKey:               encrypted,
		ID:                   pulid.MustNew("aiprv_"),
		Name:                 name,
		Kind:                 aiprovider.KindOpenAIResponses,
		BaseURL:              baseURL,
		Model:                "gpt-test",
		AllowPrivateNetwork:  true,
		StructuredOutputMode: aiprovider.StructuredOutputJSONSchema,
		MaxTokens:            2048,
		Priority:             priority,
		Tasks:                []aiprovider.Task{aiprovider.TaskDocumentExtraction},
		Enabled:              true,
	}
}

func extractionRequest() *serviceports.StructuredCompletionRequest {
	return &serviceports.StructuredCompletionRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: pulid.MustNew("org_"),
			BuID:  pulid.MustNew("bu_"),
		},
		Task:   aiprovider.TaskDocumentExtraction,
		System: "Extract the rate confirmation.",
		Context: serviceports.DelimitedContext{
			Sections: []serviceports.ContextSection{
				{Title: "Document", Trusted: false, Content: "PRO 12345"},
			},
		},
		OutputSchema: map[string]any{
			"type":       "object",
			"properties": map[string]any{"documentKind": map[string]any{"type": "string"}},
		},
		SchemaName: "extract",
	}
}

func TestSubmitBackground_DefersWhenTheProtocolCan(t *testing.T) {
	t.Parallel()

	server := newResponsesServer(t, nil)
	svc := newTestService(t, responsesProvider(t, "hosted", server.URL, 10))

	submission, err := svc.SubmitBackground(t.Context(), extractionRequest())
	require.NoError(t, err)

	assert.Equal(t, "resp_abc123", submission.Handle)
	assert.Equal(t, "gpt-test", submission.ModelIdentifier)
	assert.Equal(t, "queued", submission.RawStatus)
	assert.Nil(t, submission.Result, "a deferred call has nothing to return yet")
	assert.True(t, server.Background.Load(), "the run must be submitted in background mode")
}

func TestSubmitBackground_RunsInlineWhenNoProtocolCanDefer(t *testing.T) {
	t.Parallel()

	inline, calls := chatServer(t, http.StatusOK, `{"documentKind":"RateConfirmation"}`)
	provider := openAIChatProvider("self-hosted", inline.URL, 10)
	provider.Tasks = []aiprovider.Task{aiprovider.TaskDocumentExtraction}

	svc := newTestService(t, provider)

	submission, err := svc.SubmitBackground(t.Context(), extractionRequest())
	require.NoError(t, err)

	assert.Empty(t, submission.Handle, "there is nothing to poll for an inline run")
	require.NotNil(t, submission.Result)
	assert.Equal(t, `{"documentKind":"RateConfirmation"}`, submission.Result.Text)
	assert.Equal(t, provider.ID, submission.ProviderID)
	assert.Equal(t, int32(1), calls.Load())
}

func TestSubmitBackground_FallsBackToInlineWhenTheSubmitFails(t *testing.T) {
	t.Parallel()

	broken := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"error":{"message":"background unavailable"}}`))
	}))
	t.Cleanup(broken.Close)

	inline, inlineCalls := chatServer(t, http.StatusOK, `{"documentKind":"BillOfLading"}`)
	fallback := openAIChatProvider("self-hosted", inline.URL, 20)
	fallback.Tasks = []aiprovider.Task{aiprovider.TaskDocumentExtraction}

	svc := newTestService(t, responsesProvider(t, "hosted", broken.URL, 10), fallback)

	submission, err := svc.SubmitBackground(t.Context(), extractionRequest())
	require.NoError(t, err)

	assert.Empty(t, submission.Handle)
	require.NotNil(t, submission.Result)
	assert.Equal(t, `{"documentKind":"BillOfLading"}`, submission.Result.Text)
	assert.Positive(t, inlineCalls.Load())
}

func TestSubmitBackground_NoProviderConfigured(t *testing.T) {
	t.Parallel()

	svc := newTestService(t)

	_, err := svc.SubmitBackground(t.Context(), extractionRequest())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "No AI provider is configured")
}

func pollService(
	t *testing.T,
	poll map[string]any,
) (*Service, *aiprovider.Provider, *responsesServer) {
	t.Helper()

	server := newResponsesServer(t, poll)
	provider := responsesProvider(t, "hosted", server.URL, 10)
	svc := newTestService(t, provider)
	svc.repo = &fakeRepo{providers: []*aiprovider.Provider{provider}, byID: provider}

	return svc, provider, server
}

func pollRequest(provider *aiprovider.Provider) *serviceports.BackgroundPollRequest {
	return &serviceports.BackgroundPollRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: pulid.MustNew("org_"),
			BuID:  pulid.MustNew("bu_"),
		},
		ProviderID: provider.ID,
		Handle:     "resp_abc123",
	}
}

func TestPollBackground_Pending(t *testing.T) {
	t.Parallel()

	svc, provider, server := pollService(t, map[string]any{
		"id": "resp_abc123", "status": "in_progress", "model": "gpt-test",
	})

	outcome, err := svc.PollBackground(t.Context(), pollRequest(provider))
	require.NoError(t, err)

	assert.Equal(t, serviceports.BackgroundPending, outcome.State)
	assert.Equal(t, "in_progress", outcome.RawStatus)
	assert.Nil(t, outcome.Result)
	assert.True(t,
		strings.HasSuffix(server.PolledPath.Load().(string), "/v1/responses/resp_abc123"),
		"the handle must address the run it came from",
	)
}

func TestPollBackground_Completed(t *testing.T) {
	t.Parallel()

	svc, provider, _ := pollService(t, map[string]any{
		"id": "resp_abc123", "status": "completed", "model": "gpt-test",
		"output": []map[string]any{
			{"type": "message", "role": "assistant", "content": []map[string]any{
				{"type": "output_text", "text": `{"documentKind":"RateConfirmation"}`},
			}},
		},
		"usage": map[string]any{"input_tokens": 120, "output_tokens": 44},
	})

	outcome, err := svc.PollBackground(t.Context(), pollRequest(provider))
	require.NoError(t, err)

	assert.Equal(t, serviceports.BackgroundCompleted, outcome.State)
	require.NotNil(t, outcome.Result)
	assert.Equal(t, `{"documentKind":"RateConfirmation"}`, outcome.Result.Text)
	assert.Equal(t, 120, outcome.Result.InputTokens)
	assert.Equal(t, 44, outcome.Result.OutputTokens)
	assert.Equal(t, provider.ID, outcome.Result.ProviderID)
}

func TestPollBackground_TerminalFailureCarriesTheReason(t *testing.T) {
	t.Parallel()

	svc, provider, _ := pollService(t, map[string]any{
		"id": "resp_abc123", "status": "incomplete", "model": "gpt-test",
		"incomplete_details": map[string]any{"reason": "max_output_tokens"},
	})

	outcome, err := svc.PollBackground(t.Context(), pollRequest(provider))
	require.NoError(t, err)

	assert.Equal(t, serviceports.BackgroundFailed, outcome.State)
	assert.Equal(t, "max_output_tokens", outcome.FailureCode)
	assert.NotEmpty(t, outcome.FailureMessage)
}

func TestPollBackground_CompletedWithoutOutputIsAFailure(t *testing.T) {
	t.Parallel()

	svc, provider, _ := pollService(t, map[string]any{
		"id": "resp_abc123", "status": "completed", "model": "gpt-test",
	})

	outcome, err := svc.PollBackground(t.Context(), pollRequest(provider))
	require.NoError(t, err)

	assert.Equal(t, serviceports.BackgroundFailed, outcome.State)
	assert.Equal(t, "empty_output", outcome.FailureCode)
}

func TestPollBackground_RejectsAProtocolThatCannotDefer(t *testing.T) {
	t.Parallel()

	inline, _ := chatServer(t, http.StatusOK, `{}`)
	provider := openAIChatProvider("self-hosted", inline.URL, 10)
	svc := newTestService(t, provider)
	svc.repo = &fakeRepo{providers: []*aiprovider.Provider{provider}, byID: provider}

	_, err := svc.PollBackground(t.Context(), pollRequest(provider))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot report on background calls")
}

func TestPollBackground_RequiresAHandle(t *testing.T) {
	t.Parallel()

	svc, provider, _ := pollService(t, nil)

	req := pollRequest(provider)
	req.Handle = "  "

	_, err := svc.PollBackground(t.Context(), req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "background handle is required")
}
