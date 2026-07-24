package dvirs

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/emoss08/trenova/shared/samsara/internal/httpx"
)

type Service interface {
	Stream(ctx context.Context, params StreamParams) (StreamResponse, error)
	StreamAll(ctx context.Context, params StreamParams) ([]DVIR, error)
	Get(ctx context.Context, id string, params GetParams) (DVIRDetail, error)
	History(ctx context.Context, params HistoryParams) (HistoryResponse, error)
	HistoryAll(ctx context.Context, params HistoryParams) ([]HistoryDVIR, error)
}

type service struct {
	client httpx.Requester
}

func NewService(client httpx.Requester) Service {
	return &service{client: client}
}

//nolint:gocritic // params is intentionally passed by value.
func (s *service) Stream(ctx context.Context, params StreamParams) (StreamResponse, error) {
	if err := params.Validate(); err != nil {
		return StreamResponse{}, err
	}

	out := StreamResponse{}
	if err := s.client.Do(ctx, httpx.Request{
		Method: http.MethodGet,
		Path:   "/dvirs/stream",
		Query:  params.Query(),
		Out:    &out,
	}); err != nil {
		return StreamResponse{}, err
	}
	return out, nil
}

//nolint:gocritic // params is intentionally passed by value.
func (s *service) StreamAll(ctx context.Context, params StreamParams) ([]DVIR, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}
	if params.Limit == 0 {
		params.Limit = 200
	}

	items := make([]DVIR, 0)
	for {
		page, err := s.Stream(ctx, params)
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

//nolint:gocritic // params is intentionally passed by value.
func (s *service) Get(ctx context.Context, id string, params GetParams) (DVIRDetail, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return DVIRDetail{}, ErrIDRequired
	}

	out := DVIRDetail{}
	if err := s.client.Do(ctx, httpx.Request{
		Method: http.MethodGet,
		Path:   fmt.Sprintf("/dvirs/%s", id),
		Query:  params.Query(),
		Out:    &out,
	}); err != nil {
		return DVIRDetail{}, err
	}
	return out, nil
}

//nolint:gocritic // params is intentionally passed by value.
func (s *service) History(ctx context.Context, params HistoryParams) (HistoryResponse, error) {
	if err := params.Validate(); err != nil {
		return HistoryResponse{}, err
	}

	out := HistoryResponse{}
	if err := s.client.Do(ctx, httpx.Request{
		Method: http.MethodGet,
		Path:   "/fleet/dvirs/history",
		Query:  params.Query(),
		Out:    &out,
	}); err != nil {
		return HistoryResponse{}, err
	}
	return out, nil
}

//nolint:gocritic // params is intentionally passed by value.
func (s *service) HistoryAll(ctx context.Context, params HistoryParams) ([]HistoryDVIR, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}
	if params.Limit == 0 {
		params.Limit = 512
	}

	items := make([]HistoryDVIR, 0)
	for {
		page, err := s.History(ctx, params)
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
