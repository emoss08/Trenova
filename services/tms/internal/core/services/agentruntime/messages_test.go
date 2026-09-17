package agentruntime

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProposalRationale_UsesTheModelsOwnWordsWhenItGaveAny(t *testing.T) {
	t.Parallel()

	rationale := proposalRationale(
		"  Reassigning to the Dallas terminal because the driver is out of hours.  ",
		"reassign_move",
	)

	assert.Equal(t, "Reassigning to the Dallas terminal because the driver is out of hours.", rationale)
}

func TestProposalRationale_SaysSoWhenTheModelExplainedNothing(t *testing.T) {
	t.Parallel()

	rationale := proposalRationale("   ", "reassign_move")

	assert.Contains(t, rationale, "reassign_move")
	assert.Contains(t, rationale, "without explaining why")
}

func TestProposalRationale_TruncatesNarration(t *testing.T) {
	t.Parallel()

	long := make([]rune, maxRationaleChars*2)
	for i := range long {
		long[i] = 'a'
	}

	rationale := proposalRationale(string(long), "reassign_move")

	assert.LessOrEqual(t, len([]rune(rationale)), maxRationaleChars+1)
}
