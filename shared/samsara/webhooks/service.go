package webhooks

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/emoss08/trenova/shared/samsara/internal/httpx"
)

type Service interface {
	List(ctx context.Context, params ListParams) (ListResponse, error)
	Get(ctx context.Context, id string) (Webhook, error)
	Create(ctx context.Context, req CreateRequest) (Webhook, error)
	Update(ctx context.Context, id string, req UpdateRequest) (Webhook, error)
	Delete(ctx context.Context, id string) error
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
		Path:   "/webhooks",
		Query:  params.Query(),
		Out:    &out,
	}); err != nil {
		return ListResponse{}, err
	}
	return out, nil
}

func (s *service) Get(ctx context.Context, id string) (Webhook, error) {
	webhookID := strings.TrimSpace(id)
	if webhookID == "" {
		return Webhook{}, ErrWebhookIDRequired
	}

	out := Webhook{}
	if err := s.client.Do(ctx, httpx.Request{
		Method: http.MethodGet,
		Path:   fmt.Sprintf("/webhooks/%s", webhookID),
		Out:    &out,
	}); err != nil {
		return Webhook{}, err
	}
	return out, nil
}

func (s *service) Create(ctx context.Context, req CreateRequest) (Webhook, error) {
	if strings.TrimSpace(req.Name) == "" {
		return Webhook{}, ErrWebhookNameRequired
	}
	if strings.TrimSpace(req.Url) == "" {
		return Webhook{}, ErrWebhookURLRequired
	}

	out := Webhook{}
	if err := s.client.Do(ctx, httpx.Request{
		Method: http.MethodPost,
		Path:   "/webhooks",
		Body:   req,
		Out:    &out,
	}); err != nil {
		return Webhook{}, err
	}
	return out, nil
}

func (s *service) Update(ctx context.Context, id string, req UpdateRequest) (Webhook, error) {
	webhookID := strings.TrimSpace(id)
	if webhookID == "" {
		return Webhook{}, ErrWebhookIDRequired
	}

	out := Webhook{}
	if err := s.client.Do(ctx, httpx.Request{
		Method: http.MethodPatch,
		Path:   fmt.Sprintf("/webhooks/%s", webhookID),
		Body:   req,
		Out:    &out,
	}); err != nil {
		return Webhook{}, err
	}
	return out, nil
}

func (s *service) Delete(ctx context.Context, id string) error {
	webhookID := strings.TrimSpace(id)
	if webhookID == "" {
		return ErrWebhookIDRequired
	}

	return s.client.Do(ctx, httpx.Request{
		Method:         http.MethodDelete,
		Path:           fmt.Sprintf("/webhooks/%s", webhookID),
		ExpectedStatus: []int{http.StatusNoContent},
	})
}
