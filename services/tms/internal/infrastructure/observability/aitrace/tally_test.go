package aitrace

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func TestCompletionTally_SumsTheAttemptsOfOneCall(t *testing.T) {
	t.Parallel()

	a := AnchorFor(AnchorAssistantTurn, "atrn_tally")
	ctx, tally := WithCompletionTally(t.Context())
	require.Same(t, tally, TallyFrom(ctx))

	first := decimal.RequireFromString("0.010")
	second := decimal.RequireFromString("0.002")
	TallyFrom(ctx).Attempt(AttemptTally{
		ProviderID: "aip_1", ProviderName: "anthropic", Model: "model-a", CostUSD: &first,
	})
	TallyFrom(ctx).Restarted()
	TallyFrom(ctx).Attempt(AttemptTally{
		ProviderID: "aip_2", ProviderName: "openai", Model: "model-b", Failover: true,
		CostUSD: &second,
	})

	_, span := StartModelCall(ctx, &ModelCallSpec{Anchor: a})
	tally.Record(span)
	span.End()

	completion := spanNamed(t, a.TraceID, SpanCompletion)
	for _, want := range []any{
		AIAttempts.Int(2),
		AIFailover.Bool(true),
		AIMidReplyRestarts.Int(1),
		AIProviderID.String("aip_2"),
		GenAIProviderName.String("openai"),
		GenAIResponseModel.String("model-b"),
		AICostUSD.Float64(0.012),
	} {
		assert.Contains(t, completion.Attributes, want)
	}
	assert.Equal(t, 2, tally.Attempts())
}

func TestCompletionTally_IsSafeWithoutOne(t *testing.T) {
	t.Parallel()

	var missing *CompletionTally
	assert.Nil(t, TallyFrom(t.Context()))
	missing.Attempt(AttemptTally{})
	missing.Restarted()
	missing.Record(trace.SpanFromContext(t.Context()))
	assert.Zero(t, missing.Attempts())
}

func TestCallOrigin_RidesTheContext(t *testing.T) {
	t.Parallel()

	assert.Equal(t, CallOrigin{}, CallOriginFrom(t.Context()))
	origin := CallOrigin{ActivityAttempt: 3, Stream: true}
	assert.Equal(t, origin, CallOriginFrom(WithCallOrigin(t.Context(), origin)))
}

func TestEmitInvokeAgent_IsTheAnchoredRootWithTheRunsTotals(t *testing.T) {
	t.Parallel()

	a := AnchorFor(AnchorAgentRun, "ar_invoke_root")
	delegate := ForDelegate(pulid.ID("atrn_invoke_parent"), "call_1")
	ctx, request := tracer().Start(t.Context(), "POST /agent-runs/")
	origin := Traceparent(ctx)
	request.End()

	version := int64(4)
	cost := decimal.RequireFromString("0.0425")
	start := time.Unix(1_790_000_000, 0)
	EmitInvokeAgent(t.Context(), &InvokeAgent{
		Anchor:         a,
		AgentID:        pulid.ID("agdef_1"),
		AgentName:      "Billing desk",
		AgentVersion:   &version,
		OwnerKind:      "AgentRun",
		OwnerID:        pulid.ID("ar_invoke_root"),
		RunID:          pulid.ID("ar_invoke_root"),
		OrganizationID: pulid.ID("org_1"),
		BusinessUnitID: pulid.ID("bu_1"),
		Trigger:        "Event",
		Status:         "Failed",
		ErrorType:      "timeout",
		InputTokens:    1200,
		OutputTokens:   300,
		CostUSD:        &cost,
		ToolCalls:      5,
		Tainted:        true,
		TaintSources:   []string{"inbound_message"},
		Start:          start,
		End:            start.Add(time.Minute),
		Origin:         origin,
		Links:          []trace.Link{delegate.Link()},
	})

	root := spanNamed(t, a.TraceID, "invoke_agent Billing desk")
	assert.Equal(t, a.RootSpanID, root.SpanContext.SpanID())
	assert.True(t, root.StartTime.Equal(start))
	assert.Equal(t, codes.Error, root.Status.Code)
	for _, want := range []any{
		GenAIOperationName.String(OperationInvokeAgent),
		GenAIAgentID.String("agdef_1"),
		GenAIAgentName.String("Billing desk"),
		AIAgentVersion.Int64(4),
		AIOwnerKind.String("AgentRun"),
		AIRunID.String("ar_invoke_root"),
		AITrigger.String("Event"),
		AIStatus.String("Failed"),
		ErrorType.String("timeout"),
		AITokensInput.Int64(1200),
		AITokensOutput.Int64(300),
		AICostUSD.Float64(0.0425),
		AIToolCalls.Int(5),
		AITainted.Bool(true),
		AITaintSources.StringSlice([]string{"inbound_message"}),
		AIPurpose.String(PurposeLive),
		TenantOrganizationID.String("org_1"),
	} {
		assert.Contains(t, root.Attributes, want)
	}
	require.Len(t, root.Links, 2)
	assert.Equal(t, request.SpanContext().SpanID(), root.Links[0].SpanContext.SpanID(),
		"the root links to what started it")
	assert.Equal(t, delegate.RootSpanID, root.Links[1].SpanContext.SpanID())
}
