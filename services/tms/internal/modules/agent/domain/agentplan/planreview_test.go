package agentplan

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/stretchr/testify/assert"
)

func TestPlanNeedsReview_WhenAStepStillWaitingMustBeReviewed(t *testing.T) {
	t.Parallel()

	internal := &agent.AgentProposal{
		Status:      agent.ProposalStatusPending,
		EgressClass: agent.EgressInternal,
	}
	money := &agent.AgentProposal{
		Status:      agent.ProposalStatusPending,
		EgressClass: agent.EgressMoney,
	}
	ranMoney := &agent.AgentProposal{
		Status:      agent.ProposalStatusExecuted,
		EgressClass: agent.EgressMoney,
	}
	tainted := &agent.AgentProposal{
		Status:      agent.ProposalStatusPending,
		EgressClass: agent.EgressInternal,
		Tainted:     true,
	}

	assert.False(t, PlanNeedsReview([]*agent.AgentProposal{internal, internal}))
	assert.True(t, PlanNeedsReview([]*agent.AgentProposal{internal, money}))
	assert.False(
		t,
		PlanNeedsReview([]*agent.AgentProposal{internal, ranMoney}),
		"a step that already ran waits on nothing",
	)
	assert.True(t, PlanNeedsReview([]*agent.AgentProposal{tainted}))
	assert.False(t, PlanNeedsReview([]*agent.AgentProposal{nil}))
	assert.False(t, PlanNeedsReview(nil))
}
