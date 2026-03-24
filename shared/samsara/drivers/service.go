package drivers

import (
	"context"
	"net/http"
	"strings"

	"github.com/emoss08/trenova/shared/samsara/internal/httpx"
)

type Service interface {
	List(ctx context.Context, params ListParams) (ListResponse, error)
	ListAll(ctx context.Context, params ListParams) ([]Driver, error)
	Create(ctx context.Context, req CreateRequest) (Driver, error)
	Update(ctx context.Context, id string, req UpdateRequest) (Driver, error)
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
		Path:   "/fleet/drivers",
		Query:  params.Query(),
		Out:    &out,
	}); err != nil {
		return ListResponse{}, err
	}

	return out, nil
}

//nolint:gocritic // params is intentionally passed by value.
func (s *service) ListAll(ctx context.Context, params ListParams) ([]Driver, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}
	if params.Limit == 0 {
		params.Limit = 512
	}

	items := make([]Driver, 0)
	for {
		page, err := s.List(ctx, params)
		if err != nil {
			return nil, err
		}
		if page.Data != nil {
			items = append(items, *page.Data...)
		}
		if page.Pagination == nil ||
			!page.Pagination.HasNextPage ||
			strings.TrimSpace(page.Pagination.EndCursor) == "" {
			break
		}
		params.After = page.Pagination.EndCursor
	}
	return items, nil
}

//nolint:gocritic // request is copied intentionally to keep create validation side-effect free.
func (s *service) Create(ctx context.Context, req CreateRequest) (Driver, error) {
	if strings.TrimSpace(req.Name) == "" {
		return Driver{}, ErrDriverNameRequired
	}

	out := createResponse{}
	if err := s.client.Do(ctx, httpx.Request{
		Method: http.MethodPost,
		Path:   "/fleet/drivers",
		Body:   req,
		Out:    &out,
	}); err != nil {
		return Driver{}, err
	}
	if out.Data == nil {
		return Driver{}, nil
	}
	return *out.Data, nil
}

//nolint:gocritic // request is copied intentionally to keep update validation side-effect free.
func (s *service) Update(ctx context.Context, id string, req UpdateRequest) (Driver, error) {
	driverID := strings.TrimSpace(id)
	if driverID == "" {
		return Driver{}, ErrDriverIDRequired
	}

	out := updateResponse{}
	if err := s.client.Do(ctx, httpx.Request{
		Method: http.MethodPatch,
		Path:   "/fleet/drivers/" + driverID,
		Body:   req,
		Out:    &out,
	}); err != nil {
		return Driver{}, err
	}
	if out.Data == nil {
		return Driver{}, nil
	}
	return *out.Data, nil
}
