package aituneup_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/aituneup"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func stored(id, fingerprint string, status aituneup.Status) *aituneup.TuneUp {
	return &aituneup.TuneUp{
		ID:          pulid.ID(id),
		Kind:        aituneup.KindLeaveShadow,
		Fingerprint: fingerprint,
		Status:      status,
		ComputedAt:  now - day,
		Version:     3,
	}
}

func candidate(fingerprint string, recorded int) *aituneup.TuneUp {
	return &aituneup.TuneUp{
		Kind:        aituneup.KindLeaveShadow,
		Fingerprint: fingerprint,
		Status:      aituneup.StatusOpen,
		ComputedAt:  now,
		Evidence:    aituneup.Evidence{Recorded: recorded},
	}
}

func at(value int64) *int64 { return &value }

func TestReconcileInsertsNewAndRefreshesOpen(t *testing.T) {
	t.Parallel()

	open := stored("aitu_open", "LeaveShadow:a", aituneup.StatusOpen)
	plan := aituneup.Reconcile(
		[]*aituneup.TuneUp{open},
		[]*aituneup.TuneUp{candidate("LeaveShadow:a", 40), candidate("LeaveShadow:b", 22)},
		now,
	)

	require.Len(t, plan.Insert, 1)
	assert.Equal(t, "LeaveShadow:b", plan.Insert[0].Fingerprint)
	require.Len(t, plan.Update, 1)
	assert.Equal(t, open.ID, plan.Update[0].ID)
	assert.Equal(t, int64(3), plan.Update[0].Version, "the stored version guards the write")
	assert.Equal(t, 40, plan.Update[0].Evidence.Recorded)
	assert.Equal(t, now, plan.Update[0].ComputedAt)
	assert.Empty(t, plan.Delete)
	assert.Equal(t, 0, open.Evidence.Recorded, "the stored row is not changed in place")
}

func TestReconcileKeepsADismissalUntilItRunsOut(t *testing.T) {
	t.Parallel()

	snoozed := stored("aitu_snoozed", "LeaveShadow:a", aituneup.StatusDismissed)
	snoozed.DismissedUntil = at(now + day)
	lapsed := stored("aitu_lapsed", "LeaveShadow:b", aituneup.StatusDismissed)
	lapsed.DismissedUntil = at(now - 1)
	lapsed.DecidedByID = pulid.ID("usr_1")
	lapsed.DecidedAt = at(now - 31*day)

	plan := aituneup.Reconcile(
		[]*aituneup.TuneUp{snoozed, lapsed},
		[]*aituneup.TuneUp{candidate("LeaveShadow:a", 25), candidate("LeaveShadow:b", 25)},
		now,
	)

	require.Len(t, plan.Update, 2)
	byID := map[pulid.ID]*aituneup.TuneUp{}
	for _, tuneUp := range plan.Update {
		byID[tuneUp.ID] = tuneUp
	}
	assert.Equal(t, aituneup.StatusDismissed, byID[snoozed.ID].Status)
	assert.Equal(t, 25, byID[snoozed.ID].Evidence.Recorded, "its evidence still moves")
	assert.Equal(t, aituneup.StatusOpen, byID[lapsed.ID].Status)
	assert.Nil(t, byID[lapsed.ID].DismissedUntil)
	assert.Nil(t, byID[lapsed.ID].DecidedAt)
	assert.True(t, byID[lapsed.ID].DecidedByID.IsNil())
}

func TestReconcileLeavesARecentlyAppliedChangeAlone(t *testing.T) {
	t.Parallel()

	recent := stored("aitu_recent", "LeaveShadow:a", aituneup.StatusApplied)
	recent.DecidedAt = at(now - 2*day)
	old := stored("aitu_old", "LeaveShadow:b", aituneup.StatusApplied)
	old.DecidedAt = at(now - 40*day)

	plan := aituneup.Reconcile(
		[]*aituneup.TuneUp{recent, old},
		[]*aituneup.TuneUp{candidate("LeaveShadow:a", 25), candidate("LeaveShadow:b", 25)},
		now,
	)

	require.Len(t, plan.Update, 1)
	assert.Equal(t, old.ID, plan.Update[0].ID)
	assert.Equal(t, aituneup.StatusOpen, plan.Update[0].Status, "the condition came back after it settled")
	assert.Empty(t, plan.Insert)
}

func TestReconcileDropsWhatNoLongerApplies(t *testing.T) {
	t.Parallel()

	open := stored("aitu_open", "LeaveShadow:a", aituneup.StatusOpen)
	snoozed := stored("aitu_snoozed", "LeaveShadow:b", aituneup.StatusDismissed)
	snoozed.DismissedUntil = at(now + day)
	lapsed := stored("aitu_lapsed", "LeaveShadow:c", aituneup.StatusDismissed)
	lapsed.DismissedUntil = at(now)
	recent := stored("aitu_recent", "LeaveShadow:d", aituneup.StatusApplied)
	recent.DecidedAt = at(now - day)
	old := stored("aitu_old", "LeaveShadow:e", aituneup.StatusApplied)
	old.DecidedAt = at(now - 30*day)

	plan := aituneup.Reconcile([]*aituneup.TuneUp{open, snoozed, lapsed, recent, old}, nil, now)

	assert.ElementsMatch(t, []pulid.ID{open.ID, lapsed.ID, old.ID}, plan.Delete,
		"a dismissal and a recent apply are kept so the change is not offered again too soon")
	assert.False(t, plan.Empty())
	empty := aituneup.Reconcile(nil, nil, now)
	assert.True(t, empty.Empty())
}

func TestVisible(t *testing.T) {
	t.Parallel()

	assert.True(t, stored("a", "x", aituneup.StatusOpen).Visible(now))
	assert.False(t, stored("a", "x", aituneup.StatusApplied).Visible(now))
	snoozed := stored("a", "x", aituneup.StatusDismissed)
	snoozed.DismissedUntil = at(now + 1)
	assert.False(t, snoozed.Visible(now))
	assert.True(t, snoozed.Visible(now+1))
}

func TestClampDismissDays(t *testing.T) {
	t.Parallel()

	assert.Equal(t, aituneup.DefaultDismissDays, aituneup.ClampDismissDays(0))
	assert.Equal(t, 7, aituneup.ClampDismissDays(7))
	assert.Equal(t, aituneup.MaxDismissDays, aituneup.ClampDismissDays(1000))
}
