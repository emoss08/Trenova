package messages

import (
	"context"
	"net/http"
	"strings"

	"github.com/emoss08/trenova/shared/samsara/internal/httpx"
)

type Service interface {
	List(ctx context.Context, params ListParams) (ListResponse, error)
	Create(ctx context.Context, req CreateRequest) (CreateResponse, error)
}

type service struct {
	client httpx.Requester
}

func NewService(client httpx.Requester) Service {
	return &service{client: client}
}

func (s *service) List(ctx context.Context, params ListParams) (ListResponse, error) {
	if err := params.Validate(); err != nil {
		return ListResponse{}, err
	}

	out := ListResponse{}
	if err := s.client.Do(ctx, httpx.Request{
		Method: http.MethodGet,
		Path:   "/v1/fleet/messages",
		Query:  params.Query(),
		Out:    &out,
	}); err != nil {
		return ListResponse{}, err
	}
	return out, nil
}

func (s *service) Create(ctx context.Context, req CreateRequest) (CreateResponse, error) {
	if strings.TrimSpace(req.Text) == "" {
		return CreateResponse{}, ErrTextRequired
	}
	if len(req.DriverIds) == 0 {
		return CreateResponse{}, ErrDriverIDsRequired
	}

	out := CreateResponse{}
	if err := s.client.Do(ctx, httpx.Request{
		Method: http.MethodPost,
		Path:   "/v1/fleet/messages",
		Body:   req,
		Out:    &out,
	}); err != nil {
		return CreateResponse{}, err
	}
	return out, nil
}
