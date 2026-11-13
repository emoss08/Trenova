package demandfit_test

import (
	"fmt"
	"math/rand/v2"
	"testing"
	"time"
	_ "time/tzdata"

	"github.com/emoss08/trenova/internal/modules/operations/domain/demandfit"
	"github.com/emoss08/trenova/internal/modules/operations/domain/demandforecast"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchmarket"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScore_IsTheMissOfAKnownCountAndLessForARangeThatHoldsTheActual(t *testing.T) {
	t.Parallel()

	known := demandforecast.Count{}.Shift(4)
	assert.InDelta(t, 0.0, known.Score(4), 1e-9)
	assert.InDelta(t, 3.0, known.Score(7), 1e-9)
	assert.InDelta(t, 4.0, known.Score(0), 1e-9)

	poisson := demandforecast.Count{Mean: 3.2, Variance: 3.2}
	assert.InDelta(t, 1.161105, poisson.Score(5), 1e-4)
	assert.InDelta(t, 2.211106, poisson.Score(0), 1e-4)
	assert.InDelta(t, 35.811106, poisson.Score(40), 1e-4)
	assert.Less(t, poisson.Score(3), demandforecast.Count{}.Shift(5).Score(3),
		"a range around 3 beats a flat guess of 5 when 3 arrive")

	quiet := demandforecast.Count{Mean: 0.04, Variance: 0.04}
	assert.InDelta(t, 0.001538, quiet.Score(0), 1e-5,
		"expecting almost nothing where nothing came costs almost nothing")
	assert.InDelta(t, 0.923117, quiet.Score(1), 1e-4)

	shifted := demandforecast.Count{Mean: 2, Variance: 2}.Shift(3)
	assert.InDelta(t, 3.228494, shifted.Score(1), 1e-4)
	assert.InDelta(t, 1.378776, shifted.Score(7), 1e-4)
}

func TestWiden_StretchesTheRangeAndLeavesTheMeanAlone(t *testing.T) {
	t.Parallel()

	count := demandforecast.Count{Mean: 40, Variance: 40}
	wide := count.Widen(2.5)

	assert.InDelta(t, 40.0, wide.Mean, 1e-12)
	assert.InDelta(t, 100.0, wide.Variance, 1e-12)
	assert.Less(
		t,
		wide.Quantile(demandforecast.IntervalLow),
		count.Quantile(demandforecast.IntervalLow),
	)
	assert.Greater(
		t,
		wide.Quantile(demandforecast.IntervalHigh),
		count.Quantile(demandforecast.IntervalHigh),
	)
	assert.Equal(t, count, count.Widen(1))
	assert.Equal(t, count, count.Widen(0.4), "a range is never narrowed")
}

func swingingHistory(t *testing.T, weeks int64) *demandfit.History {
	t.Helper()
	r := rand.New(rand.NewPCG(5, 23))
	current := dispatchmarket.WeekOf(mondayMidnight)
	h := &demandfit.History{Now: nowAtWeek(current)}
	for week := current - weeks; week < current; week++ {
		level := 0.5 + r.Float64()
		for slot := range int16(dispatchmarket.SlotsPerWeek) {
			h.Counts = append(h.Counts, demandfit.LoadCount{
				CustomerID: acme, Cell: dallasCell(t), Week: week, Slot: slot,
				Loads: poisson(r, 6*level),
			})
		}
	}

	return h
}

func TestEvaluate_WidensNetworkRangesWhenWholeWeeksSwing(t *testing.T) {
	t.Parallel()

	steady, steadyDays := demandfit.Evaluate(patternedHistory(t, 26))
	swinging, swingingDays := demandfit.Evaluate(swingingHistory(t, 26))

	assert.Greater(t, swinging.Dispersion, 3.0,
		"weeks that run half or half again as busy are far wider than a steady rhythm")
	assert.Greater(t, swinging.Dispersion, steady.Dispersion)
	assert.LessOrEqual(t, swinging.Dispersion, demandfit.MaxDispersion)
	assert.Greater(t, swinging.IntervalCoverage, 0.6,
		"each tested week is ranged with what the weeks before it taught")

	require.Len(t, steadyDays, demandforecast.RecordDays)
	require.Len(t, swingingDays, demandforecast.RecordDays)
	last := swingingDays[len(swingingDays)-1]
	plain := demandforecast.Count{Mean: last.Mean, Variance: last.Mean}
	assert.Greater(t, int(last.High-last.Low),
		plain.Quantile(demandforecast.IntervalHigh)-plain.Quantile(demandforecast.IntervalLow),
		"the track record's ranges are the widened ones")
}

func TestEvaluate_CountsLoadsFromAnAreaItNeverSawAsMisses(t *testing.T) {
	t.Parallel()

	h := patternedHistory(t, 26)
	before, _ := demandfit.Evaluate(h)

	lastWeek := dispatchmarket.WeekOf(h.Now) - 1
	h.Counts = append(h.Counts, demandfit.LoadCount{
		CustomerID: acme, Cell: dallasCell(t) + 1, Week: lastWeek, Slot: 7, Loads: 30,
	})
	after, _ := demandfit.Evaluate(h)

	assert.Equal(t, before.LoadsTested+30, after.LoadsTested)
	assert.Less(t, after.Skill, before.Skill,
		"thirty loads nobody forecast are thirty loads missed, for the forecast and for last week")
}

func TestEvaluate_TrackRecordDaysAreTheOrganizationsOwnDays(t *testing.T) {
	t.Parallel()

	chicago, err := time.LoadLocation("America/Chicago")
	require.NoError(t, err)
	h := swingingHistory(t, 26)
	h.Now += 2*dispatchmarket.DaySeconds + 15*3600
	h.Grid = dispatchmarket.GridIn(chicago)

	_, days := demandfit.Evaluate(h)

	require.Len(t, days, demandforecast.RecordDays)
	for i, day := range days {
		local := time.Unix(day.DayStart, 0).In(chicago)
		assert.Zero(t, local.Hour()+local.Minute(), "day %d starts at local midnight", i)
		assert.Less(t, day.DayStart+dispatchmarket.DaySeconds-3600, h.Now, "only whole days")
		if i > 0 {
			assert.Equal(t, local.AddDate(0, 0, -1).YearDay(),
				time.Unix(days[i-1].DayStart, 0).In(chicago).YearDay())
		}

		actual := int32(0)
		for _, at := range h.Grid.Starts(day.DayStart, 6) {
			for _, count := range h.Counts {
				if count.Week == h.Grid.Week(at) && count.Slot == h.Grid.Slot(at) {
					actual += count.Loads
				}
			}
		}
		assert.Equal(t, actual, day.Actual, "day %d counts the loads of its own six slots", i)
	}
	assert.Equal(t,
		time.Unix(h.Now, 0).In(chicago).AddDate(0, 0, -1).YearDay(),
		time.Unix(days[len(days)-1].DayStart, 0).In(chicago).YearDay(),
		"the record ends yesterday")
}

func TestFit_KeepsTheBusiestCustomersOfAnAreaAndEveryAreaRate(t *testing.T) {
	t.Parallel()

	current := dispatchmarket.WeekOf(mondayMidnight)
	h := &demandfit.History{Now: nowAtWeek(current)}
	customers := demandfit.MaxAreaCustomers + 12
	for i := range customers {
		for week := current - 10; week < current; week++ {
			h.Counts = append(h.Counts, demandfit.LoadCount{
				CustomerID: pulid.ID(fmt.Sprintf("cus_%03d", i)),
				Cell:       dallasCell(t),
				Week:       week,
				Slot:       7,
				Loads:      int32(i + 1),
			})
		}
	}

	fitted := demandfit.Fit(h)

	kept := map[pulid.ID]bool{}
	areaRates := 0
	for _, rate := range fitted.Rates {
		if rate.CustomerID.IsNil() {
			areaRates++
			continue
		}
		kept[rate.CustomerID] = true
	}
	assert.Equal(t, int(dispatchmarket.SlotsPerWeek), areaRates, "the area keeps every slot")
	assert.Len(t, kept, demandfit.MaxAreaCustomers)
	assert.True(t, kept[pulid.ID(fmt.Sprintf("cus_%03d", customers-1))], "the busiest stays")
	assert.False(t, kept["cus_000"], "the quietest go")
	assert.InDelta(t,
		float64(customers*(customers+1)/2),
		areaRate(t, fitted, 7).Alpha/areaRate(t, fitted, 7).Beta,
		float64(customers*(customers+1)/2)*0.1,
		"the area's own rate still counts every customer's loads")
}
