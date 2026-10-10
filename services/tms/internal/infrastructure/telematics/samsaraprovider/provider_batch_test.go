package samsaraprovider

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/ports/services"
	sharedsamsara "github.com/emoss08/trenova/shared/samsara"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	batchTemplateID = "8c0a7f4e-6d0b-4f7a-9d63-4f0d7a1b2c3d"
	batchRevisionID = "1f2e3d4c-5b6a-4978-8a9b-0c1d2e3f4a5b"
)

type recordedRequest struct {
	path      string
	driverIDs []string
	after     string
}

type requestRecorder struct {
	mu       sync.Mutex
	requests []recordedRequest
}

func (r *requestRecorder) record(req *http.Request) recordedRequest {
	query := req.URL.Query()
	entry := recordedRequest{path: req.URL.Path, after: query.Get("after")}
	if raw := query.Get("driverIds"); raw != "" {
		entry.driverIDs = strings.Split(raw, ",")
	}
	r.mu.Lock()
	r.requests = append(r.requests, entry)
	r.mu.Unlock()
	return entry
}

func (r *requestRecorder) forPath(path string) []recordedRequest {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]recordedRequest, 0, len(r.requests))
	for _, req := range r.requests {
		if req.path == path {
			out = append(out, req)
		}
	}
	return out
}

func batchDriverIDs(n int) []string {
	ids := make([]string, 0, n)
	for i := range n {
		ids = append(ids, fmt.Sprintf("drv-%04d", i))
	}
	return ids
}

func writeJSONValue(t *testing.T, w http.ResponseWriter, status int, value any) {
	t.Helper()
	body, err := sonic.Marshal(value)
	if !assert.NoError(t, err) {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	writeJSON(w, status, string(body))
}

func paginationBody(hasNext bool, cursor string) map[string]any {
	return map[string]any{"endCursor": cursor, "hasNextPage": hasNext}
}

func splitPage(driverIDs []string, after string) ([]string, map[string]any) {
	half := (len(driverIDs) + 1) / 2
	if after == "" {
		return driverIDs[:half], paginationBody(true, "page-2")
	}
	return driverIDs[half:], paginationBody(false, "")
}

func hosLogsPage(driverIDs []string, after string) map[string]any {
	pageDrivers, pagination := splitPage(driverIDs, after)
	data := make([]map[string]any, 0, len(pageDrivers)+1)
	for _, driverID := range pageDrivers {
		data = append(data, map[string]any{
			"driver": map[string]any{"id": driverID},
			"hosLogs": []map[string]any{{
				"hosStatusType": "driving",
				"logStartTime":  "2026-03-01T14:00:00Z",
				"remark":        "first " + driverID,
			}},
		})
	}
	if after != "" {
		data = append(data, map[string]any{
			"driver": map[string]any{"id": driverIDs[0]},
			"hosLogs": []map[string]any{{
				"hosStatusType": "offDuty",
				"logStartTime":  "2026-03-01T18:00:00Z",
				"remark":        "second " + driverIDs[0],
			}},
		})
	}
	return map[string]any{"data": data, "pagination": pagination}
}

func TestListHOSLogsChunksPaginatesAndAttributesPerDriver(t *testing.T) {
	t.Parallel()

	drivers := batchDriverIDs(250)
	recorder := new(requestRecorder)
	provider := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		req := recorder.record(r)
		if req.path != "/fleet/hos/logs" {
			writeJSON(w, http.StatusNotFound, `{"message":"not found"}`)
			return
		}
		assert.LessOrEqual(t, len(req.driverIDs), hosDriverIDsPerRequest)
		assert.NotEmpty(t, r.URL.Query().Get("startTime"))
		assert.NotEmpty(t, r.URL.Query().Get("endTime"))
		writeJSONValue(t, w, http.StatusOK, hosLogsPage(req.driverIDs, req.after))
	})

	input := slices.Concat([]string{"", "  ", drivers[7]}, drivers)
	logsByDriver, err := provider.ListHOSLogs(t.Context(), input, 1772300000, 1772400000)
	require.NoError(t, err)

	requests := recorder.forPath("/fleet/hos/logs")
	require.Len(t, requests, 6)
	requested := make([]string, 0, len(drivers))
	for _, req := range requests {
		if req.after == "" {
			requested = append(requested, req.driverIDs...)
			continue
		}
		assert.Equal(t, "page-2", req.after)
	}
	assert.ElementsMatch(t, drivers, requested)

	require.Len(t, logsByDriver, len(drivers))
	for _, driverID := range drivers {
		entries := logsByDriver[driverID]
		require.NotEmpty(t, entries, driverID)
		assert.Equal(t, "first "+driverID, entries[0].Remark)
		assert.Equal(t, "driving", entries[0].HosStatusType)
		assert.Equal(
			t,
			time.Date(2026, 3, 1, 14, 0, 0, 0, time.UTC).Unix(),
			entries[0].LogStartAt,
		)
	}

	chunkHeads := []string{drivers[0], drivers[100], drivers[200]}
	for _, driverID := range chunkHeads {
		entries := logsByDriver[driverID]
		require.Len(t, entries, 2, driverID)
		assert.Equal(t, "second "+driverID, entries[1].Remark)
	}
	assert.Len(t, logsByDriver[drivers[1]], 1)
}

func TestListHOSLogsReportsOnlyTheFailedChunk(t *testing.T) {
	t.Parallel()

	drivers := batchDriverIDs(250)
	failing := drivers[150]
	recorder := new(requestRecorder)
	provider := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		req := recorder.record(r)
		if slices.Contains(req.driverIDs, failing) {
			writeJSON(w, http.StatusBadRequest, `{"message":"bad request","requestId":"r1"}`)
			return
		}
		writeJSONValue(t, w, http.StatusOK, hosLogsPage(req.driverIDs, req.after))
	})

	logsByDriver, err := provider.ListHOSLogs(t.Context(), drivers, 1772300000, 1772400000)
	require.Error(t, err)

	batchErr, ok := errors.AsType[*services.ProviderDriverBatchError](err)
	require.True(t, ok)
	assert.Equal(t, drivers[100:200], batchErr.DriverIDs)
	apiErr, ok := errors.AsType[*sharedsamsara.APIError](err)
	require.True(t, ok)
	assert.Equal(t, http.StatusBadRequest, apiErr.StatusCode)

	assert.Len(t, logsByDriver, 150)
	assert.Contains(t, logsByDriver, drivers[0])
	assert.Contains(t, logsByDriver, drivers[249])
	assert.NotContains(t, logsByDriver, failing)
	assert.Len(t, recorder.forPath("/fleet/hos/logs"), 5)
}

func TestListHOSLogsWithoutDriversMakesNoRequest(t *testing.T) {
	t.Parallel()

	recorder := new(requestRecorder)
	provider := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		recorder.record(r)
		writeJSON(w, http.StatusOK, `{"data":[],"pagination":{"endCursor":"","hasNextPage":false}}`)
	})

	logsByDriver, err := provider.ListHOSLogs(t.Context(), []string{"", " "}, 1, 2)
	require.NoError(t, err)
	assert.Empty(t, logsByDriver)
	assert.Empty(t, recorder.forPath("/fleet/hos/logs"))
}

func TestListHOSLogsReturnsContextErrorWithoutPartialResults(t *testing.T) {
	t.Parallel()

	provider := newTestProvider(t, func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, `{"data":[],"pagination":{"endCursor":"","hasNextPage":false}}`)
	})

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	logsByDriver, err := provider.ListHOSLogs(ctx, batchDriverIDs(10), 1, 2)
	require.Error(t, err)
	assert.Nil(t, logsByDriver)
	_, isBatch := errors.AsType[*services.ProviderDriverBatchError](err)
	assert.False(t, isBatch)
}

func TestListHOSDailyLogsChunksPaginatesAndAttributesPerDriver(t *testing.T) {
	t.Parallel()

	drivers := batchDriverIDs(150)
	recorder := new(requestRecorder)
	provider := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		req := recorder.record(r)
		if req.path != "/fleet/hos/daily-logs" {
			writeJSON(w, http.StatusNotFound, `{"message":"not found"}`)
			return
		}
		assert.LessOrEqual(t, len(req.driverIDs), hosDriverIDsPerRequest)
		assert.Equal(t, "2026-03-01", r.URL.Query().Get("startDate"))
		assert.Equal(t, "2026-03-07", r.URL.Query().Get("endDate"))

		pageDrivers, pagination := splitPage(req.driverIDs, req.after)
		data := make([]map[string]any, 0, len(pageDrivers)*2)
		for _, driverID := range pageDrivers {
			for day, certified := range []bool{true, false} {
				data = append(data, map[string]any{
					"driver":      map[string]any{"id": driverID, "name": "Driver " + driverID},
					"startTime":   fmt.Sprintf("2026-03-0%dT00:00:00Z", day+1),
					"endTime":     fmt.Sprintf("2026-03-0%dT00:00:00Z", day+2),
					"logMetaData": map[string]any{"isCertified": certified},
				})
			}
		}
		writeJSONValue(t, w, http.StatusOK, map[string]any{"data": data, "pagination": pagination})
	})

	daysByDriver, err := provider.ListHOSDailyLogs(
		t.Context(),
		drivers,
		"2026-03-01",
		"2026-03-07",
	)
	require.NoError(t, err)

	requests := recorder.forPath("/fleet/hos/daily-logs")
	require.Len(t, requests, 4)
	assert.Len(t, requests[0].driverIDs, 100)
	assert.Equal(t, "page-2", requests[1].after)

	require.Len(t, daysByDriver, len(drivers))
	for _, driverID := range drivers {
		days := daysByDriver[driverID]
		require.Len(t, days, 2, driverID)
		assert.True(t, days[0].IsCertified)
		assert.False(t, days[1].IsCertified)
		assert.Equal(
			t,
			time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC).Unix(),
			days[0].StartAt,
		)
	}
}

func TestListFormSubmissionsChunksToSpecLimitAndAttributesPerDriver(t *testing.T) {
	t.Parallel()

	drivers := batchDriverIDs(120)
	recorder := new(requestRecorder)
	provider := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		req := recorder.record(r)
		switch req.path {
		case "/form-templates":
			writeJSONValue(t, w, http.StatusOK, map[string]any{
				"data": []map[string]any{{
					"id":    batchTemplateID,
					"title": "Proof of delivery",
				}},
				"pagination": paginationBody(false, ""),
			})
		case "/form-submissions/stream":
			assert.LessOrEqual(t, len(req.driverIDs), formDriverIDsPerRequest)
			assert.Equal(t, "externalIds", r.URL.Query().Get("include"))

			pageDrivers, pagination := splitPage(req.driverIDs, req.after)
			data := make([]map[string]any, 0, len(pageDrivers))
			for _, driverID := range pageDrivers {
				data = append(data, map[string]any{
					"id": "sub-" + driverID,
					"formTemplate": map[string]any{
						"id":         batchTemplateID,
						"revisionId": batchRevisionID,
					},
					"submittedBy":     map[string]any{"id": driverID, "type": "driver"},
					"submittedAtTime": "2026-03-01T14:00:00Z",
					"createdAtTime":   "2026-03-01T14:00:00Z",
					"updatedAtTime":   "2026-03-01T14:00:00Z",
					"status":          "completed",
					"isRequired":      false,
					"fields":          []map[string]any{},
				})
			}
			writeJSONValue(
				t,
				w,
				http.StatusOK,
				map[string]any{"data": data, "pagination": pagination},
			)
		default:
			writeJSON(w, http.StatusNotFound, `{"message":"not found"}`)
		}
	})

	submissions, err := provider.ListFormSubmissions(t.Context(), drivers, 1772300000, 1772400000)
	require.NoError(t, err)

	streamRequests := recorder.forPath("/form-submissions/stream")
	require.Len(t, streamRequests, 6)
	firstPages := 0
	for _, req := range streamRequests {
		if req.after == "" {
			firstPages++
		}
	}
	assert.Equal(t, 3, firstPages)
	assert.Len(t, recorder.forPath("/form-templates"), 1)

	require.Len(t, submissions, len(drivers))
	byDriver := make(map[string]services.ProviderFormSubmission, len(submissions))
	for _, submission := range submissions {
		byDriver[submission.DriverID] = submission
	}
	for _, driverID := range drivers {
		submission, ok := byDriver[driverID]
		require.True(t, ok, driverID)
		assert.Equal(t, "sub-"+driverID, submission.ID)
		assert.Equal(t, batchTemplateID, submission.TemplateID)
		assert.Equal(t, "Proof of delivery", submission.TemplateName)
	}
}

func TestListFormSubmissionsWithoutResultsSkipsTemplateLookup(t *testing.T) {
	t.Parallel()

	recorder := new(requestRecorder)
	provider := newTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		recorder.record(r)
		writeJSON(w, http.StatusOK, `{"data":[],"pagination":{"endCursor":"","hasNextPage":false}}`)
	})

	submissions, err := provider.ListFormSubmissions(t.Context(), batchDriverIDs(51), 1, 2)
	require.NoError(t, err)
	assert.Empty(t, submissions)
	assert.Len(t, recorder.forPath("/form-submissions/stream"), 2)
	assert.Empty(t, recorder.forPath("/form-templates"))
}
