package migrations

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const assistantMessageProposalUp = "20261231022760_assistant_message_proposal"

func TestAssistantMessageProposalMigration_AddsANullableColumn(t *testing.T) {
	t.Parallel()

	up := compactSQL(readMigration(t, assistantMessageProposalUp))

	assert.Contains(t, up,
		`ALTER TABLE "assistant_messages" ADD COLUMN IF NOT EXISTS "proposal_id" VARCHAR(100);`)
	assert.NotContains(t, up, "NOT NULL", "only a schedule card an agent proposed carries one")
	assert.NotContains(t, up, "UPDATE", "existing messages are not rewritten")
	assert.Contains(t, up, `COMMENT ON COLUMN "assistant_messages"."proposal_id"`)
}
