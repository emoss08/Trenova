package agentresolver

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReviewRequired_FollowsTheServersApprovalRule(t *testing.T) {
	t.Parallel()

	resolver := &AgentProposalResolver{&Deps{}}
	for _, tc := range []struct {
		proposal *agent.AgentProposal
		want     bool
	}{
		{proposal: &agent.AgentProposal{EgressClass: agent.EgressInternal}, want: false},
		{proposal: &agent.AgentProposal{EgressClass: agent.EgressMoney}, want: true},
		{proposal: &agent.AgentProposal{EgressClass: agent.EgressCustomerVisible}, want: true},
		{proposal: &agent.AgentProposal{EgressClass: agent.EgressInternal, Tainted: true}, want: true},
		{proposal: &agent.AgentProposal{}, want: true},
	} {
		got, err := resolver.ReviewRequired(t.Context(), tc.proposal)
		require.NoError(t, err)
		assert.Equal(t, tc.want, got, tc.proposal.EgressClass)
	}
}
