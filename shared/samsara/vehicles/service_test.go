package vehicles

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

func TestStatsValidation(t *testing.T) {
	t.Parallel()

	svc := NewService(
		&httpxtest.MockRequester{DoFunc: func(_ context.Context, _ httpx.Request) error {
			return nil
		}},
	)

	_, err := svc.Stats(t.Context(), StatsParams{Limit: 513})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrListLimitInvalid)
}

func TestStatsQueryAndPath(t *testing.T) {
	t.Parallel()

	svc := NewService(
		&httpxtest.MockRequester{DoFunc: func(_ context.Context, req httpx.Request) error {
			assert.Equal(t, http.MethodGet, req.Method)
			assert.Equal(t, "/fleet/vehicles/stats", req.Path)
			assert.Equal(t, "veh-1,veh-2", req.Query.Get("vehicleIds"))
			assert.Equal(t, "gps,engineState", req.Query.Get("types"))
			return nil
		}},
	)

	_, err := svc.Stats(t.Context(), StatsParams{
		VehicleIDs: []string{"veh-1", "veh-2"},
		Types:      []string{"gps", "engineState"},
	})
	require.NoError(t, err)
}

func TestStatsAllPaginates(t *testing.T) {
	t.Parallel()

	calls := 0
	svc := NewService(
		&httpxtest.MockRequester{DoFunc: func(_ context.Context, req httpx.Request) error {
			calls++
			out := req.Out.(*StatsResponse)
			if calls == 1 {
				*out = StatsResponse{
					Data: []StatsData{{}},
					Pagination: samsaraspec.PaginationResponse{
						EndCursor:   "n1",
						HasNextPage: true,
					},
				}
				return nil
			}

			assert.Equal(t, "n1", req.Query.Get("after"))
			*out = StatsResponse{
				Data: []StatsData{{}, {}},
				Pagination: samsaraspec.PaginationResponse{
					EndCursor:   "",
					HasNextPage: false,
				},
			}
			return nil
		}},
	)

	items, err := svc.StatsAll(t.Context(), StatsParams{})
	require.NoError(t, err)
	assert.Len(t, items, 3)
	assert.Equal(t, 2, calls)
}

func TestStatsTypesTooMany(t *testing.T) {
	t.Parallel()

	svc := NewService(
		&httpxtest.MockRequester{DoFunc: func(_ context.Context, _ httpx.Request) error {
			return nil
		}},
	)

	_, err := svc.Stats(t.Context(), StatsParams{
		Types: []string{"gps", "engineStates", "obdOdometerMeters", "fuelPercents"},
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrStatsTypesTooMany)
}

func TestStatsFeedValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		params  StatsFeedParams
		wantErr error
	}{
		{
			name:    "missing types",
			params:  StatsFeedParams{},
			wantErr: ErrStatsTypesRequired,
		},
		{
			name: "too many types",
			params: StatsFeedParams{
				Types: []string{"gps", "engineStates", "obdOdometerMeters", "fuelPercents"},
			},
			wantErr: ErrStatsTypesTooMany,
		},
		{
			name: "limit too large",
			params: StatsFeedParams{
				Types: []string{"gps"},
				Limit: 513,
			},
			wantErr: ErrListLimitInvalid,
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

			_, err := svc.StatsFeed(t.Context(), tt.params)
			require.Error(t, err)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestStatsFeedQueryAndPath(t *testing.T) {
	t.Parallel()

	svc := NewService(
		&httpxtest.MockRequester{DoFunc: func(_ context.Context, req httpx.Request) error {
			assert.Equal(t, http.MethodGet, req.Method)
			assert.Equal(t, "/fleet/vehicles/stats/feed", req.Path)
			assert.Equal(t, "cursor-1", req.Query.Get("after"))
			assert.Equal(t, "veh-1,veh-2", req.Query.Get("vehicleIds"))
			assert.Equal(t, "tag-1", req.Query.Get("tagIds"))
			assert.Equal(t, "ptag-1", req.Query.Get("parentTagIds"))
			assert.Equal(t, "gps,engineStates", req.Query.Get("types"))
			assert.Equal(t, "obdOdometerMeters", req.Query.Get("decorations"))
			assert.Equal(t, "100", req.Query.Get("limit"))
			return nil
		}},
	)

	_, err := svc.StatsFeed(t.Context(), StatsFeedParams{
		After:        "cursor-1",
		VehicleIDs:   []string{"veh-1", "veh-2"},
		TagIDs:       []string{"tag-1"},
		ParentTagIDs: []string{"ptag-1"},
		Types:        []string{"gps", "engineStates"},
		Decorations:  []string{"obdOdometerMeters"},
		Limit:        100,
	})
	require.NoError(t, err)
}

func TestStatsHistoryValidation(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, 3, 1, 8, 0, 0, 0, time.UTC)
	end := start.Add(2 * time.Hour)

	tests := []struct {
		name    string
		params  StatsHistoryParams
		wantErr error
	}{
		{
			name:    "missing time range",
			params:  StatsHistoryParams{Types: []string{"gps"}},
			wantErr: ErrStatsTimeRangeRequired,
		},
		{
			name: "missing end time",
			params: StatsHistoryParams{
				StartTime: start,
				Types:     []string{"gps"},
			},
			wantErr: ErrStatsTimeRangeRequired,
		},
		{
			name: "missing types",
			params: StatsHistoryParams{
				StartTime: start,
				EndTime:   end,
			},
			wantErr: ErrStatsTypesRequired,
		},
		{
			name: "too many types",
			params: StatsHistoryParams{
				StartTime: start,
				EndTime:   end,
				Types:     []string{"gps", "engineStates", "obdOdometerMeters", "fuelPercents"},
			},
			wantErr: ErrStatsTypesTooMany,
		},
		{
			name: "limit too large",
			params: StatsHistoryParams{
				StartTime: start,
				EndTime:   end,
				Types:     []string{"gps"},
				Limit:     513,
			},
			wantErr: ErrListLimitInvalid,
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

			_, err := svc.StatsHistory(t.Context(), tt.params)
			require.Error(t, err)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestStatsHistoryQueryAndPath(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, 3, 1, 8, 0, 0, 0, time.UTC)
	end := start.Add(2 * time.Hour)
	svc := NewService(
		&httpxtest.MockRequester{DoFunc: func(_ context.Context, req httpx.Request) error {
			assert.Equal(t, http.MethodGet, req.Method)
			assert.Equal(t, "/fleet/vehicles/stats/history", req.Path)
			assert.Equal(t, start.Format(time.RFC3339), req.Query.Get("startTime"))
			assert.Equal(t, end.Format(time.RFC3339), req.Query.Get("endTime"))
			assert.Equal(t, "veh-1", req.Query.Get("vehicleIds"))
			assert.Equal(t, "gps,fuelPercents", req.Query.Get("types"))
			return nil
		}},
	)

	_, err := svc.StatsHistory(t.Context(), StatsHistoryParams{
		StartTime:  start,
		EndTime:    end,
		VehicleIDs: []string{"veh-1"},
		Types:      []string{"gps", "fuelPercents"},
	})
	require.NoError(t, err)
}
