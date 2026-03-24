package compliance

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/emoss08/trenova/shared/samsara/internal/httpx"
	"github.com/emoss08/trenova/shared/samsara/internal/httpxtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHOSClocksValidation(t *testing.T) {
	t.Parallel()

	svc := NewService(
		&httpxtest.MockRequester{DoFunc: func(_ context.Context, _ httpx.Request) error {
			return nil
		}},
	)

	_, err := svc.HOSClocks(t.Context(), HOSClocksParams{Limit: 513})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrListLimitInvalid)
}

func TestHOSClocksQueryAndPath(t *testing.T) {
	t.Parallel()

	svc := NewService(
		&httpxtest.MockRequester{DoFunc: func(_ context.Context, req httpx.Request) error {
			assert.Equal(t, http.MethodGet, req.Method)
			assert.Equal(t, "/fleet/hos/clocks", req.Path)
			assert.Equal(t, "d1,d2", req.Query.Get("driverIds"))
			assert.Equal(t, "100", req.Query.Get("limit"))
			assert.Equal(t, "cursor-1", req.Query.Get("after"))
			return nil
		}},
	)

	_, err := svc.HOSClocks(t.Context(), HOSClocksParams{
		DriverIDs: []string{"d1", "d2"},
		Limit:     100,
		After:     "cursor-1",
	})
	require.NoError(t, err)
}

func TestHOSLogsQueryAndPath(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, 3, 1, 8, 0, 0, 0, time.UTC)
	end := start.Add(2 * time.Hour)
	svc := NewService(
		&httpxtest.MockRequester{DoFunc: func(_ context.Context, req httpx.Request) error {
			assert.Equal(t, http.MethodGet, req.Method)
			assert.Equal(t, "/fleet/hos/logs", req.Path)
			assert.Equal(t, start.Format(time.RFC3339), req.Query.Get("startTime"))
			assert.Equal(t, end.Format(time.RFC3339), req.Query.Get("endTime"))
			return nil
		}},
	)

	_, err := svc.HOSLogs(t.Context(), HOSLogsParams{
		StartTime: &start,
		EndTime:   &end,
	})
	require.NoError(t, err)
}

func TestDriverTachographPath(t *testing.T) {
	t.Parallel()

	svc := NewService(
		&httpxtest.MockRequester{DoFunc: func(_ context.Context, req httpx.Request) error {
			assert.Equal(t, http.MethodGet, req.Method)
			assert.Equal(t, "/fleet/drivers/tachograph-files/history", req.Path)
			assert.Equal(t, "drv-1", req.Query.Get("driverIds"))
			return nil
		}},
	)

	_, err := svc.DriverTachographHistory(t.Context(), DriverTachographParams{
		DriverIDs: []string{"drv-1"},
	})
	require.NoError(t, err)
}

func TestVehicleTachographPath(t *testing.T) {
	t.Parallel()

	svc := NewService(
		&httpxtest.MockRequester{DoFunc: func(_ context.Context, req httpx.Request) error {
			assert.Equal(t, http.MethodGet, req.Method)
			assert.Equal(t, "/fleet/vehicles/tachograph-files/history", req.Path)
			assert.Equal(t, "veh-1", req.Query.Get("vehicleIds"))
			return nil
		}},
	)

	_, err := svc.VehicleTachographHistory(t.Context(), VehicleTachographParams{
		VehicleIDs: []string{"veh-1"},
	})
	require.NoError(t, err)
}
