package forms

import (
	"context"
	"net/http"
	"testing"

	"github.com/emoss08/trenova/shared/samsara/internal/httpx"
	"github.com/emoss08/trenova/shared/samsara/internal/httpxtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListTemplatesQueryAndPath(t *testing.T) {
	t.Parallel()

	svc := NewService(
		&httpxtest.MockRequester{DoFunc: func(_ context.Context, req httpx.Request) error {
			assert.Equal(t, http.MethodGet, req.Method)
			assert.Equal(t, "/form-templates", req.Path)
			assert.Equal(t, "ft-1,ft-2", req.Query.Get("ids"))
			assert.Equal(t, "cursor-1", req.Query.Get("after"))
			return nil
		}},
	)

	_, err := svc.ListTemplates(t.Context(), TemplateListParams{
		IDs:   []string{"ft-1", "ft-2"},
		After: "cursor-1",
	})
	require.NoError(t, err)
}

func TestListSubmissionsQueryAndPath(t *testing.T) {
	t.Parallel()

	svc := NewService(
		&httpxtest.MockRequester{DoFunc: func(_ context.Context, req httpx.Request) error {
			assert.Equal(t, http.MethodGet, req.Method)
			assert.Equal(t, "/form-submissions", req.Path)
			assert.Equal(t, "sub-1", req.Query.Get("ids"))
			assert.Equal(t, "fields,assignedTo", req.Query.Get("include"))
			return nil
		}},
	)

	_, err := svc.ListSubmissions(t.Context(), SubmissionListParams{
		IDs:     []string{"sub-1"},
		Include: []string{"fields", "assignedTo"},
	})
	require.NoError(t, err)
}

func TestCreateSubmissionPath(t *testing.T) {
	t.Parallel()

	svc := NewService(
		&httpxtest.MockRequester{DoFunc: func(_ context.Context, req httpx.Request) error {
			assert.Equal(t, http.MethodPost, req.Method)
			assert.Equal(t, "/form-submissions", req.Path)
			out := req.Out.(*createSubmissionResponse)
			*out = createSubmissionResponse{Data: FormSubmission{}}
			return nil
		}},
	)

	_, err := svc.CreateSubmission(t.Context(), CreateSubmissionRequest{})
	require.NoError(t, err)
}

func TestUpdateSubmissionValidation(t *testing.T) {
	t.Parallel()

	svc := NewService(
		&httpxtest.MockRequester{DoFunc: func(_ context.Context, _ httpx.Request) error {
			return nil
		}},
	)

	_, err := svc.UpdateSubmission(t.Context(), UpdateSubmissionRequest{})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrSubmissionIDRequired)
}

func TestUpdateSubmissionPath(t *testing.T) {
	t.Parallel()

	svc := NewService(
		&httpxtest.MockRequester{DoFunc: func(_ context.Context, req httpx.Request) error {
			assert.Equal(t, http.MethodPatch, req.Method)
			assert.Equal(t, "/form-submissions", req.Path)
			out := req.Out.(*updateSubmissionResponse)
			*out = updateSubmissionResponse{Data: FormSubmission{}}
			return nil
		}},
	)

	_, err := svc.UpdateSubmission(t.Context(), UpdateSubmissionRequest{Id: "sub-1"})
	require.NoError(t, err)
}
