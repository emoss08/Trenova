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
	assert.True(
		t,
		planUpsert(&escalated, &watchtower.Item{Severity: watchtower.SeverityWarning}).keepSeverity,
	)
	assert.False(t, planUpsert(nil, &watchtower.Item{}).reopen)
}

func TestPlanUpsert_TheSourcesDeadlineMovesUnlessAPersonSetTheDue(t *testing.T) {
	t.Parallel()

	due := int64(500)
	incoming := &watchtower.Item{DueAt: &due, DueSource: watchtower.DueSourceSource}

	assert.True(
		t,
		planUpsert(&watchtower.Item{DueSource: watchtower.DueSourcePolicy}, incoming).takeDeadline,
	)
	assert.True(
		t,
		planUpsert(&watchtower.Item{DueSource: watchtower.DueSourceSource}, incoming).takeDeadline,
	)
	assert.False(
		t,
		planUpsert(&watchtower.Item{DueSource: watchtower.DueSourcePerson}, incoming).takeDeadline,
	)

	now := int64(100)
	reopened := &watchtower.Item{ResolvedAt: &now, DueSource: watchtower.DueSourcePerson}
	assert.True(t, planUpsert(reopened, incoming).takeDeadline,
		"a reopened item starts again from the source's deadline")

	dropped := planUpsert(
		&watchtower.Item{DueSource: watchtower.DueSourceSource},
		&watchtower.Item{},
	)
	assert.True(
		t,
		dropped.dropDeadline,
		"a deadline the source stopped reporting is cleared for routing",
	)
	assert.False(
		t,
		planUpsert(
			&watchtower.Item{DueSource: watchtower.DueSourcePolicy},
			&watchtower.Item{},
		).dropDeadline,
	)
}

func TestUpsertWritesTheDeadlineOnlyWhenPlanned(t *testing.T) {
	t.Parallel()

	db := bun.NewDB(nil, pgdialect.New())

	taken := buildUpsert(db, &watchtower.Item{}, upsertPlan{takeDeadline: true}).String()
	assert.Contains(t, taken, "due_at = EXCLUDED.due_at", taken)
	assert.Contains(t, taken, "due_label = EXCLUDED.due_label", taken)

	dropped := buildUpsert(db, &watchtower.Item{}, upsertPlan{dropDeadline: true}).String()
	assert.Contains(t, dropped, "due_at = NULL", dropped)
	assert.Contains(t, dropped, "due_source = NULL", dropped)

	plain := buildUpsert(db, &watchtower.Item{}, upsertPlan{}).String()
	assert.NotContains(t, plain, "due_at =", plain)
	assert.Contains(t, plain, "facts = EXCLUDED.facts", plain)
	assert.Contains(t, plain, "record_label = EXCLUDED.record_label", plain)

	reopenWithDeadline := buildUpsert(
		db,
		&watchtower.Item{},
		upsertPlan{reopen: true, takeDeadline: true},
	).String()
	assert.Contains(t, reopenWithDeadline, "due_at = EXCLUDED.due_at", reopenWithDeadline)
	assert.NotContains(t, reopenWithDeadline, "due_at = NULL", reopenWithDeadline)
}

func TestPlanUpsert_ARefreshThatChangesNothingWritesNothing(t *testing.T) {
	t.Parallel()

	due := int64(500)
	stored := func() *watchtower.Item {
		return &watchtower.Item{
			Title:       "Projected late on SEED-SHP-007",
			Summary:     "Projected to reach the delivery after its window closes.",
			Severity:    watchtower.SeverityCritical,
			RecordLabel: "SEED-SHP-007",
			Facts:       []watchtower.Fact{{Label: "Customer", Value: "Northline Foods"}},
			DueAt:       &due,
			DueLabel:    "Delivery window closes",
			DueSource:   watchtower.DueSourceSource,
			OccurredAt:  100,
		}
	}

	again := stored()
	again.OccurredAt = 160
	assert.True(t, planUpsert(stored(), again).unchanged,
		"the same report, raised at a later compute time, is not a change")

	retitled := stored()
	retitled.Summary = "Now five hours out."
	assert.False(t, planUpsert(stored(), retitled).unchanged)

	moved := stored()
	later := int64(900)
	moved.DueAt = &later
	assert.False(t, planUpsert(stored(), moved).unchanged, "a deadline that moved is a change")

	lowered := stored()
	lowered.Severity = watchtower.SeverityWarning
	assert.False(t, planUpsert(stored(), lowered).unchanged)

	newFact := stored()
	newFact.Facts = append(newFact.Facts, watchtower.Fact{Label: "Driver", Value: "Will Grant"})
	assert.False(t, planUpsert(stored(), newFact).unchanged)

	now := int64(100)
	resolved := stored()
	resolved.ResolvedAt = &now
	assert.False(
		t,
		planUpsert(resolved, stored()).unchanged,
		"a report that reopens an item is a change",
	)

	assert.False(t, planUpsert(nil, stored()).unchanged, "a new item is always written")
}
