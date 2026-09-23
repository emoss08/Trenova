package watchtowerrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/watchtower"
	"github.com/stretchr/testify/assert"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

/*
A source reporting its record again updates the item it already raised.

The version is read from the stored row by its alias: beside EXCLUDED a bare
column is ambiguous, and Postgres rejects the whole statement, so no item was
ever raised or refreshed.
*/
func TestUpsertBumpsTheStoredRowsVersion(t *testing.T) {
	t.Parallel()

	sql := buildUpsert(bun.NewDB(nil, pgdialect.New()), &watchtower.Item{}).String()

	assert.Contains(t, sql, "version = wti.version + 1", sql)
	assert.NotContains(t, sql, "version = version + 1", sql)
	assert.Contains(t, sql, "resolved_at = NULL", sql)
}
