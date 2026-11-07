package dispatchcandidateservice

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/demandforecast"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchmarket"
	"github.com/emoss08/trenova/internal/modules/operations/ports/demandforecastports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var anticipateTenant = pagination.TenantInfo{OrgID: pulid.ID("org_a"), BuID: pulid.ID("bu_a")}

type fakeForecasts struct {
	demandforecastports.Service
	forecaster *demandforecast.Forecaster
	err        error
	requests   []*demandforecastports.ForecasterRequest
}

func (f *fakeForecasts) Forecaster(
	_ context.Context,
	req *demandforecastports.ForecasterRequest,
) (*demandforecast.Forecaster, error) {
	f.requests = append(f.requests, req)
	if f.err != nil {
		return nil, f.err
	}

	return f.forecaster, nil
}

func anticipating(on bool) *dispatchcontrol.DispatchControl {
	control := dispatchcontrol.NewDefaultDispatchControl(
		anticipateTenant.OrgID,
		anticipateTenant.BuID,
	)
	control.AnticipateFreight = on
	control.Economics.CoverageMarginPercent = 25
	control.Economics.AnticipationMaxCents = 30_000

	return control
}

func pointCell(t *testing.T, lat, lon float64) int64 {
	t.Helper()
	cell, ok := dispatchmarket.CellOf(dispatchmarket.Point{Latitude: lat, Longitude: lon})
	require.True(t, ok)

	return cell
}

func TestAnticipate_LeavesTheForecastOutUnlessTurnedOnWithSomewhereToGo(t *testing.T) {
	t.Parallel()

	destinations := []dispatchmarket.Point{{Latitude: chicagoLat, Longitude: chicagoLon}}
	tests := []struct {
		name         string
		control      *dispatchcontrol.DispatchControl
		destinations []dispatchmarket.Point
	}{
		{name: "no dispatch control", destinations: destinations},
		{name: "turned off", control: anticipating(false), destinations: destinations},
		{name: "no destinations", control: anticipating(true)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			forecasts := &fakeForecasts{forecaster: demandforecast.NewForecaster(
				&demandforecast.ForecasterParams{},
			)}
			svc := &Service{l: zap.NewNop(), forecasts: forecasts}
			snapshot := newTestSnapshot()
			addPlacedDriver(snapshot, "Ana", denverLat, denverLon)

			err := svc.anticipate(t.Context(), &SnapshotRequest{
				TenantInfo:   anticipateTenant,
				Control:      tt.control,
				Destinations: tt.destinations,
				Now:          testNow,
			}, snapshot)

			require.NoError(t, err)
			assert.Nil(t, snapshot.Forecast)
			assert.Empty(t, forecasts.requests)
		})
	}
}

func TestAnticipate_ForecastsTheDestinationsAndWhereDriversAreWithTheirSupplyAndEconomics(t *testing.T) {
	t.Parallel()

	forecasts := &fakeForecasts{forecaster: demandforecast.NewForecaster(
		&demandforecast.ForecasterParams{},
	)}
	svc := &Service{l: zap.NewNop(), forecasts: forecasts}
	snapshot := newTestSnapshot()
	driver := addPlacedDriver(snapshot, "Ana", denverLat, denverLon)
	addDriver(snapshot, "Ben")
	snapshot.Prepare()
	chicago, dallas := pointCell(t, chicagoLat, chicagoLon), pointCell(t, dallasLat, dallasLon)
	solving := []demandforecast.Booked{{Cell: chicago, SlotStart: testNow, Total: 1, Open: 1}}

	err := svc.anticipate(t.Context(), &SnapshotRequest{
		TenantInfo: anticipateTenant,
		Control:    anticipating(true),
		Destinations: []dispatchmarket.Point{
			{Latitude: chicagoLat, Longitude: chicagoLon},
			{Latitude: dallasLat, Longitude: dallasLon},
			{Latitude: chicagoLat, Longitude: chicagoLon},
		},
		Solving: solving,
		Now:     testNow,
	}, snapshot)
	require.NoError(t, err)

	denver := pointCell(t, denverLat, denverLon)
	cells := []int64{chicago, dallas, denver}
	slices.Sort(cells)
	assert.Same(t, forecasts.forecaster, snapshot.Forecast)
	assert.Equal(t, []*demandforecastports.ForecasterRequest{{
		TenantInfo: anticipateTenant,
		Now:        testNow,
		Cells:      cells,
		Supply: []demandforecast.Driver{{
			WorkerID: driver.WorkerID,
			Cell:     pointCell(t, denverLat, denverLon),
			FreeAt:   testNow,
		}},
		Solving:       solving,
		MarginPercent: 25,
		CapCents:      30_000,
	}}, forecasts.requests, "a driver with no known position is not supply anywhere")
}

func TestAnticipate_ReturnsTheForecastError(t *testing.T) {
	t.Parallel()

	forecastDown := errors.New("forecast down")
	svc := &Service{l: zap.NewNop(), forecasts: &fakeForecasts{err: forecastDown}}
	snapshot := newTestSnapshot()

	err := svc.anticipate(t.Context(), &SnapshotRequest{
		TenantInfo:   anticipateTenant,
		Control:      anticipating(true),
		Destinations: []dispatchmarket.Point{{Latitude: chicagoLat, Longitude: chicagoLon}},
		Now:          testNow,
	}, snapshot)

	require.ErrorIs(t, err, forecastDown)
	assert.Nil(t, snapshot.Forecast)
}
