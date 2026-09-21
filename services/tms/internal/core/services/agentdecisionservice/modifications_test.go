package agentdecisionservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The form sends every field back. Only what differs from the proposal is
// a change, and a "modified" decision that changed nothing is recorded as
// the approval it was.
func TestSettleModifications_RecordsOnlyRealChangesAndAnUnchangedFormAsApproval(t *testing.T) {
	t.Parallel()

	proposal := &agent.AgentProposal{
		ID:         pulid.MustNew("ap_"),
		ToolParams: map[string]any{"workerId": "wrk_1", "message": "Call in", "priority": "medium"},
	}
	service := &Service{}

	unchanged, err := service.settleModifications(t.Context(), proposal, &services.DecideAgentProposalRequest{
		Decision:      agent.DecisionModified,
		Modifications: map[string]any{"workerId": "wrk_1", "message": "Call in", "priority": "medium"},
	}, nil)
	require.NoError(t, err)
	assert.Equal(t, agent.DecisionAccepted, unchanged.Decision)
	assert.Nil(t, unchanged.Modifications)

	changed, err := service.settleModifications(t.Context(), proposal, &services.DecideAgentProposalRequest{
		Decision:      agent.DecisionModified,
		Modifications: map[string]any{"workerId": "wrk_1", "message": "Call dispatch now", "priority": "medium"},
	}, nil)
	require.NoError(t, err)
	assert.Equal(t, agent.DecisionModified, changed.Decision)
	assert.Equal(t, map[string]any{"message": "Call dispatch now"}, changed.Modifications)
}

func TestSettleModifications_LeavesOtherDecisionsAlone(t *testing.T) {
	t.Parallel()

	req := &services.DecideAgentProposalRequest{Decision: agent.DecisionRejected, ReasonCode: "no"}
	settled, err := (&Service{}).settleModifications(t.Context(), &agent.AgentProposal{}, req, nil)

	require.NoError(t, err)
	assert.Same(t, req, settled)
}
