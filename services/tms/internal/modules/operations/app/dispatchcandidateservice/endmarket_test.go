package dispatchcandidateservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/demandforecast"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchmarket"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchplanports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const dallasLat, dallasLon = 32.7767, -96.7970

type fixedEndMarket int64

func (f fixedEndMarket) EndMarketCents(float64, float64, int64) int64 { return int64(f) }

type forecastArea struct {
	lat, lon float64
	alpha    float64
}

func forecasterFor(t *testing.T, areas ...forecastArea) *demandforecast.Forecaster {
	t.Helper()
	params := &demandforecast.IndexParams{
		Quality: &demandforecast.Quality{Status: demandforecast.StatusReady},
		Curves: []*demandforecast.BookingCurve{
			{Shares: make([]float64, len(demandforecast.LeadHours))},
		},
	}
	params.Curves[0].Shares[0] = 1
	for _, area := range areas {
		cell, ok := dispatchmarket.CellOf(
			dispatchmarket.Point{Latitude: area.lat, Longitude: area.lon},
		)
		require.True(t, ok)
		params.Areas = append(params.Areas, &demandforecast.Area{
			Cell: cell, Label: "area", MedianRevenueCents: 100_000,
		})
		for slot := range dispatchmarket.SlotsPerWeek {
			params.Rates = append(params.Rates, &demandforecast.Rate{
				Cell: cell, Slot: slot, Alpha: area.alpha, Beta: 10,
			})
		}
	}

	return demandforecast.NewForecaster(&demandforecast.ForecasterParams{
		Now:           testNow,
		Index:         demandforecast.NewIndex(params),
		MarginPercent: 30,
	})
}

func TestEndMarket_UsesTheForecastWhereItCoversTheDropAndHistoryElsewhere(t *testing.T) {
	t.Parallel()

	s := newTestSnapshot()
	s.EndMarket = fixedEndMarket(5_000)
	s.Forecast = forecasterFor(t, forecastArea{lat: chicagoLat, lon: chicagoLon, alpha: 40})
	driver := addPlacedDriver(s, "Ana", denverLat, denverLon)

	toChicago := pickupAt(denverLat, denverLon, 900)
	toChicago.DestinationLatitude, toChicago.DestinationLongitude = new(chicagoLat), new(chicagoLon)
	toDallas := pickupAt(denverLat, denverLon, 700)
	toDallas.DestinationLatitude, toDallas.DestinationLongitude = new(dallasLat), new(dallasLon)

	forecast := evaluateFor(s, driver, toChicago, nil)
	history := evaluateFor(s, driver, toDallas, nil)

	require.NotNil(t, forecast.endMarket)
	assert.Equal(t, dispatchplanports.EndMarketForecast, forecast.endMarket.Source)
	assert.Positive(t, forecast.endMarket.DropCents)
	assert.Zero(
		t,
		forecast.endMarket.StayCents,
		"Denver is not forecast, so staying there is worth nothing",
	)
	assert.Equal(t, forecast.endMarket.Cents, forecast.features.EndMarketCents)
	require.NotNil(t, history.endMarket)
	assert.Equal(t, dispatchplanports.EndMarketHistory, history.endMarket.Source)
	assert.Equal(
		t,
		int64(5_000),
		history.endMarket.Cents,
		"never both: history fills in only where the forecast does not reach",
	)
}

func TestEndMarket_ChargesForTheValueLeftBehind(t *testing.T) {
	t.Parallel()

	s := newTestSnapshot()
	s.Forecast = forecasterFor(t,
		forecastArea{lat: denverLat, lon: denverLon, alpha: 60},
		forecastArea{lat: chicagoLat, lon: chicagoLon, alpha: 2},
	)
	driver := addPlacedDriver(s, "Ana", denverLat, denverLon)
	move := pickupAt(denverLat, denverLon, 900)
	move.DestinationLatitude, move.DestinationLongitude = new(chicagoLat), new(chicagoLon)

	eval := evaluateFor(s, driver, move, nil)

	require.NotNil(t, eval.endMarket)
	assert.Greater(t, eval.endMarket.StayCents, eval.endMarket.DropCents,
		"busy Denver is worth more to this driver than quiet Chicago")
	assert.Negative(t, eval.endMarket.Cents)
	assert.Equal(t, eval.endMarket.Cents, eval.features.EndMarketCents,
		"pulling the truck out of a busy area costs the plan")
}
