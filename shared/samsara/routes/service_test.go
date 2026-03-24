package routes

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

func TestListValidation(t *testing.T) {
	t.Parallel()

	svc := NewService(
		&httpxtest.MockRequester{DoFunc: func(_ context.Context, _ httpx.Request) error {
			return nil
		}},
	)

	_, err := svc.List(t.Context(), ListParams{Limit: 513})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrListLimitInvalid)

	start := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	end := start.Add(-time.Hour)
	_, err = svc.List(t.Context(), ListParams{StartTime: &start, EndTime: &end})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrListLimitInvalid)
}

func TestListAllPaginates(t *testing.T) {
	t.Parallel()

	calls := 0
	svc := NewService(
		&httpxtest.MockRequester{DoFunc: func(_ context.Context, req httpx.Request) error {
			calls++
			out := req.Out.(*ListResponse)
			if calls == 1 {
				*out = ListResponse{
					Data: []Route{{Id: "r1"}},
					Pagination: samsaraspec.GoaPaginationResponseResponseBody{
						EndCursor:   "next-1",
						HasNextPage: true,
					},
				}
				return nil
			}

			assert.Equal(t, "next-1", req.Query.Get("after"))
			*out = ListResponse{
				Data: []Route{{Id: "r2"}},
				Pagination: samsaraspec.GoaPaginationResponseResponseBody{
					EndCursor:   "",
					HasNextPage: false,
				},
			}
			return nil
		}},
	)

	routes, err := svc.ListAll(t.Context(), ListParams{})
	require.NoError(t, err)
	require.Len(t, routes, 2)
	assert.Equal(t, "r1", routes[0].Id)
	assert.Equal(t, "r2", routes[1].Id)
	assert.Equal(t, 2, calls)
}

func TestGetValidation(t *testing.T) {
	t.Parallel()

	svc := NewService(
		&httpxtest.MockRequester{DoFunc: func(_ context.Context, _ httpx.Request) error {
			return nil
		}},
	)

	_, err := svc.Get(t.Context(), " ")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrRouteIDRequired)
}

func TestGetPathAndResponse(t *testing.T) {
	t.Parallel()

	svc := NewService(
		&httpxtest.MockRequester{DoFunc: func(_ context.Context, req httpx.Request) error {
			assert.Equal(t, http.MethodGet, req.Method)
			assert.Equal(t, "/fleet/routes/r-1", req.Path)
			out := req.Out.(*routeResponse)
			out.Data = &Route{Id: "r-1"}
			return nil
		}},
	)

	route, err := svc.Get(t.Context(), "r-1")
	require.NoError(t, err)
	assert.Equal(t, "r-1", route.Id)
}

func TestCreateValidation(t *testing.T) {
	t.Parallel()

	svc := NewService(
		&httpxtest.MockRequester{DoFunc: func(_ context.Context, _ httpx.Request) error {
			return nil
		}},
	)

	_, err := svc.Create(t.Context(), CreateRequest{})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrRouteNameRequired)
}

func TestCreatePathAndResponse(t *testing.T) {
	t.Parallel()

	svc := NewService(
		&httpxtest.MockRequester{DoFunc: func(_ context.Context, req httpx.Request) error {
			assert.Equal(t, http.MethodPost, req.Method)
			assert.Equal(t, "/fleet/routes", req.Path)
			out := req.Out.(*createResponse)
			out.Data = &Route{Id: "r-new"}
			return nil
		}},
	)

	created, err := svc.Create(t.Context(), CreateRequest{Name: "Route A", Stops: []Stop{}})
	require.NoError(t, err)
	assert.Equal(t, "r-new", created.Id)
}

func TestUpdateValidation(t *testing.T) {
	t.Parallel()

	svc := NewService(
		&httpxtest.MockRequester{DoFunc: func(_ context.Context, _ httpx.Request) error {
			return nil
		}},
	)

	_, err := svc.Update(t.Context(), " ", UpdateRequest{})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrRouteIDRequired)
}

func TestUpdatePathAndResponse(t *testing.T) {
	t.Parallel()

	svc := NewService(
		&httpxtest.MockRequester{DoFunc: func(_ context.Context, req httpx.Request) error {
			assert.Equal(t, http.MethodPatch, req.Method)
			assert.Equal(t, "/fleet/routes/r-1", req.Path)
			out := req.Out.(*updateResponse)
			out.Data = &Route{Id: "r-1"}
			return nil
		}},
	)

	updated, err := svc.Update(t.Context(), "r-1", UpdateRequest{})
	require.NoError(t, err)
	assert.Equal(t, "r-1", updated.Id)
}

func TestDeleteValidationAndPath(t *testing.T) {
	t.Parallel()

	svc := NewService(
		&httpxtest.MockRequester{DoFunc: func(_ context.Context, req httpx.Request) error {
			assert.Equal(t, http.MethodDelete, req.Method)
			assert.Equal(t, "/fleet/routes/r-1", req.Path)
			require.Equal(t, []int{http.StatusNoContent}, req.ExpectedStatus)
			return nil
		}},
	)

	err := svc.Delete(t.Context(), " ")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrRouteIDRequired)

	err = svc.Delete(t.Context(), "r-1")
	require.NoError(t, err)
}
