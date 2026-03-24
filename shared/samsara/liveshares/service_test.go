package liveshares

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/emoss08/trenova/shared/samsara/internal/httpx"
	"github.com/emoss08/trenova/shared/samsara/internal/httpxtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListParamsValidation(t *testing.T) {
	t.Parallel()

	err := ListParams{Limit: 101}.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "between 1 and 100")

	err = ListParams{Type: "bad"}.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid live share list type")
}

func TestListAllPaginates(t *testing.T) {
	t.Parallel()

	calls := 0
	svc := NewService(
		&httpxtest.MockRequester{DoFunc: func(_ context.Context, req httpx.Request) error {
			calls++
			if req.Method != http.MethodGet || req.Path != "/live-shares" {
				return errors.New("unexpected request")
			}

			out := req.Out.(*ListPage)
			if calls == 1 {
				*out = ListPage{
					Data:       []LiveShare{{Id: "ls-1"}},
					Pagination: PaginationResponse{EndCursor: "n1", HasNextPage: true},
				}
				return nil
			}

			assert.Equal(t, "n1", req.Query.Get("after"))
			*out = ListPage{
				Data:       []LiveShare{{Id: "ls-2"}},
				Pagination: PaginationResponse{EndCursor: "", HasNextPage: false},
			}
			return nil
		}},
	)

	items, err := svc.ListAll(t.Context(), ListParams{})
	require.NoError(t, err)
	require.Len(t, items, 2)
	assert.Equal(t, "ls-1", items[0].Id)
	assert.Equal(t, "ls-2", items[1].Id)
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

	_, err = svc.Create(
		t.Context(),
		CreateRequest{Name: "Share", Type: ShareTypeAssetsNearLocation},
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "addressId is required")
}

func TestDeleteUsesExpectedStatus(t *testing.T) {
	t.Parallel()

	svc := NewService(
		&httpxtest.MockRequester{DoFunc: func(_ context.Context, req httpx.Request) error {
			assert.Equal(t, http.MethodDelete, req.Method)
			assert.Equal(t, "/live-shares", req.Path)
			require.Equal(t, []int{http.StatusNoContent}, req.ExpectedStatus)
			assert.Equal(t, "abc", req.Query.Get("id"))
			return nil
		}},
	)

	err := svc.Delete(t.Context(), "abc")
	require.NoError(t, err)
}
