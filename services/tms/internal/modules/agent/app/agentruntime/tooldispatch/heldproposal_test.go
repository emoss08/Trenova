package tooldispatch

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

/*
A proposal filed while every agent was paused said no one could approve it
yet. Once someone has, the turn that follows is told the hold was lifted, read
from that same result, so it does not take back what was true when it said it.
*/
func TestLiftedHold_ReadsTheHoldTheFilingResultNamed(t *testing.T) {
	t.Parallel()

	proposalID := pulid.MustNew("ap_")
	paused := proposedText("agent_create_schedule", proposalID, agent.TierPropose,
		organizationPausedHold)
	shadowed := proposedText("agent_create_schedule", proposalID, agent.TierPropose,
		agentShadowHold)
	plain := proposedText("agent_create_schedule", proposalID, agent.TierPropose, "")

	assert.Contains(t, LiftedHold(paused, proposalID), "agents have been resumed since")
	assert.Contains(t, LiftedHold(shadowed, proposalID), "its shadow switch has been turned off")
	assert.Empty(t, LiftedHold(plain, proposalID), "a proposal nothing held says nothing")
	assert.Empty(
		t,
		LiftedHold(paused, pulid.MustNew("ap_")),
		"another proposal's hold is not this one's",
	)
}
