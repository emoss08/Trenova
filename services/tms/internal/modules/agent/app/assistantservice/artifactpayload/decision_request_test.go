package artifactpayload

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/stretchr/testify/assert"
)

func TestDecisionStatus_FollowsTheProposal(t *testing.T) {
	t.Parallel()

	assert.Equal(t, assistantartifact.StatusPending, DecisionStatus(agent.ProposalStatusPending))
	assert.Equal(t, assistantartifact.StatusReady, DecisionStatus(agent.ProposalStatusExecuted))
	assert.Equal(t, assistantartifact.StatusFailed, DecisionStatus(agent.ProposalStatusRejected))
}
