package agentroster

import (
	"testing"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunsSinceCoversFourteenWholeDaysEndingToday(t *testing.T) {
	t.Parallel()

	now := int64(1_791_000_000)
	since := RunsSince(now)

	assert.Equal(t, int64(0), since%secondsInDay)
	assert.Equal(t, DayStart(now), since+(RunDays-1)*secondsInDay)
	assert.Less(t, since, now)
}

func TestDecisionsSinceIsThirtyDaysBack(t *testing.T) {
	t.Parallel()

	assert.Equal(t, int64(1_000_000-30*86400), DecisionsSince(1_000_000))
}

func TestAddRunsIgnoresDaysOutsideTheWindow(t *testing.T) {
	t.Parallel()

	stat := NewStat(pulid.MustNew("agdef_"))
	stat.AddRuns(0, 2)
	stat.AddRuns(RunDays-1, 3)
	stat.AddRuns(RunDays, 9)
	stat.AddRuns(-1, 9)
	stat.AddRuns(4, 0)

	require.Len(t, stat.RunsByDay, RunDays)
	assert.Equal(t, 2, stat.RunsByDay[0])
	assert.Equal(t, 3, stat.RunsByDay[RunDays-1])
	assert.Equal(t, 5, stat.Runs())
}

func TestApprovalRateCountsChangedAsApprovedAndIsAbsentUntilDecided(t *testing.T) {
	t.Parallel()

	stat := NewStat(pulid.MustNew("agdef_"))
	assert.Nil(t, stat.ApprovalRate())

	stat.Failed = 4
	assert.Nil(t, stat.ApprovalRate())

	stat.Approved, stat.Modified, stat.Rejected = 18, 6, 1
	rate := stat.ApprovalRate()
	require.NotNil(t, rate)
	assert.InDelta(t, 0.96, *rate, 1e-9)
}
