package addresses

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/emoss08/trenova/shared/samsara/internal/httpx"
	"github.com/emoss08/trenova/shared/samsara/internal/httpxtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListParamsQuery(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	params := ListParams{
		Limit:            100,
		After:            "cursor-1",
		ParentTagIDs:     []string{"p1", "p2"},
		TagIDs:           []string{"t1", "t2"},
		CreatedAfterTime: &now,
	}

	query := params.Query()
	assert.Equal(t, "100", query.Get("limit"))
	assert.Equal(t, "cursor-1", query.Get("after"))
	assert.Equal(t, "p1,p2", query.Get("parentTagIds"))
	assert.Equal(t, "t1,t2", query.Get("tagIds"))
	assert.Equal(t, now.Format(time.RFC3339), query.Get("createdAfterTime"))
}

func TestListAllPaginates(t *testing.T) {
	t.Parallel()

	calls := 0
	svc := NewService(
		&httpxtest.MockRequester{DoFunc: func(_ context.Context, req httpx.Request) error {
			calls++
			if req.Method != http.MethodGet || req.Path != "/addresses" {
				return errors.New("unexpected request")
			}

			out := req.Out.(*ListPage)
			if calls == 1 {
				assert.Equal(t, "", req.Query.Get("after"))
				*out = ListPage{
					Data: []Address{{Id: "a1"}},
					Pagination: PaginationResponse{
						EndCursor:   "next-1",
						HasNextPage: true,
					},
				}
				return nil
			}

			assert.Equal(t, "next-1", req.Query.Get("after"))
			*out = ListPage{
				Data:       []Address{{Id: "a2"}},
				Pagination: PaginationResponse{EndCursor: "", HasNextPage: false},
			}
			return nil
		}},
	)

	items, err := svc.ListAll(t.Context(), ListParams{})
	require.NoError(t, err)
	require.Len(t, items, 2)
	assert.Equal(t, "a1", items[0].Id)
	assert.Equal(t, "a2", items[1].Id)
	assert.Equal(t, 2, calls)
}

func TestCreateValidation(t *testing.T) {
	t.Parallel()

	svc := NewService(
		&httpxtest.MockRequester{
			DoFunc: func(_ context.Context, _ httpx.Request) error { return nil },
		},
	)

	_, err := svc.Create(t.Context(), CreateRequest{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "name is required")

	_, err = svc.Create(t.Context(), CreateRequest{
		Name:             "HQ",
		FormattedAddress: "350 Rhode Island St",
		Geofence: Geofence{
			Circle: &GeofenceCircle{RadiusMeters: 10},
			Polygon: &GeofencePolygon{
				Vertices: []GeofenceVertex{
					{Latitude: 1, Longitude: 1},
					{Latitude: 2, Longitude: 2},
					{Latitude: 3, Longitude: 3},
				},
			},
		},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot define both")
}

func TestListLimitValidation(t *testing.T) {
	t.Parallel()

	svc := NewService(
		&httpxtest.MockRequester{
			DoFunc: func(_ context.Context, _ httpx.Request) error { return nil },
		},
	)

	_, err := svc.List(t.Context(), ListParams{Limit: 513})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "between 1 and 512")
}

func TestQueryEncodingCompatibility(t *testing.T) {
	t.Parallel()

	values := ListParams{ParentTagIDs: []string{"123", "456"}}.Query()
	require.Equal(t, url.Values{"parentTagIds": []string{"123,456"}}, values)
}
