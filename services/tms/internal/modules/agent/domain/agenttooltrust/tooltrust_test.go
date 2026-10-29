package agenttooltrust

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdecision"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAutonomyTier_Order(t *testing.T) {
	t.Parallel()

	assert.True(t, agent.TierAutoExecute.Above(agent.TierActWithApproval))
	assert.True(t, agent.TierActWithApproval.Above(agent.TierPropose))
	assert.False(t, agent.TierPropose.Above(agent.TierPropose))
	assert.False(t, agent.AutonomyTier("bogus").Above(agent.TierPropose))

	next, ok := agent.TierPropose.Next()
	require.True(t, ok)
	assert.Equal(t, agent.TierActWithApproval, next)
	next, ok = agent.TierActWithApproval.Next()
	require.True(t, ok)
	assert.Equal(t, agent.TierAutoExecute, next)
	_, ok = agent.TierAutoExecute.Next()
	assert.False(t, ok)

	previous, ok := agent.TierAutoExecute.Previous()
	require.True(t, ok)
	assert.Equal(t, agent.TierActWithApproval, previous)
	_, ok = agent.TierPropose.Previous()
	assert.False(t, ok)
}

func TestOutcomeOfDecision(t *testing.T) {
	t.Parallel()

	assert.Equal(t, TrustOutcomeApproved, OutcomeOfDecision(agentdecision.DecisionAccepted, nil))
	assert.Equal(
		t,
		TrustOutcomeApproved,
		OutcomeOfDecision(agentdecision.DecisionAccepted, map[string]any{}),
	)
	// An acceptance that carried changes is a modification whatever it was called.
	assert.Equal(
		t,
		TrustOutcomeModified,
		OutcomeOfDecision(agentdecision.DecisionAccepted, map[string]any{"reason": "x"}),
	)
	assert.Equal(
		t,
		TrustOutcomeModified,
		OutcomeOfDecision(agentdecision.DecisionModified, map[string]any{"reason": "x"}),
	)
	assert.Equal(t, TrustOutcomeRejected, OutcomeOfDecision(agentdecision.DecisionRejected, nil))
	assert.Equal(
		t,
		TrustOutcomeRejected,
		OutcomeOfDecision(agentdecision.DecisionType("unknown"), nil),
	)
}

func TestTrustOutcome_Flags(t *testing.T) {
	t.Parallel()

	assert.True(t, TrustOutcomeApproved.Clean())
	assert.False(t, TrustOutcomeModified.Clean())

	assert.False(t, TrustOutcomeApproved.Setback())
	assert.False(t, TrustOutcomeModified.Setback())
	assert.True(t, TrustOutcomeRejected.Setback())
	assert.True(t, TrustOutcomeExecutionFailed.Setback())

	assert.False(t, TrustOutcome("nope").IsValid())
}

func TestToolTrust_Promotion(t *testing.T) {
	t.Parallel()

	row := &ToolTrust{Streak: 10}
	assert.True(t, row.ReadyForPromotion(10))
	assert.False(t, row.ReadyForPromotion(11))
	assert.False(t, row.ReadyForPromotion(0), "a threshold of zero never promotes")

	earned := &ToolTrust{EarnedTier: agent.TierAutoExecute}
	assert.True(t, earned.HoldsEarnedTier(agent.TierAutoExecute))
	assert.False(t, earned.HoldsEarnedTier(agent.TierActWithApproval), "a person moved it since")
	assert.False(t, (&ToolTrust{}).HoldsEarnedTier(agent.TierPropose), "nothing was ever earned")
}
