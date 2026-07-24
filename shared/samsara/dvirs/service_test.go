package dvirs

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

func TestStreamValidation(t *testing.T) {
	t.Parallel()

	svc := NewService(
		&httpxtest.MockRequester{DoFunc: func(_ context.Context, _ httpx.Request) error {
			return nil
		}},
	)

	start := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	tests := []struct {
		name    string
		params  StreamParams
		wantErr error
	}{
		{
			name:    "missing start time",
			params:  StreamParams{},
			wantErr: ErrStartTimeRequired,
		},
		{
			name:    "limit too high",
			params:  StreamParams{StartTime: &start, Limit: 201},
			wantErr: ErrStreamLimitInvalid,
		},
		{
			name:    "limit negative",
			params:  StreamParams{StartTime: &start, Limit: -1},
			wantErr: ErrStreamLimitInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := svc.Stream(t.Context(), tt.params)
			require.Error(t, err)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestStreamQueryAndPath(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	end := time.Date(2026, 1, 3, 3, 4, 5, 0, time.UTC)

	svc := NewService(
		&httpxtest.MockRequester{DoFunc: func(_ context.Context, req httpx.Request) error {
			assert.Equal(t, http.MethodGet, req.Method)
			assert.Equal(t, "/dvirs/stream", req.Path)
			assert.Equal(t, "2026-01-02T03:04:05Z", req.Query.Get("startTime"))
			assert.Equal(t, "2026-01-03T03:04:05Z", req.Query.Get("endTime"))
			assert.Equal(t, "unsafe,resolved", req.Query.Get("safetyStatus"))
			assert.Equal(t, "cursor-1", req.Query.Get("after"))
			assert.Equal(t, "25", req.Query.Get("limit"))
			assert.Equal(t, "true", req.Query.Get("includeExternalIds"))
			return nil
		}},
	)

	_, err := svc.Stream(t.Context(), StreamParams{
		StartTime:          &start,
		EndTime:            &end,
		SafetyStatuses:     []string{"unsafe", "resolved"},
		After:              "cursor-1",
		Limit:              25,
		IncludeExternalIDs: true,
	})
	require.NoError(t, err)
}

func TestStreamAllPaginates(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	calls := 0
	svc := NewService(
		&httpxtest.MockRequester{DoFunc: func(_ context.Context, req httpx.Request) error {
			calls++
			out := req.Out.(*StreamResponse)
			if calls == 1 {
				assert.Equal(t, "200", req.Query.Get("limit"))
				*out = StreamResponse{
					Data: []DVIR{{Id: "d-1"}, {Id: "d-2"}},
					Pagination: samsaraspec.GoaPaginationResponseResponseBody{
						EndCursor:   "n1",
						HasNextPage: true,
					},
				}
				return nil
			}

			assert.Equal(t, "n1", req.Query.Get("after"))
			*out = StreamResponse{
				Data: []DVIR{{Id: "d-3"}},
				Pagination: samsaraspec.GoaPaginationResponseResponseBody{
					EndCursor:   "",
					HasNextPage: false,
				},
			}
			return nil
		}},
	)

	items, err := svc.StreamAll(t.Context(), StreamParams{StartTime: &start})
	require.NoError(t, err)
	assert.Len(t, items, 3)
	assert.Equal(t, 2, calls)
	assert.Equal(t, "d-3", items[2].Id)
}

func TestStreamAllValidation(t *testing.T) {
	t.Parallel()

	svc := NewService(
		&httpxtest.MockRequester{DoFunc: func(_ context.Context, _ httpx.Request) error {
			return nil
		}},
	)

	_, err := svc.StreamAll(t.Context(), StreamParams{})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrStartTimeRequired)
}

func TestGetValidation(t *testing.T) {
	t.Parallel()

	svc := NewService(
		&httpxtest.MockRequester{DoFunc: func(_ context.Context, _ httpx.Request) error {
			return nil
		}},
	)

	_, err := svc.Get(t.Context(), "   ", GetParams{})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrIDRequired)
}

func TestGetQueryAndPath(t *testing.T) {
	t.Parallel()

	svc := NewService(
		&httpxtest.MockRequester{DoFunc: func(_ context.Context, req httpx.Request) error {
			assert.Equal(t, http.MethodGet, req.Method)
			assert.Equal(t, "/dvirs/dvir-123", req.Path)
			assert.Equal(t, "true", req.Query.Get("includeExternalIds"))

			out := req.Out.(*DVIRDetail)
			*out = DVIRDetail{Id: "dvir-123"}
			return nil
		}},
	)

	dvir, err := svc.Get(t.Context(), "dvir-123", GetParams{IncludeExternalIDs: true})
	require.NoError(t, err)
	assert.Equal(t, "dvir-123", dvir.Id)
}

func TestHistoryValidation(t *testing.T) {
	t.Parallel()

	svc := NewService(
		&httpxtest.MockRequester{DoFunc: func(_ context.Context, _ httpx.Request) error {
			return nil
		}},
	)

	start := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	end := time.Date(2026, 1, 3, 3, 4, 5, 0, time.UTC)

	tests := []struct {
		name    string
		params  HistoryParams
		wantErr error
	}{
		{
			name:    "missing start time",
			params:  HistoryParams{EndTime: &end},
			wantErr: ErrStartTimeRequired,
		},
		{
			name:    "missing end time",
			params:  HistoryParams{StartTime: &start},
			wantErr: ErrEndTimeRequired,
		},
		{
			name:    "limit too high",
			params:  HistoryParams{StartTime: &start, EndTime: &end, Limit: 513},
			wantErr: ErrHistoryLimitInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := svc.History(t.Context(), tt.params)
			require.Error(t, err)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestHistoryQueryAndPath(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	end := time.Date(2026, 1, 3, 3, 4, 5, 0, time.UTC)

	svc := NewService(
		&httpxtest.MockRequester{DoFunc: func(_ context.Context, req httpx.Request) error {
			assert.Equal(t, http.MethodGet, req.Method)
			assert.Equal(t, "/fleet/dvirs/history", req.Path)
			assert.Equal(t, "2026-01-02T03:04:05Z", req.Query.Get("startTime"))
			assert.Equal(t, "2026-01-03T03:04:05Z", req.Query.Get("endTime"))
			assert.Equal(t, "tag-1,tag-2", req.Query.Get("tagIds"))
			assert.Equal(t, "ptag-1", req.Query.Get("parentTagIds"))
			assert.Equal(t, "cursor-2", req.Query.Get("after"))
			assert.Equal(t, "100", req.Query.Get("limit"))
			return nil
		}},
	)

	_, err := svc.History(t.Context(), HistoryParams{
		StartTime:    &start,
		EndTime:      &end,
		TagIDs:       []string{"tag-1", "tag-2"},
		ParentTagIDs: []string{"ptag-1"},
		After:        "cursor-2",
		Limit:        100,
	})
	require.NoError(t, err)
}

func TestHistoryAllPaginates(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	end := time.Date(2026, 1, 3, 3, 4, 5, 0, time.UTC)
	calls := 0
	svc := NewService(
		&httpxtest.MockRequester{DoFunc: func(_ context.Context, req httpx.Request) error {
			calls++
			out := req.Out.(*HistoryResponse)
			if calls == 1 {
				assert.Equal(t, "512", req.Query.Get("limit"))
				*out = HistoryResponse{
					Data: []HistoryDVIR{{Id: "h-1"}, {Id: "h-2"}},
					Pagination: samsaraspec.PaginationResponse{
						EndCursor:   "n1",
						HasNextPage: true,
					},
				}
				return nil
			}

			assert.Equal(t, "n1", req.Query.Get("after"))
			*out = HistoryResponse{
				Data: []HistoryDVIR{{Id: "h-3"}},
				Pagination: samsaraspec.PaginationResponse{
					EndCursor:   "",
					HasNextPage: false,
				},
			}
			return nil
		}},
	)

	items, err := svc.HistoryAll(t.Context(), HistoryParams{StartTime: &start, EndTime: &end})
	require.NoError(t, err)
	assert.Len(t, items, 3)
	assert.Equal(t, 2, calls)
	assert.Equal(t, "h-3", items[2].Id)
}

func TestHistoryAllValidation(t *testing.T) {
	t.Parallel()

	svc := NewService(
		&httpxtest.MockRequester{DoFunc: func(_ context.Context, _ httpx.Request) error {
			return nil
		}},
	)

	start := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	_, err := svc.HistoryAll(t.Context(), HistoryParams{StartTime: &start})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrEndTimeRequired)
}
