package proposalexecutor

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const (
	executeStaleTarget = "stale_target"
	executeTenant      = "tenant_mismatch"
)

func startExecute(
	ctx context.Context,
	proposal *agent.AgentProposal,
	modifications map[string]any,
	actor *services.RequestActor,
) (context.Context, trace.Span) {
	return aitrace.StartDecide(ctx, &aitrace.DecideSpec{
		Operation:         aitrace.DecideOperationExecute,
		OrganizationID:    proposal.OrganizationID,
		BusinessUnitID:    proposal.BusinessUnitID,
		ProposalID:        proposal.ID,
		RunID:             proposal.RunID,
		ToolName:          proposal.ToolName,
		UserID:            executorOf(actor),
		ModificationCount: len(modifications),
		ProposalTraceID:   proposal.TraceID,
		ProposalSpanID:    proposal.SpanID,
	})
}

func startTool(
	ctx context.Context,
	proposal *agent.AgentProposal,
	policy services.ToolPolicy,
) (context.Context, trace.Span) {
	attrs := make([]attribute.KeyValue, 0, 6)
	attrs = append(attrs,
		aitrace.AIProposalID.String(proposal.ID.String()),
		aitrace.AITier.String(string(proposal.AutonomyTier)),
		aitrace.AITainted.Bool(proposal.Tainted),
	)
	if proposal.EgressClass != "" {
		attrs = append(attrs, aitrace.AIEgressClass.String(string(proposal.EgressClass)))
	}
	if len(proposal.HeldBy) > 0 {
		attrs = append(attrs, aitrace.AIHeldBy.StringSlice(proposal.HeldBy))
	}

	return aitrace.StartTool(ctx, &aitrace.ToolSpec{
		ToolName: proposal.ToolName,
		Effect:   string(policy.EffectiveEffect()),
		Kind:     string(policy.Kind),
		StepKey:  proposal.StepKey,
		Attrs:    attrs,
	})
}

func finishTool(span trace.Span, outcome string) {
	span.SetAttributes(aitrace.AIOutcome.String(outcome))
	if outcome == aitrace.OutcomeFailed {
		aitrace.MarkFailed(span, outcome)
	}
}

func startWrite(
	ctx context.Context,
	proposal *agent.AgentProposal,
	simulated bool,
) (context.Context, trace.Span) {
	spec := &aitrace.WriteSpec{
		EntityType: proposal.TargetResource,
		ProposalID: proposal.ID,
		Simulated:  simulated,
	}
	if proposal.TargetID.IsNotNil() {
		before := proposal.TargetVersion
		spec.EntityID = proposal.TargetID.String()
		spec.VersionBefore = &before
	}

	return aitrace.StartWrite(ctx, spec)
}

func executorOf(actor *services.RequestActor) pulid.ID {
	if actor == nil || actor.PrincipalType != services.PrincipalTypeUser {
		return pulid.Nil
	}

	return actor.UserID
}

func executeFailure(err error) string {
	switch {
	case errors.Is(err, ErrTenantMismatch):
		return executeTenant
	case errors.Is(err, ErrToolMissing):
		return aitrace.OutcomeInvalid
	case errors.Is(err, ErrTargetChanged):
		return executeStaleTarget
	case errors.Is(err, ErrTaintedNeedsPerson):
		return aitrace.OutcomeDenied
	case errors.Is(err, ErrBudgetSpent):
		return aitrace.OutcomeOverBudget
	case errortypes.IsMultiError(err):
		return aitrace.OutcomeInvalid
	default:
		return aitrace.OutcomeFailed
	}
}
