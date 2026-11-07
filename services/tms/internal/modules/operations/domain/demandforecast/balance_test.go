package demandforecast_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/demandforecast"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchmarket"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var houston = dispatchmarket.Point{Latitude: 29.7604, Longitude: -95.3698}

func houstonCell(t *testing.T) int64 {
	t.Helper()
	cell, ok := dispatchmarket.CellOf(houston)
	require.True(t, ok)

	return cell
}

type balanceWorld struct {
	now     int64
	starts  []int64
	booked  []demandforecast.Booked
	solving []demandforecast.Booked
	supply  []demandforecast.Driver
	rates   []*demandforecast.Rate
	areas   []*demandforecast.Area
	curves  []*demandforecast.BookingCurve
}

func newBalanceWorld(t *testing.T) *balanceWorld {
	t.Helper()
	now := mondayMidnight + 9*3600

	return &balanceWorld{
		now:    now,
		starts: demandforecast.OutlookStarts(now),
		areas: []*demandforecast.Area{
			{Cell: dallasCell(t), Label: "Dallas, TX", MedianRevenueCents: 45_000, Loads: 400},
			{Cell: houstonCell(t), Label: "Houston, TX", MedianRevenueCents: 40_000, Loads: 300},
		},
	}
}

func (w *balanceWorld) forecaster() *demandforecast.Forecaster {
	return demandforecast.NewForecaster(&demandforecast.ForecasterParams{
		Now:           w.now,
		Quality:       &demandforecast.Quality{Status: demandforecast.StatusReady},
		Rates:         w.rates,
		Areas:         w.areas,
		Curves:        w.curves,
		Booked:        w.booked,
		Solving:       w.solving,
		Supply:        w.supply,
		CustomerNames: map[pulid.ID]string{acme: "Acme Foods"},
		MarginPercent: 80,
	})
}

func driverAt(id string, cell, freeAt int64) demandforecast.Driver {
	return demandforecast.Driver{WorkerID: pulid.ID(id), Cell: cell, FreeAt: freeAt}
}

func TestOutlookStarts_AreTwelveFourHourWindowsFromTheOneUnderWay(t *testing.T) {
	t.Parallel()

	now := mondayMidnight + 9*3600 + 15*60
	starts := demandforecast.OutlookStarts(now)

	require.Len(t, starts, demandforecast.OutlookSlots)
	assert.Equal(t, mondayMidnight+8*3600, starts[0], "the window under way comes first")
	for i := 1; i < len(starts); i++ {
		assert.Equal(t, dispatchmarket.SlotSeconds, starts[i]-starts[i-1])
	}
}

func TestBalance_ADriverServesOneLoadAndLoadsWithNoTruckDoNotCarryOver(t *testing.T) {
	t.Parallel()

	w := newBalanceWorld(t)
	cell := dallasCell(t)
	w.booked = []demandforecast.Booked{
		{Cell: cell, SlotStart: w.starts[0], Total: 3, Open: 3},
		{Cell: cell, SlotStart: w.starts[1], Total: 2, Open: 2},
	}
	w.supply = []demandforecast.Driver{
		driverAt("wrk_a", cell, w.now),
		driverAt("wrk_b", cell, w.now-3600),
		driverAt("wrk_c", cell, w.starts[1]+3600),
	}

	balance := w.forecaster().Balance(cell, w.starts)

	assert.Equal(t, "Dallas, TX", balance.Label)
	first, second, third := balance.Slots[0], balance.Slots[1], balance.Slots[2]
	assert.InDelta(t, 3, first.Need, 1e-9)
	assert.Equal(t, 2, first.Drivers)
	assert.InDelta(t, 1, first.Short, 1e-9, "three loads, two drivers: one load has no truck")
	assert.InDelta(t, 0, first.Spare, 1e-9)
	assert.Equal(t, 3, first.Low, "booked loads are certain")
	assert.Equal(t, 3, first.High)

	assert.Equal(t, 3, second.Drivers, "drivers free by the end of the window, counted as a stock")
	assert.InDelta(t, 1, second.Short, 1e-9,
		"the two first drivers were used; only the new one serves the second window")
	assert.InDelta(
		t,
		0,
		third.Short,
		1e-9,
		"a load with no truck is not carried into later windows",
	)
}

func TestBalance_DriversLeftAfterServingAreSpareUntilFreightUsesThem(t *testing.T) {
	t.Parallel()

	w := newBalanceWorld(t)
	cell := dallasCell(t)
	w.booked = []demandforecast.Booked{
		{Cell: cell, SlotStart: w.starts[0], Total: 1, Open: 1},
		{Cell: cell, SlotStart: w.starts[3], Total: 1, Open: 1},
	}
	w.supply = []demandforecast.Driver{
		driverAt(
			"wrk_a",
			cell,
			w.now,
		), driverAt("wrk_b", cell, w.now), driverAt("wrk_c", cell, w.now),
	}

	slots := w.forecaster().Balance(cell, w.starts).Slots

	assert.InDelta(t, 2, slots[0].Spare, 1e-9)
	assert.InDelta(t, 2, slots[2].Spare, 1e-9, "spare drivers stay spare while nothing needs them")
	assert.InDelta(t, 1, slots[3].Spare, 1e-9)
	assert.InDelta(t, 0, slots[3].Short, 1e-9)
}

func TestBalance_LoadsAPlanIsAlreadyCoveringNeedNoOtherDriver(t *testing.T) {
	t.Parallel()

	w := newBalanceWorld(t)
	cell := dallasCell(t)
	w.booked = []demandforecast.Booked{{Cell: cell, SlotStart: w.starts[0], Total: 2, Open: 2}}
	w.solving = []demandforecast.Booked{{Cell: cell, SlotStart: w.starts[0], Total: 1, Open: 1}}

	slot := w.forecaster().Balance(cell, w.starts).Slots[0]

	assert.InDelta(t, 1, slot.Need, 1e-9)
	assert.Equal(t, 2, slot.Booked, "booked still counts toward the loads likely")
}

func TestNetwork_SeparatesHowManyDriversFromWhereTheyAre(t *testing.T) {
	t.Parallel()

	w := newBalanceWorld(t)
	dallas, houstonArea := dallasCell(t), houstonCell(t)
	w.booked = []demandforecast.Booked{{Cell: dallas, SlotStart: w.starts[0], Total: 2, Open: 2}}
	w.supply = []demandforecast.Driver{
		driverAt("wrk_a", houstonArea, w.now), driverAt("wrk_b", houstonArea, w.now),
	}
	f := w.forecaster()
	areas := []demandforecast.AreaBalance{
		f.Balance(dallas, w.starts),
		f.Balance(houstonArea, w.starts),
	}

	network := f.Network(areas, w.starts)

	require.Len(t, network, demandforecast.OutlookSlots)
	first := network[0]
	assert.InDelta(t, 2, first.Drivers, 1e-9)
	assert.InDelta(t, 2, first.Need, 1e-9)
	assert.InDelta(t, 0, first.Short, 1e-9, "the fleet has enough drivers")
	assert.InDelta(t, 2, first.Uncovered, 1e-9, "but Dallas has no truck for its two loads")
	assert.InDelta(t, 2, first.Idle, 1e-9, "and Houston's two drivers have nothing to haul")
}

func TestNetwork_MeasuresDriversAgainstTheLoadsStillNeedingOne(t *testing.T) {
	t.Parallel()

	w := newBalanceWorld(t)
	dallas := dallasCell(t)
	w.booked = []demandforecast.Booked{{Cell: dallas, SlotStart: w.starts[0], Total: 5, Open: 2}}
	f := w.forecaster()
	areas := []demandforecast.AreaBalance{f.Balance(dallas, w.starts)}

	first := f.Network(areas, w.starts)[0]

	assert.Equal(t, 5, first.Low, "every load likely counts the three already covered")
	assert.Equal(t, 5, first.High)
	assert.Equal(t, 2, first.NeedLow, "only the two open loads still need a driver")
	assert.Equal(t, 2, first.NeedHigh)
	assert.InDelta(t, 2, first.Need, 1e-9)
}

func TestSlot_MeasuresEachCustomerAgainstItsOwnBookingCurve(t *testing.T) {
	t.Parallel()

	w := newBalanceWorld(t)
	cell := dallasCell(t)
	start := w.starts[2]
	slot := dispatchmarket.SlotOf(start)
	shares := []float64{1, 0.9, 0.85, 0.8, 0.7, 0.4, 0.3, 0.2, 0.1, 0.05, 0.02, 0.01, 0}
	w.rates = []*demandforecast.Rate{
		{Cell: cell, Slot: slot, Alpha: 50, Beta: 10},
		{CustomerID: acme, Cell: cell, Slot: slot, Alpha: 40, Beta: 10},
	}
	w.curves = []*demandforecast.BookingCurve{{CustomerID: acme, Shares: shares}}
	w.booked = []demandforecast.Booked{
		{CustomerID: acme, Cell: cell, SlotStart: start, Total: 1, Open: 1},
	}

	forecast := w.forecaster().Slot(cell, start)

	require.Len(t, forecast.Customers, 1)
	customer := forecast.Customers[0]
	lead := float64(start+dispatchmarket.SlotSeconds/2-w.now) / 3600
	assert.Equal(t, "Acme Foods", customer.Name)
	assert.InDelta(t, 4, customer.Usual, 1e-9)
	assert.InDelta(t, 4*demandforecast.BookedShare(shares, lead), customer.ByNow, 1e-9)
	assert.Equal(t, 1, customer.Tendered)
	assert.Equal(t, 1, forecast.Booked)
	assert.Greater(t, forecast.Need, 1.0, "freight still to be booked needs drivers too")
}

func TestOneMoreDriver_IsWorthWhatTheAreaPaysAtItsWorstWindow(t *testing.T) {
	t.Parallel()

	w := newBalanceWorld(t)
	cell := dallasCell(t)
	w.booked = []demandforecast.Booked{{Cell: cell, SlotStart: w.starts[4], Total: 3, Open: 3}}
	f := w.forecaster()
	balance := f.Balance(cell, w.starts)

	index, short := balance.Worst()
	require.Equal(t, 4, index)
	assert.InDelta(t, 3, short, 1e-9)
	assert.Equal(t,
		f.ValueAt(demandforecast.WindowKey{Cell: cell, Start: w.starts[4]}, pulid.Nil).Cents,
		f.OneMoreDriver(&balance),
	)
	assert.Positive(t, f.OneMoreDriver(&balance))
}

func TestFit_KeepsTheLastEightWeeksOfEachAreaSlot(t *testing.T) {
	t.Parallel()

	h := steadyHistory(t, 4, 12, 7)
	fitted := demandforecast.Fit(h)
	rate := areaRate(t, fitted, 7)

	want := make([]int32, 0, demandforecast.RecordWeeks)
	for _, count := range h.Counts[len(h.Counts)-demandforecast.RecordWeeks:] {
		want = append(want, count.Loads)
	}
	assert.Equal(t, want, rate.Recent, "oldest first, one entry a week")
	assert.Len(t, areaRate(t, fitted, 8).Recent, demandforecast.RecordWeeks,
		"a slot with no loads still has eight weeks, all zero")
}

func TestTrackRecord_ForecastsEachDayOnlyFromTheWeeksBeforeIt(t *testing.T) {
	t.Parallel()

	h := patternedHistory(t, 26)
	days := demandforecast.TrackRecord(h)

	require.Len(t, days, demandforecast.RecordDays)
	for i, day := range days {
		assert.LessOrEqual(t, day.Low, day.High)
		assert.Equal(t, h.Now, day.FittedAt)
		if i > 0 {
			assert.Equal(t, demandforecast.DaySeconds, day.DayStart-days[i-1].DayStart)
		}
		actual := int32(0)
		week := dispatchmarket.WeekOf(day.DayStart)
		first := dispatchmarket.SlotOf(day.DayStart)
		for _, count := range h.Counts {
			if count.Week == week && count.Slot >= first && count.Slot < first+6 {
				actual += count.Loads
			}
		}
		assert.Equal(t, actual, day.Actual, "day %d counts every load that started that day", i)
	}

	lastWeek := dispatchmarket.WeekOf(h.Now) - 1
	busier := &demandforecast.History{
		Now:    h.Now,
		Counts: append([]demandforecast.LoadCount{}, h.Counts...),
	}
	for i := range busier.Counts {
		if busier.Counts[i].Week == lastWeek {
			busier.Counts[i].Loads += 50
		}
	}
	again := demandforecast.TrackRecord(busier)
	var before, after int32
	for i := range days {
		if dispatchmarket.WeekOf(days[i].DayStart) != lastWeek {
			continue
		}
		assert.Equal(t, days[i].Mean, again[i].Mean,
			"a day's forecast never sees the loads of its own week")
		before += days[i].Actual
		after += again[i].Actual
	}
	assert.Greater(t, after, before, "the busier week's days count the extra loads")
}

func TestValueAt_CountsBookedLoadsInAWindowWithNoLearnedRate(t *testing.T) {
	t.Parallel()

	w := newBalanceWorld(t)
	cell := dallasCell(t)
	w.booked = []demandforecast.Booked{{Cell: cell, SlotStart: w.starts[4], Total: 3, Open: 3}}

	value := w.forecaster().ValueAt(demandforecast.WindowKey{Cell: cell, Start: w.starts[4]}, pulid.Nil)

	assert.Positive(t, value.Cents, "a booked load with no driver is worth a driver, learned or not")
	assert.Equal(t, 3, value.Reason.Booked)
}
