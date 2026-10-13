package hosprojection

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestScheduleDrive_ShortDriveNeedsNoRest(t *testing.T) {
	t.Parallel()

	s := NewSchedule(fullClocks(), usPropertyLimits())
	elapsed, rest := s.Drive(hoursMs(3))

	assert.Equal(t, hoursMs(3), elapsed)
	assert.Equal(t, Rest{}, rest)
	assert.Equal(t, hoursMs(8), s.Clocks.DriveMs)
	assert.Equal(t, hoursMs(5), s.Clocks.BreakMs)
}

func TestScheduleDrive_InsertsBreakThenReset(t *testing.T) {
	t.Parallel()

	s := NewSchedule(fullClocks(), usPropertyLimits())
	elapsed, rest := s.Drive(hoursMs(15))

	assert.Equal(t, 1, rest.Breaks)
	assert.Equal(t, 1, rest.Resets)
	assert.Equal(t, hoursMs(10.5), rest.RestMs)
	assert.Equal(t, hoursMs(25.5), elapsed)
}

func TestScheduleDrive_CycleExhaustionTakesRestart(t *testing.T) {
	t.Parallel()

	clocks := fullClocks()
	clocks.CycleMs = hoursMs(2)
	s := NewSchedule(clocks, usPropertyLimits())
	elapsed, rest := s.Drive(hoursMs(4))

	assert.Equal(t, 1, rest.Restarts)
	assert.Equal(t, hoursMs(34), rest.RestMs)
	assert.Equal(t, hoursMs(38), elapsed)
	assert.Equal(t, hoursMs(68), s.Clocks.CycleMs)
}

func TestScheduleDrive_SpentClocksResetFirst(t *testing.T) {
	t.Parallel()

	s := NewSchedule(
		Clocks{DriveMs: 0, ShiftMs: hoursMs(3), CycleMs: hoursMs(40), BreakMs: hoursMs(8)},
		usPropertyLimits(),
	)
	elapsed, rest := s.Drive(hoursMs(1))

	assert.Equal(t, 1, rest.Resets)
	assert.Equal(t, hoursMs(11), elapsed)
}

func TestScheduleDrive_UnusableLimitsDriveStraight(t *testing.T) {
	t.Parallel()

	s := NewSchedule(fullClocks(), Limits{})
	elapsed, rest := s.Drive(hoursMs(30))

	assert.Equal(t, hoursMs(30), elapsed)
	assert.Equal(t, Rest{}, rest)
}

func TestScheduleDuty(t *testing.T) {
	t.Parallel()

	s := NewSchedule(
		Clocks{DriveMs: hoursMs(2), ShiftMs: hoursMs(4), CycleMs: hoursMs(10), BreakMs: hoursMs(1)},
		usPropertyLimits(),
	)
	s.OnDuty(hoursMs(1))
	assert.Equal(t, hoursMs(3), s.Clocks.ShiftMs)
	assert.Equal(t, hoursMs(9), s.Clocks.CycleMs)
	assert.Equal(t, hoursMs(8), s.Clocks.BreakMs)

	s.OffDuty(hoursMs(5))
	assert.Equal(t, int64(0), s.Clocks.ShiftMs)

	s.OffDuty(hoursMs(10))
	assert.Equal(t, hoursMs(11), s.Clocks.DriveMs)
	assert.Equal(t, hoursMs(14), s.Clocks.ShiftMs)
	assert.Equal(t, hoursMs(9), s.Clocks.CycleMs)

	s.OffDuty(hoursMs(34))
	assert.Equal(t, hoursMs(70), s.Clocks.CycleMs)
}
