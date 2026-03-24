package forms

import (
	"context"
	"net/http"
	"strings"

	"github.com/emoss08/trenova/shared/samsara/internal/httpx"
)

type Service interface {
	ListTemplates(ctx context.Context, params TemplateListParams) (TemplateListResponse, error)
	ListSubmissions(
		ctx context.Context,
		params SubmissionListParams,
	) (SubmissionListResponse, error)
	CreateSubmission(ctx context.Context, req CreateSubmissionRequest) (FormSubmission, error)
	UpdateSubmission(ctx context.Context, req UpdateSubmissionRequest) (FormSubmission, error)
}

type service struct {
	client httpx.Requester
}

func NewService(client httpx.Requester) Service {
	return &service{client: client}
}

func (s *service) ListTemplates(
	ctx context.Context,
	params TemplateListParams,
) (TemplateListResponse, error) {
	out := TemplateListResponse{}
	if err := s.client.Do(ctx, httpx.Request{
		Method: http.MethodGet,
		Path:   "/form-templates",
		Query:  params.Query(),
		Out:    &out,
	}); err != nil {
		return TemplateListResponse{}, err
	}
	return out, nil
}

func (s *service) ListSubmissions(
	ctx context.Context,
	params SubmissionListParams,
) (SubmissionListResponse, error) {
	out := SubmissionListResponse{}
	if err := s.client.Do(ctx, httpx.Request{
		Method: http.MethodGet,
		Path:   "/form-submissions",
		Query:  params.Query(),
		Out:    &out,
	}); err != nil {
		return SubmissionListResponse{}, err
	}
	return out, nil
}

//nolint:gocritic // request is copied intentionally to keep create validation side-effect free.
func (s *service) CreateSubmission(
	ctx context.Context,
	req CreateSubmissionRequest,
) (FormSubmission, error) {
	out := createSubmissionResponse{}
	if err := s.client.Do(ctx, httpx.Request{
		Method: http.MethodPost,
		Path:   "/form-submissions",
		Body:   req,
		Out:    &out,
	}); err != nil {
		return FormSubmission{}, err
	}
	return out.Data, nil
}

func (s *service) UpdateSubmission(
	ctx context.Context,
	req UpdateSubmissionRequest,
) (FormSubmission, error) {
	if strings.TrimSpace(req.Id) == "" {
		return FormSubmission{}, ErrSubmissionIDRequired
	}

	out := updateSubmissionResponse{}
	if err := s.client.Do(ctx, httpx.Request{
		Method: http.MethodPatch,
		Path:   "/form-submissions",
		Body:   req,
		Out:    &out,
	}); err != nil {
		return FormSubmission{}, err
	}
	return out.Data, nil
}
