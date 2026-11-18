package managedfeatures_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedfeatures"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	today      = int64(1_794_650_400)
	oneDay     = int64(86_400)
	periodEnd  = today + 30*oneDay
	nextPeriod = today + 60*oneDay
)

func planEnding(held *managedfeatures.State, endsAt, at int64) *managedfeatures.Replacement {
	return managedfeatures.Replace(held, &managedfeatures.Received{
		Features:   []platformcatalog.FeatureKey{"tms.accounting"},
		PlanEndsAt: endsAt,
	}, at)
}

func TestAPlanEndAlreadyPastIsStoredWithoutAnnouncingIt(t *testing.T) {
	t.Parallel()

	ended := planEnding(nil, today-2*oneDay, today)
	require.True(t, ended.Updated())
	assert.Nil(t, ended.PlanEnd, "an end that has passed is not news")
	assert.Equal(t, today-2*oneDay, ended.Current.PlanEndsAt)

	again := planEnding(ended.Current, today-2*oneDay, today+60)
	assert.False(t, again.Updated())

	set := planEnding(nil, periodEnd, today)
	after := planEnding(set.Current, periodEnd, periodEnd+oneDay)
	assert.False(t, after.Updated(), "the end it was told of, still served once it has passed")
	assert.Nil(t, after.PlanEnd, "a plan that ended is never announced as kept")
}

func TestAPlanEndIsRefusedBefore1970OrMoreThan400DaysAhead(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		end  int64
		ok   bool
	}{
		{name: "not ending", end: 0, ok: true},
		{name: "at the close of this period", end: periodEnd, ok: true},
		{name: "an hour ago, while the instance is being taken down", end: today - 3600, ok: true},
		{name: "a yearly period", end: today + 366*oneDay, ok: true},
		{name: "two days ago, while a slow teardown finishes", end: today - 2*oneDay, ok: true},
		{name: "more than 400 days ahead", end: today + 401*oneDay},
		{name: "before 1970", end: -1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := (&managedfeatures.Received{PlanEndsAt: tt.end}).Validate(today)
			if tt.ok {
				assert.NoError(t, err)
				return
			}
			_, refused := errortypes.RefusalOf(err)
			assert.True(t, refused)
		})
	}
}

func TestASetPlanEndIsAnnouncedOnceAndAKeptPlanOnce(t *testing.T) {
	t.Parallel()

	held := &managedfeatures.State{
		ID:       managedfeatures.CurrentStateID,
		Features: []platformcatalog.FeatureKey{"tms.accounting"},
	}

	set := planEnding(held, periodEnd, today)
	require.True(t, set.Updated())
	require.NotNil(t, set.PlanEnd)
	assert.Equal(t, managedfeatures.PlanEndSet, set.PlanEnd.Kind)
	assert.Equal(t, periodEnd, set.PlanEnd.EndsAt)
	assert.Equal(t, periodEnd, set.Current.PlanEndsAt)
	assert.Equal(t, periodEnd, set.Current.PlanEndAnnounced)
	assert.Zero(t, set.Current.PlanEndReminded, "the reminder is still to come")
	assert.False(t, set.FeaturesChanged())

	same := planEnding(set.Current, periodEnd, today+3600)
	assert.False(t, same.Updated(), "the same end again stores and announces nothing")
	assert.Nil(t, same.PlanEnd)

	kept := planEnding(set.Current, 0, today+oneDay)
	require.True(t, kept.Updated())
	require.NotNil(t, kept.PlanEnd)
	assert.Equal(t, managedfeatures.PlanEndKept, kept.PlanEnd.Kind)
	assert.Equal(t, periodEnd, kept.PlanEnd.EndsAt, "the kept notice names the end that will not happen")
	assert.Zero(t, kept.Current.PlanEndsAt)
	assert.Zero(t, kept.Current.PlanEndAnnounced)

	stillKept := planEnding(kept.Current, 0, today+2*oneDay)
	assert.False(t, stillKept.Updated(), "a plan that stays kept says nothing more")

	again := planEnding(kept.Current, nextPeriod, today+31*oneDay)
	require.NotNil(t, again.PlanEnd, "a plan set to end again in a later period is a new end")
	assert.Equal(t, managedfeatures.PlanEndSet, again.PlanEnd.Kind)
	assert.Equal(t, nextPeriod, again.PlanEnd.EndsAt)
}

func TestAPlanEndLearnedUnderThreeDaysAheadNeedsNoReminder(t *testing.T) {
	t.Parallel()

	late := planEnding(&managedfeatures.State{ID: managedfeatures.CurrentStateID}, today+2*oneDay, today)

	require.NotNil(t, late.PlanEnd)
	assert.Equal(t, managedfeatures.PlanEndSet, late.PlanEnd.Kind)
	assert.Equal(t, today+2*oneDay, late.Current.PlanEndReminded, "the ends-on notice is enough")
	assert.False(t, late.Current.PlanEndReminderDue(today+oneDay))
	assert.False(t, managedfeatures.RemindPlanEnd(late.Current, today+oneDay).Updated())
}

func TestTheReminderComesOnceInTheLastThreeDays(t *testing.T) {
	t.Parallel()

	set := planEnding(&managedfeatures.State{ID: managedfeatures.CurrentStateID}, periodEnd, today)

	early := managedfeatures.RemindPlanEnd(set.Current, periodEnd-4*oneDay)
	assert.False(t, early.Updated(), "not before three days ahead")

	due := managedfeatures.RemindPlanEnd(set.Current, periodEnd-3*oneDay+60)
	require.True(t, due.Updated())
	require.NotNil(t, due.PlanEnd)
	assert.Equal(t, managedfeatures.PlanEndReminder, due.PlanEnd.Kind)
	assert.Equal(t, periodEnd, due.PlanEnd.EndsAt)
	assert.Equal(t, periodEnd, due.Current.PlanEndReminded)

	twice := managedfeatures.RemindPlanEnd(due.Current, periodEnd-oneDay)
	assert.False(t, twice.Updated(), "the reminder is sent once")

	past := managedfeatures.RemindPlanEnd(set.Current, periodEnd+60)
	assert.False(t, past.Updated(), "nothing once the end has passed")

	assert.False(t, managedfeatures.RemindPlanEnd(nil, today).Updated())
	assert.False(t, managedfeatures.RemindPlanEnd(
		&managedfeatures.State{ID: managedfeatures.CurrentStateID},
		today,
	).Updated(), "nothing while the plan is not ending")
}

func TestAMovedEndIsANewEndAndItsReminderComesAgain(t *testing.T) {
	t.Parallel()

	set := planEnding(&managedfeatures.State{ID: managedfeatures.CurrentStateID}, periodEnd, today)
	reminded := managedfeatures.RemindPlanEnd(set.Current, periodEnd-2*oneDay)
	require.True(t, reminded.Updated())

	moved := planEnding(reminded.Current, nextPeriod, periodEnd-oneDay)
	require.NotNil(t, moved.PlanEnd)
	assert.Equal(t, managedfeatures.PlanEndSet, moved.PlanEnd.Kind)
	assert.True(t, moved.Current.PlanEndReminderDue(nextPeriod-2*oneDay))
}

func TestAChangedFinalBackupDayCountStoresWithoutANotice(t *testing.T) {
	t.Parallel()

	held := &managedfeatures.State{
		ID:              managedfeatures.CurrentStateID,
		Features:        []platformcatalog.FeatureKey{"tms.accounting"},
		FinalBackupDays: 30,
	}

	changed := managedfeatures.Replace(held, &managedfeatures.Received{
		Features:        []platformcatalog.FeatureKey{"tms.accounting"},
		FinalBackupDays: 7,
	}, today)

	require.True(t, changed.Updated())
	assert.Equal(t, 7, changed.Current.FinalBackupDays)
	assert.Nil(t, changed.PlanEnd, "a new day count is not news")
	assert.False(t, changed.Changed(), "nor a change to the plan")
}

func TestAnUntoldDayCountKeepsTheOneHeld(t *testing.T) {
	t.Parallel()

	held := &managedfeatures.State{
		ID:              managedfeatures.CurrentStateID,
		Features:        []platformcatalog.FeatureKey{"tms.accounting"},
		FinalBackupDays: 7,
	}

	same := planEnding(held, 0, today)
	assert.False(t, same.Updated(), "a request that does not say keeps the held number")

	set := planEnding(held, periodEnd, today)
	require.NotNil(t, set.PlanEnd)
	assert.Equal(t, 7, set.Current.FinalBackupDays)
	assert.Equal(t, 7, set.PlanEnd.FinalBackupDays, "the notice says the number held when it is made")

	first := planEnding(nil, periodEnd, today)
	assert.Equal(t, managedfeatures.DefaultFinalBackupDays, first.Current.FinalBackupDays)
	assert.Equal(t, managedfeatures.DefaultFinalBackupDays, first.PlanEnd.FinalBackupDays)
}

func TestTheReminderSaysTheHeldDayCount(t *testing.T) {
	t.Parallel()

	held := &managedfeatures.State{
		ID:               managedfeatures.CurrentStateID,
		PlanEndsAt:       periodEnd,
		PlanEndAnnounced: periodEnd,
		FinalBackupDays:  7,
	}

	reminder := managedfeatures.RemindPlanEnd(held, periodEnd-2*oneDay)

	require.NotNil(t, reminder.PlanEnd)
	assert.Equal(t, 7, reminder.PlanEnd.FinalBackupDays)
}

func TestAFinalBackupDayCountOutOfRangeIsRefused(t *testing.T) {
	t.Parallel()

	for days, ok := range map[int]bool{0: true, 1: true, 30: true, 3650: true, -1: false, 3651: false} {
		err := (&managedfeatures.Received{FinalBackupDays: days}).Validate(today)
		if ok {
			assert.NoError(t, err, "%d days", days)
			continue
		}
		_, refused := errortypes.RefusalOf(err)
		assert.True(t, refused, "%d days must be refused", days)
	}
}
