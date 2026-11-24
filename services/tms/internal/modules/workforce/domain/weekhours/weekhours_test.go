package weekhours_test

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/workforce/domain/timesheet"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/weekhours"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerschedule"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var weekStart = time.Date(2026, time.November, 22, 0, 0, 0, 0, time.UTC)

func at(day, hour int) int64 {
	return weekStart.AddDate(0, 0, day).Add(time.Duration(hour) * time.Hour).Unix()
}

func dockNights() *workerschedule.ShiftTemplate {
	return &workerschedule.ShiftTemplate{
		Name:            "Dock nights",
		DaysOfWeek:      "1101101",
		StartMinute:     22 * 60,
		DurationMinutes: 600,
		CycleWeeks:      1,
	}
}

func shift(clockIn, clockOut int64) timesheet.TimeClockEntry {
	return timesheet.TimeClockEntry{ClockedInAt: clockIn, ClockedOutAt: &clockOut}
}

func input(now int64) *weekhours.Input {
	return &weekhours.Input{
		WeekStart:   weekStart.Unix(),
		Location:    time.UTC,
		Now:         now,
		Assignments: []weekhours.Assignment{{Template: dockNights(), From: at(-56, 0)}},
		Entries:     []timesheet.TimeClockEntry{shift(at(0, 22), at(1, 8))},
	}
}

func TestProjectCountsWorkedShiftsAndTheRotaStillToCome(t *testing.T) {
	t.Parallel()

	week := weekhours.Project(input(at(1, 12)))

	assert.Equal(t, int32(600), week.Worked)
	assert.Equal(t, int32(2400), week.Remaining)
	assert.Equal(
		t,
		int32(2400),
		week.Threshold,
		"a worker with no timesheet takes the default line",
	)
	assert.Equal(t, int32(600), week.Overtime())
	assert.True(t, week.Over())
	assert.False(t, week.AlreadyOver())
	assert.Equal(t, "Dock nights", week.Shift)
	assert.Equal(t, 6, week.CrossDay, "the Saturday shift is the one past the line")
	assert.True(t, week.Days[6].Crosses)
	assert.False(t, week.Days[4].Crosses, "reaching the line exactly is not crossing it")

	states := make([]weekhours.DayState, 0, weekhours.DaysInWeek)
	for _, day := range week.Days {
		states = append(states, day.State)
	}
	assert.Equal(t, []weekhours.DayState{
		weekhours.DayWorked, weekhours.DayScheduled, weekhours.DayOff, weekhours.DayScheduled,
		weekhours.DayScheduled, weekhours.DayOff, weekhours.DayScheduled,
	}, states)
}

func TestProjectLeavesOutDaysAwayAndShiftsNobodyWorked(t *testing.T) {
	t.Parallel()

	in := input(at(1, 12))
	in.Entries = nil
	in.Away = []weekhours.Span{{From: at(3, 0), To: at(3, 0)}}
	week := weekhours.Project(in)

	assert.Equal(t, weekhours.DayMissed, week.Days[0].State, "Sunday's shift ended unclocked")
	assert.Equal(t, weekhours.DayAway, week.Days[3].State)
	assert.Equal(t, int32(0), week.Worked)
	assert.Equal(t, int32(1800), week.Planned())
	assert.Equal(t, int32(600), week.Room())
	assert.Equal(t, -1, week.CrossDay)
}

func TestProjectCountsAShiftInProgressAsStillToCome(t *testing.T) {
	t.Parallel()

	in := input(at(0, 23))
	in.Entries = nil
	week := weekhours.Project(in)

	assert.Equal(t, weekhours.DayScheduled, week.Days[0].State)
	assert.Equal(t, int32(3000), week.Planned())
}

func TestProjectTakesTheTimesheetsLineAndSeesAWeekAlreadyOver(t *testing.T) {
	t.Parallel()

	in := input(at(5, 12))
	in.Threshold = 1200
	in.Entries = []timesheet.TimeClockEntry{
		shift(at(0, 22), at(1, 8)),
		shift(at(1, 22), at(2, 8)),
		shift(at(3, 22), at(4, 8)),
	}
	week := weekhours.Project(in)

	assert.Equal(t, int32(1200), week.Threshold)
	assert.Equal(t, int32(1800), week.Worked)
	assert.True(t, week.AlreadyOver())
	assert.Equal(t, 3, week.CrossDay)
}

func TestProjectStopsTheRotaWhereTheAssignmentEnds(t *testing.T) {
	t.Parallel()

	in := input(at(1, 12))
	ends := at(2, 12)
	in.Assignments[0].To = &ends
	week := weekhours.Project(in)

	assert.Equal(t, weekhours.DayOff, week.Days[3].State)
	assert.Equal(t, int32(1200), week.Planned())
	assert.False(t, week.OnLine())
	assert.Equal(t, int32(1200), week.Room())
}

func TestBoundsAndTheTimesheetPeriodFollowTheOrganizationsSunday(t *testing.T) {
	t.Parallel()

	loc, err := time.LoadLocation("America/Los_Angeles")
	require.NoError(t, err)
	now := time.Date(2026, time.November, 23, 9, 41, 0, 0, loc).Unix()

	start, end := weekhours.Bounds(now, loc)

	assert.Equal(t, time.Date(2026, time.November, 22, 0, 0, 0, 0, loc).Unix(), start)
	assert.Equal(t, time.Date(2026, time.November, 29, 0, 0, 0, 0, loc).Unix(), end)
	assert.Equal(
		t,
		time.Date(2026, time.November, 22, 0, 0, 0, 0, time.UTC).Unix(),
		weekhours.TimesheetPeriod(start, loc),
		"the timesheet of the week is the one whose period starts on the same Sunday",
	)
}
