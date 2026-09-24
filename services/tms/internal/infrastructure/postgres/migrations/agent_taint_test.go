package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	agentTaintUp   = "20261231005900_agent_taint.tx.up.sql"
	agentTaintDown = "20261231005900_agent_taint.tx.down.sql"
)

func TestAgentTaintMigration_AddsTheTaintColumns(t *testing.T) {
	t.Parallel()

	up := compactSQL(readMigration(t, agentTaintUp))

	assert.Contains(t, up, `ALTER TABLE "agent_runs" ADD COLUMN IF NOT EXISTS "tainted" boolean `+
		`NOT NULL DEFAULT false, ADD COLUMN IF NOT EXISTS "taint" jsonb, ADD COLUMN IF NOT `+
		`EXISTS "tainted_at" bigint`)
	assert.Contains(t, up, `ALTER TABLE "agent_proposals" ADD COLUMN IF NOT EXISTS "tainted" `+
		`boolean NOT NULL DEFAULT false, ADD COLUMN IF NOT EXISTS "taint" jsonb, ADD COLUMN IF `+
		`NOT EXISTS "egress_class" varchar(30), ADD COLUMN IF NOT EXISTS "held_by" text[] NOT `+
		`NULL DEFAULT '{}'`)
	assert.Contains(t, up, `ALTER TABLE "assistant_threads" ADD COLUMN IF NOT EXISTS "taint" `+
		`jsonb, ADD COLUMN IF NOT EXISTS "tainted_at" bigint`)
	assert.Contains(t, up, `ALTER TABLE "agent_memories" ADD COLUMN IF NOT EXISTS "scope" `+
		`varchar(20) NOT NULL DEFAULT 'Organization', ADD COLUMN IF NOT EXISTS "tainted" `+
		`boolean NOT NULL DEFAULT false, ADD COLUMN IF NOT EXISTS "taint_run_id" varchar(100)`)
}

func TestAgentTaintMigration_IndexesOnlyTaintedProposals(t *testing.T) {
	t.Parallel()

	up := compactSQL(readMigration(t, agentTaintUp))

	assert.Contains(t, up, `CREATE INDEX IF NOT EXISTS "idx_agent_proposals_tainted" ON `+
		`"agent_proposals" ("organization_id", "business_unit_id", "created_at" DESC) `+
		`WHERE "tainted"`)
}

func TestAgentTaintMigration_CommentsEveryNewColumn(t *testing.T) {
	t.Parallel()

	up := readMigration(t, agentTaintUp)

	for _, column := range []string{
		`"agent_runs"."tainted"`,
		`"agent_runs"."taint"`,
		`"agent_runs"."tainted_at"`,
		`"agent_proposals"."tainted"`,
		`"agent_proposals"."taint"`,
		`"agent_proposals"."egress_class"`,
		`"agent_proposals"."held_by"`,
		`"assistant_threads"."taint"`,
		`"assistant_threads"."tainted_at"`,
		`"agent_memories"."scope"`,
		`"agent_memories"."tainted"`,
		`"agent_memories"."taint_run_id"`,
	} {
		assert.Contains(t, up, "COMMENT ON COLUMN "+column, column)
	}
}

func TestAgentTaintMigration_ScopesOnlyApprovedFeedbackToItsAgent(t *testing.T) {
	t.Parallel()

	up := compactSQL(readMigration(t, agentTaintUp))

	assert.Contains(t, up, `UPDATE "agent_memories" SET "scope" = 'Agent' WHERE "source" = `+
		`'Feedback' AND "agent_definition_id" IS NOT NULL;`,
		"a memory an agent wrote keeps reaching every agent; only feedback is narrowed")
	assert.Contains(t, up, `("scope" = 'Organization' OR "agent_definition_id" IS NOT NULL)`,
		"a memory kept for one agent names that agent")
}

func TestAgentTaintMigration_ClampsToolsWhoseCeilingFell(t *testing.T) {
	t.Parallel()

	up := compactSQL(readMigration(t, agentTaintUp))

	for _, tool := range []string{"schedule_report", "create_shipment"} {
		assert.Contains(t, up, `SET "tool_tiers" = "tool_tiers" || jsonb_build_object('`+tool+
			`', 'ActWithApproval') WHERE "tool_tiers" ->> '`+tool+`' = 'AutoExecute';`, tool)
	}
	assert.Contains(t, up, `UPDATE "agent_tool_trust" SET "earned_tier" = 'ActWithApproval'`)
	assert.Contains(t, up, `WHERE "tool_name" IN ('schedule_report', 'create_shipment') AND `+
		`"earned_tier" = 'AutoExecute';`)
}

func TestAgentTaintMigration_DownRemovesWhatUpAdded(t *testing.T) {
	t.Parallel()

	down := compactSQL(readMigration(t, agentTaintDown))

	for _, fragment := range []string{
		`DROP INDEX IF EXISTS "idx_agent_memories_agent_scope"`,
		`DROP INDEX IF EXISTS "idx_agent_proposals_tainted"`,
		`DROP CONSTRAINT IF EXISTS "chk_agent_memories_scope"`,
		`DROP CONSTRAINT IF EXISTS "chk_agent_proposals_egress_class"`,
		`DROP COLUMN IF EXISTS "held_by"`,
		`DROP COLUMN IF EXISTS "egress_class"`,
		`DROP COLUMN IF EXISTS "taint_run_id"`,
		`DROP COLUMN IF EXISTS "scope"`,
		`DROP COLUMN IF EXISTS "tainted_at"`,
	} {
		assert.Contains(t, down, fragment)
	}
	assert.Equal(t, 3, strings.Count(down, `DROP COLUMN IF EXISTS "tainted",`)+
		strings.Count(down, `DROP COLUMN IF EXISTS "tainted";`),
		"agent_runs, agent_proposals and agent_memories each lose tainted")
}
