package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	agentDataAccessUp = "20261231006495_agent_data_access"
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

	assert.Contains(
		t,
		up,
		`SET "data_access_ceiling" = 'Restricted' WHERE "trigger_mode" = 'Chat' OR "template" = 'CashApplication'`,
		"a chat agent never reads past its person, and the cash desk matched on amounts before",
	)
	assert.Equal(t, 1, strings.Count(strings.ToUpper(up), "UPDATE \"AGENT_DEFINITIONS\""))
}
