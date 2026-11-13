package demandforecast_test

import (
	"testing"
	"time"
	_ "time/tzdata"

	"github.com/emoss08/trenova/internal/modules/operations/domain/demandforecast"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchmarket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func everySlot(t *testing.T, alpha, beta float64) []*demandforecast.Rate {
	t.Helper()
	rates := make([]*demandforecast.Rate, 0, dispatchmarket.SlotsPerWeek)
	for slot := range dispatchmarket.SlotsPerWeek {
		rates = append(rates, &demandforecast.Rate{
			Cell: dallasCell(t), Slot: slot, Alpha: alpha, Beta: beta,
		})
	}

	return rates
}

func dallasArea(t *testing.T) []*demandforecast.Area {
	t.Helper()

	return []*demandforecast.Area{
		{Cell: dallasCell(t), Label: "Dallas, TX", MedianRevenueCents: 45_000, Loads: 400},
	}
}

func TestForecaster_ExpectsAHolidaysShareOfAnOrdinaryDay(t *testing.T) {
	t.Parallel()

	now := mondayMidnight + 9*3600
	holiday := (now + dispatchmarket.DaySeconds) / dispatchmarket.DaySeconds
	f := demandforecast.NewForecaster(&demandforecast.ForecasterParams{
		Now: now,
		Index: demandforecast.NewIndex(&demandforecast.IndexParams{
			Quality: &demandforecast.Quality{
				Status:        demandforecast.StatusReady,
				HolidayFactor: 0.25,
				Holidays:      []int64{holiday},
			},
			Rates: everySlot(t, 8, 2),
			Areas: dallasArea(t),
		}),
		MarginPercent: 80,
	})
	starts := f.OutlookStarts()

	working := f.Slot(dallasCell(t), starts[1])
	closed := f.Slot(dallasCell(t), starts[1]+dispatchmarket.DaySeconds)

	assert.InDelta(t, 4.0, working.Need, 1e-9)
	assert.InDelta(t, 1.0, closed.Need, 1e-9, "a quarter of the usual four")
	assert.Less(t, closed.Demand.Quantile(demandforecast.IntervalHigh),
		working.Demand.Quantile(demandforecast.IntervalHigh))

	window := f.ValueAt(
		demandforecast.WindowKey{Cell: dallasCell(t), Start: starts[1] + dispatchmarket.DaySeconds},
		"",
	)
	assert.InDelta(t, 3.0, window.Reason.ExpectedUnbooked, 1e-9, "three holiday slots of one load")
}

func TestForecaster_ReadsItsSlotsOnTheClockItWasFittedOn(t *testing.T) {
	t.Parallel()

	chicago, err := time.LoadLocation("America/Chicago")
	require.NoError(t, err)
	grid := dispatchmarket.GridIn(chicago)
	evening := time.Date(2026, time.July, 14, 18, 0, 0, 0, chicago).Unix()
	rates := []*demandforecast.Rate{
		{Cell: dallasCell(t), Slot: grid.Slot(evening), Alpha: 12, Beta: 2},
	}
	forecaster := func(timezone string) *demandforecast.Forecaster {
		return demandforecast.NewForecaster(&demandforecast.ForecasterParams{
			Now: evening - 30*3600,
			Index: demandforecast.NewIndex(&demandforecast.IndexParams{
				Quality: &demandforecast.Quality{
					Status: demandforecast.StatusReady, Timezone: timezone,
				},
				Rates: rates,
				Areas: dallasArea(t),
			}),
			MarginPercent: 80,
		})
	}

	local := forecaster("America/Chicago")
	starts := local.OutlookStarts()
	require.Len(t, starts, demandforecast.OutlookSlots)
	for i, start := range starts {
		assert.Zero(
			t,
			time.Unix(start, 0).In(chicago).Hour()%4,
			"slot %d starts on a local mark",
			i,
		)
	}
	slot := local.Slot(dallasCell(t), grid.SlotStart(evening))
	assert.InDelta(t, 6.0, slot.Need, 1e-9, "Tuesday evening in Chicago is the fitted slot")
	assert.Equal(t, grid.SlotEnd(evening), slot.End)
	rate, ok := local.RateAt(dallasCell(t), evening)
	require.True(t, ok)
	assert.Same(t, rates[0], rate)

	utc := forecaster("")
	assert.Zero(t, utc.Slot(dallasCell(t), dispatchmarket.SlotStart(evening)).Need,
		"on the UTC clock that hour belongs to another slot")

	balance := local.Balance(dallasCell(t), starts)
	for i := range balance.Slots[:len(balance.Slots)-1] {
		assert.Equal(t, balance.Slots[i+1].Start, balance.Slots[i].End, "slots meet end to start")
	}
}

func TestIndex_WeighsWhatItHoldsAndStaysSilentUntilReady(t *testing.T) {
	t.Parallel()

	empty := demandforecast.NewIndex(&demandforecast.IndexParams{})
	assert.False(t, empty.Ready())
	assert.Empty(t, empty.Cells())
	assert.Equal(t, 1, empty.Weight())

	full := demandforecast.NewIndex(&demandforecast.IndexParams{
		Quality: &demandforecast.Quality{Status: demandforecast.StatusReady},
		Rates:   everySlot(t, 4, 2),
		Areas:   dallasArea(t),
	})
	assert.True(t, full.Ready())
	assert.Equal(t, []int64{dallasCell(t)}, full.Cells())
	assert.Equal(t, 1+int(dispatchmarket.SlotsPerWeek)+1, full.Weight())
}
