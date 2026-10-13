//go:build integration

package watchtowerrepository_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/agent/adapters/postgres/watchtowerrepository"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

const baseTime = int64(1_700_000_000)

type harness struct {
	ctx    context.Context
	db     *bun.DB
	repo   watchtowerports.WatchtowerRepository
	tenant pagination.TenantInfo
	userID pulid.ID
}

func newHarness(t *testing.T) *harness {
	t.Helper()

	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	tenant := pagination.TenantInfo{OrgID: data.Organization.ID, BuID: data.BusinessUnit.ID}
	return &harness{
		ctx:    dbscope.WithTenant(ctx, tenant.DBTenant()),
		db:     db,
		repo:   watchtowerrepository.New(watchtowerrepository.Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()}),
		tenant: tenant,
		userID: data.User.ID,
	}
}

func (h *harness) item(sourceID string, severity watchtower.Severity) *watchtower.Item {
	return &watchtower.Item{
		OrganizationID: h.tenant.OrgID,
		BusinessUnitID: h.tenant.BuID,
		SourceKind:     watchtower.SourceServiceFailure,
		SourceID:       sourceID,
		Severity:       severity,
		Title:          "Late at " + sourceID,
		OccurredAt:     baseTime,
	}
}

func (h *harness) upsert(t *testing.T, sourceID string, severity watchtower.Severity) *watchtower.Item {
	t.Helper()
	saved, _, err := h.repo.Upsert(h.ctx, h.item(sourceID, severity))
	require.NoError(t, err)
	return saved
}

func (h *harness) update(t *testing.T, item *watchtower.Item, events ...*watchtower.Event) *watchtower.Item {
	t.Helper()
	saved, err := h.repo.UpdateWorkflow(h.ctx, &watchtowerports.UpdateWatchtowerWorkflowRequest{
		Item:            item,
		ExpectedVersion: item.Version,
		Events:          events,
	})
	require.NoError(t, err)
	return saved
}

func (h *harness) history(t *testing.T, id pulid.ID) []watchtower.EventKind {
	t.Helper()
	events, err := h.repo.ListEvents(h.ctx, watchtowerports.ListWatchtowerEventsRequest{
		TenantInfo: h.tenant,
		ItemID:     id,
	})
	require.NoError(t, err)
	kinds := make([]watchtower.EventKind, 0, len(events))
	for _, event := range events {
		kinds = append(kinds, event.Kind)
	}
	return kinds
}

func (h *harness) resolveSource(t *testing.T, sourceID string) *watchtower.Item {
	t.Helper()
	item, err := h.repo.Resolve(h.ctx, watchtowerports.ResolveWatchtowerItemRequest{
		TenantInfo: h.tenant,
		SourceKind: watchtower.SourceServiceFailure,
		SourceID:   sourceID,
		ResolvedAt: baseTime + 100,
	})
	require.NoError(t, err)
	return item
}

func (h *harness) get(t *testing.T, id pulid.ID) *watchtower.Item {
	t.Helper()
	item, err := h.repo.GetByID(h.ctx, watchtowerports.GetWatchtowerItemRequest{ID: id, TenantInfo: h.tenant})
	require.NoError(t, err)
	return item
}

func TestPersonsResolutionHoldsUntilTheSourceClears(t *testing.T) {
	h := newHarness(t)

	created := h.upsert(t, "sf_1", watchtower.SeverityWarning)
	assert.Equal(t, watchtower.StatusOpen, created.Status)

	assigned := *created
	assigned.AssigneeID = h.userID
	saved := h.update(t, &assigned,
		watchtower.NewEvent(&assigned, watchtower.EventAssigned, h.userID).WithChange("", h.userID.String()))

	resolved := *saved
	require.NoError(t, resolved.Resolve(watchtower.ResolutionCarrierDelay, "Carrier called", h.userID, baseTime+50))
	saved = h.update(t, &resolved, watchtower.NewEvent(&resolved, watchtower.EventResolved, h.userID))
	assert.Equal(t, watchtower.StatusResolved, saved.Status)

	again := h.upsert(t, "sf_1", watchtower.SeverityCritical)
	assert.False(t, again.Reopened)
	stored := h.get(t, created.ID)
	assert.Equal(t, watchtower.StatusResolved, stored.Status)
	assert.Equal(t, watchtower.ResolutionCarrierDelay, stored.ResolutionCode)

	assert.Nil(t, h.resolveSource(t, "sf_1"))
	stored = h.get(t, created.ID)
	require.NotNil(t, stored.SourceClearedAt)
	assert.Equal(t, watchtower.ResolutionCarrierDelay, stored.ResolutionCode)

	reopened := h.upsert(t, "sf_1", watchtower.SeverityWarning)
	assert.True(t, reopened.Reopened)
	stored = h.get(t, created.ID)
	assert.Equal(t, watchtower.StatusOpen, stored.Status)
	assert.Nil(t, stored.ResolvedAt)
	assert.Empty(t, stored.ResolutionCode)
	assert.Equal(t, h.userID, stored.AssigneeID)

	assert.Equal(t, []watchtower.EventKind{
		watchtower.EventCreated,
		watchtower.EventAssigned,
		watchtower.EventResolved,
		watchtower.EventSourceReopened,
	}, h.history(t, created.ID))
}

func TestSourceClearsAnOpenItemAndReopensIt(t *testing.T) {
	h := newHarness(t)

	created := h.upsert(t, "sf_2", watchtower.SeverityInfo)
	cleared := h.resolveSource(t, "sf_2")
	require.NotNil(t, cleared)
	assert.Equal(t, watchtower.StatusResolved, cleared.Status)
	assert.Equal(t, watchtower.ResolutionCleared, cleared.ResolutionCode)
	assert.True(t, cleared.ResolvedByID.IsNil())

	reopened := h.upsert(t, "sf_2", watchtower.SeverityInfo)
	assert.True(t, reopened.Reopened)
	assert.Equal(t, []watchtower.EventKind{
		watchtower.EventCreated,
		watchtower.EventSourceResolved,
		watchtower.EventSourceReopened,
	}, h.history(t, created.ID))
}

func TestEscalatedSeverityIsNotLoweredByTheSource(t *testing.T) {
	h := newHarness(t)

	created := h.upsert(t, "sf_3", watchtower.SeverityWarning)
	escalated := *created
	escalated.EscalationLevel = 1
	escalated.Severity = watchtower.SeverityCritical
	h.update(t, &escalated)

	h.upsert(t, "sf_3", watchtower.SeverityWarning)
	assert.Equal(t, watchtower.SeverityCritical, h.get(t, created.ID).Severity)
}

func TestUpdateWorkflowRefusesAStaleVersion(t *testing.T) {
	h := newHarness(t)

	created := h.upsert(t, "sf_4", watchtower.SeverityWarning)
	first := *created
	first.AssigneeID = h.userID
	h.update(t, &first)

	stale := *created
	stale.Status = watchtower.StatusInProgress
	_, err := h.repo.UpdateWorkflow(h.ctx, &watchtowerports.UpdateWatchtowerWorkflowRequest{
		Item:            &stale,
		ExpectedVersion: created.Version,
	})
	require.Error(t, err)
	assert.Equal(t, watchtower.StatusOpen, h.get(t, created.ID).Status)
}

func TestEventsAreAppendOnly(t *testing.T) {
	h := newHarness(t)

	created := h.upsert(t, "sf_5", watchtower.SeverityWarning)
	_, err := h.db.NewRaw(
		`UPDATE watchtower_item_events SET note = 'changed' WHERE item_id = ?`,
		created.ID,
	).Exec(h.ctx)
	require.Error(t, err)
}

func TestAttentionQueriesFindOverdueAndWakingItems(t *testing.T) {
	h := newHarness(t)

	overdue := h.upsert(t, "sf_6", watchtower.SeverityWarning)
	due := baseTime - 10
	withDue := *overdue
	withDue.DueAt = &due
	h.update(t, &withDue)

	capped := h.upsert(t, "sf_7", watchtower.SeverityWarning)
	cappedDue := *capped
	cappedDue.DueAt = &due
	cappedDue.EscalationLevel = 3
	h.update(t, &cappedDue)

	snoozed := h.upsert(t, "sf_8", watchtower.SeverityWarning)
	napping := *snoozed
	require.NoError(t, napping.Snooze(baseTime+5, baseTime-60))
	h.update(t, &napping)

	request := watchtowerports.ListAttentionWatchtowerItemsRequest{
		TenantInfo:         h.tenant,
		Now:                baseTime,
		BelowEscalationCap: 3,
	}
	items, err := h.repo.ListOverdue(h.ctx, request)
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, overdue.ID, items[0].ID)

	waking, err := h.repo.ListSnoozeExpired(h.ctx, request)
	require.NoError(t, err)
	assert.Empty(t, waking)
	request.Now = baseTime + 5
	waking, err = h.repo.ListSnoozeExpired(h.ctx, request)
	require.NoError(t, err)
	require.Len(t, waking, 1)
	assert.Equal(t, snoozed.ID, waking[0].ID)

	tenants, err := h.repo.ListTenantsNeedingAttention(h.ctx, baseTime+5)
	require.NoError(t, err)
	assert.Contains(t, tenants, pagination.TenantInfo{OrgID: h.tenant.OrgID, BuID: h.tenant.BuID})
}

func TestListFiltersByOwnerStatusAndOverdueAndOrdersByDue(t *testing.T) {
	h := newHarness(t)

	later := h.upsert(t, "sf_9", watchtower.SeverityWarning)
	sooner := h.upsert(t, "sf_10", watchtower.SeverityWarning)
	unowned := h.upsert(t, "sf_11", watchtower.SeverityInfo)

	laterDue, soonerDue := baseTime+600, baseTime-60
	a := *later
	a.AssigneeID, a.DueAt = h.userID, &laterDue
	h.update(t, &a)
	b := *sooner
	b.AssigneeID, b.DueAt = h.userID, &soonerDue
	h.update(t, &b)

	mine, err := h.repo.List(h.ctx, watchtowerports.ListWatchtowerItemsQuery{
		TenantInfo:     h.tenant,
		Kinds:          []watchtower.SourceKind{watchtower.SourceServiceFailure},
		UnresolvedOnly: true,
		AssigneeIDs:    []pulid.ID{h.userID},
		OrderByDue:     true,
		Limit:          10,
	})
	require.NoError(t, err)
	require.Len(t, mine, 2)
	assert.Equal(t, sooner.ID, mine[0].ID)
	assert.Equal(t, later.ID, mine[1].ID)

	overdueOnly, err := h.repo.List(h.ctx, watchtowerports.ListWatchtowerItemsQuery{
		TenantInfo: h.tenant,
		Kinds:      []watchtower.SourceKind{watchtower.SourceServiceFailure},
		OverdueAt:  baseTime,
		Limit:      10,
	})
	require.NoError(t, err)
	require.Len(t, overdueOnly, 1)
	assert.Equal(t, sooner.ID, overdueOnly[0].ID)

	nobody, err := h.repo.List(h.ctx, watchtowerports.ListWatchtowerItemsQuery{
		TenantInfo: h.tenant,
		Kinds:      []watchtower.SourceKind{watchtower.SourceServiceFailure},
		Unassigned: true,
		Statuses:   []watchtower.Status{watchtower.StatusOpen},
		Limit:      10,
	})
	require.NoError(t, err)
	require.Len(t, nobody, 1)
	assert.Equal(t, unowned.ID, nobody[0].ID)

	counts, err := h.repo.Counts(h.ctx, watchtowerports.CountWatchtowerItemsRequest{
		TenantInfo: h.tenant,
		Kinds:      []watchtower.SourceKind{watchtower.SourceServiceFailure},
		AssigneeID: h.userID,
		Now:        baseTime,
	})
	require.NoError(t, err)
	assert.Equal(t, 3, counts.Unresolved)
	assert.Equal(t, 1, counts.Overdue)
	assert.Equal(t, 1, counts.Unassigned)
	assert.Equal(t, 2, counts.Mine)
	assert.Equal(t, 1, counts.MineOverdue)
}

func TestRetentionKeepsAPersonsResolutionWhileTheSourceIsOpen(t *testing.T) {
	h := newHarness(t)

	held := h.upsert(t, "sf_12", watchtower.SeverityWarning)
	resolved := *held
	require.NoError(t, resolved.Resolve(watchtower.ResolutionFixed, "", h.userID, baseTime))
	h.update(t, &resolved)

	cleared := h.upsert(t, "sf_13", watchtower.SeverityWarning)
	require.NotNil(t, h.resolveSource(t, "sf_13"))

	deleted, err := h.repo.DeleteResolvedBefore(h.ctx, watchtowerports.DeleteResolvedWatchtowerItemsRequest{
		Before: baseTime + 1_000_000,
		Limit:  100,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, deleted)
	assert.Equal(t, watchtower.StatusResolved, h.get(t, held.ID).Status)
	_, err = h.repo.GetByID(h.ctx, watchtowerports.GetWatchtowerItemRequest{ID: cleared.ID, TenantInfo: h.tenant})
	require.Error(t, err)
}
