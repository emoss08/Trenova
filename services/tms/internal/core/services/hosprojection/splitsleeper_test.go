package hosprojection

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/telematics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProject_EightTwoSplitBeatsLimpingOnCurrentClocks(t *testing.T) {
	t.Parallel()

	result := Project(Input{
		Now:         baseTime,
		Departure:   baseTime,
		TripDriveMs: hoursMs(6),
		Clocks: Clocks{
			DriveMs: hoursMs(7),
			ShiftMs: hoursMs(1),
			CycleMs: hoursMs(40),
			BreakMs: hoursMs(4),
		},
		ClocksAt:   baseTime,
		DutyStatus: telematics.DutyStatusOffDuty,
		Limits:     usPropertyLimits(),
		Timeline: []DutyInterval{
			interval(telematics.DutyStatusSleeperBed, 14, 6),
			interval(telematics.DutyStatusDriving, 6, 2),
			interval(telematics.DutyStatusOffDuty, 2, 0),
		},
	})

	require.True(t, result.Feasible)
	assert.Equal(t, StrategySplitSleeper, result.Strategy)
	assert.Equal(t, hoursMs(7), result.DriveAvailableMs)
	assert.Equal(t, hoursMs(10), result.ShiftAvailableMs)
	assert.Equal(t, hoursMs(6), result.Trip.TotalMs)
	assert.Zero(t, result.Trip.Resets)
	assert.Contains(t, result.Detail, "split sleeper")
}

func TestProject_SevenThreeSplitWithOffDutyPeriodFirst(t *testing.T) {
	t.Parallel()

	result := Project(Input{
		Now:         baseTime,
		Departure:   baseTime,
		TripDriveMs: hoursMs(5),
		Clocks: Clocks{
			DriveMs: hoursMs(6),
			ShiftMs: hoursMs(0.5),
			CycleMs: hoursMs(40),
			BreakMs: hoursMs(3),
		},
		ClocksAt:   baseTime,
		DutyStatus: telematics.DutyStatusSleeperBed,
		Limits:     usPropertyLimits(),
		Timeline: []DutyInterval{
			interval(telematics.DutyStatusOffDuty, 17.5, 14.5),
			interval(telematics.DutyStatusDriving, 14.5, 9.5),
			interval(telematics.DutyStatusOnDuty, 9.5, 7.5),
			interval(telematics.DutyStatusSleeperBed, 7.5, 0),
		},
	})

	require.True(t, result.Feasible)
	assert.Equal(t, StrategySplitSleeper, result.Strategy)
	assert.Equal(t, hoursMs(6), result.DriveAvailableMs)
	assert.Equal(t, hoursMs(7), result.ShiftAvailableMs)
}

func TestProject_SplitNeverExtendsTheDriveClock(t *testing.T) {
	t.Parallel()

	result := Project(Input{
		Now:         baseTime,
		Departure:   baseTime,
		TripDriveMs: hoursMs(6),
		Clocks: Clocks{
			DriveMs: hoursMs(1),
			ShiftMs: hoursMs(1),
			CycleMs: hoursMs(40),
			BreakMs: hoursMs(1),
		},
		ClocksAt:   baseTime,
		DutyStatus: telematics.DutyStatusOffDuty,
		Limits:     usPropertyLimits(),
		Timeline: []DutyInterval{
			interval(telematics.DutyStatusSleeperBed, 20, 12),
			interval(telematics.DutyStatusDriving, 12, 2),
			interval(telematics.DutyStatusOffDuty, 2, 0),
		},
	})

	require.True(t, result.Feasible)
	assert.NotEqual(t, StrategySplitSleeper, result.Strategy)
	assert.Positive(t, result.Trip.Resets)
}

func TestProject_PendingSecondRestPairsWhenNothingElseFits(t *testing.T) {
	t.Parallel()

	departure := baseTime + hoursSec(3)
	result := Project(Input{
		Now:         baseTime,
		Departure:   departure,
		TripDriveMs: hoursMs(4),
		Clocks: Clocks{
			DriveMs: hoursMs(5),
			ShiftMs: hoursMs(1),
			CycleMs: hoursMs(40),
			BreakMs: hoursMs(2),
		},
		ClocksAt:   baseTime,
		DutyStatus: telematics.DutyStatusDriving,
		Limits:     usPropertyLimits(),
		Timeline: []DutyInterval{
			interval(telematics.DutyStatusSleeperBed, 13, 5),
			interval(telematics.DutyStatusDriving, 5, 0),
		},
	})

	require.True(t, result.Feasible)
	assert.Equal(t, StrategySplitSleeper, result.Strategy)
	assert.Equal(t, departure-hoursSec(2), result.RestStartDeadline)
	assert.Contains(t, result.Detail, "rest must start within")
}

func TestProject_OpenSleeperCompletingThePairAtDeparture(t *testing.T) {
	t.Parallel()

	departure := baseTime + hoursSec(3)
	result := Project(Input{
		Now:         baseTime,
		Departure:   departure,
		TripDriveMs: hoursMs(4),
		Clocks: Clocks{
			DriveMs: hoursMs(5),
			ShiftMs: hoursMs(1),
			CycleMs: hoursMs(40),
			BreakMs: hoursMs(2),
		},
		ClocksAt:   baseTime,
		DutyStatus: telematics.DutyStatusSleeperBed,
		Limits:     usPropertyLimits(),
		Timeline: []DutyInterval{
			interval(telematics.DutyStatusOffDuty, 13, 10.5),
			interval(telematics.DutyStatusDriving, 10.5, 4.5),
			interval(telematics.DutyStatusSleeperBed, 4.5, 0),
		},
	})

	require.True(t, result.Feasible)
	assert.Equal(t, StrategySplitSleeper, result.Strategy)
	assert.Equal(t, hoursMs(5), result.DriveAvailableMs)
	assert.Equal(t, hoursMs(8), result.ShiftAvailableMs)
}

func TestProject_SplitSkippedForNonStandardDriveLimits(t *testing.T) {
	t.Parallel()

	limits := usPropertyLimits()
	limits.DriveMs = 10 * hourMs

	result := Project(Input{
		Now:         baseTime,
		Departure:   baseTime,
		TripDriveMs: hoursMs(6),
		Clocks: Clocks{
			DriveMs: hoursMs(7),
			ShiftMs: hoursMs(1),
			CycleMs: hoursMs(40),
			BreakMs: hoursMs(4),
		},
		ClocksAt:   baseTime,
		DutyStatus: telematics.DutyStatusOffDuty,
		Limits:     limits,
		Timeline: []DutyInterval{
			interval(telematics.DutyStatusSleeperBed, 14, 6),
			interval(telematics.DutyStatusDriving, 6, 2),
			interval(telematics.DutyStatusOffDuty, 2, 0),
		},
	})

	if result.Feasible {
		assert.NotEqual(t, StrategySplitSleeper, result.Strategy)
	}
}

func TestProject_ShortRestsNeverQualifyAsAPair(t *testing.T) {
	t.Parallel()

	result := Project(Input{
		Now:         baseTime,
		Departure:   baseTime,
		TripDriveMs: hoursMs(6),
		Clocks: Clocks{
			DriveMs: hoursMs(7),
			ShiftMs: hoursMs(1),
			CycleMs: hoursMs(40),
			BreakMs: hoursMs(4),
		},
		ClocksAt:   baseTime,
		DutyStatus: telematics.DutyStatusOffDuty,
		Limits:     usPropertyLimits(),
		Timeline: []DutyInterval{
			interval(telematics.DutyStatusSleeperBed, 14, 7.5),
			interval(telematics.DutyStatusDriving, 7.5, 2),
			interval(telematics.DutyStatusOffDuty, 2, 0),
		},
	})

	if result.Feasible {
		assert.NotEqual(t, StrategySplitSleeper, result.Strategy)
	}
}
