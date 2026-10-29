package proposaloutcomes

import (
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentturn"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
)

// Pending names the undecided proposals for the prompt.
func Pending(
	outcomes []agentruntimeports.ProposalOutcome,
) []agentturn.PendingProposal {
	pending := make([]agentturn.PendingProposal, 0, len(outcomes))
	for idx := range outcomes {
		outcome := &outcomes[idx]
		if outcome.Pending() {
			pending = append(pending, agentturn.PendingProposal{
				ProposalID: outcome.ProposalID,
				PlanID:     outcome.PlanID,
				ToolName:   outcome.ToolName,
				Rationale:  outcome.Rationale,
			})
		}
	}

	return pending
}
