package forms

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/emoss08/trenova/shared/samsara/internal/httpx"
	"github.com/emoss08/trenova/shared/samsara/internal/httpxtest"
	samsaraspec "github.com/emoss08/trenova/shared/samsara/internal/samsaraspec"
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

func TestStreamSubmissionsValidation(t *testing.T) {
	t.Parallel()

	svc := NewService(
		&httpxtest.MockRequester{DoFunc: func(_ context.Context, _ httpx.Request) error {
			return nil
		}},
	)

	_, err := svc.StreamSubmissions(t.Context(), SubmissionStreamParams{})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrStreamStartTimeRequired)

	_, err = svc.StreamSubmissionsAll(t.Context(), SubmissionStreamParams{})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrStreamStartTimeRequired)
}

func TestStreamSubmissionsQueryAndPath(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	end := time.Date(2026, 1, 3, 3, 4, 5, 0, time.UTC)

	svc := NewService(
		&httpxtest.MockRequester{DoFunc: func(_ context.Context, req httpx.Request) error {
			assert.Equal(t, http.MethodGet, req.Method)
			assert.Equal(t, "/form-submissions/stream", req.Path)
			assert.Equal(t, "2026-01-02T03:04:05Z", req.Query.Get("startTime"))
			assert.Equal(t, "2026-01-03T03:04:05Z", req.Query.Get("endTime"))
			assert.Equal(t, "ft-1,ft-2", req.Query.Get("formTemplateIds"))
			assert.Equal(t, "user-1", req.Query.Get("userIds"))
			assert.Equal(t, "driver-1", req.Query.Get("driverIds"))
			assert.Equal(t, "fields,assignedTo", req.Query.Get("include"))
			assert.Equal(t, "stop-1", req.Query.Get("assignedToRouteStopIds"))
			assert.Equal(t, "cursor-1", req.Query.Get("after"))
			return nil
		}},
	)

	_, err := svc.StreamSubmissions(t.Context(), SubmissionStreamParams{
		StartTime:              &start,
		EndTime:                &end,
		FormTemplateIDs:        []string{"ft-1", "ft-2"},
		UserIDs:                []string{"user-1"},
		DriverIDs:              []string{"driver-1"},
		Include:                []string{"fields", "assignedTo"},
		AssignedToRouteStopIDs: []string{"stop-1"},
		After:                  "cursor-1",
	})
	require.NoError(t, err)
}

func TestStreamSubmissionsAllPaginates(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	calls := 0
	svc := NewService(
		&httpxtest.MockRequester{DoFunc: func(_ context.Context, req httpx.Request) error {
			calls++
			out := req.Out.(*SubmissionStreamResponse)
			if calls == 1 {
				*out = SubmissionStreamResponse{
					Data: []FormSubmission{{Id: "sub-1"}, {Id: "sub-2"}},
					Pagination: samsaraspec.GoaPaginationResponseResponseBody{
						EndCursor:   "n1",
						HasNextPage: true,
					},
				}
				return nil
			}

			assert.Equal(t, "n1", req.Query.Get("after"))
			*out = SubmissionStreamResponse{
				Data: []FormSubmission{{Id: "sub-3"}},
				Pagination: samsaraspec.GoaPaginationResponseResponseBody{
					EndCursor:   "",
					HasNextPage: false,
				},
			}
			return nil
		}},
	)

	items, err := svc.StreamSubmissionsAll(t.Context(), SubmissionStreamParams{StartTime: &start})
	require.NoError(t, err)
	assert.Len(t, items, 3)
	assert.Equal(t, 2, calls)
	assert.Equal(t, "sub-3", items[2].Id)
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
