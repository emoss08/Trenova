package briefingrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/briefing"
	"github.com/stretchr/testify/assert"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

/*
A rerun of the day's briefing replaces the stored one and bumps its version.

The version has to be read from the stored row by its alias: beside EXCLUDED a
bare column is ambiguous, and Postgres rejects the whole statement.
*/
func TestUpsertBumpsTheStoredRowsVersion(t *testing.T) {
	t.Parallel()

	sql := buildUpsert(bun.NewDB(nil, pgdialect.New()), &briefing.Briefing{}).String()

	assert.Contains(t, sql, "version = abrf.version + 1", sql)
	assert.NotContains(t, sql, "version = version + 1", sql)
	assert.Contains(t, sql, "COALESCE(user_id, '')) DO UPDATE", sql)
}

func TestDeleteBeforeSelectsTheOldestBatchByID(t *testing.T) {
	t.Parallel()

	sql := buildDeleteBefore(bun.NewDB(nil, pgdialect.New()), "2026-01-01", 500).String()

	assert.Contains(t, sql, "abrf.id IN (SELECT abrf.id FROM", sql)
	assert.Contains(t, sql, "abrf.briefing_date < '2026-01-01' ORDER BY abrf.briefing_date LIMIT 500)", sql)
	assert.NotContains(t, sql, "IN (?)", sql)
}
