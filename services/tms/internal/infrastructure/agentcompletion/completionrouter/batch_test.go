package completionrouter

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiusage"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiusageports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	routerBatchCreated = `{"id":"msgbatch_7","type":"message_batch","processing_status":"in_progress",` +
		`"request_counts":{"processing":2,"succeeded":0,"errored":0,"canceled":0,"expired":0},` +
		`"created_at":"2026-10-27T03:00:00Z","expires_at":"2026-10-28T03:00:00Z"}`
	routerBatchEnded = `{"id":"msgbatch_7","type":"message_batch","processing_status":"ended",` +
		`"request_counts":{"processing":0,"succeeded":2,"errored":1,"canceled":0,"expired":1},` +
		`"created_at":"2026-10-27T03:00:00Z","expires_at":"2026-10-28T03:00:00Z"}`
	routerBatchResults = `{"custom_id":"aeval_1","result":{"type":"succeeded","message":` +
		`{"id":"msg_1","type":"message","role":"assistant","model":"claude-sonnet-5-5",` +
		`"stop_reason":"end_turn","content":[{"type":"text","text":"{\"score\":1,\"rationale\":\"Right.\"}"}],` +
		`"usage":{"input_tokens":1000000,"output_tokens":100000}}}}` + "\n" +
		`{"custom_id":"aeval_2","result":{"type":"errored","error":{"type":"error",` +
		`"error":{"type":"api_error","message":"internal error"}}}}` + "\n" +
		`{"custom_id":"aeval_3","result":{"type":"expired"}}` + "\n" +
		`{"custom_id":"aeval_4","result":{"type":"succeeded","message":` +
		`{"id":"msg_4","type":"message","role":"assistant","model":"claude-sonnet-5-5",` +
		`"stop_reason":"end_turn","content":[{"type":"text","text":"I think it is fine."}],` +
		`"usage":{"input_tokens":900,"output_tokens":12}}}}` + "\n"
)

type claudeRoute struct{ target *url.URL }

func (c claudeRoute) RoundTrip(req *http.Request) (*http.Response, error) {
	routed := req.Clone(req.Context())
	routed.URL.Scheme = c.target.Scheme
	routed.URL.Host = c.target.Host
	routed.Host = c.target.Host

	return http.DefaultTransport.RoundTrip(routed)
}

type batchAPI struct {
	mu     sync.Mutex
	posted []map[string]any
	paths  []string
}

func (b *batchAPI) requests() []map[string]any {
	b.mu.Lock()
	defer b.mu.Unlock()

	return append([]map[string]any(nil), b.posted...)
}

func newBatchService(
	t *testing.T,
	providers ...*aiprovider.Provider,
) (*Service, *batchAPI, *fakeUsage) {
	t.Helper()

	api := &batchAPI{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		api.mu.Lock()
		api.paths = append(api.paths, r.Method+" "+r.URL.Path)
		if r.Method == http.MethodPost {
			body := map[string]any{}
			_ = sonic.ConfigDefault.NewDecoder(r.Body).Decode(&body)
			api.posted = append(api.posted, body)
		}
		api.mu.Unlock()

		switch {
		case r.Method == http.MethodPost:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(routerBatchCreated))
		case strings.HasSuffix(r.URL.Path, "/results"):
			w.Header().Set("Content-Type", "application/x-jsonl")
			_, _ = w.Write([]byte(routerBatchResults))
		default:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(routerBatchEnded))
		}
	}))
	t.Cleanup(server.Close)

	target, err := url.Parse(server.URL)
	require.NoError(t, err)

	svc := newTestService(t, providers...)
	for _, provider := range providers {
		key, keyErr := svc.encryption.EncryptString("sk-ant-test")
		require.NoError(t, keyErr)
		provider.APIKey = key
	}
	svc.clients[false] = &http.Client{
		Timeout:   5 * time.Second,
		Transport: claudeRoute{target: target},
	}
	usage := &fakeUsage{}
	svc.usage = usage
	if len(providers) > 0 {
		svc.repo.(*fakeRepo).byID = providers[0]
	}

	return svc, api, usage
}

func claudeJudge() *aiprovider.Provider {
	return priced(&aiprovider.Provider{
		ID:                   pulid.MustNew("aiprv_"),
		Name:                 "Claude",
		Kind:                 aiprovider.KindAnthropicMessages,
		BaseURL:              "https://api.anthropic.com",
		Model:                "claude-sonnet-5-5",
		StructuredOutputMode: aiprovider.StructuredOutputJSONSchema,
		MaxTokens:            2048,
		Tasks:                []aiprovider.Task{aiprovider.TaskEvaluationJudge},
		Enabled:              true,
	}, "2", "10")
}

func judgeTenant() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
}

func judgeItem(tenant pagination.TenantInfo, id, question string) llmports.BatchCompletionItem {
	return llmports.BatchCompletionItem{
		CustomID: id,
		Request: llmports.StructuredCompletionRequest{
			TenantInfo: tenant,
			Task:       aiprovider.TaskEvaluationJudge,
			System:     "You grade one answer.",
			Context: llmports.DelimitedContext{Sections: []llmports.ContextSection{
				{Title: "Answer to grade", Content: question},
			}},
			OutputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{"score": map[string]any{"type": "number"}},
			},
			SchemaName: "evaluation_judgement",
			MaxTokens:  600,
		},
	}
}

func TestSubmitBatch_SendsEveryRequestToTheClaudeAPIInOneBatch(t *testing.T) {
	t.Parallel()

	provider := claudeJudge()
	svc, api, _ := newBatchService(t, provider)
	tenant := judgeTenant()

	handle, err := svc.SubmitBatch(t.Context(), &llmports.BatchCompletionRequest{
		TenantInfo: tenant,
		Task:       aiprovider.TaskEvaluationJudge,
		Items: []llmports.BatchCompletionItem{
			judgeItem(tenant, "aeval_1", "It was sent on Tuesday."),
			judgeItem(tenant, "aeval_2", "Load 42 is late."),
		},
	})
	require.NoError(t, err)
	assert.Equal(t, provider.ID, handle.ProviderID)
	assert.Equal(t, "msgbatch_7", handle.BatchID)
	assert.Equal(t, []string{"aeval_1", "aeval_2"}, handle.CustomIDs)
	assert.True(
		t,
		handle.EstimatedCostUSD.IsPositive(),
		"a priced provider's batch has an estimate",
	)
	assert.Positive(t, handle.SubmittedAt)

	posted := api.requests()
	require.Len(t, posted, 1, "one request carries the whole batch")
	items, ok := posted[0]["requests"].([]any)
	require.True(t, ok)
	assert.Len(t, items, 2)
}

func TestSubmitBatch_TakesOnlyWhatTheSpendLimitAffords(t *testing.T) {
	t.Parallel()

	provider := claudeJudge()
	svc, api, _ := newBatchService(t, provider)
	tenant := judgeTenant()
	first := judgeItem(tenant, "aeval_1", "It was sent on Tuesday.")
	second := judgeItem(tenant, "aeval_2", "Load 42 is late.")

	one := batchEstimate(provider, aiprovider.TaskEvaluationJudge, structuredRun(&first.Request))
	require.NotNil(t, one)
	full := provider.CostForTask(aiprovider.TaskEvaluationJudge, aiprovider.TokenUsage{
		Input:  1,
		Output: 600,
	})
	require.NotNil(t, full)
	assert.True(t, one.LessThan(*full), "the estimate is at the batch rate")

	handle, err := svc.SubmitBatch(t.Context(), &llmports.BatchCompletionRequest{
		TenantInfo:    tenant,
		Task:          aiprovider.TaskEvaluationJudge,
		Items:         []llmports.BatchCompletionItem{first, second},
		SpendLimitUSD: one,
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"aeval_1"}, handle.CustomIDs)
	assert.True(t, handle.EstimatedCostUSD.Equal(*one))

	nothing := decimal.Zero
	_, err = svc.SubmitBatch(t.Context(), &llmports.BatchCompletionRequest{
		TenantInfo:    tenant,
		Task:          aiprovider.TaskEvaluationJudge,
		Items:         []llmports.BatchCompletionItem{first},
		SpendLimitUSD: &nothing,
	})
	require.ErrorIs(t, err, llmports.ErrBatchOverBudget)
	assert.Len(t, api.requests(), 1, "a batch nothing fits is never sent")
}

func TestSubmitBatch_RefusesAProviderWithoutABatchAPI(t *testing.T) {
	t.Parallel()

	chat := openAIChatProvider("local", "http://127.0.0.1:1", 1)
	chat.Tasks = []aiprovider.Task{aiprovider.TaskEvaluationJudge}
	svc, api, _ := newBatchService(t, chat)
	tenant := judgeTenant()

	_, err := svc.SubmitBatch(t.Context(), &llmports.BatchCompletionRequest{
		TenantInfo: tenant,
		Task:       aiprovider.TaskEvaluationJudge,
		Items:      []llmports.BatchCompletionItem{judgeItem(tenant, "aeval_1", "Yes.")},
	})
	require.ErrorIs(t, err, llmports.ErrBatchUnsupported)
	assert.Empty(t, api.requests())
}

func TestSubmitBatch_RefusesARepeatedID(t *testing.T) {
	t.Parallel()

	svc, api, _ := newBatchService(t, claudeJudge())
	tenant := judgeTenant()

	_, err := svc.SubmitBatch(t.Context(), &llmports.BatchCompletionRequest{
		TenantInfo: tenant,
		Task:       aiprovider.TaskEvaluationJudge,
		Items: []llmports.BatchCompletionItem{
			judgeItem(tenant, "aeval_1", "Yes."),
			judgeItem(tenant, "aeval_1", "No."),
		},
	})
	require.Error(t, err)
	assert.Empty(t, api.requests())
}

func TestBatchStatus_SaysWhenTheBatchHasEnded(t *testing.T) {
	t.Parallel()

	provider := claudeJudge()
	svc, _, _ := newBatchService(t, provider)

	status, err := svc.BatchStatus(t.Context(), &llmports.BatchStatusRequest{
		TenantInfo: judgeTenant(),
		ProviderID: provider.ID,
		BatchID:    "msgbatch_7",
	})
	require.NoError(t, err)
	assert.True(t, status.Ended)
	assert.Equal(t, 2, status.Succeeded)
	assert.Equal(t, 1, status.Errored)
	assert.Equal(t, 1, status.Expired)
}

func TestBatchResults_RecordsEachRunItemOnceAtTheBatchRate(t *testing.T) {
	t.Parallel()

	provider := claudeJudge()
	svc, _, usage := newBatchService(t, provider)
	tenant := judgeTenant()
	agentID := pulid.MustNew("agd_")
	attribution := func(run string) aiusageports.AIUsageAttribution {
		return aiusageports.AIUsageAttribution{
			AgentDefinitionID: agentID,
			RunID:             pulid.ID(run),
			Purpose:           aiusageports.AIUsagePurposeEvaluation,
			Feature:           aiusage.FeatureAgentEvaluation,
		}
	}
	req := &llmports.BatchResultsRequest{
		TenantInfo:  tenant,
		Task:        aiprovider.TaskEvaluationJudge,
		ProviderID:  provider.ID,
		BatchID:     "msgbatch_7",
		SubmittedAt: time.Date(2026, time.October, 27, 3, 0, 0, 0, time.UTC).Unix(),
		Attributions: map[string]aiusageports.AIUsageAttribution{
			"aeval_1": attribution("aeval_1"),
			"aeval_2": attribution("aeval_2"),
			"aeval_3": attribution("aeval_3"),
			"aeval_4": attribution("aeval_4"),
		},
	}

	results, err := svc.BatchResults(t.Context(), req)
	require.NoError(t, err)
	require.Len(t, results, 4)

	judged := results["aeval_1"]
	require.True(t, judged.Succeeded())
	assert.JSONEq(t, `{"score":1,"rationale":"Right."}`, judged.Result.Text)
	full := provider.CostForTask(aiprovider.TaskEvaluationJudge, aiprovider.TokenUsage{
		Input:  1_000_000,
		Output: 100_000,
	})
	require.NotNil(t, full)
	require.NotNil(t, judged.Result.CostUSD)
	assert.True(t, judged.Result.CostUSD.Equal(full.Div(decimal.NewFromInt(2))),
		judged.Result.CostUSD.String())

	assert.False(t, results["aeval_2"].Succeeded())
	assert.Equal(t, "api_error", results["aeval_2"].FailureCode)
	assert.Equal(t, "expired", results["aeval_3"].FailureCode)
	assert.False(
		t,
		results["aeval_4"].Succeeded(),
		"prose where an object was asked for is unusable",
	)
	assert.Equal(t, "unusable", results["aeval_4"].FailureCode)

	rows := usage.recorded(t, 3)
	require.Len(t, rows, 3, "an expired request was never run and costs nothing")
	byRun := map[pulid.ID]*aiusage.AIUsageRecord{}
	for _, row := range rows {
		assert.True(t, row.Batched)
		assert.Equal(t, aiusage.SurfaceEvaluation, row.Surface)
		assert.Equal(t, "aiu_", row.ID.Prefix())
		byRun[row.RunID] = row
	}
	require.Contains(t, byRun, pulid.ID("aeval_1"))
	assert.True(t, byRun["aeval_1"].Succeeded)
	require.NotNil(t, byRun["aeval_1"].CostUSD)
	assert.True(t, byRun["aeval_1"].CostUSD.Equal(*judged.Result.CostUSD))
	assert.False(t, byRun["aeval_2"].Succeeded)
	assert.False(t, byRun["aeval_4"].Succeeded)

	_, err = svc.BatchResults(t.Context(), req)
	require.NoError(t, err)
	assert.Len(t, usage.recorded(t, 3), 3, "reading the same results again records nothing twice")
}
