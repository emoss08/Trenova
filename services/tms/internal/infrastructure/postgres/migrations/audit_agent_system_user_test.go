package migrations

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	auditAgentSystemUserUp   = "20261231007230_audit_agent_system_user.tx.up.sql"
	auditAgentSystemUserDown = "20261231007230_audit_agent_system_user.tx.down.sql"
)

func TestAuditAgentSystemUserMigration_LetsAnAgentNameTheSystemUserItRanAs(t *testing.T) {
	t.Parallel()

	up := compactSQL(readMigration(t, auditAgentSystemUserUp))

	assert.Contains(t, up,
		`DROP CONSTRAINT IF EXISTS "chk_audit_entries_principal_consistency"`)
	assert.Contains(t, up,
		`("principal_type" = 'agent' AND "api_key_id" IS NULL AND "principal_id" IS NOT NULL `+
			`AND ("user_id" IS NULL OR "principal_id" <> "user_id"))`,
		"an agent may carry a user, never an api key, and is never that user")
	assert.Contains(t, up,
		`("principal_type" = 'system' AND "user_id" IS NULL AND "api_key_id" IS NULL `+
			`AND "principal_id" IS NOT NULL)`,
		"the system principal still names no user")
	assert.Contains(t, up,
		`("principal_type" = 'session_user' AND "user_id" IS NOT NULL AND "api_key_id" IS NULL `+
			`AND "principal_id" = "user_id")`)
	assert.Contains(t, up,
		`("principal_type" = 'api_key' AND "user_id" IS NULL AND "api_key_id" IS NOT NULL `+
			`AND "principal_id" = "api_key_id")`)
	assert.NotContains(t, up, "chk_audit_entries_principal_type",
		"the principal types are unchanged")
	assert.NotContains(t, up, "UPDATE", "audit_entries is append-only")
}

func TestAuditAgentSystemUserMigration_DownKeepsRowsWrittenSince(t *testing.T) {
	t.Parallel()

	down := compactSQL(readMigration(t, auditAgentSystemUserDown))

	assert.Contains(t, down,
		`("principal_type" IN ('system', 'agent') AND "user_id" IS NULL AND "api_key_id" IS NULL `+
			`AND "principal_id" IS NOT NULL)`)
	assert.Contains(t, down, ") NOT VALID;",
		"rows an agent wrote as the system user stay, since audit_entries cannot be updated")
	assert.NotContains(t, down, "UPDATE")
	assert.NotContains(t, down, "DELETE")
}
