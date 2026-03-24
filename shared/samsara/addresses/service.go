package addresses

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/emoss08/trenova/shared/samsara/internal/httpx"
)

type Service interface {
	List(ctx context.Context, params ListParams) (ListPage, error)
	ListAll(ctx context.Context, params ListParams) ([]Address, error)
	Create(ctx context.Context, req CreateRequest) (Address, error)
	Get(ctx context.Context, id string) (Address, error)
	Update(ctx context.Context, id string, req UpdateRequest) (Address, error)
	Delete(ctx context.Context, id string) error
}

type service struct {
	client httpx.Requester
}

func NewService(client httpx.Requester) Service {
	return &service{client: client}
}

//nolint:gocritic // params is intentionally passed by value.
func (s *service) List(
	ctx context.Context,
	params ListParams,
) (ListPage, error) {
	if err := params.Validate(); err != nil {
		return ListPage{}, err
	}

	out := ListPage{}
	if err := s.client.Do(ctx, httpx.Request{
		Method: http.MethodGet,
		Path:   "/addresses",
		Query:  params.Query(),
		Out:    &out,
	}); err != nil {
		return ListPage{}, err
	}

	return out, nil
}

//nolint:gocritic // params is intentionally passed by value.
func (s *service) ListAll(
	ctx context.Context,
	params ListParams,
) ([]Address, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}
	if params.Limit == 0 {
		params.Limit = 512
	}

	addresses := make([]Address, 0)
	for {
		page, err := s.List(ctx, params)
		if err != nil {
			return nil, err
		}
		addresses = append(addresses, page.Data...)
		if !page.Pagination.HasNextPage || strings.TrimSpace(page.Pagination.EndCursor) == "" {
			break
		}
		params.After = page.Pagination.EndCursor
	}
	return addresses, nil
}

//nolint:gocritic // request is copied intentionally to avoid caller mutation during validation.
func (s *service) Create(
	ctx context.Context,
	req CreateRequest,
) (Address, error) {
	if strings.TrimSpace(req.Name) == "" {
		return Address{}, ErrNameRequired
	}
	if strings.TrimSpace(req.FormattedAddress) == "" {
		return Address{}, ErrFormattedAddressRequired
	}
	if req.Geofence.Circle == nil && req.Geofence.Polygon == nil {
		return Address{}, ErrGeofenceRequired
	}
	if req.Geofence.Circle != nil && req.Geofence.Polygon != nil {
		return Address{}, ErrGeofenceMutuallyExclusive
	}
	if req.Geofence.Polygon != nil {
		if len(req.Geofence.Polygon.Vertices) < 3 || len(req.Geofence.Polygon.Vertices) > 40 {
			return Address{}, ErrGeofencePolygonVerticesBounds
		}
	}

	out := AddressResponse{}
	if err := s.client.Do(ctx, httpx.Request{
		Method: http.MethodPost,
		Path:   "/addresses",
		Body:   req,
		Out:    &out,
	}); err != nil {
		return Address{}, err
	}

	return out.Data, nil
}

func (s *service) Get(ctx context.Context, id string) (Address, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return Address{}, ErrIDRequired
	}

	out := AddressResponse{}
	if err := s.client.Do(ctx, httpx.Request{
		Method: http.MethodGet,
		Path:   fmt.Sprintf("/addresses/%s", id),
		Out:    &out,
	}); err != nil {
		return Address{}, err
	}
	return out.Data, nil
}

//nolint:gocritic // request is copied intentionally to keep update validation side-effect free.
func (s *service) Update(
	ctx context.Context,
	id string,
	req UpdateRequest,
) (Address, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return Address{}, ErrIDRequired
	}

	out := AddressResponse{}
	if err := s.client.Do(ctx, httpx.Request{
		Method: http.MethodPatch,
		Path:   fmt.Sprintf("/addresses/%s", id),
		Body:   req,
		Out:    &out,
	}); err != nil {
		return Address{}, err
	}
	return out.Data, nil
}

func (s *service) Delete(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return ErrIDRequired
	}

	return s.client.Do(ctx, httpx.Request{
		Method:         http.MethodDelete,
		Path:           fmt.Sprintf("/addresses/%s", id),
		ExpectedStatus: []int{http.StatusNoContent},
	})
}
