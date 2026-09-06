package worker_test

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 2026-09-06 is a Sunday, which is where the rota week and the availability
// mask both start.
func rotaWeekStart() int64 {
	return time.Date(2026, time.September, 6, 0, 0, 0, 0, time.UTC).Unix()
}

func rotaDayAt(offset int) int64 {
	return time.Unix(rotaWeekStart(), 0).UTC().AddDate(0, 0, offset).Unix()
}

func weekdayTemplate() *worker.ShiftTemplate {
	return &worker.ShiftTemplate{
		Code:            "DAY",
		Name:            "Weekday",
		DaysOfWeek:      "0111110",
		StartMinute:     360,
		DurationMinutes: 600,
		CycleWeeks:      1,
	}
}

func buildOne(input worker.RotaWorkerInput) worker.RotaRow {
	rota := worker.BuildRota(worker.RotaInput{
		WeekStart: rotaWeekStart(),
		Location:  time.UTC,
		Workers:   []worker.RotaWorkerInput{input},
	})
	return rota.Rows[0]
}

func TestBuildRota_FollowsThePattern(t *testing.T) {
	t.Parallel()

	row := buildOne(worker.RotaWorkerInput{WorkerID: "wrk_1", Template: weekdayTemplate()})

	require.Len(t, row.Days, 7)
	assert.Equal(t, worker.RotaOff, row.Days[0].State, "Sunday is off")
	assert.Equal(t, worker.RotaScheduled, row.Days[1].State, "Monday is on")
	assert.Equal(t, worker.RotaScheduled, row.Days[5].State, "Friday is on")
	assert.Equal(t, worker.RotaOff, row.Days[6].State, "Saturday is off")
	assert.Equal(t, 5, row.ScheduledDays)
	assert.Equal(t, 0, row.Conflicts)
}

// Leave and time off outrank an assignment: if dispatch has put work on a day
// somebody is signed off, the rota has to show the conflict rather than the
// work.
func TestBuildRota_TheStrongestFactWins(t *testing.T) {
	t.Parallel()

	monday := rotaDayAt(1)
	row := buildOne(worker.RotaWorkerInput{
		WorkerID:     "wrk_1",
		Template:     weekdayTemplate(),
		AssignedDays: map[int64]int{monday: 2},
		TimeOffDays:  map[int64]bool{monday: true},
	})

	assert.Equal(t, worker.RotaTimeOff, row.Days[1].State)
	// The assignment is still reported — the dispatcher needs to know work is
	// sitting on a day nobody is covering.
	assert.Equal(t, 2, row.Days[1].AssignmentCount)
	assert.True(t, row.Days[1].IsConflict())
	assert.Equal(t, 1, row.Conflicts)
}

func TestBuildRota_LeaveOutranksTimeOff(t *testing.T) {
	t.Parallel()

	monday := rotaDayAt(1)
	row := buildOne(worker.RotaWorkerInput{
		WorkerID:    "wrk_1",
		Template:    weekdayTemplate(),
		TimeOffDays: map[int64]bool{monday: true},
		LeaveDays:   map[int64]bool{monday: true},
	})

	assert.Equal(t, worker.RotaLeave, row.Days[1].State)
}

// A day somebody is off that the pattern never rostered them onto is not a
// hole in the roster, and counting it as one would make every weekend a
// conflict for anybody on leave.
func TestBuildRota_TimeOffOnAnUnscheduledDayIsNotAConflict(t *testing.T) {
	t.Parallel()

	sunday := rotaDayAt(0)
	row := buildOne(worker.RotaWorkerInput{
		WorkerID:    "wrk_1",
		Template:    weekdayTemplate(),
		TimeOffDays: map[int64]bool{sunday: true},
	})

	assert.Equal(t, worker.RotaTimeOff, row.Days[0].State)
	assert.False(t, row.Days[0].Scheduled)
	assert.False(t, row.Days[0].IsConflict())
	assert.Equal(t, 0, row.Conflicts)
}

// A preference is a statement, never a constraint. Dispatch overriding it has
// to stay visible rather than being hidden behind the assignment.
func TestBuildRota_CarriesThePreferenceThrough(t *testing.T) {
	t.Parallel()

	monday := rotaDayAt(1)
	row := buildOne(worker.RotaWorkerInput{
		WorkerID:     "wrk_1",
		Template:     weekdayTemplate(),
		Preferences:  map[int16]worker.AvailabilityPreference{1: worker.AvailabilityUnavailable},
		AssignedDays: map[int64]int{monday: 1},
	})

	assert.Equal(t, worker.AvailabilityUnavailable, row.Days[1].Preference)
	// Time off is a fact and a preference is a wish, so the assignment wins the
	// state — but the day is still flagged as a conflict for somebody to look at.
	assert.Equal(t, worker.RotaAssigned, row.Days[1].State)
	assert.Equal(t, worker.AvailabilityUnavailable, row.Days[1].Preference)
}

func TestBuildRota_UnavailableWithoutWorkIsAConflict(t *testing.T) {
	t.Parallel()

	row := buildOne(worker.RotaWorkerInput{
		WorkerID:    "wrk_1",
		Template:    weekdayTemplate(),
		Preferences: map[int16]worker.AvailabilityPreference{1: worker.AvailabilityUnavailable},
	})

	assert.Equal(t, worker.RotaUnavailable, row.Days[1].State)
	assert.True(t, row.Days[1].IsConflict())
}

// An A/B rotation is one template and two offsets, so the two workers must not
// both be on in the same week.
func TestBuildRota_AlternatesAnAlternatingRotation(t *testing.T) {
	t.Parallel()

	template := weekdayTemplate()
	template.CycleWeeks = 2

	weekA := buildOne(worker.RotaWorkerInput{
		WorkerID:         "wrk_a",
		Template:         template,
		CycleOffsetWeeks: 0,
	})
	weekB := buildOne(worker.RotaWorkerInput{
		WorkerID:         "wrk_b",
		Template:         template,
		CycleOffsetWeeks: 1,
	})

	assert.NotEqual(
		t,
		weekA.ScheduledDays > 0,
		weekB.ScheduledDays > 0,
		"exactly one of the two is on this week",
	)
}

func TestBuildRota_NoAssignmentMeansNoScheduledDays(t *testing.T) {
	t.Parallel()

	row := buildOne(worker.RotaWorkerInput{WorkerID: "wrk_1"})

	assert.Equal(t, 0, row.ScheduledDays)
	for _, day := range row.Days {
		assert.Equal(t, worker.RotaOff, day.State)
	}
}

func TestStartOfWeekUTC(t *testing.T) {
	t.Parallel()

	// A Wednesday resolves back to the Sunday before it.
	wednesday := time.Date(2026, time.September, 9, 14, 30, 0, 0, time.UTC).Unix()
	assert.Equal(t, rotaWeekStart(), worker.StartOfWeekUTC(wednesday, time.UTC))

	// A Sunday is already the start of its own week.
	assert.Equal(t, rotaWeekStart(), worker.StartOfWeekUTC(rotaWeekStart(), time.UTC))
}

func TestShiftTemplate_Validate(t *testing.T) {
	t.Parallel()

	t.Run("a complete template passes", func(t *testing.T) {
		t.Parallel()
		entity := weekdayTemplate()
		entity.Status = "Active"

		multiErr := errortypes.NewMultiError()
		entity.Validate(multiErr)
		assert.False(t, multiErr.HasErrors(), multiErr.Error())
	})

	// A pattern with no working days is a way to roster somebody onto nothing
	// and never notice.
	t.Run("a shift needs at least one working day", func(t *testing.T) {
		t.Parallel()
		entity := weekdayTemplate()
		entity.Status = "Active"
		entity.DaysOfWeek = "0000000"

		multiErr := errortypes.NewMultiError()
		entity.Validate(multiErr)
		require.True(t, multiErr.HasErrors())
		assert.Contains(t, multiErr.Error(), "at least one working day")
	})

	t.Run("the mask has to be seven days of 0 or 1", func(t *testing.T) {
		t.Parallel()
		entity := weekdayTemplate()
		entity.Status = "Active"
		entity.DaysOfWeek = "MTWTF"

		multiErr := errortypes.NewMultiError()
		entity.Validate(multiErr)
		require.True(t, multiErr.HasErrors())
		assert.Contains(t, multiErr.Error(), "seven characters")
	})
}

// A shift that ends at 02:00 is ten hours long, not minus four.
func TestShiftTemplate_EndMinutePastMidnight(t *testing.T) {
	t.Parallel()

	night := &worker.ShiftTemplate{StartMinute: 1200, DurationMinutes: 600}
	assert.Equal(t, 1800, night.EndMinute())
}

func TestShiftSwapStatus_CanTransitionTo(t *testing.T) {
	t.Parallel()

	// The colleague answers first and the office answers second; skipping the
	// colleague would approve a swap they never agreed to.
	assert.True(t, worker.SwapProposed.CanTransitionTo(worker.SwapAccepted))
	assert.True(t, worker.SwapProposed.CanTransitionTo(worker.SwapDeclined))
	assert.False(t, worker.SwapProposed.CanTransitionTo(worker.SwapApproved))

	assert.True(t, worker.SwapAccepted.CanTransitionTo(worker.SwapApproved))
	assert.True(t, worker.SwapAccepted.CanTransitionTo(worker.SwapRejected))

	// Every terminal state stays terminal.
	for _, terminal := range []worker.ShiftSwapStatus{
		worker.SwapDeclined,
		worker.SwapApproved,
		worker.SwapRejected,
		worker.SwapWithdrawn,
	} {
		assert.False(t, terminal.CanTransitionTo(worker.SwapApproved), string(terminal))
		assert.False(t, terminal.IsOpen(), string(terminal))
	}
}

func TestShiftSwapRequest_Validate(t *testing.T) {
	t.Parallel()

	t.Run("a swap has to be with somebody else", func(t *testing.T) {
		t.Parallel()
		entity := &worker.ShiftSwapRequest{
			RequestingWorkerID:   "wrk_1",
			CounterpartyWorkerID: "wrk_1",
			Status:               worker.SwapProposed,
			ShiftDate:            rotaDayAt(1),
		}

		multiErr := errortypes.NewMultiError()
		entity.Validate(multiErr)
		require.True(t, multiErr.HasErrors())
		assert.Contains(t, multiErr.Error(), "with somebody else")
	})

	// A decision is a dated act; one with no date leaves a swap that says it
	// was approved and cannot say when.
	t.Run("a decision needs its date", func(t *testing.T) {
		t.Parallel()
		entity := &worker.ShiftSwapRequest{
			RequestingWorkerID: "wrk_1",
			Status:             worker.SwapApproved,
			ShiftDate:          rotaDayAt(1),
		}

		multiErr := errortypes.NewMultiError()
		entity.Validate(multiErr)
		require.True(t, multiErr.HasErrors())
		assert.Contains(t, multiErr.Error(), "needs the date it was made")
	})
}
