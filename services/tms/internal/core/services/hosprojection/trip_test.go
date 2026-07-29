package hosprojection

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/telematics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fullClocks() Clocks {
	return Clocks{
		DriveMs: 11 * hourMs,
		ShiftMs: 14 * hourMs,
		CycleMs: 70 * hourMs,
		BreakMs: 8 * hourMs,
	}
}

func TestPlanTrip_ShortTripNeedsNoRest(t *testing.T) {
	t.Parallel()

	plan, limiter := PlanTrip(hoursMs(4), fullClocks(), usPropertyLimits())

	require.Equal(t, LimiterNone, limiter)
	assert.Equal(t, hoursMs(4), plan.TotalMs)
	assert.Zero(t, plan.Breaks)
	assert.Zero(t, plan.Resets)
	assert.Equal(t, hoursMs(7), plan.MarginMs)
}

func TestPlanTrip_TenHourDriveInsertsOneBreak(t *testing.T) {
	t.Parallel()

	plan, limiter := PlanTrip(hoursMs(10), fullClocks(), usPropertyLimits())

	require.Equal(t, LimiterNone, limiter)
	assert.Equal(t, 1, plan.Breaks)
	assert.Zero(t, plan.Resets)
	assert.Equal(t, hoursMs(10.5), plan.TotalMs)
	assert.Equal(t, hoursMs(1), plan.MarginMs)
}

func TestPlanTrip_ResetCapMakesExtremeTripsInfeasible(t *testing.T) {
	t.Parallel()

	_, limiter := PlanTrip(hoursMs(35), fullClocks(), usPropertyLimits())

	assert.Equal(t, LimiterDrive, limiter)
}

func TestPlanTrip_CycleExhaustionIsTerminal(t *testing.T) {
	t.Parallel()

	clocks := fullClocks()
	clocks.CycleMs = hoursMs(5)

	plan, limiter := PlanTrip(hoursMs(8), clocks, usPropertyLimits())

	assert.Equal(t, LimiterCycle, limiter)
	assert.Equal(t, hoursMs(5), plan.TotalMs)
}

func TestPlanTrip_ZeroDriveTripCompletesImmediately(t *testing.T) {
	t.Parallel()

	plan, limiter := PlanTrip(0, fullClocks(), usPropertyLimits())

	require.Equal(t, LimiterNone, limiter)
	assert.Zero(t, plan.TotalMs)
	assert.Equal(t, hoursMs(11), plan.MarginMs)
}

func TestPlanTrip_ExhaustedShiftWithNoBreakRoomForcesReset(t *testing.T) {
	t.Parallel()

	clocks := fullClocks()
	clocks.ShiftMs = hoursMs(0.25)
	clocks.BreakMs = 0

	plan, limiter := PlanTrip(hoursMs(2), clocks, usPropertyLimits())

	require.Equal(t, LimiterNone, limiter)
	assert.Equal(t, 1, plan.Resets)
	assert.Equal(t, hoursMs(12), plan.TotalMs)
}

func TestBuildTimeline_CapsOpenEntriesAndClipsOverlaps(t *testing.T) {
	t.Parallel()

	endAt := baseTime - hoursSec(6)
	logs := []*telematics.WorkerHOSLog{
		{
			DutyStatus: telematics.DutyStatusSleeperBed,
			LogStartAt: baseTime - hoursSec(4),
		},
		{
			DutyStatus: telematics.DutyStatusDriving,
			LogStartAt: baseTime - hoursSec(10),
			LogEndAt:   &endAt,
		},
		{
			DutyStatus: telematics.DutyStatusOnDuty,
			LogStartAt: baseTime - hoursSec(7),
		},
	}

	timeline := BuildTimeline(logs, baseTime)

	require.Len(t, timeline, 3)
	assert.Equal(t, telematics.DutyStatusDriving, timeline[0].Status)
	assert.Equal(t, baseTime-hoursSec(7), timeline[0].EndAt)
	assert.Equal(t, telematics.DutyStatusOnDuty, timeline[1].Status)
	assert.Equal(t, baseTime-hoursSec(4), timeline[1].EndAt)
	assert.Equal(t, telematics.DutyStatusSleeperBed, timeline[2].Status)
	assert.Equal(t, baseTime, timeline[2].EndAt)
}

func TestBuildTimeline_DropsEmptyAndFutureEntries(t *testing.T) {
	t.Parallel()

	logs := []*telematics.WorkerHOSLog{
		nil,
		{DutyStatus: telematics.DutyStatusDriving, LogStartAt: 0},
		{DutyStatus: telematics.DutyStatusDriving, LogStartAt: baseTime + hoursSec(1)},
		{DutyStatus: telematics.DutyStatusOffDuty, LogStartAt: baseTime - hoursSec(2)},
	}

	timeline := BuildTimeline(logs, baseTime)

	require.Len(t, timeline, 1)
	assert.Equal(t, telematics.DutyStatusOffDuty, timeline[0].Status)
}

func TestRestRuns_MergesAdjacentRestStatusesAndTracksSleeper(t *testing.T) {
	t.Parallel()

	timeline := []DutyInterval{
		interval(telematics.DutyStatusOffDuty, 14, 12),
		interval(telematics.DutyStatusSleeperBed, 12, 4),
		interval(telematics.DutyStatusPersonalConveyance, 4, 3),
		interval(telematics.DutyStatusDriving, 3, 1),
		interval(telematics.DutyStatusOffDuty, 1, 0),
	}

	runs := restRuns(timeline, baseTime)

	require.Len(t, runs, 2)
	assert.Equal(t, baseTime-hoursSec(14), runs[0].startAt)
	assert.Equal(t, baseTime-hoursSec(3), runs[0].endAt)
	assert.Equal(t, baseTime-hoursSec(12), runs[0].sleeperStart)
	assert.Equal(t, baseTime-hoursSec(4), runs[0].sleeperEnd)
	assert.False(t, runs[0].open)
	assert.True(t, runs[1].open)
}
