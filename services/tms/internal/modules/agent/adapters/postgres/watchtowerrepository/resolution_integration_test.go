//go:build integration

package watchtowerrepository_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower/watchtowerdb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (h *harness) resolutions(t *testing.T, itemID pulid.ID) []*watchtower.Resolution {
	t.Helper()
	wr := watchtowerdb.Resolutions
	rows, err := wr.Where(wr.ItemID.Eq(itemID)).
		Order(wr.ItemVersion.Asc()).
		Limit(10).
		List(h.ctx, h.db)
	require.NoError(t, err)
	return rows
}

func (h *harness) resolveAs(
	t *testing.T,
	item *watchtower.Item,
	via watchtower.ResolvedVia,
	at int64,
) *watchtower.Item {
	t.Helper()
	working := *item
	require.NoError(t, working.Resolve(watchtower.ResolutionCodeCarrierDelay, "", h.userID, at))
	saved, err := h.repo.UpdateWorkflow(h.ctx, &watchtowerports.UpdateWatchtowerWorkflowRequest{
		Item:            &working,
		ExpectedVersion: item.Version,
		Events: []*watchtower.Event{
			watchtower.NewEvent(&working, watchtower.EventResolved, h.userID),
		},
		ResolvedVia: via,
	})
	require.NoError(t, err)
	return saved
}

func TestAPersonsResolutionIsRecordedWithTheResolve(t *testing.T) {
	h := newHarness(t)

	raised := h.upsert(t, "sf_r1", watchtower.SeverityCritical)
	assigned := *raised
	assigned.AssigneeID = h.userID
	due := baseTime + 60
	assigned.DueAt = &due
	open := h.update(t, &assigned)

	resolved := h.resolveAs(t, open, watchtower.ResolvedViaPerson, baseTime+600)

	rows := h.resolutions(t, raised.ID)
	require.Len(t, rows, 1)
	row := rows[0]
	assert.Equal(t, resolved.Version, row.ItemVersion)
	assert.Equal(t, raised.ID, row.CaseID, "an item about nothing else is its own case")
	assert.Equal(t, watchtower.SourceKindServiceFailure, row.Kind)
	assert.Equal(t, watchtower.SeverityCritical, row.Severity)
	assert.Equal(t, watchtower.ResolvedViaPerson, row.ResolvedVia)
	assert.Equal(t, watchtower.ResolutionCodeCarrierDelay, row.ResolutionCode)
	assert.Equal(t, h.userID, row.AssigneeID)
	assert.Equal(t, h.userID, row.ResolvedByID)
	assert.Equal(t, int64(600), row.SecondsToResolve)
	assert.True(t, row.PastDue, "resolved ten minutes after a one-minute due time")
}

func TestASourceClearIsRecordedWithNobodyAsTheResolver(t *testing.T) {
	h := newHarness(t)

	raised := h.upsert(t, "sf_r2", watchtower.SeverityWarning)
	require.NotNil(t, h.resolveSource(t, "sf_r2"))

	rows := h.resolutions(t, raised.ID)
	require.Len(t, rows, 1)
	assert.Equal(t, watchtower.ResolvedViaSourceCleared, rows[0].ResolvedVia)
	assert.Equal(t, watchtower.ResolutionCodeCleared, rows[0].ResolutionCode)
	assert.True(t, rows[0].ResolvedByID.IsNil())
	assert.False(t, rows[0].PastDue)

	other := h.upsert(t, "sf_r3", watchtower.SeverityWarning)
	resolved, err := h.repo.ResolveMissing(
		h.ctx,
		&watchtowerports.ResolveMissingWatchtowerItemsRequest{
			TenantInfo:    h.tenant,
			SourceKind:    watchtower.SourceKindServiceFailure,
			OpenSourceIDs: []string{"sf_r2"},
			ResolvedAt:    baseTime + 200,
		},
	)
	require.NoError(t, err)
	assert.Equal(t, 1, resolved)
	swept := h.resolutions(t, other.ID)
	require.Len(t, swept, 1, "a bulk clear records each item it resolves")
	assert.Equal(t, watchtower.ResolvedViaSourceCleared, swept[0].ResolvedVia)
	assert.Equal(t, int64(200), swept[0].SecondsToResolve)
}

func TestAReopenedItemResolvedAgainIsASecondResolution(t *testing.T) {
	h := newHarness(t)

	raised := h.upsert(t, "sf_r4", watchtower.SeverityWarning)
	first := h.resolveAs(t, raised, watchtower.ResolvedViaPerson, baseTime+100)

	reopened := *first
	require.NoError(t, reopened.Reopen())
	open := h.update(t, &reopened)
	h.resolveAs(t, open, watchtower.ResolvedViaAgent, baseTime+300)

	rows := h.resolutions(t, raised.ID)
	require.Len(t, rows, 2)
	assert.Equal(t, watchtower.ResolvedViaPerson, rows[0].ResolvedVia)
	assert.Equal(t, watchtower.ResolvedViaAgent, rows[1].ResolvedVia)
	assert.Equal(t, h.userID, rows[1].ResolvedByID, "an agent's resolution is the person's")
	assert.Less(t, rows[0].ItemVersion, rows[1].ItemVersion)
}

func TestTheSameResolutionIsRecordedOnce(t *testing.T) {
	h := newHarness(t)

	raised := h.upsert(t, "sf_r5", watchtower.SeverityInfo)
	resolved := h.resolveAs(t, raised, watchtower.ResolvedViaPerson, baseTime+100)

	again := watchtower.ResolutionOf(resolved, watchtower.ResolvedViaPerson)
	again.ID = pulid.MustNew("wtr_")
	inserted, err := watchtowerdb.Resolutions.CreateOrSkip(
		h.ctx,
		h.db,
		[]*watchtower.Resolution{again},
		watchtowerdb.Resolutions.OrganizationID,
		watchtowerdb.Resolutions.BusinessUnitID,
		watchtowerdb.Resolutions.ItemID,
		watchtowerdb.Resolutions.ItemVersion,
	)
	require.NoError(t, err)
	assert.Zero(t, inserted)
	assert.Len(t, h.resolutions(t, raised.ID), 1)
}

func TestResolutionsOutliveTheItemAndArePrunedOnTheirOwnRetention(t *testing.T) {
	h := newHarness(t)

	raised := h.upsert(t, "sf_r6", watchtower.SeverityWarning)
	require.NotNil(t, h.resolveSource(t, "sf_r6"))

	deleted, err := h.repo.DeleteResolvedBefore(
		h.ctx,
		watchtowerports.DeleteResolvedWatchtowerItemsRequest{Before: baseTime + 1_000, Limit: 100},
	)
	require.NoError(t, err)
	assert.Equal(t, 1, deleted)
	assert.Len(t, h.resolutions(t, raised.ID), 1, "the item is gone and its resolution stays")

	pruned, err := h.repo.DeleteResolutionsBefore(
		h.ctx,
		watchtowerports.DeleteResolvedWatchtowerItemsRequest{Before: baseTime + 1_000, Limit: 100},
	)
	require.NoError(t, err)
	assert.Equal(t, 1, pruned)
	assert.Empty(t, h.resolutions(t, raised.ID))
}
