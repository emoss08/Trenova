package migrations

import (
	"io/fs"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	agentAccessUp   = "20261231004800_agent_access.tx.up.sql"
	agentAccessDown = "20261231004800_agent_access.tx.down.sql"
)

func readMigration(t *testing.T, name string) string {
	t.Helper()

	body, err := fs.ReadFile(sqlMigrations, name)
	require.NoError(t, err, "read %s", name)

	return string(body)
}

func compactSQL(sql string) string {
	return strings.Join(strings.Fields(sql), " ")
}

func TestAgentAccessMigration_DefaultsEveryAgentToEveryone(t *testing.T) {
	t.Parallel()

	up := compactSQL(readMigration(t, agentAccessUp))

	assert.Contains(t, up,
		`ADD COLUMN IF NOT EXISTS "access_mode" VARCHAR(20) NOT NULL DEFAULT 'Everyone'`)
	assert.Contains(t, up,
		`CHECK ("system_key" IS NULL OR "access_mode" = 'Everyone')`,
		"a system agent must be open to everyone at the database too")
	assert.Contains(t, up, `COMMENT ON COLUMN "agent_definitions"."access_mode"`)
}

func TestAgentAccessMigration_BackfillsNoGrants(t *testing.T) {
	t.Parallel()

	up := strings.ToUpper(readMigration(t, agentAccessUp))

	assert.NotContains(t, up, "INSERT INTO",
		"existing agents stay open to everyone; no role is granted anything on migration")
	assert.NotContains(t, up, "UPDATE \"AGENT_DEFINITIONS\"",
		"no agent is restricted on migration")
}

func TestAgentAccessMigration_GrantTableIsTenantConsistent(t *testing.T) {
	t.Parallel()

	up := compactSQL(readMigration(t, agentAccessUp))

	assert.Contains(t, up,
		`FOREIGN KEY ("role_id", "business_unit_id", "organization_id") REFERENCES "roles"("id", `+
			`"business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE`,
		"a grant names a role in its own tenant and goes with it")
	assert.Contains(t, up,
		`FOREIGN KEY ("agent_definition_id", "business_unit_id", "organization_id") REFERENCES `+
			`"agent_definitions"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION `+
			`ON DELETE CASCADE`,
		"a grant names an agent in its own tenant and goes with it")
	assert.Regexp(t,
		regexp.MustCompile(`CREATE UNIQUE INDEX IF NOT EXISTS "\w+" ON "role_agent_grants" `+
			`\("role_id", "agent_definition_id"\)`),
		up,
		"a role is granted an agent once")
	assert.Contains(t, up, `ON "role_agent_grants" ("organization_id", "role_id")`)
	assert.Contains(t, up,
		`ON "role_agent_grants" ("organization_id", "business_unit_id", "agent_definition_id")`)

	for _, column := range []string{"role_id", "agent_definition_id", "granted_by", "granted_at"} {
		assert.Contains(t, up, `COMMENT ON COLUMN "role_agent_grants"."`+column+`"`)
	}
	assert.Contains(t, up, `COMMENT ON TABLE "role_agent_grants"`)
}

func TestAgentAccessMigration_DownUndoesUp(t *testing.T) {
	t.Parallel()

	down := compactSQL(readMigration(t, agentAccessDown))

	assert.Contains(t, down, `DROP TABLE IF EXISTS "role_agent_grants"`)
	assert.Contains(t, down, `DROP CONSTRAINT IF EXISTS "ck_agent_definitions_system_access"`)
	assert.Contains(t, down, `DROP CONSTRAINT IF EXISTS "ck_agent_definitions_access_mode"`)
	assert.Contains(t, down, `DROP COLUMN IF EXISTS "access_mode"`)
	assert.Less(t,
		strings.Index(down, "role_agent_grants"),
		strings.Index(down, "access_mode"),
		"the grants go before the column that gives them meaning")
}
