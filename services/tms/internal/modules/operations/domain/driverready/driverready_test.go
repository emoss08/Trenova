package driverready_test

import (
	"testing"
	"time"
	_ "time/tzdata"

	"github.com/emoss08/trenova/internal/modules/operations/domain/driverready"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerschedule"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	now  = int64(1_800_000_000)
	hour = int64(3600)
)

func clocks(drive, shift, cycle, tomorrow int64) *driverready.Clocks {
	return &driverready.Clocks{
		RecordedAt:           now - 600,
		DriveSeconds:         drive * hour,
		ShiftSeconds:         shift * hour,
		CycleSeconds:         cycle * hour,
		CycleTomorrowSeconds: tomorrow * hour,
	}
}

func TestAt_SaysWhenADriverCanTakeALoad(t *testing.T) {
	t.Parallel()

	stale := clocks(0, 0, 0, 0)
	stale.RecordedAt = now - driverready.ClocksMaxAgeSeconds - 1
	tests := []struct {
		name string
		in   driverready.Input
		want int64
	}{
		{
			"no clocks: when their work ends",
			driverready.Input{Now: now, FreeAt: now + 5*hour},
			now + 5*hour,
		},
		{"free in the past is free now", driverready.Input{Now: now, FreeAt: now - 9*hour}, now},
		{
			"hours to spare",
			driverready.Input{Now: now, FreeAt: now, Clocks: clocks(8, 10, 40, 0)},
			now,
		},
		{
			"out of drive time now: after a ten-hour rest",
			driverready.Input{Now: now, FreeAt: now, Clocks: clocks(1, 6, 40, 0)},
			now + driverready.ResetSeconds,
		},
		{
			"delivers with too little of the shift left",
			driverready.Input{Now: now, FreeAt: now + 6*hour, Clocks: clocks(8, 8, 40, 0)},
			now + 6*hour + driverready.ResetSeconds,
		},
		{
			"delivers with half a shift left",
			driverready.Input{Now: now, FreeAt: now + 3*hour, Clocks: clocks(9, 11, 40, 0)},
			now + 3*hour,
		},
		{
			"a run longer than this shift: they will have rested on the way",
			driverready.Input{Now: now, FreeAt: now + 30*hour, Clocks: clocks(9, 11, 60, 0)},
			now + 30*hour,
		},
		{
			"out of cycle with nothing coming back: a 34-hour restart",
			driverready.Input{Now: now, FreeAt: now, Clocks: clocks(8, 10, 1, 0)},
			now + driverready.RestartSeconds,
		},
		{
			"out of cycle but hours come back tomorrow",
			driverready.Input{Now: now, FreeAt: now, Clocks: clocks(8, 10, 1, 9)},
			now + driverready.ResetSeconds,
		},
		{
			"the run uses up the cycle",
			driverready.Input{Now: now, FreeAt: now + 5*hour, Clocks: clocks(10, 12, 6, 0)},
			now + 5*hour + driverready.RestartSeconds,
		},
		{"clocks too old to trust", driverready.Input{Now: now, FreeAt: now, Clocks: stale}, now},
		{
			"on approved time off",
			driverready.Input{Now: now, FreeAt: now, TimeOffUntil: now + 72*hour},
			now + 72*hour,
		},
		{
			"time off that ends before the rest does",
			driverready.Input{
				Now: now, FreeAt: now, Clocks: clocks(0, 0, 40, 0), TimeOffUntil: now + 4*hour,
			},
			now + driverready.ResetSeconds,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, driverready.At(tt.in))
		})
	}
}

func weekdayPattern(cycleWeeks, offset int16) *driverready.Pattern {
	return &driverready.Pattern{
		Template: &workerschedule.ShiftTemplate{
			DaysOfWeek: "0111110", StartMinute: 7 * 60, CycleWeeks: cycleWeeks,
		},
		CycleOffsetWeeks: offset,
	}
}

func TestAt_HoldsADriverUntilTheNextDayTheirPatternCovers(t *testing.T) {
	t.Parallel()

	denver, err := time.LoadLocation("America/Denver")
	require.NoError(t, err)
	at := func(day, hour int) int64 {
		return time.Date(2026, time.November, day, hour, 0, 0, 0, denver).Unix()
	}

	friday, saturday, sunday := at(13, 22), at(14, 9), at(15, 23)
	monday := at(16, 7)
	weekly := weekdayPattern(1, 0)
	for name, free := range map[string]int64{"late Friday": friday} {
		assert.Equal(t, free, driverready.At(driverready.Input{
			Now: free, FreeAt: free, Pattern: weekly, Location: denver,
		}), "%s is still a working day", name)
	}
	for name, free := range map[string]int64{"Saturday": saturday, "Sunday night": sunday} {
		assert.Equal(t, monday, driverready.At(driverready.Input{
			Now: free, FreeAt: free, Pattern: weekly, Location: denver,
		}), "%s waits for Monday's start", name)
	}

	assert.Equal(t, saturday, driverready.At(driverready.Input{
		Now: saturday, FreeAt: saturday, Location: denver,
	}), "no pattern holds nobody")
	assert.Equal(t, saturday, driverready.At(driverready.Input{
		Now: saturday, FreeAt: saturday, Location: denver,
		Pattern: &driverready.Pattern{
			Template: &workerschedule.ShiftTemplate{DaysOfWeek: "0000000"},
		},
	}), "a pattern with no working day says nothing")

	onWeek, offWeek := weekdayPattern(2, 0), weekdayPattern(2, 1)
	tuesday := at(17, 10)
	a := driverready.At(
		driverready.Input{Now: tuesday, FreeAt: tuesday, Pattern: onWeek, Location: denver},
	)
	b := driverready.At(
		driverready.Input{Now: tuesday, FreeAt: tuesday, Pattern: offWeek, Location: denver},
	)
	assert.True(t, (a == tuesday) != (b == tuesday),
		"the two halves of an alternating rotation are never both on")
	later := max(a, b)
	assert.Equal(t, at(23, 7), later, "the half that is off this week is back next Monday")

	rested := driverready.At(driverready.Input{
		Now: friday, FreeAt: friday, Clocks: clocks(0, 0, 40, 0), Pattern: weekly, Location: denver,
	})
	assert.Equal(t, monday, rested, "a ten-hour rest that ends on Saturday still waits for Monday")
}
