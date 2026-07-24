package forms

import (
	"context"
	"net/http"
	"strings"

	"github.com/emoss08/trenova/shared/samsara/internal/httpx"
)

const submissionsPath = "/form-submissions"

type Service interface {
	ListTemplates(ctx context.Context, params TemplateListParams) (TemplateListResponse, error)
	ListSubmissions(
		ctx context.Context,
		params SubmissionListParams,
	) (SubmissionListResponse, error)
	StreamSubmissions(
		ctx context.Context,
		params SubmissionStreamParams,
	) (SubmissionStreamResponse, error)
	StreamSubmissionsAll(
		ctx context.Context,
		params SubmissionStreamParams,
	) ([]FormSubmission, error)
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
		Path:   submissionsPath,
		Query:  params.Query(),
		Out:    &out,
	}); err != nil {
		return SubmissionListResponse{}, err
	}
	return out, nil
}

//nolint:gocritic // params is intentionally passed by value.
func (s *service) StreamSubmissions(
	ctx context.Context,
	params SubmissionStreamParams,
) (SubmissionStreamResponse, error) {
	if err := params.Validate(); err != nil {
		return SubmissionStreamResponse{}, err
	}

	out := SubmissionStreamResponse{}
	if err := s.client.Do(ctx, httpx.Request{
		Method: http.MethodGet,
		Path:   "/form-submissions/stream",
		Query:  params.Query(),
		Out:    &out,
	}); err != nil {
		return SubmissionStreamResponse{}, err
	}
	return out, nil
}

//nolint:gocritic // params is intentionally passed by value.
func (s *service) StreamSubmissionsAll(
	ctx context.Context,
	params SubmissionStreamParams,
) ([]FormSubmission, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}

	items := make([]FormSubmission, 0)
	for {
		page, err := s.StreamSubmissions(ctx, params)
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

//nolint:gocritic // request is copied intentionally to keep create validation side-effect free.
func (s *service) CreateSubmission(
	ctx context.Context,
	req CreateSubmissionRequest,
) (FormSubmission, error) {
	out := createSubmissionResponse{}
	if err := s.client.Do(ctx, httpx.Request{
		Method: http.MethodPost,
		Path:   submissionsPath,
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
		Path:   submissionsPath,
		Body:   req,
		Out:    &out,
	}); err != nil {
		return FormSubmission{}, err
	}
	return out.Data, nil
}
