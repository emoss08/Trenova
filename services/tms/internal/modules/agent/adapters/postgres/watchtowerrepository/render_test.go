package watchtowerrepository

import (
	"strconv"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower/watchtowerdb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func resolvedItem() *watchtower.Item {
	at := int64(100)
	due := int64(400)
	return &watchtower.Item{
		ID:              "wt_1",
		Severity:        watchtower.SeverityCritical,
		Status:          watchtower.StatusResolved,
		ResolvedAt:      &at,
		ResolutionCode:  watchtower.ResolutionCleared,
		ResolutionNote:  "cleared",
		SourceClearedAt: &at,
		AcknowledgedAt:  &at,
		EscalationLevel: 2,
		LastEscalatedAt: &at,
		DueAt:           &due,
		DueLabel:        "Pickup window",
		DueSource:       watchtower.DueSourcePerson,
		Version:         4,
	}
}

func TestApplyUpsertRefreshesTheDescriptionOnly(t *testing.T) {
	t.Parallel()

	existing := resolvedItem()
	incoming := &watchtower.Item{
		Title:       "Late",
		Summary:     "Now two hours out",
		Severity:    watchtower.SeverityWarning,
		RecordLabel: "SHP-1",
		Facts:       []watchtower.Fact{{Label: "Customer", Value: "Northline"}},
		OccurredAt:  300,
	}

	merged := applyUpsert(existing, incoming, upsertPlan{})

	assert.Equal(t, "Late", merged.Title)
	assert.Equal(t, "Now two hours out", merged.Summary)
	assert.Equal(t, "SHP-1", merged.RecordLabel)
	assert.Equal(t, incoming.Facts, merged.Facts)
	assert.Equal(t, int64(300), merged.OccurredAt)
	assert.Equal(t, watchtower.SeverityWarning, merged.Severity)
	assert.Equal(t, existing.ResolvedAt, merged.ResolvedAt, "the resolution is left alone")
	assert.Equal(t, existing.DueAt, merged.DueAt, "an unplanned deadline is left alone")
	assert.Equal(t, int64(4), merged.Version, "Save compares the version the row was read at")
	assert.Equal(t, watchtower.StatusResolved, existing.Status, "the stored row is not changed")
}

func TestApplyUpsertReopensWithACleanSlateAndKeepsAnEscalatedSeverity(t *testing.T) {
	t.Parallel()

	reopened := applyUpsert(resolvedItem(), &watchtower.Item{}, upsertPlan{reopen: true})
	assert.Nil(t, reopened.ResolvedAt)
	assert.Equal(t, watchtower.StatusOpen, reopened.Status)
	assert.Empty(t, reopened.ResolutionCode)
	assert.Empty(t, reopened.ResolutionNote)
	assert.True(t, reopened.ResolvedByID.IsNil())
	assert.Nil(t, reopened.SourceClearedAt)
	assert.Nil(t, reopened.AcknowledgedAt)
	assert.Nil(t, reopened.SnoozedUntil)
	assert.Zero(t, reopened.EscalationLevel)
	assert.Nil(t, reopened.LastEscalatedAt)
	assert.Nil(t, reopened.DueAt, "a reopened item without the source's deadline starts with none")
	assert.Empty(t, reopened.DueSource)

	kept := applyUpsert(
		resolvedItem(),
		&watchtower.Item{Severity: watchtower.SeverityWarning},
		upsertPlan{keepSeverity: true},
	)
	assert.Equal(t, watchtower.SeverityCritical, kept.Severity)
}

func TestApplyUpsertWritesTheDeadlineOnlyWhenPlanned(t *testing.T) {
	t.Parallel()

	due := int64(900)
	incoming := &watchtower.Item{
		DueAt:     &due,
		DueLabel:  "Delivery window closes",
		DueSource: watchtower.DueSourceSource,
	}

	taken := applyUpsert(resolvedItem(), incoming, upsertPlan{takeDeadline: true})
	assert.Equal(t, &due, taken.DueAt)
	assert.Equal(t, "Delivery window closes", taken.DueLabel)
	assert.Equal(t, watchtower.DueSourceSource, taken.DueSource)

	reopenWithDeadline := applyUpsert(
		resolvedItem(),
		incoming,
		upsertPlan{reopen: true, takeDeadline: true},
	)
	assert.Equal(t, &due, reopenWithDeadline.DueAt)

	dropped := applyUpsert(resolvedItem(), &watchtower.Item{}, upsertPlan{dropDeadline: true})
	assert.Nil(t, dropped.DueAt)
	assert.Empty(t, dropped.DueLabel)
	assert.Empty(t, dropped.DueSource)
}

func TestListOrderedByDueContinuesAfterTheCursor(t *testing.T) {
	t.Parallel()

	sql, args := keeltest.Render(t, listQuery(&watchtowerports.ListWatchtowerItemsQuery{
		OrderByDue:  true,
		BeforeID:    pulid.ID("wt_9"),
		BeforeDueAt: 500,
		Unassigned:  true,
		AssigneeIDs: []pulid.ID{"usr_1"},
		Limit:       1000,
	}))

	due := `coalesce(wi."due_at", ` + strconv.FormatInt(watchtower.NoDueSortKey, 10) + `)`
	assert.Contains(t, sql, "ROW("+due+`, wi."id") > ROW($`, sql)
	assert.Contains(t, sql, `ORDER BY `+due+` ASC, wi."id" ASC LIMIT 200`, sql)
	assert.Contains(t, sql, `(wi."assignee_id" IS NULL OR wi."assignee_id" = ANY(`, sql)
	assert.Contains(t, args, int64(500))
	assert.Contains(t, args, "wt_9")
}

func TestListNewestFirstContinuesBeforeTheCursor(t *testing.T) {
	t.Parallel()

	sql, _ := keeltest.Render(t, listQuery(&watchtowerports.ListWatchtowerItemsQuery{
		BeforeID:         pulid.ID("wt_9"),
		BeforeOccurredAt: 700,
		UnresolvedOnly:   true,
	}))

	assert.Contains(t, sql, `ROW(wi."occurred_at", wi."id") < ROW($`, sql)
	assert.Contains(t, sql, `wi."resolved_at" IS NULL`, sql)
	assert.Contains(t, sql, `ORDER BY wi."occurred_at" DESC, wi."id" DESC LIMIT 50`, sql)
}

func TestOverdueSkipsItemsEscalatedTooRecently(t *testing.T) {
	t.Parallel()

	wi := watchtowerdb.Items
	req := &watchtowerports.ListAttentionWatchtowerItemsRequest{
		Now:                1_000,
		BelowEscalationCap: 3,
		EscalateEvery:      100,
	}
	sql, args := keeltest.Render(t, wi.Where(overduePred(req)))

	assert.Contains(t, sql, `wi."due_source" IS DISTINCT FROM $`, sql)
	assert.Contains(t, sql, `wi."last_escalated_at" IS NULL`, sql)
	assert.Contains(t, sql, `wi."last_escalated_at" <= $`, sql)
	assert.Contains(t, args, int64(900))
}

func TestCountsLeaveMineEmptyWithoutAnAssignee(t *testing.T) {
	t.Parallel()

	sql, _ := keeltest.Render(t, countTotalsQuery(&watchtowerports.CountWatchtowerItemsRequest{
		ReaderID: pulid.ID("usr_1"),
		SeenAt:   10,
		Now:      20,
	}))

	assert.Contains(t, sql, `count(*) FILTER (WHERE wi."assignee_id" IS NULL)`, sql)
	assert.Contains(t, sql, `FILTER (WHERE FALSE)`, sql)
	assert.Contains(t, sql, `NOT EXISTS (SELECT`, sql)
}

func TestTenantsNeedingAttentionAreReadAcrossTenantsOnce(t *testing.T) {
	t.Parallel()

	sql, _ := keeltest.RenderIn(
		t,
		dbscope.WithSystem(t.Context(), "test"),
		tenantsNeedingAttentionQuery(1_000),
	)

	assert.Contains(t, sql, `SELECT wi."organization_id", wi."business_unit_id"`, sql)
	assert.NotContains(t, sql, `wi."organization_id" = $`, sql)
	assert.Contains(t, sql, `GROUP BY wi."organization_id", wi."business_unit_id"`, sql)
}

func TestRetentionDeletesOnlyItemsTheSourceIsDoneWith(t *testing.T) {
	t.Parallel()

	wi := watchtowerdb.Items
	sql, args := keeltest.RenderIn(
		t,
		dbscope.WithSystem(t.Context(), "test"),
		wi.AllTenants().Where(resolvedBefore(1_000, 500)),
	)

	assert.Contains(t, sql, `."id" IN (SELECT stale."id" FROM "watchtower_items" stale`, sql)
	assert.Contains(t, sql, `(stale."resolved_by_id" IS NULL OR stale."source_cleared_at" IS NOT NULL)`, sql)
	assert.Contains(t, sql, "LIMIT 500)", sql)
	assert.Equal(t, []any{int64(1_000)}, args)
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
