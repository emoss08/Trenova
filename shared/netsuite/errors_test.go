package netsuite_test

import (
	"errors"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/emoss08/trenova/shared/netsuite"
	"github.com/emoss08/trenova/shared/restx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type predicate struct {
	name string
	fn   func(error) bool
}

var predicates = []predicate{
	{"IsNotFound", netsuite.IsNotFound},
	{"IsConcurrencyLimit", netsuite.IsConcurrencyLimit},
	{"IsUnauthorized", netsuite.IsUnauthorized},
	{"IsForbidden", netsuite.IsForbidden},
	{"IsIdempotencyConflict", netsuite.IsIdempotencyConflict},
	{"IsDuplicateExternalID", netsuite.IsDuplicateExternalID},
	{"IsClosedPeriod", netsuite.IsClosedPeriod},
	{"IsInvalidReference", netsuite.IsInvalidReference},
	{"IsPermission", netsuite.IsPermission},
	{"IsUnsupportedTable", netsuite.IsUnsupportedTable},
	{"IsTransient", netsuite.IsTransient},
}

func writeError(t *testing.T, status int, name string) error {
	t.Helper()
	client, _ := newAPIClient(t, serveFixture(t, status, name))
	_, err := client.Create(t.Context(), "invoice", map[string]any{})
	require.Error(t, err)
	return err
}

func TestErrorClassification(t *testing.T) {
	t.Parallel()

	cases := []struct {
		fixture string
		status  int
		code    string
		matches []string
	}{
		{"error_401_invalid_login.json", http.StatusUnauthorized, "INVALID_LOGIN",
			[]string{"IsUnauthorized"}},
		{"error_404_nonexistent.json", http.StatusNotFound, "NONEXISTENT_ID",
			[]string{"IsNotFound"}},
		{"error_429_concurrency.json", http.StatusTooManyRequests, "CONCURRENCY_LIMIT_EXCEEDED",
			[]string{"IsConcurrencyLimit"}},
		{"error_period_closed.json", http.StatusBadRequest, "USER_ERROR",
			[]string{"IsClosedPeriod"}},
		{"error_idempotency.json", http.StatusConflict, "IDEMPOTENCY_ERROR",
			[]string{"IsIdempotencyConflict"}},
		{"error_duplicate_external_id.json", http.StatusBadRequest, "USER_ERROR",
			[]string{"IsDuplicateExternalID"}},
		{"error_invalid_reference.json", http.StatusBadRequest, "INVALID_CONTENT",
			[]string{"IsInvalidReference"}},
		{"error_permission.json", http.StatusBadRequest, "USER_ERROR",
			[]string{"IsPermission"}},
		{"error_500.json", http.StatusInternalServerError, "UNEXPECTED_ERROR",
			[]string{"IsTransient"}},
		{"error_unsupported_table.json", http.StatusBadRequest, "INVALID_PARAMETER",
			[]string{"IsUnsupportedTable"}},
	}
	for _, tc := range cases {
		err := writeError(t, tc.status, tc.fixture)
		var apiErr *netsuite.APIError
		require.ErrorAs(t, err, &apiErr, tc.fixture)
		assert.Equal(t, tc.status, apiErr.Status, tc.fixture)
		assert.Equal(t, tc.code, apiErr.Code(), tc.fixture)
		assert.NotEmpty(t, apiErr.Title, tc.fixture)
		assert.NotEmpty(t, apiErr.Detail(), tc.fixture)
		assert.Contains(t, err.Error(), tc.code, tc.fixture)
		for _, p := range predicates {
			want := false
			for _, name := range tc.matches {
				want = want || name == p.name
			}
			assert.Equal(t, want, p.fn(err), "%s %s", tc.fixture, p.name)
		}
	}
}

func TestErrorDetailPath(t *testing.T) {
	t.Parallel()

	err := writeError(t, http.StatusBadRequest, "error_invalid_reference.json")
	var apiErr *netsuite.APIError
	require.ErrorAs(t, err, &apiErr)
	require.Len(t, apiErr.Details, 1)
	assert.Equal(t, netsuite.ErrorDetail{
		Detail: "Error while accessing a resource. Invalid entity reference key 99999.",
		Code:   "INVALID_CONTENT",
		Path:   "entity",
	}, apiErr.Details[0])
	var restErr *restx.APIError
	assert.True(t, errors.As(err, &restErr), "the transport error is kept as the cause")
}

func TestCodesClassifyWithoutTheirStatus(t *testing.T) {
	t.Parallel()

	assert.True(t, netsuite.IsNotFound(writeError(t, http.StatusBadRequest,
		"error_404_nonexistent.json")))
	assert.True(t, netsuite.IsConcurrencyLimit(writeError(t, http.StatusBadRequest,
		"error_429_concurrency.json")))
	assert.True(t, netsuite.IsUnauthorized(writeError(t, http.StatusBadRequest,
		"error_401_invalid_login.json")))
}

func TestClosedPeriodPhrases(t *testing.T) {
	t.Parallel()

	closed := []string{
		"The transaction date is not within an open accounting period.",
		"The accounting period is closed.",
		"Accounting Period is locked for A/R.",
		"The period is locked.",
		"You cannot post to a closed period.",
		"This transaction falls in a locked posting period.",
		"A/R period locked for Oct 2026",
		"Period Oct 2026 has been closed",
	}
	for _, detail := range closed {
		err := detailError(t, "USER_ERROR", detail)
		assert.True(t, netsuite.IsClosedPeriod(err), detail)
	}
	for _, detail := range []string{
		"Please enter value(s) for: Terms",
		"The record is locked by another user.",
	} {
		assert.False(t, netsuite.IsClosedPeriod(detailError(t, "USER_ERROR", detail)), detail)
	}
	assert.False(t, netsuite.IsClosedPeriod(detailError(t, "INVALID_CONTENT",
		"The accounting period is closed.")), "only a USER_ERROR is a closed period")
}

func detailError(t *testing.T, code, detail string) error {
	t.Helper()
	client, _ := newAPIClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"title":"Bad Request","status":400,"o:errorDetails":[` +
			`{"detail":"` + detail + `","o:errorCode":"` + code + `"}]}`))
	})
	_, err := client.Create(t.Context(), "invoice", map[string]any{})
	require.Error(t, err)
	return err
}

func TestConcurrencyLimitIsRetriedForReadsAndCarriesRetryAfter(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	client, _ := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Retry-After", "0")
		serveFixture(t, http.StatusTooManyRequests, "error_429_concurrency.json")(w, r)
	})
	var out map[string]any
	err := client.Get(t.Context(), netsuite.RecordRead{Type: "invoice", ID: "1"}, &out)
	require.Error(t, err)
	assert.True(t, netsuite.IsConcurrencyLimit(err))
	assert.True(t, netsuite.IsRateLimited(err))
	assert.Equal(t, int32(3), calls.Load())

	client, _ = newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "7")
		serveFixture(t, http.StatusTooManyRequests, "error_429_concurrency.json")(w, r)
	})
	_, err = client.Create(t.Context(), "invoice", map[string]any{})
	require.Error(t, err)
	assert.Equal(t, 7*time.Second, netsuite.RetryAfter(err))

	err = writeError(t, http.StatusTooManyRequests, "error_429_concurrency.json")
	assert.Zero(t, netsuite.RetryAfter(err))
}

func TestErrorsNeverLeakTheAccessToken(t *testing.T) {
	t.Parallel()

	client, _ := newAPIClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"title":"Unauthorized","status":401,"o:errorDetails":[` +
			`{"detail":"token ` + testAccessToken + ` rejected","o:errorCode":"INVALID_LOGIN"}]}`))
	})
	_, err := client.Create(t.Context(), "invoice", map[string]any{})
	require.Error(t, err)
	assert.True(t, netsuite.IsUnauthorized(err))
	assert.NotContains(t, err.Error(), "super-secret-access")
}

func TestPlainTextErrorBody(t *testing.T) {
	t.Parallel()

	client, _ := newAPIClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("<html>bad gateway</html>"))
	})
	_, err := client.Create(t.Context(), "invoice", map[string]any{})
	var apiErr *netsuite.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.Equal(t, "Bad Gateway", apiErr.Title)
	assert.Empty(t, apiErr.Details)
	assert.True(t, netsuite.IsTransient(err))
}
