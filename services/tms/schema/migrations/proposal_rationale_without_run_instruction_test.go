package migrations

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const proposalRationaleUp = "20261231023225_proposal_rationale_without_run_instruction"

func TestProposalRationaleMigration_RewritesOnlyTheRuntimesOwnInstructions(t *testing.T) {
	t.Parallel()

	up := compactSQL(readMigration(t, proposalRationaleUp))

	for _, opening := range []string{
		"An event started this run",
		`This is a scheduled run\.`,
		`This is one pass of a continuous run\.`,
		"An earlier run parked this work on a wait",
		`A person started this run\.`,
	} {
		assert.Contains(t, up, opening,
			"every opening agentmessage.BackgroundRunInput writes is rewritten")
	}
	assert.Contains(t, up, `'The agent asked to \1 without explaining why.'`,
		"the same words turnhistory.ProposalRationale writes for a call with no reason")
	assert.NotContains(t, up, "DELETE", "proposals are reworded, never dropped")
}
