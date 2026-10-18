package agent

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPlanNeedsReview_WhenAStepStillWaitingMustBeReviewed(t *testing.T) {
	t.Parallel()

	internal := &AgentProposal{Status: ProposalStatusPending, EgressClass: EgressInternal}
	money := &AgentProposal{Status: ProposalStatusPending, EgressClass: EgressMoney}
	ranMoney := &AgentProposal{Status: ProposalStatusExecuted, EgressClass: EgressMoney}
	tainted := &AgentProposal{Status: ProposalStatusPending, EgressClass: EgressInternal, Tainted: true}

	assert.False(t, PlanNeedsReview([]*AgentProposal{internal, internal}))
	assert.True(t, PlanNeedsReview([]*AgentProposal{internal, money}))
	assert.False(t, PlanNeedsReview([]*AgentProposal{internal, ranMoney}), "a step that already ran waits on nothing")
	assert.True(t, PlanNeedsReview([]*AgentProposal{tainted}))
	assert.False(t, PlanNeedsReview([]*AgentProposal{nil}))
	assert.False(t, PlanNeedsReview(nil))
}
