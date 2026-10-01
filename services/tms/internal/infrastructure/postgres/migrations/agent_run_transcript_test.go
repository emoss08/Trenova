package migrations

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	agentRunTranscriptUp   = "20261231007090_agent_run_transcript.tx.up.sql"
	agentRunTranscriptDown = "20261231007090_agent_run_transcript.tx.down.sql"
)

func TestAgentRunTranscriptMigration_AddsANullableColumnToTheRun(t *testing.T) {
	t.Parallel()

	up := compactSQL(readMigration(t, agentRunTranscriptUp))

	assert.Contains(t, up, `ALTER TABLE "agent_runs" ADD COLUMN IF NOT EXISTS "transcript" JSONB;`)
	assert.NotContains(t, up, "NOT NULL", "a run filed before transcripts were kept has none")
	assert.NotContains(t, up, "CREATE TABLE", "the transcript lives and dies with its run row")
	assert.NotContains(t, up, "POLICY", "agent_runs keeps the row-level security it already has")
	assert.Contains(t, up, `COMMENT ON COLUMN "agent_runs"."transcript"`)
}

func TestAgentRunTranscriptMigration_DownDropsTheColumn(t *testing.T) {
	t.Parallel()

	down := compactSQL(readMigration(t, agentRunTranscriptDown))

	assert.Contains(t, down, `DROP COLUMN IF EXISTS "transcript"`)
}
