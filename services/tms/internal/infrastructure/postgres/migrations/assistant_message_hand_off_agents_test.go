package migrations

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	assistantMessageHandOffAgentsUp   = "20261231008630_assistant_message_hand_off_agents.tx.up.sql"
	assistantMessageHandOffAgentsDown = "20261231008630_assistant_message_hand_off_agents.tx.down.sql"
)

func TestAssistantMessageHandOffAgentsMigration_AddsANullableColumn(t *testing.T) {
	t.Parallel()

	up := compactSQL(readMigration(t, assistantMessageHandOffAgentsUp))

	assert.Contains(t, up,
		`ALTER TABLE "assistant_messages" ADD COLUMN IF NOT EXISTS "hand_off_agents" JSONB;`)
	assert.NotContains(t, up, "NOT NULL", "only a find_tools result that named an agent carries one")
	assert.NotContains(t, up, "UPDATE", "existing messages are not rewritten")
	assert.Contains(t, up, `COMMENT ON COLUMN "assistant_messages"."hand_off_agents"`)
}

func TestAssistantMessageHandOffAgentsMigration_DownDropsTheColumn(t *testing.T) {
	t.Parallel()

	down := compactSQL(readMigration(t, assistantMessageHandOffAgentsDown))

	assert.Contains(t, down, `DROP COLUMN IF EXISTS "hand_off_agents"`)
}
