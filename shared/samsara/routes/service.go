package routes

import (
	"context"
	"net/http"
	"strings"

	"github.com/emoss08/trenova/shared/samsara/internal/httpx"
)

type Service interface {
	List(ctx context.Context, params ListParams) (ListResponse, error)
	ListAll(ctx context.Context, params ListParams) ([]Route, error)
	Get(ctx context.Context, id string) (Route, error)
	Create(ctx context.Context, req CreateRequest) (Route, error)
	Update(ctx context.Context, id string, req UpdateRequest) (Route, error)
	Delete(ctx context.Context, id string) error
}

type service struct {
	client httpx.Requester
}

func NewService(client httpx.Requester) Service {
	return &service{client: client}
}

//nolint:gocritic // params is intentionally passed by value.
func (s *service) List(ctx context.Context, params ListParams) (ListResponse, error) {
	if err := params.Validate(); err != nil {
		return ListResponse{}, err
	}

	out := ListResponse{}
	if err := s.client.Do(ctx, httpx.Request{
		Method: http.MethodGet,
		Path:   "/fleet/routes",
		Query:  params.Query(),
		Out:    &out,
	}); err != nil {
		return ListResponse{}, err
	}
	return out, nil
}

//nolint:gocritic // params is intentionally passed by value.
func (s *service) ListAll(ctx context.Context, params ListParams) ([]Route, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}
	if params.Limit == 0 {
		params.Limit = maxListLimit
	}

	items := make([]Route, 0)
	for {
		page, err := s.List(ctx, params)
		if err != nil {
			return nil, err
		}
		items = append(items, page.Data...)
		if !page.Pagination.HasNextPage || strings.TrimSpace(page.Pagination.EndCursor) == "" {
			break
		}
		params.After = page.Pagination.EndCursor
	}
	return items, nil
}

func (s *service) Get(ctx context.Context, id string) (Route, error) {
	routeID := strings.TrimSpace(id)
	if routeID == "" {
		return Route{}, ErrRouteIDRequired
	}

	out := routeResponse{}
	if err := s.client.Do(ctx, httpx.Request{
		Method: http.MethodGet,
		Path:   httpx.PathWithID("/fleet/routes", routeID),
		Out:    &out,
	}); err != nil {
		return Route{}, err
	}
	if out.Data == nil {
		return Route{}, nil
	}
	return *out.Data, nil
}

//nolint:gocritic // request is copied intentionally to keep create validation side-effect free.
func (s *service) Create(ctx context.Context, req CreateRequest) (Route, error) {
	if err := ValidateCreateRequest(req); err != nil {
		return Route{}, err
	}

	out := createResponse{}
	if err := s.client.Do(ctx, httpx.Request{
		Method: http.MethodPost,
		Path:   "/fleet/routes",
		Body:   req,
		Out:    &out,
	}); err != nil {
		return Route{}, err
	}
	if out.Data == nil {
		return Route{}, nil
	}
	return *out.Data, nil
}

func (s *service) Update(ctx context.Context, id string, req UpdateRequest) (Route, error) {
	routeID := strings.TrimSpace(id)
	if routeID == "" {
		return Route{}, ErrRouteIDRequired
	}
	if err := ValidateUpdateRequest(req); err != nil {
		return Route{}, err
	}

	out := updateResponse{}
	if err := s.client.Do(ctx, httpx.Request{
		Method: http.MethodPatch,
		Path:   httpx.PathWithID("/fleet/routes", routeID),
		Body:   req,
		Out:    &out,
	}); err != nil {
		return Route{}, err
	}
	if out.Data == nil {
		return Route{}, nil
	}
	return *out.Data, nil
}

func (s *service) Delete(ctx context.Context, id string) error {
	routeID := strings.TrimSpace(id)
	if routeID == "" {
		return ErrRouteIDRequired
	}

	return s.client.Do(ctx, httpx.Request{
		Method:         http.MethodDelete,
		Path:           httpx.PathWithID("/fleet/routes", routeID),
		ExpectedStatus: []int{http.StatusNoContent},
	})
}
