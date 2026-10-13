package watchtowerrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
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

	sql := buildUpsert(bun.NewDB(nil, pgdialect.New()), &watchtower.Item{}, upsertPlan{}).String()

	assert.Contains(t, sql, "version = wti.version + 1", sql)
	assert.NotContains(t, sql, "version = version + 1", sql)
	assert.NotContains(t, sql, "resolved_at = NULL", sql)
	assert.Contains(t, sql, "severity = EXCLUDED.severity", sql)
}

/*
A report on an item the source had cleared, or the system had, opens it again
with a clean slate; a report on an item a person resolved while the source was
still open leaves the person's resolution alone, and an escalated severity is
not lowered by the source.
*/
func TestUpsertReopensOnlyWhenPlannedAndKeepsAnEscalatedSeverity(t *testing.T) {
	t.Parallel()

	db := bun.NewDB(nil, pgdialect.New())
	sql := buildUpsert(db, &watchtower.Item{}, upsertPlan{reopen: true}).String()
	assert.Contains(t, sql, "resolved_at = NULL", sql)
	assert.Contains(t, sql, "resolution_code = NULL", sql)
	assert.Contains(t, sql, "escalation_level = 0", sql)

	kept := buildUpsert(db, &watchtower.Item{}, upsertPlan{keepSeverity: true}).String()
	assert.NotContains(t, kept, "severity = EXCLUDED.severity", kept)
}

func TestPlanUpsert(t *testing.T) {
	t.Parallel()

	now := int64(100)
	person := watchtower.Item{ResolvedAt: &now, ResolvedByID: "usr_1"}
	assert.False(t, planUpsert(&person, &watchtower.Item{}).reopen)

	cleared := person
	cleared.SourceClearedAt = &now
	assert.True(t, planUpsert(&cleared, &watchtower.Item{}).reopen)

	system := watchtower.Item{ResolvedAt: &now}
	assert.True(t, planUpsert(&system, &watchtower.Item{}).reopen)

	escalated := watchtower.Item{EscalationLevel: 1, Severity: watchtower.SeverityCritical}
	assert.True(t, planUpsert(&escalated, &watchtower.Item{Severity: watchtower.SeverityWarning}).keepSeverity)
	assert.False(t, planUpsert(nil, &watchtower.Item{}).reopen)
}
