package tooldispatch

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentshadow"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

type ShadowVerdicts interface {
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

// liftedHolds pairs each hold a filed proposal's result names with what is
// true once the proposal was approved anyway: approval is refused while the
// hold is on (agentdecisionservice), so the hold was lifted in between. The
// turn that follows the approval is told so. Told only "approved, and it ran",
// an agent read its earlier, true, "no one can approve it yet" as a mistake
// and took it back to the person.
var liftedHolds = []struct{ hold, lifted string }{
	{
		hold: organizationPausedHold,
		lifted: "When you proposed it, every agent in this organization was paused in AI " +
			"Control, so no one could approve it then; agents have been resumed since, which " +
			"is how it was approved.",
	},
	{
		hold: agentShadowHold,
		lifted: "When you proposed it, this agent was in shadow, so no one could approve it " +
			"then; its shadow switch has been turned off since, which is how it was approved.",
	},
}

// LiftedHold reads the result of the call that filed a proposal. When it said
// no one could approve the proposal yet, it is what changed before someone
// did; otherwise it is empty.
func LiftedHold(toolResult string, proposalID pulid.ID) string {
	if proposalID.IsNil() || !strings.Contains(toolResult, "(proposal "+proposalID.String()+")") {
		return ""
	}
	for _, held := range liftedHolds {
		if strings.Contains(toolResult, held.hold) {
			return held.lifted
		}
	}

	return ""
}

func (s *Dispatcher) proposalHold(ctx context.Context, req *agentruntimeports.RunRequest) string {
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
