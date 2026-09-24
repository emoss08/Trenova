package migrations

import (
	"io/fs"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	aiUsageFeatureSubjectUp   = "20261231006520_ai_usage_feature_subject.tx.up.sql"
	aiUsageFeatureSubjectDown = "20261231006520_ai_usage_feature_subject.tx.down.sql"
	dropAILogsUp              = "20261231006540_drop_ai_logs.tx.up.sql"
	dropAILogsDown            = "20261231006540_drop_ai_logs.tx.down.sql"
	dropAILogsVersion         = "20261231006540"
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

func TestAIUsageFeatureSubjectMigration_DownRemovesWhatUpAdded(t *testing.T) {
	t.Parallel()

	down := compactSQL(readMigration(t, aiUsageFeatureSubjectDown))

	for _, fragment := range []string{
		`DROP INDEX IF EXISTS "idx_ai_usage_records_tenant_feature_time"`,
		`DROP CONSTRAINT IF EXISTS "ck_ai_usage_records_subject_pair"`,
		`DROP CONSTRAINT IF EXISTS "ck_ai_usage_records_subject_type"`,
		`DROP CONSTRAINT IF EXISTS "ck_ai_usage_records_feature"`,
		`DROP COLUMN IF EXISTS "subject_id"`,
		`DROP COLUMN IF EXISTS "subject_type"`,
		`DROP COLUMN IF EXISTS "feature"`,
	} {
		assert.Contains(t, down, fragment)
	}
}

func TestDropAILogsMigration_RetiresTheTableWithoutCarryingRows(t *testing.T) {
	t.Parallel()

	up := compactSQL(readMigration(t, dropAILogsUp))

	assert.Contains(t, up, `DROP TABLE IF EXISTS "ai_logs";`)
	assert.Contains(t, up, `DROP FUNCTION IF EXISTS prevent_ai_logs_modification();`)
	assert.Contains(t, up, `DROP TYPE IF EXISTS "operation_enum";`)
	assert.NotContains(t, up, "INSERT", "the decision is no backfill")
}

func TestDropAILogsMigration_DownRestoresEveryOperationTheTypeHeld(t *testing.T) {
	t.Parallel()

	created := regexp.MustCompile(`(?s)CREATE TYPE "operation_enum" AS ENUM\((.*?)\);`)
	added := regexp.MustCompile(`ALTER TYPE "?operation_enum"? ADD VALUE IF NOT EXISTS '([^']+)'`)
	quoted := regexp.MustCompile(`'([^']+)'`)

	held := make(map[string]struct{})
	for _, file := range embeddedMigrationFiles(t) {
		if file.direction != "up" || file.version >= dropAILogsVersion {
			continue
		}
		body, err := fs.ReadFile(sqlMigrations, file.name)
		require.NoError(t, err)

		if match := created.FindSubmatch(body); match != nil {
			for _, value := range quoted.FindAllSubmatch(match[1], -1) {
				held[string(value[1])] = struct{}{}
			}
		}
		for _, value := range added.FindAllSubmatch(body, -1) {
			held[string(value[1])] = struct{}{}
		}
	}
	require.NotEmpty(t, held, "no earlier migration defines operation_enum")

	restored := created.FindStringSubmatch(readMigration(t, dropAILogsDown))
	require.NotNil(t, restored, "the down migration does not recreate operation_enum")
	for value := range held {
		assert.Contains(t, restored[1], "'"+value+"'", "rollback loses operation %q", value)
	}

	down := compactSQL(readMigration(t, dropAILogsDown))
	for _, fragment := range []string{
		`CREATE TABLE IF NOT EXISTS "ai_logs"(`,
		`"model" varchar(200) NOT NULL`,
		`"provider_kind" varchar(50) NOT NULL DEFAULT ''`,
		`"provider_id" varchar(100)`,
		`CREATE TRIGGER enforce_ai_logs_append_only`,
		`CREATE INDEX IF NOT EXISTS "idx_ai_logs_search_vector" ON "ai_logs" ` +
			`USING GIN("search_vector");`,
	} {
		assert.Contains(t, down, fragment)
	}
}
