//go:build integration

package watchtowerrepository_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTheSourcesDeadlineSurvivesARefreshButNotAPersonsChoice(t *testing.T) {
	h := newHarness(t)

	incoming := h.item("sf_deadline", watchtower.SeverityCritical)
	incoming.ApplyDeadline(&watchtower.Deadline{At: baseTime + 3600, Label: "Dock closes"})
	incoming.Facts = []watchtower.Fact{{Label: "Customer", Value: "Northline Foods"}}
	saved, _, err := h.repo.Upsert(h.ctx, incoming)
	require.NoError(t, err)
	require.NotNil(t, saved.DueAt)
	assert.Equal(t, "Dock closes", saved.DueLabel)
	assert.Equal(t, incoming.Facts, h.get(t, saved.ID).Facts)

	moved := h.item("sf_deadline", watchtower.SeverityCritical)
	moved.ApplyDeadline(&watchtower.Deadline{At: baseTime + 7200, Label: "Dock closes"})
	refreshed, _, err := h.repo.Upsert(h.ctx, moved)
	require.NoError(t, err)
	assert.Equal(t, baseTime+7200, *refreshed.DueAt, "the source moved its own deadline")

	chosen := h.get(t, saved.ID)
	require.NoError(t, chosen.SetDue(nil, baseTime))
	h.update(t, chosen)

	again := h.item("sf_deadline", watchtower.SeverityCritical)
	again.ApplyDeadline(&watchtower.Deadline{At: baseTime + 9000, Label: "Dock closes"})
	_, _, err = h.repo.Upsert(h.ctx, again)
	require.NoError(t, err)
	after := h.get(t, saved.ID)
	assert.Nil(t, after.DueAt, "a person cleared the due time and the source does not undo it")
	assert.Equal(t, watchtower.DueSourcePerson, after.DueSource)
}

func TestReadsCountsAndClearedFollowTheReader(t *testing.T) {
	h := newHarness(t)

	opened := h.upsert(t, "sf_read", watchtower.SeverityCritical)
	h.upsert(t, "sf_unread", watchtower.SeverityCritical)
	snoozed := h.upsert(t, "sf_snoozed", watchtower.SeverityCritical)
	resolved := h.upsert(t, "sf_resolved", watchtower.SeverityWarning)

	require.NoError(t, h.repo.MarkRead(h.ctx, &watchtower.Read{
		ItemID:         opened.ID,
		UserID:         h.userID,
		OrganizationID: h.tenant.OrgID,
		BusinessUnitID: h.tenant.BuID,
		SeenAt:         baseTime,
	}))
	require.NoError(t, h.repo.MarkRead(h.ctx, &watchtower.Read{
		ItemID:         opened.ID,
		UserID:         h.userID,
		OrganizationID: h.tenant.OrgID,
		BusinessUnitID: h.tenant.BuID,
		SeenAt:         baseTime + 10,
	}), "reading an item twice is one read")

	read, err := h.repo.ListReadItemIDs(h.ctx, watchtowerports.ListWatchtowerReadsRequest{
		TenantInfo: h.tenant,
		UserID:     h.userID,
		ItemIDs:    []pulid.ID{opened.ID, snoozed.ID},
	})
	require.NoError(t, err)
	assert.Equal(t, map[pulid.ID]struct{}{opened.ID: {}}, read)

	require.NoError(t, snoozed.Snooze(baseTime+3600, baseTime))
	h.update(t, snoozed, watchtower.NewEvent(snoozed, watchtower.EventSnoozed, h.userID))
	require.NoError(t, resolved.Resolve(watchtower.ResolutionFixed, "", h.userID, baseTime+20))
	h.update(t, resolved)

	counts, err := h.repo.Counts(h.ctx, watchtowerports.CountWatchtowerItemsRequest{
		TenantInfo: h.tenant,
		Kinds:      []watchtower.SourceKind{watchtower.SourceServiceFailure},
		SeenAt:     0,
		ReaderID:   h.userID,
		Now:        baseTime,
	})
	require.NoError(t, err)
	assert.Equal(t, 3, counts.Unresolved)
	assert.Equal(t, 2, counts.Active, "a snoozed item is open but not in the queue")
	assert.Equal(t, 2, counts.ActiveCritical)
	assert.Equal(t, 2, counts.UnseenCritical, "the opened item no longer counts as new")

	cleared, err := h.repo.CountCleared(h.ctx, watchtowerports.CountClearedWatchtowerItemsRequest{
		TenantInfo: h.tenant,
		Kinds:      []watchtower.SourceKind{watchtower.SourceServiceFailure},
		UserID:     h.userID,
		Since:      baseTime - 60,
	})
	require.NoError(t, err)
	assert.Equal(t, 2, cleared, "one resolved and one snoozed by the reader")
}

func TestTheQueueReadsSoonestDeadlineFirstAndUndatedLast(t *testing.T) {
	h := newHarness(t)

	later := h.item("sf_later", watchtower.SeverityInfo)
	later.ApplyDeadline(&watchtower.Deadline{At: baseTime + 9000, Label: "Later"})
	soon := h.item("sf_soon", watchtower.SeverityInfo)
	soon.ApplyDeadline(&watchtower.Deadline{At: baseTime + 60, Label: "Soon"})
	undated := h.item("sf_undated", watchtower.SeverityInfo)
	for _, item := range []*watchtower.Item{later, undated, soon} {
		_, _, err := h.repo.Upsert(h.ctx, item)
		require.NoError(t, err)
	}

	items, err := h.repo.List(h.ctx, watchtowerports.ListWatchtowerItemsQuery{
		TenantInfo:     h.tenant,
		Kinds:          []watchtower.SourceKind{watchtower.SourceServiceFailure},
		UnresolvedOnly: true,
		Statuses:       []watchtower.Status{watchtower.StatusOpen},
		OrderByDue:     true,
		Limit:          10,
	})
	require.NoError(t, err)
	order := make([]string, 0, len(items))
	for _, item := range items {
		order = append(order, item.SourceID)
	}
	assert.Equal(t, []string{"sf_soon", "sf_later", "sf_undated"}, order)
}
