package liveshares

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/emoss08/trenova/shared/samsara/internal/httpx"
)

type Service interface {
	List(ctx context.Context, params ListParams) (ListPage, error)
	ListAll(ctx context.Context, params ListParams) ([]LiveShare, error)
	Create(ctx context.Context, req CreateRequest) (LiveShare, error)
	Update(ctx context.Context, id string, req UpdateRequest) (LiveShare, error)
	Delete(ctx context.Context, id string) error
}

type service struct {
	client httpx.Requester
}

func NewService(client httpx.Requester) Service {
	return &service{client: client}
}

func (s *service) List(ctx context.Context, params ListParams) (ListPage, error) {
	if err := params.Validate(); err != nil {
		return ListPage{}, err
	}

	out := ListPage{}
	if err := s.client.Do(ctx, httpx.Request{
		Method: http.MethodGet,
		Path:   "/live-shares",
		Query:  params.Query(),
		Out:    &out,
	}); err != nil {
		return ListPage{}, err
	}

	return out, nil
}

func (s *service) ListAll(ctx context.Context, params ListParams) ([]LiveShare, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}
	if params.Limit == 0 {
		params.Limit = 100
	}

	shares := make([]LiveShare, 0)
	for {
		page, err := s.List(ctx, params)
		if err != nil {
			return nil, err
		}
		shares = append(shares, page.Data...)
		if !page.Pagination.HasNextPage || strings.TrimSpace(page.Pagination.EndCursor) == "" {
			break
		}
		params.After = page.Pagination.EndCursor
	}

	return shares, nil
}

func (s *service) Create(
	ctx context.Context,
	req CreateRequest,
) (LiveShare, error) {
	if err := validateCreateRequest(req); err != nil {
		return LiveShare{}, err
	}

	out := createResponse{}
	if err := s.client.Do(ctx, httpx.Request{
		Method: http.MethodPost,
		Path:   "/live-shares",
		Body:   req,
		Out:    &out,
	}); err != nil {
		return LiveShare{}, err
	}
	return out.Data, nil
}

func (s *service) Update(ctx context.Context, id string, req UpdateRequest) (LiveShare, error) {
	if strings.TrimSpace(id) == "" {
		return LiveShare{}, ErrIDRequired
	}
	if strings.TrimSpace(req.Name) == "" {
		return LiveShare{}, ErrNameRequired
	}

	out := updateResponse{}
	if err := s.client.Do(ctx, httpx.Request{
		Method: http.MethodPatch,
		Path:   "/live-shares",
		Query:  paramsWithID(id),
		Body:   req,
		Out:    &out,
	}); err != nil {
		return LiveShare{}, err
	}
	return out.Data, nil
}

func (s *service) Delete(ctx context.Context, id string) error {
	if strings.TrimSpace(id) == "" {
		return ErrIDRequired
	}

	return s.client.Do(ctx, httpx.Request{
		Method:         http.MethodDelete,
		Path:           "/live-shares",
		Query:          paramsWithID(id),
		ExpectedStatus: []int{http.StatusNoContent},
	})
}

func validateCreateRequest(
	req CreateRequest,
) error {
	if strings.TrimSpace(req.Name) == "" {
		return ErrNameRequired
	}
	switch req.Type {
	case ShareTypeAssetsLocation:
		if req.AssetsLocationLinkConfig == nil {
			return ErrAssetsLocationConfigRequired
		}
	case ShareTypeAssetsNearLocation:
		if req.AssetsNearLocationLinkConfig == nil ||
			strings.TrimSpace(req.AssetsNearLocationLinkConfig.AddressId) == "" {
			return ErrAssetsNearLocationAddressRequired
		}
	case ShareTypeAssetsOnRoute:
		if req.AssetsOnRouteLinkConfig == nil ||
			strings.TrimSpace(req.AssetsOnRouteLinkConfig.RecurringRouteId) == "" {
			return ErrRecurringRouteIDRequired
		}
	default:
		return fmt.Errorf("invalid live share type: %s", req.Type)
	}
	return nil
}

func paramsWithID(id string) url.Values {
	values := url.Values{}
	values.Set("id", id)
	return values
}
