package aitrace

import (
	"context"
	"crypto/sha256"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"go.opentelemetry.io/otel/trace"
)

const (
	anchorDerivePrefix = "trenova/ai-trace/v1|"
	anchorKindSep      = "|"
	anchorRootSuffix   = "|root"
	delegateKeySep     = ":"
)

type AnchorKind string

const (
	AnchorAgentRun      = AnchorKind("AgentRun")
	AnchorAssistantTurn = AnchorKind("AssistantTurn")
	AnchorDelegate      = AnchorKind("Delegate")
	AnchorEvaluation    = AnchorKind("Evaluation")
)

func (k AnchorKind) IsValid() bool {
	switch k {
	case AnchorAgentRun, AnchorAssistantTurn, AnchorDelegate, AnchorEvaluation:
		return true
	default:
		return false
	}
}

type Anchor struct {
	TraceID    trace.TraceID
	RootSpanID trace.SpanID
	Sampled    bool
}

func AnchorFor(kind AnchorKind, key string) Anchor {
	if !kind.IsValid() || key == "" {
		return Anchor{}
	}

	material := make(
		[]byte,
		0,
		len(anchorDerivePrefix)+len(kind)+len(anchorKindSep)+len(key)+len(anchorRootSuffix),
	)
	material = append(material, anchorDerivePrefix...)
	material = append(material, kind...)
	material = append(material, anchorKindSep...)
	material = append(material, key...)
	traceSum := sha256.Sum256(material)
	material = append(material, anchorRootSuffix...)
	spanSum := sha256.Sum256(material)

	var traceID trace.TraceID
	copy(traceID[:], traceSum[:len(traceID)])
	if !traceID.IsValid() {
		copy(traceID[:], traceSum[len(traceID):])
	}
	if !traceID.IsValid() {
		traceID[len(traceID)-1] = 1
	}

	var spanID trace.SpanID
	copy(spanID[:], spanSum[:len(spanID)])
	if !spanID.IsValid() {
		copy(spanID[:], spanSum[len(spanID):2*len(spanID)])
	}
	if !spanID.IsValid() {
		spanID[len(spanID)-1] = 1
	}

	return Anchor{TraceID: traceID, RootSpanID: spanID, Sampled: sampledByAIRate(traceID)}
}

func ForDelegate(ownerID pulid.ID, delegateCallID string) Anchor {
	if ownerID.IsNil() || delegateCallID == "" {
		return Anchor{}
	}

	return AnchorFor(AnchorDelegate, ownerID.String()+delegateKeySep+delegateCallID)
}

func ForRun(owner serviceports.RunStepOwner, delegation *serviceports.Delegation) Anchor {
	callID := ""
	if delegation != nil {
		callID = delegation.CallID
	}

	return forOwner(owner.Kind, owner.ID, callID)
}

func ForAttribution(attribution *serviceports.AIUsageAttribution) Anchor {
	if attribution == nil {
		return Anchor{}
	}
	if attribution.OwnerID.IsNotNil() {
		return forOwner(attribution.OwnerKind, attribution.OwnerID, attribution.DelegateCallID)
	}
	if attribution.RunID.IsNotNil() {
		return forOwner(serviceports.RunStepOwnerAgentRun, attribution.RunID, "")
	}

	return Anchor{}
}

func forOwner(kind serviceports.RunStepOwnerKind, id pulid.ID, delegateCallID string) Anchor {
	if id.IsNil() {
		return Anchor{}
	}
	if delegateCallID != "" {
		return ForDelegate(id, delegateCallID)
	}

	switch kind {
	case serviceports.RunStepOwnerAgentRun:
		if id.Prefix() == agent.EvaluationIDPrefix {
			return AnchorFor(AnchorEvaluation, id.String())
		}
		return AnchorFor(AnchorAgentRun, id.String())
	case serviceports.RunStepOwnerAssistantTurn:
		return AnchorFor(AnchorAssistantTurn, id.String())
	}

	return Anchor{}
}

func (a Anchor) IsValid() bool {
	return a.TraceID.IsValid() && a.RootSpanID.IsValid()
}

func (a Anchor) SpanContext() trace.SpanContext {
	if !a.IsValid() {
		return trace.SpanContext{}
	}

	var flags trace.TraceFlags
	if a.Sampled {
		flags = trace.FlagsSampled
	}

	return trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    a.TraceID,
		SpanID:     a.RootSpanID,
		TraceFlags: flags,
		Remote:     true,
	})
}

func ContextWithAnchor(ctx context.Context, a Anchor) context.Context {
	if !a.IsValid() {
		return ctx
	}

	return trace.ContextWithRemoteSpanContext(ctx, a.SpanContext())
}

func Parent(ctx context.Context, a Anchor) (context.Context, []trace.Link) {
	if !a.IsValid() {
		return ctx, nil
	}

	current := trace.SpanContextFromContext(ctx)
	if current.IsValid() && current.TraceID() == a.TraceID {
		return ctx, nil
	}

	anchored := ContextWithAnchor(ctx, a)
	if !current.IsValid() {
		return anchored, nil
	}

	return anchored, []trace.Link{{SpanContext: current}}
}
