package migrations

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	assistantMessageToolVerdictUp   = "20261231007080_assistant_message_tool_verdict.tx.up.sql"
	assistantMessageToolVerdictDown = "20261231007080_assistant_message_tool_verdict.tx.down.sql"
)

func TestAssistantMessageToolVerdictMigration_AddsANullableColumn(t *testing.T) {
	t.Parallel()

	up := compactSQL(readMigration(t, assistantMessageToolVerdictUp))

	assert.Contains(t, up,
		`ALTER TABLE "assistant_messages" ADD COLUMN IF NOT EXISTS "tool_verdict" VARCHAR(50);`)
	assert.NotContains(t, up, "NOT NULL", "a result saved before the verdict was kept has none")
	assert.NotContains(t, up, "UPDATE", "existing messages are not rewritten")
	assert.Contains(t, up, `COMMENT ON COLUMN "assistant_messages"."tool_verdict"`)
}

func TestAssistantMessageToolVerdictMigration_DownDropsTheColumn(t *testing.T) {
	t.Parallel()

	down := compactSQL(readMigration(t, assistantMessageToolVerdictDown))

	assert.Contains(t, down, `DROP COLUMN IF EXISTS "tool_verdict"`)
}
