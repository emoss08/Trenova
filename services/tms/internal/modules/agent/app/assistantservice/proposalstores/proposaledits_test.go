package proposalstores

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

// The list carries what was saved while the proposal waits, and nothing once
// it is decided: the decision's own modifications say what was approved.
func TestToAssistantProposal_CarriesPendingModificationsOnlyWhilePending(t *testing.T) {
	t.Parallel()

	pending := ToAssistantProposal(&agent.AgentProposal{
		ID:                   pulid.MustNew("ap_"),
		Status:               agent.ProposalStatusPending,
		PendingModifications: map[string]any{"subject": "PO needed"},
	}, nil)
	assert.Equal(t, map[string]any{"subject": "PO needed"}, pending.PendingModifications)

	decided := ToAssistantProposal(&agent.AgentProposal{
		ID:                   pulid.MustNew("ap_"),
		Status:               agent.ProposalStatusAccepted,
		PendingModifications: map[string]any{"subject": "PO needed"},
	}, nil)
	assert.Nil(t, decided.PendingModifications)
}

func TestToAssistantProposal_SaysWhenApprovingNeedsThePreviewShown(t *testing.T) {
	t.Parallel()

	for class, want := range map[agent.EgressClass]bool{
		agent.EgressInternal:          false,
		agent.EgressPersonal:          false,
		agent.EgressMoney:             true,
		agent.EgressCustomerVisible:   true,
		agent.EgressDriverVisible:     true,
		agent.EgressExternalRecipient: true,
		"":                            true,
	} {
		got := ToAssistantProposal(&agent.AgentProposal{EgressClass: class}, nil)
		assert.Equal(t, want, got.ReviewRequired, class)
	}

	tainted := ToAssistantProposal(&agent.AgentProposal{
		EgressClass: agent.EgressInternal,
		Tainted:     true,
	}, nil)
	assert.True(t, tainted.ReviewRequired, "a tainted run's write is reviewed whatever it reaches")
}
