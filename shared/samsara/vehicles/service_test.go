package vehicles

import (
	"context"
	"net/http"
	"testing"

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
