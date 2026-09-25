package agentruntime

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"github.com/emoss08/trenova/shared/pulid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func (s *Service) startToolSpan(
	ctx context.Context,
	req *serviceports.RunRequest,
	call *serviceports.ToolCall,
	key string,
) (context.Context, trace.Span) {
	spec := &aitrace.ToolSpec{
		Anchor:   runAnchor(req),
		ToolName: call.Name,
		CallID:   call.ID,
		StepKey:  key,
	}
	if policy, ok := serviceports.PolicyOf(s.toolNamed(call.Name)); ok {
		spec.Effect = string(policy.EffectiveEffect())
		spec.Kind = string(policy.Kind)
	}
	if req.Definition != nil {
		spec.Attrs = append(spec.Attrs,
			aitrace.GenAIAgentID.String(req.Definition.ID.String()),
			aitrace.AIAgentVersion.Int64(req.Definition.Version),
		)
	}
	if req.Delegation != nil && req.Delegation.CallID != "" {
		spec.Attrs = append(spec.Attrs, aitrace.AIDelegateCallID.String(req.Delegation.CallID))
	}

	return aitrace.StartTool(ctx, spec)
}

func runAnchor(req *serviceports.RunRequest) aitrace.Anchor {
	if anchor := aitrace.ForRun(req.StepOwner, req.Delegation); anchor.IsValid() {
		return anchor
	}

	return aitrace.ForAttribution(&serviceports.AIUsageAttribution{RunID: req.RunID})
}

func stepState(state serviceports.StepState) string {
	switch state {
	case serviceports.StepFresh:
		return aitrace.StepStateFresh
	case serviceports.StepCompleted, serviceports.StepFailed:
		return aitrace.StepStateReplayed
	case serviceports.StepUnknown:
		return aitrace.StepStateUnknown
	default:
		return ""
	}
}

func finishToolSpan(span trace.Span, outcome *toolOutcome, afterExternal bool) {
	verdict := outcome.verdictOrDerived()
	attrs := make([]attribute.KeyValue, 0, 8)
	attrs = append(attrs,
		aitrace.AIOutcome.String(verdict),
		aitrace.AIAfterExternalContent.Bool(afterExternal),
	)
	if action := outcome.action; action != nil {
		attrs = append(attrs, aitrace.AITainted.Bool(action.Tainted))
		attrs = appendAttr(attrs, aitrace.AITier, string(action.Tier))
		attrs = appendAttr(attrs, aitrace.AITierSource, string(action.TierSource))
		attrs = appendAttr(attrs, aitrace.AIEgressClass, string(action.Egress))
		attrs = appendAttr(attrs, aitrace.AIProposalID, action.ProposalID.String())
		if len(action.HeldBy) > 0 {
			attrs = append(attrs, aitrace.AIHeldBy.StringSlice(action.HeldBy))
		}
	}
	span.SetAttributes(attrs...)

	if failedVerdict(verdict) {
		aitrace.MarkFailed(span, verdict)
	}
}

func failedVerdict(verdict string) bool {
	switch verdict {
	case aitrace.OutcomeFailed, aitrace.OutcomeUnknown, aitrace.OutcomeDenied,
		aitrace.OutcomeInvalid, aitrace.OutcomeOverBudget:
		return true
	default:
		return false
	}
}

func appendAttr(attrs []attribute.KeyValue, key attribute.Key, value string) []attribute.KeyValue {
	if value == "" {
		return attrs
	}

	return append(attrs, key.String(value))
}

func (o *toolOutcome) verdictOrDerived() string {
	if o.verdict != "" {
		return o.verdict
	}

	return derivedVerdict(o.failed, o.action)
}

func derivedVerdict(failed bool, action *serviceports.PendingAction) string {
	switch {
	case action != nil && action.Simulated:
		return aitrace.OutcomeSimulated
	case action != nil && action.Executed && action.ExecutionError != "":
		return aitrace.OutcomeFailed
	case action != nil && action.Executed:
		return aitrace.OutcomeRan
	case action != nil:
		return aitrace.OutcomeProposed
	case failed:
		return aitrace.OutcomeFailed
	default:
		return aitrace.OutcomeRan
	}
}

func stampAction(
	ctx context.Context,
	action *serviceports.PendingAction,
	source agent.TierSource,
	stepKey string,
) {
	action.ProposalID = pulid.MustNew("ap_")
	action.TraceID, action.SpanID = aitrace.IDs(ctx)
	action.TierSource = source
	action.StepKey = stepKey
}
