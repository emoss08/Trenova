package assistantproposals

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/timeutils"
)

type proposalEditStore interface {
	SetPendingModifications(
		ctx context.Context,
		req agentports.SetPendingModificationsRequest,
	) (*agent.AgentProposal, error)
}

// editable refuses a proposal no longer waiting on a decision: an edit kept
// on it would go with no approval.
func editable(proposal *agent.AgentProposal) error {
	if proposal.Status != agent.ProposalStatusPending {
		return errortypes.NewBusinessError(
			"This proposal has already been decided: it is {0}",
			strings.ToLower(string(proposal.Status)),
		)
	}
	if proposal.Expired(timeutils.NowUnix()) {
		return errortypes.NewBusinessError(
			"This proposal expired without a decision. Ask the agent again for a current one",
		)
	}

	return nil
}
