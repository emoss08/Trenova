package migrations

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const triageUp = "20261231008800_triage.tx.up.sql"

func TestTriageMigration_RepliesAndActivitiesGoWithTheirMessage(t *testing.T) {
	t.Parallel()

	up := compactSQL(readMigration(t, triageUp))

	assert.Contains(t, up, `CONSTRAINT "fk_inbound_message_activities_message" FOREIGN KEY `+
		`("message_id", "business_unit_id", "organization_id") REFERENCES "inbound_messages"`+
		`("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE`,
		"retention deletes settled messages by id; their activity lines must go with them")
	assert.Contains(t, up, `CONSTRAINT "fk_inbound_replies_message" FOREIGN KEY `+
		`("message_id", "business_unit_id", "organization_id") REFERENCES "inbound_messages"`+
		`("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE`,
		"an outbound reply never outlives the message it answered")
}
