package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	agentDataAccessUp   = "20261231006495_agent_data_access.tx.up.sql"
	agentDataAccessDown = "20261231006495_agent_data_access.tx.down.sql"
)

func TestAgentDataAccessMigration_DefaultsANewAgentToInternal(t *testing.T) {
	t.Parallel()

	up := compactSQL(readMigration(t, agentDataAccessUp))

	assert.Contains(t, up,
		`ADD COLUMN IF NOT EXISTS "data_access_ceiling" VARCHAR(20) NOT NULL DEFAULT 'Internal'`)
	assert.Contains(t, up,
		`CHECK ("data_access_ceiling" IN ('Internal', 'Restricted'))`)
	assert.Contains(t, up, `COMMENT ON COLUMN "agent_definitions"."data_access_ceiling"`)
}

func TestAgentDataAccessMigration_KeepsWhatExistingAgentsAlreadyRead(t *testing.T) {
	t.Parallel()

	up := compactSQL(readMigration(t, agentDataAccessUp))

	assert.Contains(t, up,
		`SET "data_access_ceiling" = 'Restricted' WHERE "trigger_mode" = 'Chat' OR "template" = 'CashApplication'`,
		"a chat agent never reads past its person, and the cash desk matched on amounts before")
	assert.Equal(t, 1, strings.Count(strings.ToUpper(up), "UPDATE \"AGENT_DEFINITIONS\""))
}

func TestAgentDataAccessMigration_DownDropsTheColumn(t *testing.T) {
	t.Parallel()

	down := compactSQL(readMigration(t, agentDataAccessDown))

	assert.Contains(t, down, `DROP CONSTRAINT IF EXISTS "ck_agent_definitions_data_access_ceiling"`)
	assert.Contains(t, down, `DROP COLUMN IF EXISTS "data_access_ceiling"`)
}
