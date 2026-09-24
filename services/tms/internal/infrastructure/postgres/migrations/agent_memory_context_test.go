package migrations

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	agentMemoryContextUp   = "20261231006200_agent_memory_context.tx.up.sql"
	agentMemoryContextDown = "20261231006200_agent_memory_context.tx.down.sql"
)

func TestAgentMemoryContextMigration_IndexesTheWordsOfEveryMemory(t *testing.T) {
	t.Parallel()

	up := compactSQL(readMigration(t, agentMemoryContextUp))

	assert.Contains(t, up, `ADD COLUMN IF NOT EXISTS "search_vector" tsvector GENERATED ALWAYS AS `+
		`( setweight(to_tsvector('simple', coalesce("subject_label", '')), 'A') || `+
		`setweight(to_tsvector('simple', coalesce("content", '')), 'B') || `+
		`setweight(to_tsvector('simple', coalesce("tool_name", '')), 'C') ) STORED`)
	assert.Contains(t, up, `CREATE INDEX IF NOT EXISTS "idx_agent_memories_search" ON `+
		`"agent_memories" USING gin ("search_vector")`)
	assert.NotContains(t, up, `to_tsvector('english'`,
		"memory keeps words as written so the prefix fallback reads the same lexemes")
}

func TestAgentMemoryContextMigration_BoundsTheTokenBudget(t *testing.T) {
	t.Parallel()

	up := compactSQL(readMigration(t, agentMemoryContextUp))

	assert.Contains(t, up, `ALTER TABLE "agent_definitions" ADD COLUMN IF NOT EXISTS `+
		`"memory_token_budget" integer;`)
	assert.Contains(t, up, `CHECK ( "memory_token_budget" IS NULL OR "memory_token_budget" `+
		`BETWEEN 1000 AND 16000 )`)
	for _, column := range []string{
		`"agent_memories"."search_vector"`,
		`"agent_definitions"."memory_token_budget"`,
	} {
		assert.Contains(t, up, "COMMENT ON COLUMN "+column, column)
	}
}

func TestAgentMemoryContextMigration_DownRemovesWhatUpAdded(t *testing.T) {
	t.Parallel()

	down := compactSQL(readMigration(t, agentMemoryContextDown))

	for _, fragment := range []string{
		`DROP CONSTRAINT IF EXISTS "chk_agent_definitions_memory_token_budget"`,
		`DROP COLUMN IF EXISTS "memory_token_budget"`,
		`DROP INDEX IF EXISTS "idx_agent_memories_search"`,
		`DROP COLUMN IF EXISTS "search_vector"`,
	} {
		assert.Contains(t, down, fragment)
	}
}
