package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	aiTraceLinksUp     = "20261231006700_ai_trace_links.tx.up.sql"
	aiTraceLinksDown   = "20261231006700_ai_trace_links.tx.down.sql"
	aiTraceIndexesUp   = "20261231006710_ai_trace_link_indexes.up.sql"
	aiTraceIndexesDown = "20261231006710_ai_trace_link_indexes.down.sql"
)

func splitStatements(body string) []string {
	parts := strings.Split(body, "--bun:split")
	statements := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			statements = append(statements, compactSQL(trimmed))
		}
	}

	return statements
}

func TestAITraceLinksMigration_AddsOnlyOptionalColumns(t *testing.T) {
	t.Parallel()

	up := compactSQL(readMigration(t, aiTraceLinksUp))

	for _, fragment := range []string{
		`ALTER TABLE "agent_run_steps" ADD COLUMN IF NOT EXISTS "trace_id" varchar(32), ` +
			`ADD COLUMN IF NOT EXISTS "span_id" varchar(16), ` +
			`ADD COLUMN IF NOT EXISTS "agent_definition_id" varchar(100), ` +
			`ADD COLUMN IF NOT EXISTS "agent_definition_version" bigint, ` +
			`ADD COLUMN IF NOT EXISTS "delegate_call_id" varchar(200);`,
		`ALTER TABLE "agent_proposals" ADD COLUMN IF NOT EXISTS "trace_id" varchar(32), ` +
			`ADD COLUMN IF NOT EXISTS "span_id" varchar(16), ` +
			`ADD COLUMN IF NOT EXISTS "step_key" varchar(120), ` +
			`ADD COLUMN IF NOT EXISTS "executed_by_user_id" varchar(100), ` +
			`ADD COLUMN IF NOT EXISTS "executed_target_version" bigint;`,
		`ALTER TABLE "agent_decisions" ADD COLUMN IF NOT EXISTS "trace_id" varchar(32);`,
		`ALTER TABLE "agent_runs" ADD COLUMN IF NOT EXISTS "trace_id" varchar(32), ` +
			`ADD COLUMN IF NOT EXISTS "turn_id" varchar(100), ` +
			`ADD COLUMN IF NOT EXISTS "parent_owner_kind" varchar(20), ` +
			`ADD COLUMN IF NOT EXISTS "parent_owner_id" varchar(100), ` +
			`ADD COLUMN IF NOT EXISTS "delegate_call_id" varchar(200);`,
		`ALTER TABLE "assistant_turns" ADD COLUMN IF NOT EXISTS "trace_id" varchar(32);`,
		`ADD COLUMN IF NOT EXISTS "attempt" integer,`,
		`ADD COLUMN IF NOT EXISTS "failover" boolean NOT NULL DEFAULT false,`,
		`ADD COLUMN IF NOT EXISTS "cache_read_tokens" integer NOT NULL DEFAULT 0,`,
		`ADD COLUMN IF NOT EXISTS "cache_write_tokens" integer NOT NULL DEFAULT 0;`,
		`ADD COLUMN IF NOT EXISTS "ai_audit_retention_period" integer NOT NULL DEFAULT 2555;`,
		`CHECK ("ai_audit_retention_period" >= 365)`,
	} {
		assert.Contains(t, up, fragment)
	}
}

func TestAITraceLinksMigration_ChecksLargeTablesWithoutScanningThem(t *testing.T) {
	t.Parallel()

	up := compactSQL(readMigration(t, aiTraceLinksUp))
	indexes := compactSQL(readMigration(t, aiTraceIndexesUp))

	for _, constraint := range []string{
		`ADD CONSTRAINT "ck_agent_runs_parent_owner_kind" CHECK ( "parent_owner_kind" IS NULL ` +
			`OR "parent_owner_kind" IN ('AgentRun', 'AssistantTurn') ) NOT VALID;`,
		`ADD CONSTRAINT "ck_agent_runs_parent_owner_pair" CHECK ( ("parent_owner_kind" IS NULL) = ` +
			`("parent_owner_id" IS NULL) AND ("delegate_call_id" IS NULL OR "parent_owner_id" IS NOT NULL) ) NOT VALID;`,
		`ADD CONSTRAINT "ck_ai_usage_records_owner_kind" CHECK ( "owner_kind" IS NULL ` +
			`OR "owner_kind" IN ('AgentRun', 'AssistantTurn') ) NOT VALID;`,
	} {
		assert.Contains(t, up, constraint)
	}

	for _, validate := range []string{
		`ALTER TABLE "agent_runs" VALIDATE CONSTRAINT "ck_agent_runs_parent_owner_kind";`,
		`ALTER TABLE "agent_runs" VALIDATE CONSTRAINT "ck_agent_runs_parent_owner_pair";`,
		`ALTER TABLE "ai_usage_records" VALIDATE CONSTRAINT "ck_ai_usage_records_owner_kind";`,
	} {
		assert.Contains(t, indexes, validate)
	}
}

func TestAITraceLinkIndexesMigration_BuildsEachIndexConcurrentlyOnItsOwn(t *testing.T) {
	t.Parallel()

	statements := splitStatements(readMigration(t, aiTraceIndexesUp))
	expected := []string{
		`CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_agent_run_steps_updated" ON "agent_run_steps"("updated_at", "id");`,
		`CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_agent_proposals_updated" ON "agent_proposals"("updated_at", "id");`,
		`CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_agent_decisions_created" ON "agent_decisions"("created_at", "id");`,
		`CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_agent_runs_updated" ON "agent_runs"("updated_at", "id");`,
		`CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_assistant_turns_updated" ON "assistant_turns"("updated_at", "id");`,
		`CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_ai_usage_records_created" ON "ai_usage_records"("created_at", "id");`,
	}

	for _, statement := range expected {
		assert.Contains(t, statements, statement)
	}
	for _, statement := range statements {
		assert.Equal(t, 1, strings.Count(statement, ";"),
			"a concurrent index build cannot share a statement batch: %s", statement)
	}
}

func TestAITraceLinksMigration_DownRemovesWhatUpAdded(t *testing.T) {
	t.Parallel()

	down := compactSQL(readMigration(t, aiTraceLinksDown))
	for _, fragment := range []string{
		`DROP CONSTRAINT IF EXISTS "ck_data_retention_ai_audit_retention_period"`,
		`DROP COLUMN IF EXISTS "ai_audit_retention_period"`,
		`DROP CONSTRAINT IF EXISTS "ck_ai_usage_records_owner_kind"`,
		`DROP COLUMN IF EXISTS "cache_write_tokens"`,
		`DROP CONSTRAINT IF EXISTS "ck_agent_runs_parent_owner_pair"`,
		`DROP CONSTRAINT IF EXISTS "ck_agent_runs_parent_owner_kind"`,
		`DROP COLUMN IF EXISTS "parent_owner_kind"`,
		`DROP COLUMN IF EXISTS "executed_target_version"`,
		`DROP COLUMN IF EXISTS "agent_definition_version"`,
	} {
		assert.Contains(t, down, fragment)
	}

	indexesDown := splitStatements(readMigration(t, aiTraceIndexesDown))
	require.Len(t, indexesDown, 6)
	for _, statement := range indexesDown {
		assert.True(
			t,
			strings.HasPrefix(statement, "DROP INDEX CONCURRENTLY IF EXISTS "),
			statement,
		)
	}
}
