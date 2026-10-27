package migrations

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	agentProposalPendingModificationsUp = "20261231008120_agent_proposal_pending_modifications"
)

func TestAgentProposalPendingModificationsMigration_AddsANullableColumn(t *testing.T) {
	t.Parallel()

	up := compactSQL(readMigration(t, agentProposalPendingModificationsUp))

	assert.Contains(t, up, `ALTER TABLE "agent_proposals"`)
	assert.Contains(t, up, `ADD COLUMN IF NOT EXISTS "pending_modifications" jsonb;`)
	assert.NotContains(t, up, "NOT NULL", "a proposal nobody edited has no pending wording")
}
