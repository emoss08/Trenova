package agentproposalrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/stretchr/testify/assert"
)

// Deciding a proposal ends the wording a person had not yet approved with;
// an approval still in its undo window keeps it, so taking the approval back
// leaves the edit where it was.
func TestClearsPendingModifications(t *testing.T) {
	t.Parallel()

	for _, status := range []agent.ProposalStatus{
		agent.ProposalStatusAccepted,
		agent.ProposalStatusModified,
		agent.ProposalStatusRejected,
		agent.ProposalStatusExpired,
		agent.ProposalStatusSuperseded,
		agent.ProposalStatusSkipped,
	} {
		assert.True(t, clearsPendingModifications(status), status)
	}

	assert.False(t, clearsPendingModifications(agent.ProposalStatusPending))
	assert.False(t, clearsPendingModifications(agent.ProposalStatusApproving))
}
