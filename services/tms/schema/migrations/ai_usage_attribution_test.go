package migrations

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	aiUsageFeatureSubjectUp = "20261231006520_ai_usage_feature_subject"
	dropAILogsUp            = "20261231006540_drop_ai_logs"
)

func TestAIUsageFeatureSubjectMigration_AddsNullableAttribution(t *testing.T) {
	t.Parallel()

	up := compactSQL(readMigration(t, aiUsageFeatureSubjectUp))

	for _, fragment := range []string{
		`ALTER TABLE "ai_usage_records" ADD COLUMN IF NOT EXISTS "feature" varchar(50);`,
		`ALTER TABLE "ai_usage_records" ADD COLUMN IF NOT EXISTS "subject_type" varchar(50);`,
		`ALTER TABLE "ai_usage_records" ADD COLUMN IF NOT EXISTS "subject_id" varchar(100);`,
		`CHECK (("subject_type" IS NULL) = ("subject_id" IS NULL))`,
		`CREATE INDEX IF NOT EXISTS "idx_ai_usage_records_tenant_feature_time" ON ` +
			`"ai_usage_records"("organization_id", "business_unit_id", ` +
			`"feature", "created_at" DESC);`,
	} {
		assert.Contains(t, up, fragment)
	}
	assert.NotContains(t, up, "NOT NULL",
		"rows written before the cutover carry no feature and must stay valid")
}

func TestDropAILogsMigration_RetiresTheTableWithoutCarryingRows(t *testing.T) {
	t.Parallel()

	up := compactSQL(readMigration(t, dropAILogsUp))

	assert.Contains(t, up, `DROP TABLE IF EXISTS "ai_logs";`)
	assert.Contains(t, up, `DROP FUNCTION IF EXISTS prevent_ai_logs_modification();`)
	assert.Contains(t, up, `DROP TYPE IF EXISTS "operation_enum";`)
	assert.NotContains(t, up, "INSERT", "the decision is no backfill")
}
