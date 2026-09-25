package completionrouter

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/aiusage"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace/aitracetest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func turnAttribution() serviceports.AIUsageAttribution {
	version := int64(7)

	return serviceports.AIUsageAttribution{
		UserID:            pulid.MustNew("usr_"),
		AgentDefinitionID: pulid.MustNew("agdef_"),
		ThreadID:          pulid.MustNew("athr_"),
		Feature:           aiusage.FeatureAgentTurn,
		OwnerKind:         serviceports.RunStepOwnerAssistantTurn,
		OwnerID:           pulid.MustNew("atrn_"),
		DefinitionVersion: &version,
	}
}

func cachedChatServer(t *testing.T) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"model":"model-a-served",` +
			`"choices":[{"finish_reason":"stop","message":{"role":"assistant","content":"Answered."}}],` +
			`"usage":{"prompt_tokens":2400,"completion_tokens":9,"prompt_tokens_details":{"cached_tokens":2048}}}`))
	}))
	t.Cleanup(server.Close)

	return server
}

func spanIDOf(t *testing.T, value string) trace.SpanID {
	t.Helper()

	id, err := trace.SpanIDFromHex(value)
	require.NoError(t, err)

	return id
}

func TestCompleteChat_TracesEachAttemptAndFailsOverUnderTheTurn(t *testing.T) {
	t.Parallel()
	aitracetest.Install()

	busy, busyCalls := busyThenOK(t, 10, http.StatusServiceUnavailable, "")
	healthy := cachedChatServer(t)
	first := chatProvider("first", busy.URL, 10)
	first.Model = "model-a"
	second := chatProvider("second", healthy.URL, 20)
	second.Model = "model-b"
	usage := &fakeUsage{}
	svc := newTestService(t, first, second)
	svc.usage = usage
	waits := recordingPauses(svc)

	attribution := turnAttribution()
	req := chatRequest(pulid.Nil)
	req.Attribution = attribution
	ctx, tally := aitrace.WithCompletionTally(t.Context())

	result, err := svc.CompleteChat(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, "Answered.", result.Text)
	assert.Equal(t, int32(maxBusyAttempts), busyCalls.Load())
	require.Len(t, *waits, maxBusyAttempts-1)

	anchor := aitrace.ForAttribution(&attribution)
	failed := aitracetest.One(t, anchor.TraceID, "chat model-a")
	answered := aitracetest.One(t, anchor.TraceID, "chat model-b")

	assert.Equal(t, trace.SpanKindClient, failed.SpanKind)
	assert.Equal(t, anchor.RootSpanID, failed.Parent.SpanID())
	assert.Equal(t, codes.Error, failed.Status.Code)
	assert.Contains(t, failed.Attributes, aitrace.ErrorType.String("provider_unavailable"))
	assert.Contains(t, failed.Attributes, aitrace.AIAttempt.Int(1))
	assert.Contains(t, failed.Attributes, aitrace.AIFailover.Bool(false))
	assert.Contains(t, failed.Attributes, aitrace.GenAIProviderName.String("openai"))
	assert.Len(t, failed.Events, maxBusyAttempts-1, "each wait on the busy provider is an event")
	for _, event := range failed.Events {
		assert.Equal(t, aitrace.EventProviderBusyWait, event.Name)
	}

	assert.Contains(t, answered.Attributes, aitrace.AIAttempt.Int(2))
	assert.Contains(t, answered.Attributes, aitrace.AIFailover.Bool(true))
	assert.Contains(t, answered.Attributes, aitrace.GenAIResponseModel.String("model-a-served"))
	assert.Contains(t, answered.Attributes, aitrace.GenAIUsageInputTokens.Int64(2400))
	assert.Contains(t, answered.Attributes, aitrace.GenAIUsageCacheReadTokens.Int64(2048))
	assert.Contains(t, answered.Attributes,
		aitrace.GenAIResponseFinishReasons.StringSlice([]string{"stop"}))
	assert.Equal(t, codes.Unset, answered.Status.Code)

	rows := usage.recorded(t, 2)
	require.Len(t, rows, 2)
	byAttempt := map[int]*aiusage.AIUsageRecord{rows[0].Attempt: rows[0], rows[1].Attempt: rows[1]}
	require.Contains(t, byAttempt, 1)
	require.Contains(t, byAttempt, 2)

	for attempt, span := range map[int]trace.SpanID{
		1: failed.SpanContext.SpanID(),
		2: answered.SpanContext.SpanID(),
	} {
		row := byAttempt[attempt]
		assert.Equal(t, anchor.TraceID.String(), row.TraceID)
		assert.Equal(t, span, spanIDOf(t, row.SpanID), "the row names the attempt's own span")
		assert.Equal(t, agent.RunOwnerAssistantTurn, row.OwnerKind)
		assert.Equal(t, attribution.OwnerID, row.OwnerID)
		require.NotNil(t, row.AgentDefinitionVersion)
		assert.Equal(t, int64(7), *row.AgentDefinitionVersion)
	}
	assert.False(t, byAttempt[1].Failover)
	assert.False(t, byAttempt[1].Succeeded)
	assert.True(t, byAttempt[2].Failover)
	assert.Equal(t, 2048, byAttempt[2].CacheReadTokens)
	assert.Zero(t, byAttempt[2].CacheWriteTokens)

	assert.Equal(t, 2, tally.Attempts())
}

func TestCompleteChat_ADelegatesAttemptIsTracedUnderItsOwnTask(t *testing.T) {
	t.Parallel()
	aitracetest.Install()

	server := cachedChatServer(t)
	provider := chatProvider("only", server.URL, 10)
	provider.Model = "model-c"
	usage := &fakeUsage{}
	svc := newTestService(t, provider)
	svc.usage = usage

	attribution := turnAttribution()
	attribution.DelegateCallID = "call_delegate_1"
	req := chatRequest(pulid.Nil)
	req.Attribution = attribution

	_, err := svc.CompleteChat(t.Context(), req)
	require.NoError(t, err)

	delegate := aitrace.ForDelegate(attribution.OwnerID, "call_delegate_1")
	span := aitracetest.One(t, delegate.TraceID, "chat model-c")
	assert.Equal(t, delegate.RootSpanID, span.Parent.SpanID())

	rows := usage.recorded(t, 1)
	require.Len(t, rows, 1)
	assert.Equal(t, "call_delegate_1", rows[0].DelegateCallID)
	assert.Equal(t, delegate.TraceID.String(), rows[0].TraceID)
	assert.Equal(t, attribution.OwnerID, rows[0].OwnerID)
}

func TestCompleteStructured_TracesItsAttemptInTheCallersTrace(t *testing.T) {
	t.Parallel()
	aitracetest.Install()

	healthy, _ := chatServer(t, http.StatusOK, `{"answer":"ok"}`)
	provider := openAIChatProvider("only", healthy.URL, 10)
	provider.Model = "model-d"
	usage := &fakeUsage{}
	svc := newTestService(t, provider)
	svc.usage = usage

	ctx, request := otelTracer().Start(t.Context(), "POST /graphql")
	_, err := svc.CompleteStructured(ctx, generalRequest())
	request.End()
	require.NoError(t, err)

	span := aitracetest.One(t, request.SpanContext().TraceID(), "chat model-d")
	assert.Equal(t, request.SpanContext().SpanID(), span.Parent.SpanID())
	rows := usage.recorded(t, 1)
	require.Len(t, rows, 1)
	assert.Equal(t, request.SpanContext().TraceID().String(), rows[0].TraceID)
	assert.Equal(t, 1, rows[0].Attempt)
	assert.Empty(t, rows[0].OwnerKind, "a one-shot call belongs to no run or turn")
}

func otelTracer() trace.Tracer {
	return otel.Tracer("completionrouter-test")
}
