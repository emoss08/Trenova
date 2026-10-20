package decisionsresolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/loaders"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentshadow"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

// loadAgentRun reads the run behind a proposal or plan through the
// per-request loader, so a page of rows asks once. A run that is gone leaves
// the field empty rather than failing the row.
func loadAgentRun(ctx context.Context, runID pulid.ID) (*agent.AgentRun, error) {
	if runID.IsNil() {
		return nil, nil
	}

	if !loaders.Available(ctx) {
		return nil, nil
	}

	run, err := loaders.AgentRunByID(ctx, runID.String())
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, nil
		}

		return nil, err
	}

	return run, nil
}

func pendingDecisionHold(hold *agentports.ProposalHold) *gqlmodel.PendingDecisionHold {
	if hold == nil {
		return nil
	}

	return &gqlmodel.PendingDecisionHold{
		Reason:    gqlmodel.PendingDecisionHoldReason(hold.Reason),
		AgentName: hold.AgentName,
	}
}

func loadHold(
	ctx context.Context,
	organizationID, runID pulid.ID,
) (*gqlmodel.PendingDecisionHold, error) {
	if !loaders.Available(ctx) {
		return nil, nil //nolint:nilnil // without loaders nothing is known to hold it
	}

	paused, err := loaders.AgentsPaused(ctx, organizationID.String())
	if err != nil {
		return nil, err
	}
	definition := (*agentdefinition.Definition)(nil)
	if !paused {
		if definition, err = loadRunDefinition(ctx, runID); err != nil {
			return nil, err
		}
	}

	return pendingDecisionHold(agentshadow.VerdictFor(paused, definition).Hold()), nil
}

func loadRunDefinition(ctx context.Context, runID pulid.ID) (*agentdefinition.Definition, error) {
	run, err := loadAgentRun(ctx, runID)
	if err != nil || run == nil || run.AgentDefinitionID.IsNil() {
		return nil, err
	}

	definition, err := loaders.AgentDefinitionByID(ctx, run.AgentDefinitionID.String())
	if err != nil && errortypes.IsNotFoundError(err) {
		return nil, nil //nolint:nilnil // a deleted agent follows the organization's switch alone
	}

	return definition, err
}

func proposalHold(
	ctx context.Context,
	proposal *agent.AgentProposal,
) (*gqlmodel.PendingDecisionHold, error) {
	if proposal.Status != agent.ProposalStatusPending {
		return nil, nil //nolint:nilnil // a decided proposal is held by nothing
	}

	return loadHold(ctx, proposal.OrganizationID, proposal.RunID)
}

func planHold(ctx context.Context, plan *agent.AgentPlan) (*gqlmodel.PendingDecisionHold, error) {
	if plan.Status != agent.PlanStatusPending || !loaders.Available(ctx) {
		return nil, nil //nolint:nilnil // a decided plan is held by nothing
	}

	hold, err := loaders.AgentPlanHold(ctx, plan.ID.String())
	if err != nil {
		return nil, err
	}

	return pendingDecisionHold(hold), nil
}

func pendingDecisionSummary(
	summary *agentports.PendingDecisionSummary,
	actionable int,
) *gqlmodel.PendingDecisionSummary {
	out := &gqlmodel.PendingDecisionSummary{
		Total:   summary.Total,
		Held:    max(summary.Total-actionable, 0),
		ByAgent: make([]*gqlmodel.PendingDecisionAgentCount, 0, len(summary.ByAgent)),
		ByTool:  make([]*gqlmodel.PendingDecisionToolCount, 0, len(summary.ByTool)),
	}
	for _, row := range summary.ByAgent {
		out.ByAgent = append(out.ByAgent, &gqlmodel.PendingDecisionAgentCount{
			AgentDefinitionID: row.AgentDefinitionID.String(),
			AgentName:         row.AgentName,
			Count:             row.Count,
		})
	}
	for _, row := range summary.ByTool {
		out.ByTool = append(out.ByTool, &gqlmodel.PendingDecisionToolCount{
			ToolName: row.ToolName,
			Count:    row.Count,
		})
	}
	if summary.OldestAt != nil {
		oldest := int(*summary.OldestAt)
		out.OldestAt = &oldest
	}

	return out
}
