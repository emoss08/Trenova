package completionrouter

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aitask"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiusage"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiusageports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeUsage struct {
	mu   sync.Mutex
	rows []*aiusage.AIUsageRecord
	err  error
}

func (f *fakeUsage) Create(_ context.Context, record *aiusage.AIUsageRecord) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rows = append(f.rows, record)

	return f.err
}

func (f *fakeUsage) CreateOrSkip(
	_ context.Context,
	records []*aiusage.AIUsageRecord,
) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return 0, f.err
	}

	var inserted int64
	for _, record := range records {
		known := false
		for _, row := range f.rows {
			if row.ID == record.ID {
				known = true

				break
			}
		}
		if !known {
			f.rows = append(f.rows, record)
			inserted++
		}
	}

	return inserted, nil
}

func (f *fakeUsage) Summary(
	context.Context,
	aiusageports.AIUsageSummaryRequest,
) (*aiusageports.AIUsageSummaryRow, error) {
	return &aiusageports.AIUsageSummaryRow{}, nil
}

func (f *fakeUsage) recorded(t *testing.T, n int) []*aiusage.AIUsageRecord {
	t.Helper()
	require.Eventually(t, func() bool {
		f.mu.Lock()
		defer f.mu.Unlock()

		return len(f.rows) >= n
	}, 2*time.Second, 10*time.Millisecond, "usage rows are written off the request path")

	f.mu.Lock()
	defer f.mu.Unlock()

	return append([]*aiusage.AIUsageRecord(nil), f.rows...)
}

func priced(provider *aiprovider.Provider, in, out string) *aiprovider.Provider {
	inCost := decimal.RequireFromString(in)
	outCost := decimal.RequireFromString(out)
	provider.InputCostPerMillion = &inCost
	provider.OutputCostPerMillion = &outCost

	return provider
}

// A turn that fell through one provider before a second answered was two
// calls, two latencies and two bills. Each is a row of its own; the answer
// carries only its own latency and cost.
func TestCompleteStructured_RecordsEveryAttempt(t *testing.T) {
	t.Parallel()

	failing, _ := chatServer(t, http.StatusInternalServerError, "")
	healthy, _ := chatServer(t, http.StatusOK, `{"answer":"second"}`)
	usage := &fakeUsage{}
	svc := newTestService(t,
		openAIChatProvider("primary", failing.URL, 10),
		priced(openAIChatProvider("secondary", healthy.URL, 20), "1", "10"),
	)
	svc.usage = usage

	req := generalRequest()
	req.Attribution = aiusageports.AIUsageAttribution{UserID: pulid.MustNew("usr_")}
	result, err := svc.CompleteStructured(t.Context(), req)
	require.NoError(t, err)

	rows := usage.recorded(t, 2)
	require.Len(t, rows, 2)

	first, second := rows[0], rows[1]
	assert.False(t, first.Succeeded)
	assert.Equal(t, "provider_unavailable", first.ErrorClass)
	assert.Zero(t, first.InputTokens, "a provider that errored reported no tokens")
	assert.Nil(t, first.CostUSD)

	assert.True(t, second.Succeeded)
	assert.Equal(t, aiusage.SurfaceStructured, second.Surface)
	assert.Equal(t, aitask.TaskGeneral, second.Task)
	assert.Equal(t, req.Attribution.UserID, second.UserID)
	assert.Equal(t, 11, second.InputTokens)
	assert.Equal(t, 7, second.OutputTokens)
	assert.GreaterOrEqual(t, second.LatencyMs, int64(0))
	require.NotNil(t, second.CostUSD)
	// 11 in at $1/M plus 7 out at $10/M
	assert.True(
		t,
		second.CostUSD.Equal(decimal.RequireFromString("0.000081")),
		second.CostUSD.String(),
	)

	require.NotNil(t, result.CostUSD)
	assert.True(t, result.CostUSD.Equal(*second.CostUSD))
	assert.Equal(t, second.LatencyMs, result.LatencyMs)
	assert.Equal(t, req.TenantInfo.OrgID, second.OrganizationID)
}

func TestCompleteStructured_RecordsAnEvaluationApartFromTheLiveAgent(t *testing.T) {
	t.Parallel()

	healthy, _ := chatServer(t, http.StatusOK, `{"answer":"replayed"}`)
	usage := &fakeUsage{}
	svc := newTestService(t, priced(openAIChatProvider("only", healthy.URL, 10), "1", "10"))
	svc.usage = usage

	req := generalRequest()
	req.Attribution = aiusageports.AIUsageAttribution{
		AgentDefinitionID: pulid.MustNew("agd_"),
		RunID:             pulid.MustNew("aeval_"),
	}
	_, err := svc.CompleteStructured(t.Context(), req)
	require.NoError(t, err)

	rows := usage.recorded(t, 1)
	require.Len(t, rows, 1)
	assert.Equal(t, aiusage.SurfaceEvaluation, rows[0].Surface)
	assert.Equal(t, req.Attribution.AgentDefinitionID, rows[0].AgentDefinitionID)
}

func TestSurfaceFor(t *testing.T) {
	t.Parallel()

	live := aiusageports.AIUsageAttribution{RunID: pulid.MustNew("ar_")}
	byRun := aiusageports.AIUsageAttribution{RunID: pulid.MustNew("aeval_")}
	byPurpose := aiusageports.AIUsageAttribution{Purpose: aiusageports.AIUsagePurposeEvaluation}

	assert.Equal(t, aiusage.SurfaceChat, surfaceFor(aiusage.SurfaceChat, live))
	assert.Equal(t, aiusage.SurfaceStructured, surfaceFor(aiusage.SurfaceStructured, live))
	assert.Equal(t, aiusage.SurfaceEvaluation, surfaceFor(aiusage.SurfaceChat, byRun))
	assert.Equal(t, aiusage.SurfaceEvaluation, surfaceFor(aiusage.SurfaceStructured, byPurpose))
}

// A slow or failing usage table never slows an answer or turns it into an
// error: the row is telemetry, not the transaction.
func TestCompleteStructured_AnswersWhenTheUsageWriteFails(t *testing.T) {
	t.Parallel()

	server, _ := chatServer(t, http.StatusOK, `{"answer":"ok"}`)
	usage := &fakeUsage{err: errors.New("usage table is away")}
	svc := newTestService(t, openAIChatProvider("local", server.URL, 10))
	svc.usage = usage

	result, err := svc.CompleteStructured(t.Context(), generalRequest())
	require.NoError(t, err)
	assert.Equal(t, `{"answer":"ok"}`, result.Text)
	usage.recorded(t, 1)
}

func TestCompleteStructured_RecordsNothingWithoutARecorder(t *testing.T) {
	t.Parallel()

	server, _ := chatServer(t, http.StatusOK, `{"answer":"ok"}`)
	svc := newTestService(t, openAIChatProvider("local", server.URL, 10))

	_, err := svc.CompleteStructured(t.Context(), generalRequest())
	require.NoError(t, err)
}

type timeoutErr struct{}

func (timeoutErr) Error() string   { return "i/o timeout" }
func (timeoutErr) Timeout() bool   { return true }
func (timeoutErr) Temporary() bool { return true }

func TestClassifyError(t *testing.T) {
	t.Parallel()

	var netTimeout net.Error = timeoutErr{}
	assert.Empty(t, classifyError(nil))
	assert.Equal(t, "refused", classifyError(errRefused))
	assert.Equal(t, "timeout", classifyError(context.DeadlineExceeded))
	assert.Equal(t, "cancelled", classifyError(context.Canceled))
	assert.Equal(t, "timeout", classifyError(netTimeout))
	assert.Equal(t, "provider_error", classifyError(errors.New("bad request")))
	assert.Equal(t, "rate_limited", classifyError(&modeladapter.TransportError{
		StatusCode: http.StatusTooManyRequests,
		Retryable:  true,
	}))
	assert.Equal(t, "auth", classifyError(&modeladapter.TransportError{
		StatusCode: http.StatusUnauthorized,
	}))
	assert.Equal(t, "auth", classifyError(&modeladapter.TransportError{
		StatusCode: http.StatusForbidden,
	}))
	assert.Equal(t, "provider_unavailable", classifyError(&modeladapter.TransportError{
		StatusCode: http.StatusBadGateway,
		Retryable:  true,
	}))
	assert.Equal(t, "provider_error", classifyError(&modeladapter.TransportError{
		StatusCode: http.StatusBadRequest,
	}))
}

func (f *fakeUsage) RecentFailures(
	context.Context,
	aiusageports.AIUsageFailuresRequest,
) ([]aiusageports.AIUsageFailureRow, error) {
	return nil, nil
}

func (f *fakeUsage) CostByDefinition(
	context.Context,
	aiusageports.AIUsageCostRequest,
) (*aiusageports.AIUsageCost, error) {
	return &aiusageports.AIUsageCost{}, nil
}

func (f *fakeUsage) Daily(
	context.Context,
	aiusageports.AIUsageDailyQuery,
) ([]aiusageports.AIUsageDayTotals, error) {
	return []aiusageports.AIUsageDayTotals{}, nil
}

func (f *fakeUsage) ProviderTaskTotals(
	context.Context,
	aiusageports.AIUsageProviderTaskRequest,
) ([]aiusageports.AIUsageProviderTaskTotals, error) {
	return []aiusageports.AIUsageProviderTaskTotals{}, nil
}

func (f *fakeUsage) SurfaceCost(
	context.Context,
	aiusageports.AIUsageSurfaceCostRequest,
) (*aiusageports.AIUsageCost, error) {
	return &aiusageports.AIUsageCost{CostUSD: decimal.Zero}, nil
}

func (f *fakeUsage) EvaluationCost(
	context.Context,
	aiusageports.AIUsageEvaluationCostRequest,
) (*aiusageports.AIUsageCost, error) {
	return &aiusageports.AIUsageCost{CostUSD: decimal.Zero}, nil
}

func (f *fakeUsage) SpendByProvider(
	context.Context,
	aiusageports.AIUsageProviderSpendRequest,
) (map[pulid.ID]decimal.Decimal, error) {
	return map[pulid.ID]decimal.Decimal{}, nil
}

// OpenAI counts the prompt it served from its cache inside prompt_tokens, and
// the row priced all of it as fresh input. The cached part is priced on its
// own, at a tenth of input when no cache price is entered.
func TestCompleteStructured_PricesCachedPromptTokensApart(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		encoded, _ := sonic.Marshal(map[string]any{
			"model": "test-model",
			"choices": []map[string]any{{"finish_reason": "stop", "message": map[string]any{
				"role": "assistant", "content": `{"answer":"cached"}`,
			}}},
			"usage": map[string]any{
				"prompt_tokens":         1000,
				"completion_tokens":     7,
				"prompt_tokens_details": map[string]any{"cached_tokens": 800},
			},
		})
		_, _ = w.Write(encoded)
	}))
	t.Cleanup(server.Close)

	usage := &fakeUsage{}
	svc := newTestService(t, priced(openAIChatProvider("cached", server.URL, 10), "1", "10"))
	svc.usage = usage

	result, err := svc.CompleteStructured(t.Context(), generalRequest())
	require.NoError(t, err)

	rows := usage.recorded(t, 1)
	require.Len(t, rows, 1)
	assert.Equal(t, 800, rows[0].CacheReadTokens)
	require.NotNil(t, rows[0].CostUSD)
	// 200 fresh at $1/M, 800 cached at $0.10/M, 7 out at $10/M
	assert.True(
		t,
		rows[0].CostUSD.Equal(decimal.RequireFromString("0.00035")),
		rows[0].CostUSD.String(),
	)
	require.NotNil(t, result.CostUSD)
	assert.True(t, result.CostUSD.Equal(*rows[0].CostUSD))
}
