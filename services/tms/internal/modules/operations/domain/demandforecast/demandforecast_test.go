package demandforecast_test

import (
	"math"
	"math/rand/v2"
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/demandforecast"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchmarket"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	dallas = dispatchmarket.Point{Latitude: 32.7767, Longitude: -96.7970}
	acme   = pulid.ID("cus_acme")
)

func dallasCell(t *testing.T) int64 {
	t.Helper()
	cell, ok := dispatchmarket.CellOf(dallas)
	require.True(t, ok)

	return cell
}

const mondayMidnight = int64(1_793_577_600)

func nowAtWeek(week int64) int64 {
	return mondayMidnight + (week-dispatchmarket.WeekOf(mondayMidnight))*dispatchmarket.WeekSeconds
}

func poisson(r *rand.Rand, mean float64) int32 {
	limit, k, p := math.Exp(-mean), int32(0), 1.0
	for {
		p *= r.Float64()
		if p <= limit {
			return k
		}
		k++
	}
}

func steadyHistory(t *testing.T, mean float64, weeks int64, slot int16) *demandforecast.History {
	t.Helper()
	r := rand.New(rand.NewPCG(7, 11))
	current := dispatchmarket.WeekOf(mondayMidnight) + weeks
	h := &demandforecast.History{Now: nowAtWeek(current)}
	for week := current - weeks; week < current; week++ {
		h.Counts = append(h.Counts, demandforecast.LoadCount{
			CustomerID: acme, Cell: dallasCell(t), Week: week, Slot: slot, Loads: poisson(r, mean),
		})
	}

	return h
}

func areaRate(t *testing.T, fitted *demandforecast.Fitted, slot int16) *demandforecast.Rate {
	t.Helper()
	for _, rate := range fitted.Rates {
		if rate.CustomerID.IsNil() && rate.Slot == slot {
			return rate
		}
	}
	t.Fatalf("no area rate for slot %d", slot)

	return nil
}

func TestFit_RecoversAKnownWeeklyRate(t *testing.T) {
	t.Parallel()

	fitted := demandforecast.Fit(steadyHistory(t, 4, 26, 7))

	rate := areaRate(t, fitted, 7)
	assert.InDelta(t, 4.0, rate.Alpha/rate.Beta, 0.6)
}

func TestFit_ShrinksASparseSlotTowardTheAreaRhythm(t *testing.T) {
	t.Parallel()

	h := steadyHistory(t, 6, 26, 7)
	h.Counts = append(h.Counts, demandforecast.LoadCount{
		CustomerID: acme, Cell: dallasCell(t), Week: dispatchmarket.WeekOf(h.Now) - 3, Slot: 8, Loads: 1,
	})
	fitted := demandforecast.Fit(h)

	sparse := areaRate(t, fitted, 8)
	busy := areaRate(t, fitted, 7)
	assert.Greater(t, sparse.Alpha/sparse.Beta, 1.0/26, "borrows from the area's rhythm")
	assert.Less(t, sparse.Alpha/sparse.Beta, busy.Alpha/busy.Beta)
}

func TestFit_WeighsRecentWeeksMore(t *testing.T) {
	t.Parallel()

	current := dispatchmarket.WeekOf(mondayMidnight) + 26
	h := &demandforecast.History{Now: nowAtWeek(current)}
	for week := current - 26; week < current; week++ {
		loads := int32(2)
		if week >= current-8 {
			loads = 6
		}
		h.Counts = append(h.Counts, demandforecast.LoadCount{
			CustomerID: acme, Cell: dallasCell(t), Week: week, Slot: 7, Loads: loads,
		})
	}

	rate := areaRate(t, demandforecast.Fit(h), 7)
	assert.Greater(t, rate.Alpha/rate.Beta, 4.0, "the last 8 weeks at 6 outweigh 18 earlier weeks at 2")
}

func TestFit_StartsTheWindowAtTheFirstWeekOfHistory(t *testing.T) {
	t.Parallel()

	rate := areaRate(t, demandforecast.Fit(steadyHistory(t, 4, 6, 7)), 7)

	assert.InDelta(t, 4.0, rate.Alpha/rate.Beta, 1.0,
		"six weeks of history are not diluted by twenty empty weeks before it")
}

func TestBookingCurve_ShrinksASmallCustomerToTheTenant(t *testing.T) {
	t.Parallel()

	big, small := pulid.ID("cus_big"), pulid.ID("cus_small")
	fitted := demandforecast.Fit(&demandforecast.History{
		Now: mondayMidnight,
		Leads: []demandforecast.LeadCount{
			{CustomerID: big, Bucket: 7, Loads: 200},
			{CustomerID: small, Bucket: 0, Loads: 2},
		},
	})

	shares := map[pulid.ID][]float64{}
	for _, curve := range fitted.Curves {
		shares[curve.CustomerID] = curve.Shares
	}
	require.Len(t, shares, 3)
	assert.InDelta(t, 1.0, shares[pulid.Nil][0], 1e-9, "every load is booked at least 0 h ahead")
	assert.Greater(t, shares[small][7], 0.8,
		"two same-day loads do not convince the model the customer books late")
	assert.InDelta(t, 1.0, shares[big][7], 0.01)
}

func TestBookedShare_InterpolatesBetweenBuckets(t *testing.T) {
	t.Parallel()

	shares := []float64{1, 1, 1, 1, 1, 0.8, 0.6, 0.4, 0.2, 0.1, 0.05, 0.02, 0}

	assert.InDelta(t, 0.7, demandforecast.BookedShare(shares, 30), 1e-9)
	assert.InDelta(t, 1.0, demandforecast.BookedShare(shares, 0), 1e-9)
	assert.InDelta(t, 0.0, demandforecast.BookedShare(shares, 500), 1e-9)
}

func TestLeadBucket_PutsALoadInTheLargestBucketItReaches(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 0, demandforecast.LeadBucket(-3600))
	assert.Equal(t, 5, demandforecast.LeadBucket(30*3600))
	assert.Equal(t, 12, demandforecast.LeadBucket(500*3600))
}

func TestNowcast_MoreBookedThanUsualRaisesTheRemainder(t *testing.T) {
	t.Parallel()

	usual := demandforecast.Nowcast(38.8, 10.4, 2, 0.4)
	busy := demandforecast.Nowcast(38.8, 10.4, 5, 0.4)

	assert.InDelta(t, 1.48, usual.Mean, 0.01)
	assert.InDelta(t, 1.59, busy.Mean, 0.01)
	assert.Greater(t, busy.Variance, busy.Mean, "the remainder is overdispersed, not Poisson")
}

func TestNowcast_FullyBookedWindowExpectsNothingMore(t *testing.T) {
	t.Parallel()

	assert.Equal(t, demandforecast.Count{}, demandforecast.Nowcast(38.8, 10.4, 4, 0))
}

func TestCount_ExceedsAndQuantileAgreeWithAPoisson(t *testing.T) {
	t.Parallel()

	c := demandforecast.Count{Mean: 3, Variance: 3}

	assert.InDelta(t, 0.5768, c.Exceeds(2), 1e-3)
	assert.InDelta(t, 1.0, c.Exceeds(-1), 1e-9)
	assert.Equal(t, 3, c.Quantile(0.5))
}

type forecastWorld struct {
	rates  []*demandforecast.Rate
	areas  []*demandforecast.Area
	curves []*demandforecast.BookingCurve
	now    int64
	window int64
}

func dallasWorld(t *testing.T) forecastWorld {
	t.Helper()
	cell := dallasCell(t)
	now := mondayMidnight + 18*3600
	window := now + 18*3600
	rates := make([]*demandforecast.Rate, 0, 6)
	for i := range int64(3) {
		slot := dispatchmarket.SlotOf(window + i*dispatchmarket.SlotSeconds)
		rates = append(rates,
			&demandforecast.Rate{Cell: cell, Slot: slot, Alpha: 11.6, Beta: 10},
			&demandforecast.Rate{CustomerID: acme, Cell: cell, Slot: slot, Alpha: 9, Beta: 10},
		)
	}

	return forecastWorld{
		rates:  rates,
		areas:  []*demandforecast.Area{{Cell: cell, Label: "Dallas, TX", MedianRevenueCents: 45_000, Loads: 400}},
		curves: []*demandforecast.BookingCurve{{Shares: []float64{1, 0.9, 0.85, 0.8, 0.7, 0.4, 0.3, 0.2, 0.1, 0.05, 0.02, 0.01, 0}}},
		now:    now,
		window: window,
	}
}

func (w forecastWorld) forecaster(supply ...demandforecast.Driver) *demandforecast.Forecaster {
	return demandforecast.NewForecaster(&demandforecast.ForecasterParams{
		Now:           w.now,
		Quality:       &demandforecast.Quality{Status: demandforecast.StatusReady},
		Rates:         w.rates,
		Areas:         w.areas,
		Curves:        w.curves,
		Supply:        supply,
		CustomerNames: map[pulid.ID]string{acme: "Acme Foods"},
		MarginPercent: 80,
	})
}

func idleDrivers(t *testing.T, n int, at int64) []demandforecast.Driver {
	t.Helper()
	drivers := make([]demandforecast.Driver, 0, n)
	for i := range n {
		drivers = append(drivers, demandforecast.Driver{
			WorkerID: pulid.ID("wrk_" + string(rune('a'+i))),
			Cell:     dallasCell(t),
			FreeAt:   at,
		})
	}

	return drivers
}

func TestCredit_FallsAsSupplyRises(t *testing.T) {
	t.Parallel()

	w := dallasWorld(t)
	query := demandforecast.Query{Point: dallas, FreeAt: w.window, WorkerID: pulid.ID("wrk_me")}

	previous := int64(math.MaxInt64)
	for _, k := range []int{0, 1, 2, 4, 8} {
		value, ok := w.forecaster(idleDrivers(t, k, w.now)...).Value(query)
		require.True(t, ok)
		assert.LessOrEqual(t, value.Cents, previous, "credit with %d other drivers", k)
		assert.Equal(t, k, value.Reason.DriversExpected)
		previous = value.Cents
	}
	none, _ := w.forecaster().Value(query)
	crowded, _ := w.forecaster(idleDrivers(t, 8, w.now)...).Value(query)
	assert.Greater(t, none.Cents, int64(10_000), "the first driver into a likely market is worth over $100")
	assert.Less(t, crowded.Cents, none.Cents/4, "the ninth is worth a fraction of that")
}

func TestCredit_DoesNotCountTheDriverItValues(t *testing.T) {
	t.Parallel()

	w := dallasWorld(t)
	me := pulid.ID("wrk_me")
	self := []demandforecast.Driver{{WorkerID: me, Cell: dallasCell(t), FreeAt: w.now}}

	value, ok := w.forecaster(self...).Value(demandforecast.Query{Point: dallas, FreeAt: w.window, WorkerID: me})

	require.True(t, ok)
	assert.Zero(t, value.Reason.DriversExpected)
}

func TestCredit_IsCappedByTheSetting(t *testing.T) {
	t.Parallel()

	w := dallasWorld(t)
	w.areas[0].MedianRevenueCents = 10_000_000
	f := demandforecast.NewForecaster(&demandforecast.ForecasterParams{
		Now: w.now, Quality: &demandforecast.Quality{Status: demandforecast.StatusReady},
		Rates: w.rates, Areas: w.areas, Curves: w.curves, MarginPercent: 80, CapCents: 40_000,
	})

	value, ok := f.Value(demandforecast.Query{Point: dallas, FreeAt: w.window})

	require.True(t, ok)
	assert.Equal(t, int64(40_000), value.Cents)
}

/*
A fixed $400 default capped every area of a fleet whose loads earn thousands, so every window
read $400 and the forecast could not say which area mattered more. Unset, the cap is what a
typical load is worth to the organization: its load-weighted median revenue times its margin.
*/
func TestAutomaticCap_IsWhatATypicalLoadIsWorth(t *testing.T) {
	t.Parallel()

	areas := []*demandforecast.Area{
		{Cell: 1, MedianRevenueCents: 100_000, Loads: 10},
		{Cell: 2, MedianRevenueCents: 300_000, Loads: 70},
		{Cell: 3, MedianRevenueCents: 900_000, Loads: 20},
		{Cell: 4, MedianRevenueCents: 0, Loads: 500},
	}

	assert.Equal(t, int64(90_000), demandforecast.AutomaticCapCents(areas, 30),
		"the typical load earns $3,000 and the margin is 30%%")
	assert.Zero(t, demandforecast.AutomaticCapCents(nil, 30))
	assert.Equal(t, int64(30_000), demandforecast.AutomaticCapCents(
		[]*demandforecast.Area{{Cell: 9, MedianRevenueCents: 100_000}}, 30,
	), "an area with no load count still counts once")

	w := dallasWorld(t)
	f := w.forecaster()
	value, ok := f.Value(demandforecast.Query{Point: dallas, FreeAt: w.window})
	require.True(t, ok)
	assert.Equal(t, int64(36_000), f.CapCents(), "one area: its own $450 at an 80%% margin")
	assert.Less(t, value.Cents, f.CapCents(), "the typical area is never held at the cap")
}

func TestCredit_BookedButUnplannedLoadsAreDemandTooUnlessBeingPlanned(t *testing.T) {
	t.Parallel()

	w := dallasWorld(t)
	busy := demandforecast.NewForecaster(&demandforecast.ForecasterParams{
		Now: w.now, Quality: &demandforecast.Quality{Status: demandforecast.StatusReady},
		Rates: w.rates, Areas: w.areas, Curves: w.curves, MarginPercent: 80,
		Supply:  idleDrivers(t, 3, w.now),
		Booked:  []demandforecast.Booked{{Cell: dallasCell(t), SlotStart: w.window, Total: 4, Open: 4}},
		Solving: []demandforecast.Booked{{Cell: dallasCell(t), SlotStart: w.window, Open: 1}},
	})
	plain := w.forecaster(idleDrivers(t, 3, w.now)...)
	query := demandforecast.Query{Point: dallas, FreeAt: w.window}

	withBooked, _ := busy.Value(query)
	without, _ := plain.Value(query)

	assert.Greater(t, withBooked.Cents, without.Cents,
		"three loads booked for the window and not in this solve still need drivers")
	assert.Equal(t, 4, withBooked.Reason.Booked)
}

func TestForecaster_StaysSilentUntilTheForecastIsReady(t *testing.T) {
	t.Parallel()

	w := dallasWorld(t)
	f := demandforecast.NewForecaster(&demandforecast.ForecasterParams{
		Now: w.now, Quality: &demandforecast.Quality{Status: demandforecast.StatusNoSkill},
		Rates: w.rates, Areas: w.areas, Curves: w.curves, MarginPercent: 80,
	})

	_, ok := f.Value(demandforecast.Query{Point: dallas, FreeAt: w.window})

	assert.False(t, ok)
	assert.False(t, f.Covers(dallasCell(t)))
}

func TestForecaster_WithSupplyRecountsWithoutChangingTheOriginal(t *testing.T) {
	t.Parallel()

	w := dallasWorld(t)
	base := w.forecaster()
	crowded := base.WithSupply(idleDrivers(t, 5, w.now))
	query := demandforecast.Query{Point: dallas, FreeAt: w.window}

	before, _ := base.Value(query)
	after, _ := crowded.Value(query)

	assert.Zero(t, before.Reason.DriversExpected)
	assert.Equal(t, 5, after.Reason.DriversExpected)
	assert.Less(t, after.Cents, before.Cents)
}

func patternedHistory(t *testing.T, weeks int64) *demandforecast.History {
	t.Helper()
	r := rand.New(rand.NewPCG(3, 5))
	current := dispatchmarket.WeekOf(mondayMidnight) + weeks
	h := &demandforecast.History{Now: nowAtWeek(current)}
	means := map[int16]float64{1: 4, 7: 3, 13: 5, 25: 2, 31: 1}
	for week := current - weeks; week < current; week++ {
		for slot, mean := range means {
			h.Counts = append(h.Counts, demandforecast.LoadCount{
				CustomerID: acme, Cell: dallasCell(t), Week: week, Slot: slot, Loads: poisson(r, mean),
			})
		}
	}

	return h
}

func TestBacktest_BeatsSameSlotLastWeekOnARegularRhythm(t *testing.T) {
	t.Parallel()

	quality := demandforecast.Backtest(patternedHistory(t, 26))

	assert.Equal(t, demandforecast.StatusReady, quality.Status)
	assert.EqualValues(t, demandforecast.BacktestWeeks, quality.WeeksTested)
	assert.Greater(t, quality.Skill, 0.2, "beats repeating last week by more than 20%%")
	assert.Greater(t, quality.IntervalCoverage, 0.7)
	assert.LessOrEqual(t, quality.IntervalCoverage, 1.0)
}

func TestBacktest_SaysWhenHistoryIsTooShort(t *testing.T) {
	t.Parallel()

	quality := demandforecast.Backtest(patternedHistory(t, 5))

	assert.Equal(t, demandforecast.StatusInsufficientHistory, quality.Status)
	assert.EqualValues(t, 5, quality.HistoryWeeks)
}

func TestCount_ShiftAddsAKnownCountNotASpread(t *testing.T) {
	t.Parallel()

	booked := demandforecast.Count{}.Shift(4)

	assert.InDelta(t, 1.0, booked.Exceeds(3), 1e-9)
	assert.InDelta(t, 0.0, booked.Exceeds(4), 1e-9)
	assert.Equal(t, 4, booked.Quantile(0.1))
	assert.Equal(t, 4, booked.Quantile(0.9))

	withMore := demandforecast.Count{Mean: 2, Variance: 2}.Shift(4)
	assert.InDelta(t, demandforecast.Count{Mean: 2, Variance: 2}.Exceeds(1), withMore.Exceeds(5), 1e-12)
	assert.GreaterOrEqual(t, withMore.Quantile(0.1), 4)
}
