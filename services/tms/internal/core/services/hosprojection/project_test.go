package hosprojection

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/telematics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const baseTime = int64(1_800_000_000)

func hoursMs(h float64) int64  { return int64(h * float64(hourMs)) }
func hoursSec(h float64) int64 { return int64(h * 3600) }

func usPropertyLimits() Limits {
	return Limits{
		DriveMs: 11 * hourMs,
		ShiftMs: 14 * hourMs,
		CycleMs: 70 * hourMs,
		BreakMs: 8 * hourMs,
	}
}

func interval(status telematics.DutyStatus, startHoursAgo, endHoursAgo float64) DutyInterval {
	return DutyInterval{
		Status:  status,
		StartAt: baseTime - hoursSec(startHoursAgo),
		EndAt:   baseTime - hoursSec(endHoursAgo),
	}
}

func TestProject_PlentyOfHoursRunsOnCurrentClocks(t *testing.T) {
	t.Parallel()

	result := Project(Input{
		Now:         baseTime,
		Departure:   baseTime,
		TripDriveMs: hoursMs(4),
		Clocks: Clocks{
			DriveMs: hoursMs(9),
			ShiftMs: hoursMs(12),
			CycleMs: hoursMs(40),
			BreakMs: hoursMs(8),
		},
		ClocksAt:   baseTime,
		DutyStatus: telematics.DutyStatusOnDuty,
		Limits:     usPropertyLimits(),
	})

	require.True(t, result.Feasible)
	assert.Equal(t, StrategyCurrentClocks, result.Strategy)
	assert.Equal(t, LimiterNone, result.Limiter)
	assert.Zero(t, result.Trip.Breaks)
	assert.Zero(t, result.Trip.Resets)
	assert.Equal(t, hoursMs(4), result.Trip.TotalMs)
	assert.Equal(t, hoursMs(5), result.Trip.MarginMs)
	assert.Equal(t, baseTime+hoursSec(4), result.Trip.Arrival)
	assert.Zero(t, result.RestStartDeadline)
}

func TestProject_TenHourResetMakesDepletedDriverLegal(t *testing.T) {
	t.Parallel()

	departure := baseTime + hoursSec(12)
	result := Project(Input{
		Now:         baseTime,
		Departure:   departure,
		TripDriveMs: hoursMs(6),
		Clocks: Clocks{
			DriveMs: hoursMs(2),
			ShiftMs: hoursMs(3),
			CycleMs: hoursMs(30),
			BreakMs: hoursMs(8),
		},
		ClocksAt:   baseTime,
		DutyStatus: telematics.DutyStatusOffDuty,
		Limits:     usPropertyLimits(),
	})

	require.True(t, result.Feasible)
	assert.Equal(t, StrategyTenHourReset, result.Strategy)
	assert.Equal(t, usPropertyLimits().DriveMs, result.DriveAvailableMs)
	assert.Equal(t, usPropertyLimits().ShiftMs, result.ShiftAvailableMs)
	assert.Equal(t, departure-hoursSec(10), result.RestStartDeadline)
	assert.Contains(t, result.Detail, "10-hour reset")
	assert.Contains(t, result.Detail, "clocks only")
}

func TestProject_Restart34RecoversAnExhaustedCycle(t *testing.T) {
	t.Parallel()

	departure := baseTime + hoursSec(40)
	result := Project(Input{
		Now:         baseTime,
		Departure:   departure,
		TripDriveMs: hoursMs(8),
		Clocks: Clocks{
			DriveMs: hoursMs(8),
			ShiftMs: hoursMs(10),
			CycleMs: hoursMs(1),
			BreakMs: hoursMs(8),
		},
		ClocksAt:   baseTime,
		DutyStatus: telematics.DutyStatusOffDuty,
		Limits:     usPropertyLimits(),
	})

	require.True(t, result.Feasible)
	assert.Equal(t, StrategyRestart34, result.Strategy)
	assert.Equal(t, usPropertyLimits().CycleMs, result.CycleAvailableMs)
	assert.Equal(t, departure-hoursSec(34), result.RestStartDeadline)
	assert.Contains(t, result.Detail, "34-hour")
}

func TestProject_ExhaustedCycleWithNoRoomForRestartIsInfeasible(t *testing.T) {
	t.Parallel()

	result := Project(Input{
		Now:         baseTime,
		Departure:   baseTime + hoursSec(20),
		TripDriveMs: hoursMs(8),
		Clocks: Clocks{
			DriveMs: hoursMs(8),
			ShiftMs: hoursMs(10),
			CycleMs: hoursMs(1),
			BreakMs: hoursMs(8),
		},
		ClocksAt:   baseTime,
		DutyStatus: telematics.DutyStatusOffDuty,
		Limits:     usPropertyLimits(),
	})

	require.False(t, result.Feasible)
	assert.Equal(t, LimiterCycle, result.Limiter)
	assert.Contains(t, result.Detail, "34-hour restart")
}

func TestProject_TenHourResetNeverRestoresTheCycle(t *testing.T) {
	t.Parallel()

	result := Project(Input{
		Now:         baseTime,
		Departure:   baseTime + hoursSec(12),
		TripDriveMs: hoursMs(6),
		Clocks: Clocks{
			DriveMs: hoursMs(2),
			ShiftMs: hoursMs(3),
			CycleMs: hoursMs(2),
			BreakMs: hoursMs(8),
		},
		ClocksAt:   baseTime,
		DutyStatus: telematics.DutyStatusOffDuty,
		Limits:     usPropertyLimits(),
	})

	require.False(t, result.Feasible)
	assert.Equal(t, LimiterCycle, result.Limiter)
}

func TestProject_MidSleeperDriverBanksRestAlreadyTaken(t *testing.T) {
	t.Parallel()

	departure := baseTime + hoursSec(9)
	result := Project(Input{
		Now:         baseTime,
		Departure:   departure,
		TripDriveMs: hoursMs(4),
		Clocks: Clocks{
			DriveMs: 0,
			ShiftMs: hoursMs(1),
			CycleMs: hoursMs(30),
			BreakMs: hoursMs(8),
		},
		ClocksAt:   baseTime,
		DutyStatus: telematics.DutyStatusSleeperBed,
		Limits:     usPropertyLimits(),
		Timeline: []DutyInterval{
			interval(telematics.DutyStatusDriving, 12, 1),
			interval(telematics.DutyStatusSleeperBed, 1, 0),
		},
	})

	require.True(t, result.Feasible)
	assert.Equal(t, StrategyTenHourReset, result.Strategy)
	assert.Equal(t, baseTime-hoursSec(1), result.RestStartDeadline)
}

func TestProject_MultiDayTripInsertsBreakAndReset(t *testing.T) {
	t.Parallel()

	result := Project(Input{
		Now:         baseTime,
		Departure:   baseTime,
		TripDriveMs: hoursMs(16),
		Clocks: Clocks{
			DriveMs: hoursMs(11),
			ShiftMs: hoursMs(14),
			CycleMs: hoursMs(60),
			BreakMs: hoursMs(8),
		},
		ClocksAt:   baseTime,
		DutyStatus: telematics.DutyStatusOnDuty,
		Limits:     usPropertyLimits(),
	})

	require.True(t, result.Feasible)
	assert.Equal(t, StrategyCurrentClocks, result.Strategy)
	assert.Equal(t, 1, result.Trip.Breaks)
	assert.Equal(t, 1, result.Trip.Resets)
	assert.Equal(t, hoursMs(26.5), result.Trip.TotalMs)
	assert.Equal(t, hoursMs(6), result.Trip.MarginMs)
	assert.Contains(t, result.Detail, "en route")
}

func TestProject_BreakDueNowIsTakenBeforeDriving(t *testing.T) {
	t.Parallel()

	result := Project(Input{
		Now:         baseTime,
		Departure:   baseTime,
		TripDriveMs: hoursMs(2),
		Clocks: Clocks{
			DriveMs: hoursMs(4),
			ShiftMs: hoursMs(5),
			CycleMs: hoursMs(30),
			BreakMs: 0,
		},
		ClocksAt:   baseTime,
		DutyStatus: telematics.DutyStatusDriving,
		Limits:     usPropertyLimits(),
	})

	require.True(t, result.Feasible)
	assert.Equal(t, 1, result.Trip.Breaks)
	assert.Equal(t, hoursMs(2.5), result.Trip.TotalMs)
}

func TestProject_ClockOnlyFallbackStillProjectsResets(t *testing.T) {
	t.Parallel()

	result := Project(Input{
		Now:         baseTime,
		Departure:   baseTime + hoursSec(14),
		TripDriveMs: hoursMs(8),
		Clocks: Clocks{
			DriveMs: hoursMs(1),
			ShiftMs: hoursMs(1),
			CycleMs: hoursMs(40),
			BreakMs: hoursMs(1),
		},
		ClocksAt:   baseTime,
		DutyStatus: telematics.DutyStatusOnDuty,
		Limits:     usPropertyLimits(),
	})

	require.True(t, result.Feasible)
	assert.Equal(t, StrategyTenHourReset, result.Strategy)
}

func TestProject_CycleRecoversAsOldDutyDaysAgeOut(t *testing.T) {
	t.Parallel()

	departure := baseTime + hoursSec(36)
	result := Project(Input{
		Now:         baseTime,
		Departure:   departure,
		TripDriveMs: hoursMs(8),
		Clocks: Clocks{
			DriveMs: hoursMs(11),
			ShiftMs: hoursMs(14),
			CycleMs: hoursMs(60),
			BreakMs: hoursMs(8),
		},
		ClocksAt:   baseTime,
		DutyStatus: telematics.DutyStatusOffDuty,
		Limits:     usPropertyLimits(),
		Timeline: []DutyInterval{
			interval(telematics.DutyStatusOnDuty, 190, 180),
			interval(telematics.DutyStatusOffDuty, 180, 0),
		},
	})

	require.True(t, result.Feasible)
	assert.Equal(t, hoursMs(70), result.CycleAvailableMs)
}

func TestProject_CanadianJurisdictionSkipsSplitAndRestart(t *testing.T) {
	t.Parallel()

	limits := Limits{
		DriveMs: 13 * hourMs,
		ShiftMs: 16 * hourMs,
		CycleMs: 70 * hourMs,
		BreakMs: 8 * hourMs,
	}
	result := Project(Input{
		Now:          baseTime,
		Departure:    baseTime + hoursSec(40),
		TripDriveMs:  hoursMs(8),
		Clocks:       Clocks{DriveMs: hoursMs(8), ShiftMs: hoursMs(10), CycleMs: hoursMs(1), BreakMs: hoursMs(8)},
		ClocksAt:     baseTime,
		DutyStatus:   telematics.DutyStatusOffDuty,
		Limits:       limits,
		Jurisdiction: "CS",
	})

	require.False(t, result.Feasible)
	assert.Equal(t, LimiterCycle, result.Limiter)
}

func TestProject_PastDepartureIsClampedToNow(t *testing.T) {
	t.Parallel()

	result := Project(Input{
		Now:         baseTime,
		Departure:   baseTime - hoursSec(2),
		TripDriveMs: hoursMs(2),
		Clocks: Clocks{
			DriveMs: hoursMs(6),
			ShiftMs: hoursMs(8),
			CycleMs: hoursMs(30),
			BreakMs: hoursMs(8),
		},
		ClocksAt:   baseTime,
		DutyStatus: telematics.DutyStatusOnDuty,
		Limits:     usPropertyLimits(),
	})

	require.True(t, result.Feasible)
	assert.Equal(t, baseTime+hoursSec(2), result.Trip.Arrival)
}
