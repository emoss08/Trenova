package freightoutlookservice

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/operations/domain/demandforecast"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchmarket"
	"github.com/emoss08/trenova/internal/modules/operations/domain/driverpositioning"
	"github.com/emoss08/trenova/internal/modules/operations/domain/freightmove"
	"github.com/emoss08/trenova/internal/modules/operations/ports/demandforecastports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/freightoutlookports"
	"github.com/emoss08/trenova/internal/testutil/mocks/dispatchcontrolportsmock"
	"github.com/emoss08/trenova/internal/testutil/positioningtest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const testNow = int64(1_793_577_600 + 9*3600)

var (
	tenant = pagination.TenantInfo{OrgID: pulid.ID("org_1"), BuID: pulid.ID("bu_1")}
	dallas = dispatchmarket.Point{Latitude: 32.7767, Longitude: -96.7970}
	okc    = dispatchmarket.Point{Latitude: 35.4676, Longitude: -97.5164}
)

func cellOf(t *testing.T, p dispatchmarket.Point) int64 {
	t.Helper()
	cell, ok := dispatchmarket.CellOf(p)
	require.True(t, ok)
	return cell
}

type fakeForecasts struct {
	demandforecastports.Service

	quality *demandforecast.Quality
	areas   []*demandforecast.Area
	rates   []*demandforecast.Rate
	booked  []demandforecast.Booked
	request *demandforecastports.ForecasterRequest
}

func (f *fakeForecasts) Quality(
	context.Context,
	pagination.TenantInfo,
) (*demandforecast.Quality, error) {
	return f.quality, nil
}

func (f *fakeForecasts) Forecaster(
	_ context.Context,
	req *demandforecastports.ForecasterRequest,
) (*demandforecast.Forecaster, error) {
	f.request = req
	return demandforecast.NewForecaster(&demandforecast.ForecasterParams{
		Now:           req.Now,
		Quality:       f.quality,
		Rates:         f.rates,
		Areas:         f.areas,
		Booked:        f.booked,
		Supply:        req.Supply,
		MarginPercent: req.MarginPercent,
		CapCents:      req.CapCents,
	}), nil
}

func (f *fakeForecasts) Location(context.Context, pagination.TenantInfo) *time.Location {
	return time.UTC
}

type fakeRepo struct {
	demandforecastports.Repository

	starts     []demandforecastports.DriverStart
	days       []*demandforecast.Day
	startsRead int
	window     *demandforecastports.WindowRequest
	locations  []pulid.ID
	areaCell   int64
}

func (f *fakeRepo) ListDriverStarts(
	context.Context,
	*demandforecastports.DriverStartsRequest,
) ([]demandforecastports.DriverStart, error) {
	f.startsRead++
	return f.starts, nil
}

func (f *fakeRepo) ListOpenLanes(
	_ context.Context,
	req *demandforecastports.WindowRequest,
) ([]demandforecastports.OpenLane, error) {
	f.window = req
	return nil, nil
}

func (f *fakeRepo) ListDepartures(
	context.Context,
	*demandforecastports.WindowRequest,
) ([]demandforecastports.Departure, error) {
	return nil, nil
}

func (f *fakeRepo) ListAreaLocations(
	_ context.Context,
	req *demandforecastports.AreaLocationsRequest,
) ([]pulid.ID, error) {
	f.areaCell = req.Cell
	return f.locations, nil
}

func (f *fakeRepo) Days(context.Context, pagination.TenantInfo) ([]*demandforecast.Day, error) {
	return f.days, nil
}

func newTestService(
	t *testing.T,
	forecasts *fakeForecasts,
	repo *fakeRepo,
	live positioningtest.Fixed,
) *Service {
	t.Helper()
	controls := dispatchcontrolportsmock.NewMockDispatchControlRepository(t)
	controls.EXPECT().
		GetOrCreate(mock.Anything, tenant.OrgID, tenant.BuID).
		Return(dispatchcontrol.NewDefaultDispatchControl(tenant.OrgID, tenant.BuID), nil).
		Maybe()

	return &Service{
		l:            zap.NewNop(),
		forecasts:    forecasts,
		repo:         repo,
		controls:     controls,
		positionings: live,
		clock:        func() int64 { return testNow },
	}
}

func readyWorld(t *testing.T) (*fakeForecasts, *fakeRepo) {
	t.Helper()
	starts := demandforecast.OutlookStarts(testNow)
	dallasCell := cellOf(t, dallas)

	return &fakeForecasts{
		quality: &demandforecast.Quality{Status: demandforecast.StatusReady, FittedAt: testNow},
		areas: []*demandforecast.Area{
			{Cell: dallasCell, Label: "Dallas, TX", MedianRevenueCents: 250_000, Loads: 400},
			{
				Cell:               cellOf(t, okc),
				Label:              "Oklahoma City, OK",
				MedianRevenueCents: 30_000,
				Loads:              200,
			},
		},
		rates: []*demandforecast.Rate{{
			Cell: dallasCell, Slot: dispatchmarket.SlotOf(starts[4]), Alpha: 30, Beta: 10,
			Recent: []int32{3, 2, 4, 3, 3, 5, 2, 3},
		}},
		booked: []demandforecast.Booked{
			{Cell: dallasCell, SlotStart: starts[4], Total: 3, Open: 3},
		},
	}, &fakeRepo{
		starts: []demandforecastports.DriverStart{
			{
				WorkerID:  "wrk_ana",
				Name:      "Ana Ruiz",
				Latitude:  okc.Latitude,
				Longitude: okc.Longitude,
				FreeAt:    testNow,
			},
			{
				WorkerID:  "wrk_ben",
				Name:      "Ben Ito",
				Latitude:  okc.Latitude,
				Longitude: okc.Longitude,
				FreeAt:    testNow,
			},
		},
		days: []*demandforecast.Day{{DayStart: testNow - 86_400, Low: 3, High: 8, Actual: 5}},
	}
}

func TestOutlook_ReadsNothingElseUntilTheForecastIsReady(t *testing.T) {
	t.Parallel()

	for _, quality := range []*demandforecast.Quality{
		{Status: demandforecast.StatusInsufficientHistory},
		{Status: demandforecast.StatusNoSkill},
	} {
		repo := &fakeRepo{}
		svc := newTestService(t, &fakeForecasts{quality: quality}, repo, positioningtest.Fixed{})

		outlook, err := svc.Outlook(t.Context(), &freightoutlookports.Request{TenantInfo: tenant})
		require.NoError(t, err)

		assert.Same(t, quality, outlook.Quality)
		assert.Equal(t, testNow, outlook.GeneratedAt)
		assert.Len(t, outlook.Starts, demandforecast.OutlookSlots)
		assert.Empty(t, outlook.Areas)
		assert.Empty(t, outlook.Steps)
		assert.NotNil(t, outlook.Positionings)
		assert.Zero(t, repo.startsRead, "driver starts are not read for a forecast nobody sees")
	}
}

func TestOutlook_ReadsEveryAreaTheMovesThatFixThemAndTheTrackRecord(t *testing.T) {
	t.Parallel()

	forecasts, repo := readyWorld(t)
	positioning := &driverpositioning.Positioning{
		ID: "dpo_1", WorkerID: "wrk_ben", ToCell: cellOf(t, dallas), Status: driverpositioning.StatusActive,
	}
	svc := newTestService(t, forecasts, repo, positioningtest.Fixed{
		Rows: []*driverpositioning.Positioning{positioning},
	})

	outlook, err := svc.Outlook(t.Context(), &freightoutlookports.Request{TenantInfo: tenant})
	require.NoError(t, err)

	require.NotNil(t, forecasts.request)
	assert.True(
		t,
		forecasts.request.AllCells,
		"the outlook reads every area, not the ones a plan touches",
	)
	assert.Equal(t, horizonSeconds, forecasts.request.Horizon)
	assert.Len(t, forecasts.request.Supply, 2)
	require.NotNil(t, repo.window)
	assert.Equal(t, testNow, repo.window.From)
	assert.Equal(t, testNow+horizonSeconds, repo.window.Until)

	assert.Equal(t, 2, outlook.AreaCount)
	require.Len(t, outlook.Areas, 2)
	assert.Equal(t, "Dallas, TX", outlook.Areas[0].Balance.Label,
		"an area a move names comes first")
	assert.Positive(t, outlook.Areas[0].ValueCents)
	assert.Equal(t, dispatchmarket.CenterOf(cellOf(t, dallas)), outlook.Areas[0].Point)
	assert.Len(t, outlook.Network, demandforecast.OutlookSlots)

	require.NotEmpty(t, outlook.Steps)
	assert.Equal(t, freightmove.KindSend, outlook.Steps[0].Kind)
	assert.Equal(t, "Dallas, TX", outlook.Steps[0].ToLabel)
	assert.Equal(t, "Oklahoma City, OK", outlook.Steps[0].FromLabel)

	assert.Equal(t, repo.days, outlook.Days)
	require.Len(t, outlook.Positionings, 1)
	assert.Same(t, positioning, outlook.Positionings[0].Positioning)
	assert.Equal(t, "Ben Ito", outlook.Positionings[0].WorkerName)
}

func TestArea_ReadsTheWorstWindowWhenNoneIsAskedAndItsRecentWeeks(t *testing.T) {
	t.Parallel()

	forecasts, repo := readyWorld(t)
	repo.locations = []pulid.ID{"loc_dock_a", "loc_dock_b"}
	svc := newTestService(t, forecasts, repo, positioningtest.Fixed{})
	starts := demandforecast.OutlookStarts(testNow)

	detail, err := svc.Area(t.Context(), &freightoutlookports.AreaRequest{
		TenantInfo: tenant,
		Cell:       cellOf(t, dallas),
	})
	require.NoError(t, err)

	assert.Equal(t, cellOf(t, dallas), repo.areaCell)
	assert.Equal(t, []pulid.ID{"loc_dock_a", "loc_dock_b"}, detail.DeliveryLocationIDs,
		"the area names its locations, so its loads can be read")

	assert.Equal(t, starts[4], detail.SlotStart)
	assert.Equal(t, "Dallas, TX", detail.Area.Balance.Label)
	assert.True(t, detail.Bounds.Contains(dallas), "the area's bounds hold the place it is named for")
	assert.Less(t, detail.Bounds.West, detail.Bounds.East)
	assert.Less(t, detail.Bounds.South, detail.Bounds.North)
	assert.Equal(t, []int32{3, 2, 4, 3, 3, 5, 2, 3}, detail.Recent)
	typical := demandforecast.GammaPoisson(30, 10)
	assert.Equal(t, typical.Quantile(demandforecast.IntervalLow), detail.RecentLow)
	assert.Equal(t, typical.Quantile(demandforecast.IntervalHigh), detail.RecentHigh)
	require.NotEmpty(t, detail.Reach)
	assert.Equal(t, freightmove.ReachOnTime, detail.Reach[0].Reach)
	for _, step := range detail.Steps {
		assert.True(
			t,
			step.ToCell == detail.Area.Balance.Cell || step.FromCell == detail.Area.Balance.Cell,
		)
	}

	asked, err := svc.Area(t.Context(), &freightoutlookports.AreaRequest{
		TenantInfo: tenant, Cell: cellOf(t, dallas), SlotStart: starts[2],
	})
	require.NoError(t, err)
	assert.Equal(t, starts[2], asked.SlotStart, "a window in the outlook is read as asked")
}

func TestArea_RefusesAnAreaTheForecastDoesNotHave(t *testing.T) {
	t.Parallel()

	forecasts, repo := readyWorld(t)
	svc := newTestService(t, forecasts, repo, positioningtest.Fixed{})

	_, err := svc.Area(t.Context(), &freightoutlookports.AreaRequest{TenantInfo: tenant, Cell: 1})

	var notFound *errortypes.NotFoundError
	require.ErrorAs(t, err, &notFound)
}

func TestRelevant_PutsAreasAMoveNamesFirstThenTheBiggestGapAndStopsAtForty(t *testing.T) {
	t.Parallel()

	balances := make([]demandforecast.AreaBalance, 0, 45)
	for i := range 45 {
		balances = append(balances, demandforecast.AreaBalance{
			Cell:  int64(i + 1),
			Label: fmt.Sprintf("Area %d", i+1),
			Slots: []demandforecast.SlotBalance{{Short: float64(i) / 10}},
		})
	}
	steps := []freightmove.Step{{Kind: freightmove.KindSend, ToCell: 3, FromCell: 7}}

	got := relevant(balances, steps)

	require.Len(t, got, freightoutlookports.MaxAreas)
	assert.ElementsMatch(t, []int64{3, 7}, []int64{got[0].Cell, got[1].Cell})
	assert.Equal(t, int64(45), got[2].Cell, "then the most short")
	for _, balance := range got {
		assert.NotEqual(t, int64(1), balance.Cell, "the least relevant fall past the cut")
	}
}
