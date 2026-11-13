package demandfit_test

import (
	"math/rand/v2"
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/demandfit"
	"github.com/emoss08/trenova/internal/modules/operations/domain/demandforecast"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchmarket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const weekdaySlot = int16(2)

func weekdayHistory(
	t *testing.T,
	weeks int64,
	mean func(week int64, day int) float64,
) *demandfit.History {
	t.Helper()
	r := rand.New(rand.NewPCG(17, 29))
	current := dispatchmarket.WeekOf(mondayMidnight) + weeks
	h := &demandfit.History{Now: nowAtWeek(current)}
	for week := current - weeks; week < current; week++ {
		for day := range 5 {
			h.Counts = append(h.Counts, demandfit.LoadCount{
				CustomerID: acme,
				Cell:       dallasCell(t),
				Week:       week,
				Slot:       int16(day)*int16(dispatchmarket.SlotsPerDay) + weekdaySlot,
				Loads:      poisson(r, mean(week, day)),
			})
		}
	}

	return h
}

func weeklyMean(fitted *demandforecast.Fitted) float64 {
	total := 0.0
	for _, rate := range fitted.Rates {
		if rate.CustomerID.IsNil() {
			total += rate.Alpha / rate.Beta
		}
	}

	return total
}

func TestTrain_LearnsWhatAHolidayIsWorthAndKeepsItOutOfTheWorkingRhythm(t *testing.T) {
	t.Parallel()

	current := dispatchmarket.WeekOf(mondayMidnight) + 30
	holidays := demandforecast.Holidays{}
	closed := map[int64]bool{}
	for _, back := range []int64{3, 9, 14, 20, 25} {
		week := current - back
		closed[week] = true
		holidays[dispatchmarket.WallDay(week, 0)] = struct{}{}
	}
	h := weekdayHistory(t, 30, func(week int64, day int) float64 {
		if day == 0 && closed[week] {
			return 6
		}
		return 30
	})

	plain := demandfit.Train(h)
	h.Holidays = holidays
	aware := demandfit.Train(h)

	assert.InDelta(t, 0.2, aware.Quality.HolidayFactor, 0.08,
		"a holiday brings about a fifth of a Monday's loads")
	monday := areaRate(t, aware.Fitted, weekdaySlot)
	assert.InDelta(t, 30, monday.Alpha/monday.Beta, 2.5,
		"a working Monday is forecast from working Mondays")
	assert.Less(
		t,
		areaRate(t, plain.Fitted, weekdaySlot).Alpha/areaRate(t, plain.Fitted, weekdaySlot).Beta,
		monday.Alpha/monday.Beta-2,
		"without the calendar the closed Mondays drag every Monday down",
	)
	assert.Greater(t, aware.Quality.Skill, plain.Quality.Skill,
		"and the weeks with a holiday in them are forecast better")
}

func TestTrain_KeepsTheHolidaysAheadForTheForecast(t *testing.T) {
	t.Parallel()

	h := weekdayHistory(t, 12, func(int64, int) float64 { return 10 })
	today := h.Grid.Day(h.Now)
	h.Holidays = demandforecast.Holidays{
		today - 40: {}, today - 1: {}, today + 3: {}, today + demandfit.HolidayHorizonDays: {},
		today + demandfit.HolidayHorizonDays + 1: {},
	}

	quality, _ := demandfit.Evaluate(h)

	assert.Equal(t, []int64{today - 1, today + 3, today + demandfit.HolidayHorizonDays},
		quality.Holidays)
	assert.Equal(t, "UTC", quality.Timezone)
}

func TestTrain_FollowsAStepInVolumeWithinTwoWeeksAndLeavesASteadyBookAlone(t *testing.T) {
	t.Parallel()

	current := dispatchmarket.WeekOf(mondayMidnight) + 30
	steady := demandfit.Train(weekdayHistory(t, 30, func(int64, int) float64 { return 40 }))
	stepped := demandfit.Train(weekdayHistory(t, 30, func(week int64, _ int) float64 {
		if week >= current-2 {
			return 60
		}
		return 40
	}))

	assert.InDelta(t, 1.0, steady.Quality.Level, 0.06, "noise alone barely moves the level")
	assert.InDelta(t, 200, weeklyMean(steady.Fitted), 12)

	require.Greater(t, stepped.Quality.Level, 1.2,
		"two weeks half again as busy lift the level most of the way")
	assert.Greater(t, weeklyMean(stepped.Fitted), 270.0,
		"where the eight-week memory alone would still say about 230")
	assert.Less(t, weeklyMean(stepped.Fitted), 305.0, "and never past what the two weeks showed")
	assert.LessOrEqual(t, stepped.Quality.Level, demandfit.MaxLevel)
}
