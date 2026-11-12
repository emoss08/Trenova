package agenttoolpolicy_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolpolicy"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/stretchr/testify/assert"
)

func TestDecide_AnAgentThatRunsOnItsOwnNeverWritesUnaskedInAConversation(t *testing.T) {
	t.Parallel()

	decide := func(trigger agentdefinition.TriggerMode, conversation bool) agenttoolpolicy.Decision {
		desk := definition(agent.TierAutoExecute, map[string]agent.AutonomyTier{
			"dispatch_assign_move": agent.TierAutoExecute,
		})
		desk.TriggerMode = trigger

		return agenttoolpolicy.Decide(t.Context(), agenttoolpolicy.DecideInput{
			Policy:          registeredPolicy(t, "dispatch_assign_move"),
			Params:          call(person(), nil),
			Definition:      desk,
			TierSetByPerson: true,
			Taint:           clean(),
			Conversation:    conversation,
		})
	}

	onItsOwn := decide(agentdefinition.TriggerEvent, false)
	assert.Equal(t, agent.TierAutoExecute, onItsOwn.Tier, "its own run keeps the tier it holds")

	asked := decide(agentdefinition.TriggerEvent, true)
	assert.Equal(t, agent.TierActWithApproval, asked.Tier)
	assert.Contains(t, asked.HeldBy, agenttoolpolicy.HeldByConversation)

	scheduled := decide(agentdefinition.TriggerScheduled, true)
	assert.Equal(t, agent.TierActWithApproval, scheduled.Tier)

	chat := decide(agentdefinition.TriggerChat, true)
	assert.Equal(t, agent.TierAutoExecute, chat.Tier, "a chat agent's tiers were set for chat")
	assert.NotContains(t, chat.HeldBy, agenttoolpolicy.HeldByConversation)
}
