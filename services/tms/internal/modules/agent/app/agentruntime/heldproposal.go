package agentruntime

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentshadow"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

type shadowVerdicts interface {
	ForDefinition(
		ctx context.Context,
		tenant pagination.TenantInfo,
		definition *agentdefinition.Definition,
	) (agentshadow.Verdict, error)
}

const organizationPausedHold = "every agent in this organization is paused in AI Control, so a " +
	"person must resume agents there before anyone can approve it. Tell the person that, and " +
	"do not say it waits on their approval."

const agentShadowHold = "this agent is in shadow, so its shadow switch must be turned off in " +
	"AI Control before anyone can approve it. Tell the person that, and do not say it waits " +
	"on their approval."

func (s *Service) proposalHold(ctx context.Context, req *agentports.RunRequest) string {
	if s.shadow == nil || req.Actor == nil || req.Definition == nil {
		return ""
	}

	verdict, err := s.shadow.ForDefinition(ctx, pagination.TenantInfo{
		OrgID: req.Actor.OrganizationID,
		BuID:  req.Actor.BusinessUnitID,
	}, req.Definition)
	if err != nil {
		s.logger.Warn("could not read whether a proposal is held", zap.Error(err))
		return ""
	}

	switch verdict.Cause {
	case agentshadow.CauseOrganization:
		return organizationPausedHold
	case agentshadow.CauseDefinition:
		return agentShadowHold
	default:
		return ""
	}
}

func proposedText(
	toolName string,
	proposalID pulid.ID,
	tier agent.AutonomyTier,
	hold string,
) string {
	if hold != "" {
		return fmt.Sprintf(
			"Recorded a proposal to run %q (proposal %s) at the %s tier. It has not run, and "+
				"no one can approve it yet: %s",
			toolName, proposalID, tier, hold,
		)
	}

	return fmt.Sprintf(
		"Recorded a proposal to run %q (proposal %s). It is awaiting a person's review at "+
			"the %s tier and has not run.",
		toolName, proposalID, tier,
	)
}
