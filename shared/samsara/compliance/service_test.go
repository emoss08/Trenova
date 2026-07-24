package compliance

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/emoss08/trenova/shared/samsara/internal/httpx"
	"github.com/emoss08/trenova/shared/samsara/internal/httpxtest"
	samsaraspec "github.com/emoss08/trenova/shared/samsara/internal/samsaraspec"
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

func TestHOSClocksAllPaginates(t *testing.T) {
	t.Parallel()

	calls := 0
	svc := NewService(
		&httpxtest.MockRequester{DoFunc: func(_ context.Context, req httpx.Request) error {
			calls++
			out := req.Out.(*HOSClocksResponse)
			if calls == 1 {
				assert.Equal(t, "512", req.Query.Get("limit"))
				*out = HOSClocksResponse{
					Data: []HOSClock{{}},
					Pagination: samsaraspec.PaginationResponse{
						EndCursor:   "n1",
						HasNextPage: true,
					},
				}
				return nil
			}

			assert.Equal(t, "n1", req.Query.Get("after"))
			*out = HOSClocksResponse{
				Data: []HOSClock{{}, {}},
				Pagination: samsaraspec.PaginationResponse{
					EndCursor:   "",
					HasNextPage: false,
				},
			}
			return nil
		}},
	)

	items, err := svc.HOSClocksAll(t.Context(), HOSClocksParams{})
	require.NoError(t, err)
	assert.Len(t, items, 3)
	assert.Equal(t, 2, calls)
}

func TestHOSClocksAllValidation(t *testing.T) {
	t.Parallel()

	svc := NewService(
		&httpxtest.MockRequester{DoFunc: func(_ context.Context, _ httpx.Request) error {
			return nil
		}},
	)

	_, err := svc.HOSClocksAll(t.Context(), HOSClocksParams{Limit: 513})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrListLimitInvalid)
}

func TestHOSDailyLogsValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		params HOSDailyLogsParams
	}{
		{
			name:   "invalid start date",
			params: HOSDailyLogsParams{StartDate: "03-01-2026"},
		},
		{
			name:   "invalid end date",
			params: HOSDailyLogsParams{EndDate: "2026-13-45"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc := NewService(
				&httpxtest.MockRequester{DoFunc: func(_ context.Context, _ httpx.Request) error {
					return nil
				}},
			)

			_, err := svc.HOSDailyLogs(t.Context(), tt.params)
			require.Error(t, err)
			assert.ErrorIs(t, err, ErrDateFormatInvalid)
		})
	}
}

func TestHOSDailyLogsQueryAndPath(t *testing.T) {
	t.Parallel()

	svc := NewService(
		&httpxtest.MockRequester{DoFunc: func(_ context.Context, req httpx.Request) error {
			assert.Equal(t, http.MethodGet, req.Method)
			assert.Equal(t, "/fleet/hos/daily-logs", req.Path)
			assert.Equal(t, "d1,d2", req.Query.Get("driverIds"))
			assert.Equal(t, "2026-03-01", req.Query.Get("startDate"))
			assert.Equal(t, "2026-03-07", req.Query.Get("endDate"))
			assert.Equal(t, "tag-1", req.Query.Get("tagIds"))
			assert.Equal(t, "ptag-1", req.Query.Get("parentTagIds"))
			assert.Equal(t, "active", req.Query.Get("driverActivationStatus"))
			assert.Equal(t, "cursor-1", req.Query.Get("after"))
			assert.Equal(t, "vehicle", req.Query.Get("expand"))
			return nil
		}},
	)

	_, err := svc.HOSDailyLogs(t.Context(), HOSDailyLogsParams{
		DriverIDs:              []string{"d1", "d2"},
		StartDate:              "2026-03-01",
		EndDate:                "2026-03-07",
		TagIDs:                 []string{"tag-1"},
		ParentTagIDs:           []string{"ptag-1"},
		DriverActivationStatus: "active",
		After:                  "cursor-1",
		Expand:                 []string{"vehicle"},
	})
	require.NoError(t, err)
}

func TestHOSViolationsQueryAndPath(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)
	svc := NewService(
		&httpxtest.MockRequester{DoFunc: func(_ context.Context, req httpx.Request) error {
			assert.Equal(t, http.MethodGet, req.Method)
			assert.Equal(t, "/fleet/hos/violations", req.Path)
			assert.Equal(t, "d1", req.Query.Get("driverIds"))
			assert.Equal(t, start.Format(time.RFC3339), req.Query.Get("startTime"))
			assert.Equal(t, end.Format(time.RFC3339), req.Query.Get("endTime"))
			assert.Equal(t, "tag-1", req.Query.Get("tagIds"))
			assert.Equal(t, "ptag-1", req.Query.Get("parentTagIds"))
			assert.Equal(t, "shiftHours,dailyDrivingHours", req.Query.Get("types"))
			assert.Equal(t, "cursor-1", req.Query.Get("after"))
			return nil
		}},
	)

	_, err := svc.HOSViolations(t.Context(), HOSViolationsParams{
		DriverIDs:    []string{"d1"},
		StartTime:    &start,
		EndTime:      &end,
		TagIDs:       []string{"tag-1"},
		ParentTagIDs: []string{"ptag-1"},
		Types:        []string{"shiftHours", "dailyDrivingHours"},
		After:        "cursor-1",
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
