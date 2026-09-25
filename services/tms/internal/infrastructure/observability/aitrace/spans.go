package aitrace

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type ModelCallSpec struct {
	Anchor          Anchor
	Feature         string
	ActivityAttempt int
	Stream          bool
	Looped          bool
	Attrs           []attribute.KeyValue
}

type AttemptSpec struct {
	Anchor        Anchor
	Operation     string
	ProviderKind  aiprovider.Kind
	ProviderID    pulid.ID
	Model         string
	MaxTokens     int
	ServerAddress string
	Attempt       int
	Feature       string
	Attrs         []attribute.KeyValue
}

type ToolSpec struct {
	Anchor   Anchor
	ToolName string
	CallID   string
	Effect   string
	Kind     string
	StepKey  string
	Attrs    []attribute.KeyValue
}

type WriteSpec struct {
	Anchor        Anchor
	EntityType    string
	EntityID      string
	VersionBefore *int64
	ProposalID    pulid.ID
	Simulated     bool
	Attrs         []attribute.KeyValue
}

const decideSpanPrefix = "trenova.ai.proposal."

type DecideOperation string

const (
	DecideOperationDecide  = DecideOperation("decide")
	DecideOperationExecute = DecideOperation("execute")
	DecideOperationExpire  = DecideOperation("expire")
)

type DecideSpec struct {
	Operation         DecideOperation
	OrganizationID    pulid.ID
	BusinessUnitID    pulid.ID
	ProposalID        pulid.ID
	RunID             pulid.ID
	ToolName          string
	Decision          string
	UserID            pulid.ID
	ReasonCode        string
	ModificationCount int
	ProposalTraceID   string
	ProposalSpanID    string
	Attrs             []attribute.KeyValue
}

type Usage struct {
	ResponseModel    string
	FinishReasons    []string
	InputTokens      int64
	OutputTokens     int64
	CacheReadTokens  int64
	CacheWriteTokens int64
	ReasoningTokens  int64
	CostUSD          *decimal.Decimal
	Truncated        bool
}

func StartModelCall(ctx context.Context, spec *ModelCallSpec) (context.Context, trace.Span) {
	attrs := make([]attribute.KeyValue, 0, len(spec.Attrs)+4)
	attrs = append(attrs,
		AIActivityAttempt.Int(spec.ActivityAttempt),
		AIStream.Bool(spec.Stream),
		AILooped.Bool(spec.Looped),
	)
	attrs = appendString(attrs, AIFeature, spec.Feature)
	attrs = append(attrs, spec.Attrs...)

	return start(ctx, spec.Anchor, SpanCompletion, trace.SpanKindInternal, attrs, nil)
}

func StartAttempt(ctx context.Context, spec *AttemptSpec) (context.Context, trace.Span) {
	operation := spec.Operation
	if operation == "" {
		operation = OperationChat
	}

	attrs := make([]attribute.KeyValue, 0, len(spec.Attrs)+8)
	attrs = append(attrs,
		GenAIOperationName.String(operation),
		GenAIProviderName.String(ProviderName(spec.ProviderKind)),
		AIAttempt.Int(spec.Attempt),
	)
	attrs = appendString(attrs, GenAIRequestModel, spec.Model)
	attrs = appendString(attrs, AIProviderID, spec.ProviderID.String())
	attrs = appendString(attrs, ServerAddress, spec.ServerAddress)
	attrs = appendString(attrs, AIFeature, spec.Feature)
	if spec.MaxTokens > 0 {
		attrs = append(attrs, GenAIRequestMaxTokens.Int(spec.MaxTokens))
	}
	attrs = append(attrs, spec.Attrs...)

	return start(
		ctx,
		spec.Anchor,
		spanName(operation, spec.Model),
		trace.SpanKindClient,
		attrs,
		nil,
	)
}

func StartTool(ctx context.Context, spec *ToolSpec) (context.Context, trace.Span) {
	attrs := make([]attribute.KeyValue, 0, len(spec.Attrs)+7)
	attrs = append(attrs,
		GenAIOperationName.String(OperationExecuteTool),
		GenAIToolName.String(spec.ToolName),
		GenAIToolType.String(ToolTypeFunction),
	)
	attrs = appendString(attrs, GenAIToolCallID, spec.CallID)
	attrs = appendString(attrs, AIToolEffect, spec.Effect)
	attrs = appendString(attrs, AIToolKind, spec.Kind)
	attrs = appendString(attrs, AIStepKey, spec.StepKey)
	attrs = append(attrs, spec.Attrs...)

	return start(
		ctx,
		spec.Anchor,
		spanName(OperationExecuteTool, spec.ToolName),
		trace.SpanKindInternal,
		attrs,
		nil,
	)
}

func StartWrite(ctx context.Context, spec *WriteSpec) (context.Context, trace.Span) {
	attrs := make([]attribute.KeyValue, 0, len(spec.Attrs)+5)
	attrs = append(attrs, AISimulated.Bool(spec.Simulated))
	attrs = appendString(attrs, AIEntityType, spec.EntityType)
	attrs = appendString(attrs, AIEntityID, spec.EntityID)
	attrs = appendString(attrs, AIProposalID, spec.ProposalID.String())
	if spec.VersionBefore != nil {
		attrs = append(attrs, AIVersionBefore.Int64(*spec.VersionBefore))
	}
	attrs = append(attrs, spec.Attrs...)

	return start(
		ctx,
		spec.Anchor,
		spanName(SpanWrite, spec.EntityType),
		trace.SpanKindInternal,
		attrs,
		nil,
	)
}

func StartDecide(ctx context.Context, spec *DecideSpec) (context.Context, trace.Span) {
	attrs := make([]attribute.KeyValue, 0, len(spec.Attrs)+9)
	attrs = appendString(attrs, TenantOrganizationID, spec.OrganizationID.String())
	attrs = appendString(attrs, TenantBusinessUnitID, spec.BusinessUnitID.String())
	attrs = appendString(attrs, AIProposalID, spec.ProposalID.String())
	attrs = appendString(attrs, AIRunID, spec.RunID.String())
	attrs = appendString(attrs, GenAIToolName, spec.ToolName)
	attrs = appendString(attrs, AIDecision, spec.Decision)
	attrs = appendString(attrs, UserID, spec.UserID.String())
	attrs = appendString(attrs, AIReasonCode, spec.ReasonCode)
	if spec.ModificationCount > 0 {
		attrs = append(attrs, AIModificationCount.Int(spec.ModificationCount))
	}
	attrs = append(attrs, spec.Attrs...)

	var links []trace.Link
	if link, ok := LinkTo(spec.ProposalTraceID, spec.ProposalSpanID); ok {
		links = []trace.Link{link}
	}

	return start(
		ctx,
		Anchor{},
		decideSpanPrefix+string(spec.Operation),
		trace.SpanKindInternal,
		attrs,
		links,
	)
}

func RecordUsage(span trace.Span, usage *Usage) {
	if usage == nil || !span.IsRecording() {
		return
	}

	attrs := make([]attribute.KeyValue, 0, 9)
	attrs = append(attrs,
		GenAIUsageInputTokens.Int64(usage.InputTokens),
		GenAIUsageOutputTokens.Int64(usage.OutputTokens),
		AITruncated.Bool(usage.Truncated),
	)
	attrs = appendString(attrs, GenAIResponseModel, usage.ResponseModel)
	if usage.CacheReadTokens > 0 {
		attrs = append(attrs, GenAIUsageCacheReadTokens.Int64(usage.CacheReadTokens))
	}
	if usage.CacheWriteTokens > 0 {
		attrs = append(attrs, GenAIUsageCacheCreationTokens.Int64(usage.CacheWriteTokens))
	}
	if usage.ReasoningTokens > 0 {
		attrs = append(attrs, AIReasoningTokens.Int64(usage.ReasoningTokens))
	}
	if len(usage.FinishReasons) > 0 {
		attrs = append(attrs, GenAIResponseFinishReasons.StringSlice(usage.FinishReasons))
	}
	if usage.CostUSD != nil {
		attrs = append(attrs, AICostUSD.Float64(usage.CostUSD.InexactFloat64()))
	}

	span.SetAttributes(attrs...)
}

func MarkFailed(span trace.Span, errorType string) {
	if errorType == "" {
		errorType = OutcomeFailed
	}

	span.SetAttributes(ErrorType.String(errorType))
	span.SetStatus(codes.Error, errorType)
}

func RecordBusyWait(span trace.Span, wait time.Duration) {
	span.AddEvent(EventProviderBusyWait, trace.WithAttributes(
		EventWaitSeconds.Float64(wait.Seconds()),
	))
}

func RecordResting(span trace.Span) {
	span.AddEvent(EventProviderResting)
}

func start(
	ctx context.Context,
	a Anchor,
	name string,
	kind trace.SpanKind,
	attrs []attribute.KeyValue,
	links []trace.Link,
) (context.Context, trace.Span) {
	ctx, parentLinks := Parent(ctx, a)
	if len(parentLinks) > 0 {
		links = append(links, parentLinks...)
	}

	return tracer().Start( //nolint:spancheck // the caller ends the span it is handed
		ctx,
		name,
		trace.WithSpanKind(kind),
		trace.WithAttributes(attrs...),
		trace.WithLinks(links...),
	)
}

func appendString(
	attrs []attribute.KeyValue,
	key attribute.Key,
	value string,
) []attribute.KeyValue {
	if value == "" {
		return attrs
	}

	return append(attrs, key.String(value))
}
