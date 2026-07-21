package recurringshipment

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mustUnix(t *testing.T, value, timezone string) int64 {
	t.Helper()

	loc, err := time.LoadLocation(timezone)
	require.NoError(t, err)

	parsed, err := time.ParseInLocation("2006-01-02 15:04", value, loc)
	require.NoError(t, err)

	return parsed.Unix()
}

func weekdaySeries(timezone string) *RecurringShipment {
	return &RecurringShipment{
		CronExpression:  "0 8 * * 1-5",
		Timezone:        timezone,
		ExceptionPolicy: ExceptionPolicySkip,
	}
}

func TestNextOccurrence_WeekdayCadence(t *testing.T) {
	t.Parallel()

	series := weekdaySeries("America/New_York")
	after := mustUnix(t, "2026-07-17 09:00", "America/New_York")

	occurrence, err := series.NextOccurrence(after)
	require.NoError(t, err)
	require.NotNil(t, occurrence)

	assert.Equal(t, mustUnix(t, "2026-07-20 08:00", "America/New_York"), occurrence.At)
	assert.False(t, occurrence.Shifted)
}

func TestNextOccurrence_HonorsStartDate(t *testing.T) {
	t.Parallel()

	series := weekdaySeries("America/New_York")
	series.StartDate = mustUnix(t, "2026-08-03 00:00", "America/New_York")

	occurrence, err := series.NextOccurrence(mustUnix(t, "2026-07-17 09:00", "America/New_York"))
	require.NoError(t, err)
	require.NotNil(t, occurrence)

	assert.Equal(t, mustUnix(t, "2026-08-03 08:00", "America/New_York"), occurrence.At)
}

func TestNextOccurrence_EndDateExhaustsSeries(t *testing.T) {
	t.Parallel()

	series := weekdaySeries("America/New_York")
	endDate := mustUnix(t, "2026-07-17 12:00", "America/New_York")
	series.EndDate = &endDate

	occurrence, err := series.NextOccurrence(mustUnix(t, "2026-07-17 09:00", "America/New_York"))
	require.NoError(t, err)
	assert.Nil(t, occurrence)
}

func TestNextOccurrence_BlackoutSkipPolicy(t *testing.T) {
	t.Parallel()

	series := weekdaySeries("America/New_York")
	series.BlackoutDates = []string{"2026-07-20"}

	occurrence, err := series.NextOccurrence(mustUnix(t, "2026-07-17 09:00", "America/New_York"))
	require.NoError(t, err)
	require.NotNil(t, occurrence)

	assert.Equal(t, mustUnix(t, "2026-07-21 08:00", "America/New_York"), occurrence.At)
	assert.False(t, occurrence.Shifted)
}

func TestNextOccurrence_BlackoutNextBusinessDayShifts(t *testing.T) {
	t.Parallel()

	series := weekdaySeries("America/New_York")
	series.ExceptionPolicy = ExceptionPolicyNextBusinessDay
	series.BlackoutDates = []string{"2026-07-20"}

	occurrence, err := series.NextOccurrence(mustUnix(t, "2026-07-17 09:00", "America/New_York"))
	require.NoError(t, err)
	require.NotNil(t, occurrence)

	assert.Equal(t, mustUnix(t, "2026-07-21 08:00", "America/New_York"), occurrence.At)
	assert.Equal(t, mustUnix(t, "2026-07-20 08:00", "America/New_York"), occurrence.OriginalAt)
	assert.True(t, occurrence.Shifted)
}

func TestNextOccurrence_MonthlyWeekendPreviousBusinessDay(t *testing.T) {
	t.Parallel()

	series := &RecurringShipment{
		CronExpression:  "30 6 1 * *",
		Timezone:        "America/Chicago",
		SkipWeekends:    true,
		ExceptionPolicy: ExceptionPolicyPreviousBusinessDay,
	}

	occurrence, err := series.NextOccurrence(mustUnix(t, "2026-07-25 00:00", "America/Chicago"))
	require.NoError(t, err)
	require.NotNil(t, occurrence)

	assert.Equal(t, mustUnix(t, "2026-07-31 06:30", "America/Chicago"), occurrence.At)
	assert.Equal(t, mustUnix(t, "2026-08-01 06:30", "America/Chicago"), occurrence.OriginalAt)
	assert.True(t, occurrence.Shifted)
}

func TestNextOccurrence_WeekendSkipPolicyAdvances(t *testing.T) {
	t.Parallel()

	series := &RecurringShipment{
		CronExpression:  "0 8 * * *",
		Timezone:        "America/New_York",
		SkipWeekends:    true,
		ExceptionPolicy: ExceptionPolicySkip,
	}

	occurrence, err := series.NextOccurrence(mustUnix(t, "2026-07-17 09:00", "America/New_York"))
	require.NoError(t, err)
	require.NotNil(t, occurrence)

	assert.Equal(t, mustUnix(t, "2026-07-20 08:00", "America/New_York"), occurrence.At)
}

func TestNextOccurrence_InvalidCron(t *testing.T) {
	t.Parallel()

	series := weekdaySeries("America/New_York")
	series.CronExpression = "not-a-cron"

	occurrence, err := series.NextOccurrence(time.Now().Unix())
	require.Error(t, err)
	assert.Nil(t, occurrence)
}

func TestGenerationDueAt(t *testing.T) {
	t.Parallel()

	series := weekdaySeries("America/New_York")
	series.LeadTimeDays = 2

	occurrenceAt := mustUnix(t, "2026-07-22 08:00", "America/New_York")
	assert.Equal(
		t,
		mustUnix(t, "2026-07-20 08:00", "America/New_York"),
		series.GenerationDueAt(occurrenceAt),
	)
}

func TestReachedOccurrenceLimit(t *testing.T) {
	t.Parallel()

	series := weekdaySeries("America/New_York")
	assert.False(t, series.ReachedOccurrenceLimit())

	limit := int32(3)
	series.MaxOccurrences = &limit
	series.GenerationCount = 2
	assert.False(t, series.ReachedOccurrenceLimit())

	series.GenerationCount = 3
	assert.True(t, series.ReachedOccurrenceLimit())
}
