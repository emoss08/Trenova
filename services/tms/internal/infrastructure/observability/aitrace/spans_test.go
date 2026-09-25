package aitrace

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func TestStartModelCall_OpensTheCompletionUnderTheAnchor(t *testing.T) {
	t.Parallel()

	a := AnchorFor(AnchorAssistantTurn, "atrn_model_call")
	ctx, span := StartModelCall(t.Context(), &ModelCallSpec{
		Anchor:          a,
		Feature:         "AgentTurn",
		ActivityAttempt: 2,
		Stream:          true,
	})
	_, attempt := StartAttempt(ctx, &AttemptSpec{
		ProviderKind:  aiprovider.KindAnthropicMessages,
		ProviderID:    pulid.ID("aip_1"),
		Model:         "model-a",
		MaxTokens:     4096,
		ServerAddress: "api.example.com",
		Attempt:       1,
	})
	cost := decimal.RequireFromString("0.012500")
	RecordUsage(attempt, &Usage{
		ResponseModel:    "model-a-2026",
		FinishReasons:    []string{"end_turn"},
		InputTokens:      1200,
		OutputTokens:     300,
		CacheReadTokens:  800,
		CacheWriteTokens: 64,
		ReasoningTokens:  12,
		CostUSD:          &cost,
	})
	RecordBusyWait(attempt, 1500*time.Millisecond)
	RecordResting(attempt)
	attempt.End()
	span.End()

	completion := spanNamed(t, a.TraceID, SpanCompletion)
	assert.Equal(t, a.RootSpanID, completion.Parent.SpanID())
	assert.Contains(t, completion.Attributes, AIActivityAttempt.Int(2))
	assert.Contains(t, completion.Attributes, AIStream.Bool(true))
	assert.Contains(t, completion.Attributes, AIFeature.String("AgentTurn"))

	chat := spanNamed(t, a.TraceID, "chat model-a")
	assert.Equal(t, trace.SpanKindClient, chat.SpanKind)
	assert.Equal(t, completion.SpanContext.SpanID(), chat.Parent.SpanID())
	for _, want := range []attribute.KeyValue{
		GenAIOperationName.String(OperationChat),
		GenAIProviderName.String("anthropic"),
		GenAIRequestModel.String("model-a"),
		GenAIRequestMaxTokens.Int(4096),
		ServerAddress.String("api.example.com"),
		AIProviderID.String("aip_1"),
		AIAttempt.Int(1),
		GenAIResponseModel.String("model-a-2026"),
		GenAIResponseFinishReasons.StringSlice([]string{"end_turn"}),
		GenAIUsageInputTokens.Int64(1200),
		GenAIUsageOutputTokens.Int64(300),
		GenAIUsageCacheReadTokens.Int64(800),
		GenAIUsageCacheCreationTokens.Int64(64),
		AIReasoningTokens.Int64(12),
		AICostUSD.Float64(0.0125),
	} {
		assert.Contains(t, chat.Attributes, want)
	}
	require.Len(t, chat.Events, 2)
	assert.Equal(t, EventProviderBusyWait, chat.Events[0].Name)
	assert.Contains(t, chat.Events[0].Attributes, EventWaitSeconds.Float64(1.5))
	assert.Equal(t, EventProviderResting, chat.Events[1].Name)
}

func TestStartTool_ReparentsADelegateCallAndMarksFailure(t *testing.T) {
	t.Parallel()

	turn := AnchorFor(AnchorAssistantTurn, "atrn_tool_delegate")
	delegate := ForDelegate(pulid.ID("atrn_tool_delegate"), "call_9")
	activityCtx, activity := tracer().Start(ContextWithAnchor(t.Context(), turn), "RunActivity")

	_, span := StartTool(activityCtx, &ToolSpec{
		Anchor:   delegate,
		ToolName: "assign_move",
		CallID:   "call_x",
		Effect:   "change",
		Kind:     "action",
		StepKey:  "step-1",
		Attrs:    []attribute.KeyValue{AIOutcome.String(OutcomeProposed)},
	})
	MarkFailed(span, "")
	span.End()
	activity.End()

	tool := spanNamed(t, delegate.TraceID, "execute_tool assign_move")
	assert.Equal(t, delegate.RootSpanID, tool.Parent.SpanID())
	require.Len(t, tool.Links, 1)
	assert.Equal(t, activity.SpanContext(), tool.Links[0].SpanContext)
	for _, want := range []attribute.KeyValue{
		GenAIOperationName.String(OperationExecuteTool),
		GenAIToolName.String("assign_move"),
		GenAIToolCallID.String("call_x"),
		GenAIToolType.String(ToolTypeFunction),
		AIToolEffect.String("change"),
		AIToolKind.String("action"),
		AIStepKey.String("step-1"),
		AIOutcome.String(OutcomeProposed),
		ErrorType.String(OutcomeFailed),
	} {
		assert.Contains(t, tool.Attributes, want)
	}
	assert.Equal(t, codes.Error, tool.Status.Code)
}

func TestStartWrite_DescribesTheRecordWithoutItsContent(t *testing.T) {
	t.Parallel()

	a := AnchorFor(AnchorAgentRun, "ar_write")
	version := int64(0)
	_, span := StartWrite(ContextWithAnchor(t.Context(), a), &WriteSpec{
		EntityType:    "shipment",
		EntityID:      "shp_1",
		VersionBefore: &version,
		ProposalID:    pulid.ID("ap_1"),
		Simulated:     true,
	})
	span.End()

	write := spanNamed(t, a.TraceID, "trenova.ai.write shipment")
	for _, want := range []attribute.KeyValue{
		AIEntityType.String("shipment"),
		AIEntityID.String("shp_1"),
		AIVersionBefore.Int64(0),
		AIProposalID.String("ap_1"),
		AISimulated.Bool(true),
	} {
		assert.Contains(t, write.Attributes, want)
	}
}

func TestStartDecide_LinksBackToTheProposingSpan(t *testing.T) {
	t.Parallel()

	proposing := AnchorFor(AnchorAgentRun, "ar_decide_origin")
	ctx, request := tracer().Start(t.Context(), "POST /graphql")
	_, span := StartDecide(ctx, &DecideSpec{
		Operation:         DecideOperationDecide,
		OrganizationID:    pulid.ID("org_1"),
		BusinessUnitID:    pulid.ID("bu_1"),
		ProposalID:        pulid.ID("ap_1"),
		RunID:             pulid.ID("ar_decide_origin"),
		ToolName:          "assign_move",
		Decision:          "Modified",
		UserID:            pulid.ID("usr_1"),
		ReasonCode:        "wrong_driver",
		ModificationCount: 2,
		ProposalTraceID:   proposing.TraceID.String(),
		ProposalSpanID:    proposing.RootSpanID.String(),
	})
	span.End()
	request.End()

	decide := spanNamed(t, request.SpanContext().TraceID(), "trenova.ai.proposal.decide")
	assert.Equal(t, request.SpanContext().SpanID(), decide.Parent.SpanID())
	require.Len(t, decide.Links, 1)
	assert.Equal(t, proposing.TraceID, decide.Links[0].SpanContext.TraceID())
	assert.Equal(t, proposing.RootSpanID, decide.Links[0].SpanContext.SpanID())
	for _, want := range []attribute.KeyValue{
		TenantOrganizationID.String("org_1"),
		TenantBusinessUnitID.String("bu_1"),
		AIProposalID.String("ap_1"),
		AIRunID.String("ar_decide_origin"),
		GenAIToolName.String("assign_move"),
		AIDecision.String("Modified"),
		UserID.String("usr_1"),
		AIReasonCode.String("wrong_driver"),
		AIModificationCount.Int(2),
	} {
		assert.Contains(t, decide.Attributes, want)
	}

	_, unlinked := StartDecide(t.Context(), &DecideSpec{
		Operation:       DecideOperationExpire,
		ProposalTraceID: "not-a-trace",
	})
	unlinked.End()
	expired := spanNamed(t, unlinked.SpanContext().TraceID(), "trenova.ai.proposal.expire")
	assert.Empty(t, expired.Links)
}

func TestProviderName_MapsProtocolsToGenAIProviders(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "anthropic", ProviderName(aiprovider.KindAnthropicMessages))
	assert.Equal(t, "openai", ProviderName(aiprovider.KindOpenAIResponses))
	assert.Equal(t, "openai", ProviderName(aiprovider.KindOpenAIChat))
	assert.Equal(t, "ollama", ProviderName(aiprovider.KindOllama))
	assert.Equal(t, "someday", ProviderName(aiprovider.Kind("Someday")))
}

func TestTenant_NamesBothScopes(t *testing.T) {
	t.Parallel()

	assert.Equal(t, []attribute.KeyValue{
		TenantOrganizationID.String("org_1"),
		TenantBusinessUnitID.String("bu_1"),
	}, Tenant(pulid.ID("org_1"), pulid.ID("bu_1")))
}
