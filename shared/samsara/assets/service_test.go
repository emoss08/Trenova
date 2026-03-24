package assets

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/emoss08/trenova/shared/samsara/internal/httpx"
	"github.com/emoss08/trenova/shared/samsara/internal/httpxtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreatePassesRequestThrough(t *testing.T) {
	t.Parallel()

	svc := NewService(
		&httpxtest.MockRequester{DoFunc: func(_ context.Context, req httpx.Request) error {
			out := req.Out.(*createResponse)
			*out = createResponse{
				Data: Asset{Id: "asset-1"},
			}
			return nil
		}},
	)

	_, err := svc.Create(t.Context(), CreateRequest{})
	require.NoError(t, err)
}

func TestListQuery(t *testing.T) {
	t.Parallel()

	called := false
	svc := NewService(
		&httpxtest.MockRequester{DoFunc: func(_ context.Context, req httpx.Request) error {
			called = true
			assert.Equal(t, http.MethodGet, req.Method)
			assert.Equal(t, "/assets", req.Path)
			assert.Equal(t, "vehicle", req.Query.Get("type"))
			assert.Equal(t, "abc", req.Query.Get("after"))
			assert.Equal(t, "a1,a2", req.Query.Get("ids"))
			assert.Equal(t, "true", req.Query.Get("includeTags"))
			assert.Equal(t, "true", req.Query.Get("includeExternalIds"))
			out := req.Out.(*ListResponse)
			*out = ListResponse{
				Data:       []Asset{{Id: "a1"}},
				Pagination: PaginationResponse{HasNextPage: false, EndCursor: ""},
			}
			return nil
		}},
	)

	page, err := svc.List(t.Context(), ListParams{
		Type:               TypeVehicle,
		After:              "abc",
		IDs:                []string{"a1", "a2"},
		IncludeTags:        true,
		IncludeExternalIDs: true,
	})
	require.NoError(t, err)
	require.True(t, called)
	require.Len(t, page.Data, 1)
	assert.Equal(t, "a1", page.Data[0].Id)
}

func TestDeleteIDsValidation(t *testing.T) {
	t.Parallel()

	svc := NewService(
		&httpxtest.MockRequester{DoFunc: func(_ context.Context, _ httpx.Request) error {
			return nil
		}},
	)

	err := svc.Delete(t.Context(), nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrAssetIDsRequired)
}

func TestDeleteByID(t *testing.T) {
	t.Parallel()

	calls := 0
	svc := NewService(
		&httpxtest.MockRequester{DoFunc: func(_ context.Context, req httpx.Request) error {
			calls++
			assert.Equal(t, http.MethodDelete, req.Method)
			assert.Equal(t, "/assets", req.Path)
			require.Equal(t, []int{http.StatusNoContent}, req.ExpectedStatus)
			if calls == 1 {
				assert.Equal(t, "id-1", req.Query.Get("id"))
			} else {
				assert.Equal(t, "id-2", req.Query.Get("id"))
			}
			return nil
		}},
	)

	err := svc.Delete(t.Context(), []string{"id-1", " ", "id-2"})
	require.NoError(t, err)
	assert.Equal(t, 2, calls)
}

func TestStreamLocationValidation(t *testing.T) {
	t.Parallel()

	svc := NewService(
		&httpxtest.MockRequester{DoFunc: func(_ context.Context, _ httpx.Request) error {
			return nil
		}},
	)

	_, err := svc.StreamLocationAndSpeed(t.Context(), LocationStreamParams{})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrLocationStartTimeRequired)
}

func TestStreamLocationPagesPaginates(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, 3, 1, 8, 0, 0, 0, time.UTC)
	cursor1 := "cursor-1"
	calls := 0
	svc := NewService(
		&httpxtest.MockRequester{DoFunc: func(_ context.Context, req httpx.Request) error {
			calls++
			assert.Equal(t, http.MethodGet, req.Method)
			assert.Equal(t, "/assets/location-and-speed/stream", req.Path)
			out := req.Out.(*LocationStreamResponse)
			if calls == 1 {
				assert.Equal(t, "", req.Query.Get("after"))
				*out = LocationStreamResponse{
					Data: []StreamRecord{
						{
							Asset:          StreamAsset{Id: "a1"},
							HappenedAtTime: start.Format(time.RFC3339),
						},
					},
					Pagination: StreamPaginationResponse{
						EndCursor:   &cursor1,
						HasNextPage: true,
					},
				}
				return nil
			}
			assert.Equal(t, "cursor-1", req.Query.Get("after"))
			*out = LocationStreamResponse{
				Data: []StreamRecord{
					{
						Asset:          StreamAsset{Id: "a2"},
						HappenedAtTime: start.Add(time.Minute).Format(time.RFC3339),
					},
				},
				Pagination: StreamPaginationResponse{EndCursor: nil, HasNextPage: false},
			}
			return nil
		}},
	)

	var seen []string
	err := svc.StreamLocationPages(
		t.Context(),
		LocationStreamParams{StartTime: &start},
		func(page *LocationStreamResponse) error {
			for _, item := range page.Data {
				seen = append(seen, item.Asset.Id)
			}
			return nil
		},
	)
	require.NoError(t, err)
	assert.Equal(t, []string{"a1", "a2"}, seen)
	assert.Equal(t, 2, calls)
}

func TestCurrentLocationsLatestByAsset(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 1, 9, 30, 0, 0, time.UTC)
	first := now.Add(-10 * time.Minute)
	second := now.Add(-5 * time.Minute)
	other := now.Add(-4 * time.Minute)
	calls := 0
	svc := newService(
		&httpxtest.MockRequester{DoFunc: func(_ context.Context, req httpx.Request) error {
			calls++
			out := req.Out.(*LocationStreamResponse)
			*out = LocationStreamResponse{
				Data: []StreamRecord{
					{
						Asset:          StreamAsset{Id: "asset-1"},
						HappenedAtTime: first.Format(time.RFC3339),
					},
					{
						Asset:          StreamAsset{Id: "asset-1"},
						HappenedAtTime: second.Format(time.RFC3339),
					},
					{
						Asset:          StreamAsset{Id: "asset-2"},
						HappenedAtTime: other.Format(time.RFC3339),
					},
				},
				Pagination: StreamPaginationResponse{EndCursor: nil, HasNextPage: false},
			}
			return nil
		}},
		func() time.Time {
			return now
		},
	)

	current, err := svc.CurrentLocations(t.Context(), CurrentLocationsParams{})
	require.NoError(t, err)
	require.Equal(t, 1, calls)
	require.Len(t, current.Data, 2)
	assert.Equal(t, "asset-1", current.Data[0].Asset.Id)
	assert.Equal(t, second.Format(time.RFC3339), current.Data[0].HappenedAtTime)
	assert.Equal(t, "asset-2", current.Data[1].Asset.Id)
}

func TestHistoricalLocationsCollectsAndSorts(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, 3, 1, 7, 0, 0, 0, time.UTC)
	end := start.Add(2 * time.Hour)
	svc := NewService(
		&httpxtest.MockRequester{DoFunc: func(_ context.Context, req httpx.Request) error {
			out := req.Out.(*LocationStreamResponse)
			*out = LocationStreamResponse{
				Data: []StreamRecord{
					{
						Asset:          StreamAsset{Id: "asset-2"},
						HappenedAtTime: start.Add(15 * time.Minute).Format(time.RFC3339),
					},
					{
						Asset:          StreamAsset{Id: "asset-1"},
						HappenedAtTime: start.Add(10 * time.Minute).Format(time.RFC3339),
					},
				},
				Pagination: StreamPaginationResponse{EndCursor: nil, HasNextPage: false},
			}
			return nil
		}},
	)

	history, err := svc.HistoricalLocations(t.Context(), HistoricalLocationsParams{
		StartTime: start,
		EndTime:   end,
	})
	require.NoError(t, err)
	require.Len(t, history.Data, 2)
	assert.Equal(t, "asset-1", history.Data[0].Asset.Id)
	assert.Equal(t, "asset-2", history.Data[1].Asset.Id)
}

func TestStreamLocationPagesRequiresCallback(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, 3, 1, 7, 0, 0, 0, time.UTC)
	svc := NewService(
		&httpxtest.MockRequester{DoFunc: func(_ context.Context, _ httpx.Request) error {
			return nil
		}},
	)

	err := svc.StreamLocationPages(t.Context(), LocationStreamParams{StartTime: &start}, nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrCallbackNil)
}

func TestStreamLocationPagesBubblesCallbackError(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, 3, 1, 7, 0, 0, 0, time.UTC)
	expected := errors.New("boom")
	svc := NewService(
		&httpxtest.MockRequester{DoFunc: func(_ context.Context, req httpx.Request) error {
			out := req.Out.(*LocationStreamResponse)
			*out = LocationStreamResponse{
				Data:       []StreamRecord{},
				Pagination: StreamPaginationResponse{EndCursor: nil, HasNextPage: false},
			}
			return nil
		}},
	)

	err := svc.StreamLocationPages(
		t.Context(),
		LocationStreamParams{StartTime: &start},
		func(_ *LocationStreamResponse) error {
			return expected
		},
	)
	require.Error(t, err)
	assert.ErrorIs(t, err, expected)
}
