package vehicles

import (
	"context"
	"net/http"
	"strings"

	"github.com/emoss08/trenova/shared/samsara/internal/httpx"
)

type Service interface {
	Stats(ctx context.Context, params StatsParams) (StatsResponse, error)
	StatsAll(ctx context.Context, params StatsParams) ([]StatsData, error)
}

type service struct {
	client httpx.Requester
}

func NewService(client httpx.Requester) Service {
	return &service{client: client}
}

//nolint:gocritic // params is intentionally passed by value.
func (s *service) Stats(ctx context.Context, params StatsParams) (StatsResponse, error) {
	if err := params.Validate(); err != nil {
		return StatsResponse{}, err
	}

	out := StatsResponse{}
	if err := s.client.Do(ctx, httpx.Request{
		Method: http.MethodGet,
		Path:   "/fleet/vehicles/stats",
		Query:  params.Query(),
		Out:    &out,
	}); err != nil {
		return StatsResponse{}, err
	}
	return out, nil
}

//nolint:gocritic // params is intentionally passed by value.
func (s *service) StatsAll(ctx context.Context, params StatsParams) ([]StatsData, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}
	if params.Limit == 0 {
		params.Limit = 512
	}

	items := make([]StatsData, 0)
	for {
		page, err := s.Stats(ctx, params)
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
