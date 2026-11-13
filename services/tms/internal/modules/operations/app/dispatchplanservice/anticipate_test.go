package dispatchplanservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/app/dispatchcandidateservice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/demandforecast"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatcheconomics"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchmarket"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchboardports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchplanports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const anticipateNow = int64(1_793_577_600)

func likelyDallas(t *testing.T) (*demandforecast.Forecaster, demandforecast.WindowKey) {
	t.Helper()
	cell, ok := dispatchmarket.CellOf(dispatchmarket.Point{Latitude: 32.7767, Longitude: -96.7970})
	require.True(t, ok)
	shares := make([]float64, len(demandforecast.LeadHours))
	shares[0] = 1
	params := &demandforecast.IndexParams{
		Quality: &demandforecast.Quality{Status: demandforecast.StatusReady},
		Areas: []*demandforecast.Area{
			{Cell: cell, Label: "Dallas, TX", MedianRevenueCents: 100_000},
		},
		Curves: []*demandforecast.BookingCurve{{Shares: shares}},
	}
	for slot := range dispatchmarket.SlotsPerWeek {
		params.Rates = append(
			params.Rates,
			&demandforecast.Rate{Cell: cell, Slot: slot, Alpha: 4, Beta: 10},
		)
	}
	forecaster := demandforecast.NewForecaster(&demandforecast.ForecasterParams{
		Now:           anticipateNow,
		Index:         demandforecast.NewIndex(params),
		MarginPercent: 50,
	})
	key, ok := forecaster.Window(cell, anticipateNow+6*3600)
	require.True(t, ok)

	return forecaster, key
}

/*
Pass one prices every arc with the drivers the snapshot knows, so two drivers both headed for
the one load Dallas usually books were each credited as if they would be the only truck there.
Pass two recounts supply from where the first solve leaves every driver.
*/
func TestReanticipate_DoesNotCreditTwoDriversForOneLikelyLoad(t *testing.T) {
	t.Parallel()

	forecaster, key := likelyDallas(t)
	lat, lon := 32.7767, -96.7970
	drivers := []*dispatchboardports.BoardDriver{
		{WorkerID: pulid.ID("wrk_a")},
		{WorkerID: pulid.ID("wrk_b")},
	}
	graph := &dispatchcandidateservice.Graph{
		Drivers:  drivers,
		Coverage: []int64{1_000_000, 1_000_000},
	}
	firstCredit := int64(0)
	for row := range 2 {
		graph.Moves = append(graph.Moves, &dispatchboardports.BoardMove{
			MoveID:               pulid.ID("smv_" + string(rune('a'+row))),
			DestinationLatitude:  &lat,
			DestinationLongitude: &lon,
		})
		var arcs []dispatchcandidateservice.Arc
		for col, driver := range drivers {
			value := forecaster.ValueAt(key, driver.WorkerID)
			firstCredit = value.Cents
			base := int64(10_000 + 1_000*abs(row-col))
			arcs = append(arcs, dispatchcandidateservice.Arc{
				Driver:    col,
				CostCents: base - value.Cents,
				Candidate: &dispatchplanports.CandidateScore{
					WorkerID:        driver.WorkerID,
					ProjectedFreeAt: key.Start,
					HardCents:       base - value.Cents,
					Cost: dispatcheconomics.Cost{
						TotalCents: base - value.Cents,
						Lines: []dispatcheconomics.Line{
							{Component: dispatcheconomics.ComponentEmptyMiles, Cents: base},
							{Component: dispatcheconomics.ComponentEndMarket, Cents: -value.Cents},
						},
					},
					EndMarket: &dispatchplanports.EndMarket{
						Source:    dispatchplanports.EndMarketForecast,
						Cents:     value.Cents,
						DropCents: value.Cents,
						Window:    key,
						Reason:    value.Reason,
					},
				},
			})
		}
		graph.Arcs = append(graph.Arcs, arcs)
	}
	snapshot := &dispatchcandidateservice.FleetSnapshot{Now: anticipateNow, Forecast: forecaster}

	reanticipate(graph, snapshot, stability{})

	require.Positive(t, firstCredit)
	for _, arcs := range graph.Arcs {
		for _, arc := range arcs {
			market := arc.Candidate.EndMarket
			assert.Less(t, market.Cents, firstCredit, "the other driver headed for Dallas counts")
			assert.Equal(t, 1, market.Reason.DriversExpected)
			total := int64(0)
			for _, line := range arc.Candidate.Cost.Lines {
				total += line.Cents
			}
			assert.Equal(t, arc.Candidate.Cost.TotalCents, total, "lines still add up to the total")
			assert.Equal(t, arc.CostCents, arc.Candidate.HardCents,
				"the arc and the candidate moved together")
		}
	}
}

func TestReanticipate_LeavesAPlanWithoutForecastArcsAlone(t *testing.T) {
	t.Parallel()

	forecaster, _ := likelyDallas(t)
	graph := &dispatchcandidateservice.Graph{
		Drivers: []*dispatchboardports.BoardDriver{{WorkerID: pulid.ID("wrk_a")}},
		Moves:   []*dispatchboardports.BoardMove{{MoveID: pulid.ID("smv_a")}},
		Arcs: [][]dispatchcandidateservice.Arc{{{
			Driver: 0, CostCents: 5_000,
			Candidate: &dispatchplanports.CandidateScore{WorkerID: pulid.ID("wrk_a")},
		}}},
		Coverage: []int64{100_000},
	}

	reanticipate(graph, &dispatchcandidateservice.FleetSnapshot{Forecast: forecaster}, stability{})

	assert.Equal(t, int64(5_000), graph.Arcs[0][0].CostCents)
}

func abs(v int) int {
	if v < 0 {
		return -v
	}

	return v
}
