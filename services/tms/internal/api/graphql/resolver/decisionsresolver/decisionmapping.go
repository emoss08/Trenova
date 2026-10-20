package decisionsresolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/loaders"
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
	if paused {
		return &gqlmodel.PendingDecisionHold{
			Reason: gqlmodel.PendingDecisionHoldReasonOrganizationPaused,
		}, nil
	}

	definition, err := loadRunDefinition(ctx, runID)
	if err != nil {
		return nil, err
	}

	return shadowHold(definition), nil
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

func shadowHold(definition *agentdefinition.Definition) *gqlmodel.PendingDecisionHold {
	if definition == nil || !definition.EffectiveShadow(false) {
		return nil
	}

	return &gqlmodel.PendingDecisionHold{
		Reason:    gqlmodel.PendingDecisionHoldReasonAgentShadow,
		AgentName: definition.Name,
	}
}
