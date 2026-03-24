package drivers

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
}

func TestListQueryAndPath(t *testing.T) {
	t.Parallel()

	svc := NewService(
		&httpxtest.MockRequester{DoFunc: func(_ context.Context, req httpx.Request) error {
			assert.Equal(t, http.MethodGet, req.Method)
			assert.Equal(t, "/fleet/drivers", req.Path)
			assert.Equal(t, "active", req.Query.Get("driverActivationStatus"))
			assert.Equal(t, "abc", req.Query.Get("after"))
			assert.Equal(t, "one,two", req.Query.Get("attributeValueIds"))
			require.Equal(t, []string{"attr1", "attr2"}, req.Query["attributes"])
			return nil
		}},
	)

	_, err := svc.List(t.Context(), ListParams{
		DriverActivationStatus: "active",
		After:                  "abc",
		AttributeValueIDs:      []string{"one", "two"},
		Attributes:             []string{"attr1", "attr2"},
	})
	require.NoError(t, err)
}

func TestListAllPaginates(t *testing.T) {
	t.Parallel()

	calls := 0
	svc := NewService(
		&httpxtest.MockRequester{DoFunc: func(_ context.Context, req httpx.Request) error {
			calls++
			out := req.Out.(*ListResponse)
			if calls == 1 {
				cursor := "n1"
				data := []Driver{{}, {}}
				*out = ListResponse{
					Data: &data,
					Pagination: &samsaraspec.PaginationResponse{
						EndCursor:   cursor,
						HasNextPage: true,
					},
				}
				return nil
			}

			assert.Equal(t, "n1", req.Query.Get("after"))
			data := []Driver{{}}
			*out = ListResponse{
				Data:       &data,
				Pagination: &samsaraspec.PaginationResponse{EndCursor: "", HasNextPage: false},
			}
			return nil
		}},
	)

	items, err := svc.ListAll(t.Context(), ListParams{})
	require.NoError(t, err)
	assert.Len(t, items, 3)
	assert.Equal(t, 2, calls)
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
	assert.ErrorIs(t, err, ErrDriverNameRequired)
}

func TestCreatePathAndResponse(t *testing.T) {
	t.Parallel()

	name := "Driver 1"
	svc := NewService(
		&httpxtest.MockRequester{DoFunc: func(_ context.Context, req httpx.Request) error {
			assert.Equal(t, http.MethodPost, req.Method)
			assert.Equal(t, "/fleet/drivers", req.Path)

			out := req.Out.(*createResponse)
			nameValue := samsaraspec.DriverName(name)
			out.Data = &Driver{Name: &nameValue}
			return nil
		}},
	)

	driver, err := svc.Create(t.Context(), CreateRequest{Name: name})
	require.NoError(t, err)
	require.NotNil(t, driver.Name)
	assert.Equal(t, name, string(*driver.Name))
}

func TestUpdateValidation(t *testing.T) {
	t.Parallel()

	svc := NewService(
		&httpxtest.MockRequester{DoFunc: func(_ context.Context, _ httpx.Request) error {
			return nil
		}},
	)

	_, err := svc.Update(t.Context(), "   ", UpdateRequest{})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrDriverIDRequired)
}

func TestUpdatePathAndResponse(t *testing.T) {
	t.Parallel()

	driverID := "driver-123"
	name := "Driver Updated"
	svc := NewService(
		&httpxtest.MockRequester{DoFunc: func(_ context.Context, req httpx.Request) error {
			assert.Equal(t, http.MethodPatch, req.Method)
			assert.Equal(t, "/fleet/drivers/"+driverID, req.Path)

			out := req.Out.(*updateResponse)
			nameValue := samsaraspec.DriverName(name)
			out.Data = &Driver{Name: &nameValue}
			return nil
		}},
	)

	driver, err := svc.Update(t.Context(), driverID, UpdateRequest{})
	require.NoError(t, err)
	require.NotNil(t, driver.Name)
	assert.Equal(t, name, string(*driver.Name))
}
