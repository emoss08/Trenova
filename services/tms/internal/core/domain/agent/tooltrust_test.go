package agent

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAutonomyTier_Order(t *testing.T) {
	t.Parallel()

	assert.True(t, TierAutoExecute.Above(TierActWithApproval))
	assert.True(t, TierActWithApproval.Above(TierPropose))
	assert.False(t, TierPropose.Above(TierPropose))
	assert.False(t, AutonomyTier("bogus").Above(TierPropose))

	next, ok := TierPropose.Next()
	require.True(t, ok)
	assert.Equal(t, TierActWithApproval, next)
	next, ok = TierActWithApproval.Next()
	require.True(t, ok)
	assert.Equal(t, TierAutoExecute, next)
	_, ok = TierAutoExecute.Next()
	assert.False(t, ok)

	previous, ok := TierAutoExecute.Previous()
	require.True(t, ok)
	assert.Equal(t, TierActWithApproval, previous)
	_, ok = TierPropose.Previous()
	assert.False(t, ok)
}

func TestOutcomeOfDecision(t *testing.T) {
	t.Parallel()

	assert.Equal(t, TrustOutcomeApproved, OutcomeOfDecision(DecisionAccepted, nil))
	assert.Equal(t, TrustOutcomeApproved, OutcomeOfDecision(DecisionAccepted, map[string]any{}))
	// An acceptance that carried changes is a modification whatever it was called.
	assert.Equal(
		t,
		TrustOutcomeModified,
		OutcomeOfDecision(DecisionAccepted, map[string]any{"reason": "x"}),
	)
	assert.Equal(
		t,
		TrustOutcomeModified,
		OutcomeOfDecision(DecisionModified, map[string]any{"reason": "x"}),
	)
	assert.Equal(t, TrustOutcomeRejected, OutcomeOfDecision(DecisionRejected, nil))
	assert.Equal(t, TrustOutcomeRejected, OutcomeOfDecision(DecisionType("unknown"), nil))
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

	earned := &ToolTrust{EarnedTier: TierAutoExecute}
	assert.True(t, earned.HoldsEarnedTier(TierAutoExecute))
	assert.False(t, earned.HoldsEarnedTier(TierActWithApproval), "a person moved it since")
	assert.False(t, (&ToolTrust{}).HoldsEarnedTier(TierPropose), "nothing was ever earned")
}
