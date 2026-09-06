package worker_test

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func hours(count int) int64 { return int64(count) * 3600 }

func closedEntry(startHour, endHour int) *worker.TimeClockEntry {
	start := time.Date(2026, time.September, 7, startHour, 0, 0, 0, time.UTC).Unix()
	end := time.Date(2026, time.September, 7, endHour, 0, 0, 0, time.UTC).Unix()
	return &worker.TimeClockEntry{
		WorkerID:     "wrk_1",
		Source:       worker.TimeEntrySourceClock,
		ClockedInAt:  start,
		ClockedOutAt: &end,
	}
}

func TestTimeClockEntry_PaidMinutes(t *testing.T) {
	t.Parallel()

	entry := closedEntry(8, 17)
	assert.Equal(t, int32(540), entry.PaidMinutes())

	// The unpaid break comes out of the period rather than shortening it.
	entry.BreakMinutes = 30
	assert.Equal(t, int32(510), entry.PaidMinutes())
}

// Paying for a shift somebody has not finished would make every roll-up depend
// on when it was read.
func TestTimeClockEntry_AnOpenEntryIsWorthNothingYet(t *testing.T) {
	t.Parallel()

	entry := &worker.TimeClockEntry{
		WorkerID:    "wrk_1",
		Source:      worker.TimeEntrySourceClock,
		ClockedInAt: hours(10),
	}

	assert.True(t, entry.IsOpen())
	assert.Equal(t, int32(0), entry.PaidMinutes())
}

func TestTimeClockEntry_Validate(t *testing.T) {
	t.Parallel()

	t.Run("a closed entry passes", func(t *testing.T) {
		t.Parallel()
		multiErr := errortypes.NewMultiError()
		closedEntry(8, 17).Validate(multiErr)
		assert.False(t, multiErr.HasErrors(), multiErr.Error())
	})

	t.Run("an entry cannot end before it began", func(t *testing.T) {
		t.Parallel()
		entry := closedEntry(17, 8)
		multiErr := errortypes.NewMultiError()
		entry.Validate(multiErr)
		require.True(t, multiErr.HasErrors())
		assert.Contains(t, multiErr.Error(), "end before it began")
	})

	// A punch nobody closed until the next day is a forgotten clock-out rather
	// than a day somebody worked straight through.
	t.Run("an entry cannot be longer than a day", func(t *testing.T) {
		t.Parallel()
		start := time.Date(2026, time.September, 7, 8, 0, 0, 0, time.UTC).Unix()
		end := start + hours(30)
		entry := &worker.TimeClockEntry{
			WorkerID:     "wrk_1",
			Source:       worker.TimeEntrySourceClock,
			ClockedInAt:  start,
			ClockedOutAt: &end,
		}

		multiErr := errortypes.NewMultiError()
		entry.Validate(multiErr)
		require.True(t, multiErr.HasErrors())
		assert.Contains(t, multiErr.Error(), "longer than a day")
	})

	t.Run("a break cannot swallow the entry", func(t *testing.T) {
		t.Parallel()
		entry := closedEntry(8, 9)
		entry.BreakMinutes = 60

		multiErr := errortypes.NewMultiError()
		entry.Validate(multiErr)
		require.True(t, multiErr.HasErrors())
		assert.Contains(t, multiErr.Error(), "nothing would be paid")
	})

	// Subtracting a break from a running entry would make the same shift
	// shrink as it goes.
	t.Run("a running entry cannot carry a break yet", func(t *testing.T) {
		t.Parallel()
		entry := &worker.TimeClockEntry{
			WorkerID:     "wrk_1",
			Source:       worker.TimeEntrySourceClock,
			ClockedInAt:  hours(10),
			BreakMinutes: 30,
		}

		multiErr := errortypes.NewMultiError()
		entry.Validate(multiErr)
		require.True(t, multiErr.HasErrors())
		assert.Contains(t, multiErr.Error(), "recorded when the entry is closed")
	})
}

func TestTimesheetStatus_CanTransitionTo(t *testing.T) {
	t.Parallel()

	assert.True(t, worker.TimesheetOpen.CanTransitionTo(worker.TimesheetSubmitted))
	// A week cannot be approved without somebody handing it over first.
	assert.False(t, worker.TimesheetOpen.CanTransitionTo(worker.TimesheetApproved))

	assert.True(t, worker.TimesheetSubmitted.CanTransitionTo(worker.TimesheetApproved))
	assert.True(t, worker.TimesheetSubmitted.CanTransitionTo(worker.TimesheetRejected))

	// The point of rejecting a week is to have it fixed, so it goes back round.
	assert.True(t, worker.TimesheetRejected.CanTransitionTo(worker.TimesheetSubmitted))

	assert.True(t, worker.TimesheetApproved.CanTransitionTo(worker.TimesheetLocked))
	// Voiding the export is what reopens the week.
	assert.True(t, worker.TimesheetApproved.CanTransitionTo(worker.TimesheetOpen))
	assert.True(t, worker.TimesheetLocked.CanTransitionTo(worker.TimesheetApproved))
	assert.False(t, worker.TimesheetLocked.CanTransitionTo(worker.TimesheetOpen))
}

// A submitted sheet is frozen because its totals are what somebody is being
// asked to sign; an approved one because they already did.
func TestTimesheetStatus_IsEditable(t *testing.T) {
	t.Parallel()

	assert.True(t, worker.TimesheetOpen.IsEditable())
	assert.True(t, worker.TimesheetRejected.IsEditable())
	assert.False(t, worker.TimesheetSubmitted.IsEditable())
	assert.False(t, worker.TimesheetApproved.IsEditable())
	assert.False(t, worker.TimesheetLocked.IsEditable())
}

func TestTimesheet_ApplyTotals(t *testing.T) {
	t.Parallel()

	t.Run("splits at the threshold", func(t *testing.T) {
		t.Parallel()
		sheet := &worker.Timesheet{OvertimeThresholdMinutes: 2400}
		sheet.ApplyTotals(2700, 0, 5)

		assert.Equal(t, int32(2400), sheet.RegularMinutes)
		assert.Equal(t, int32(300), sheet.OvertimeMinutes)
		assert.Equal(t, int32(5), sheet.EntryCount)
	})

	t.Run("a short week is all regular", func(t *testing.T) {
		t.Parallel()
		sheet := &worker.Timesheet{OvertimeThresholdMinutes: 2400}
		sheet.ApplyTotals(1800, 0, 4)

		assert.Equal(t, int32(1800), sheet.RegularMinutes)
		assert.Equal(t, int32(0), sheet.OvertimeMinutes)
	})

	// Time nobody worked does not earn overtime. Counting it would pay a
	// premium on a holiday somebody spent at home.
	t.Run("paid leave sits outside the overtime split", func(t *testing.T) {
		t.Parallel()
		sheet := &worker.Timesheet{OvertimeThresholdMinutes: 2400}
		sheet.ApplyTotals(2400, 480, 5)

		assert.Equal(t, int32(2400), sheet.RegularMinutes)
		assert.Equal(t, int32(0), sheet.OvertimeMinutes)
		assert.Equal(t, int32(480), sheet.PaidLeaveMinutes)
		assert.Equal(t, int32(2880), sheet.TotalMinutes())
	})

	t.Run("falls back to the standard week when no threshold was set", func(t *testing.T) {
		t.Parallel()
		sheet := &worker.Timesheet{}
		sheet.ApplyTotals(2500, 0, 5)

		assert.Equal(t, int32(2400), sheet.RegularMinutes)
		assert.Equal(t, int32(100), sheet.OvertimeMinutes)
	})
}

func TestTimesheet_Validate(t *testing.T) {
	t.Parallel()

	weekStart := worker.StartOfWeekUTC(
		time.Date(2026, time.September, 7, 0, 0, 0, 0, time.UTC).Unix(),
		time.UTC,
	)

	complete := func() *worker.Timesheet {
		return &worker.Timesheet{
			WorkerID:                 "wrk_1",
			Status:                   worker.TimesheetOpen,
			PeriodStart:              weekStart,
			PeriodEnd:                weekStart + 7*86400,
			OvertimeThresholdMinutes: 2400,
		}
	}

	t.Run("a complete sheet passes", func(t *testing.T) {
		t.Parallel()
		multiErr := errortypes.NewMultiError()
		complete().Validate(multiErr)
		assert.False(t, multiErr.HasErrors(), multiErr.Error())
	})

	// More than a week of hours in a week is an entry somebody never closed,
	// and approving it would pay for it.
	t.Run("a week cannot hold more hours than a week has", func(t *testing.T) {
		t.Parallel()
		sheet := complete()
		sheet.RegularMinutes = 2400
		sheet.OvertimeMinutes = 9000

		multiErr := errortypes.NewMultiError()
		sheet.Validate(multiErr)
		require.True(t, multiErr.HasErrors())
		assert.Contains(t, multiErr.Error(), "never closed")
	})

	t.Run("an approval needs its date", func(t *testing.T) {
		t.Parallel()
		sheet := complete()
		sheet.Status = worker.TimesheetApproved

		multiErr := errortypes.NewMultiError()
		sheet.Validate(multiErr)
		require.True(t, multiErr.HasErrors())
		assert.Contains(t, multiErr.Error(), "needs the date it was made")
	})

	t.Run("a period cannot end before it begins", func(t *testing.T) {
		t.Parallel()
		sheet := complete()
		sheet.PeriodEnd = weekStart - 1

		multiErr := errortypes.NewMultiError()
		sheet.Validate(multiErr)
		require.True(t, multiErr.HasErrors())
		assert.Contains(t, multiErr.Error(), "end before it begins")
	})
}

// Voiding a payroll run reopens every week in it, so it is refused without a
// reason: nobody can explain the reopening afterwards otherwise.
func TestPayrollExport_VoidingNeedsAReason(t *testing.T) {
	t.Parallel()

	export := &worker.PayrollExport{
		Status:      worker.PayrollExportVoided,
		PeriodStart: hours(24),
		PeriodEnd:   hours(24 * 8),
	}

	multiErr := errortypes.NewMultiError()
	export.Validate(multiErr)
	require.True(t, multiErr.HasErrors())
	assert.Contains(t, multiErr.Error(), "needs a reason")
}
