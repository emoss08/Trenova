package businesscentral_test

import (
	"errors"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/emoss08/trenova/shared/businesscentral"
	"github.com/emoss08/trenova/shared/restx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func apiError(t *testing.T, err error) *businesscentral.APIError {
	t.Helper()
	var apiErr *businesscentral.APIError
	require.ErrorAs(t, err, &apiErr)
	return apiErr
}

func TestUnauthorizedIsAuthAndNeverLeaksTheToken(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	client := newAPIClient(t, func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.Header().Set("request-id", "req-401")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"code":"Authentication_InvalidCredentials",` +
			`"message":"token ` + testAccessToken + ` rejected"}}`))
	})

	_, err := client.Currencies(t.Context())
	require.Error(t, err)
	apiErr := apiError(t, err)
	assert.Equal(t, http.StatusUnauthorized, apiErr.Status)
	assert.Equal(t, "Authentication_InvalidCredentials", apiErr.Code)
	assert.Equal(t, "req-401", apiErr.RequestID)
	assert.True(t, businesscentral.IsAuth(err))
	assert.False(t, businesscentral.IsForbidden(err))
	assert.False(t, businesscentral.IsTransient(err))
	assert.NotContains(t, err.Error(), "super-secret-access")
	assert.NotContains(t, apiErr.Message, "super-secret-access")
	assert.Equal(t, int32(1), calls.Load())
}

func TestAuthenticationCodeIsAuthWithoutStatus(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, serveFixture(t, http.StatusBadRequest, "error_401.json"))
	_, err := client.Currencies(t.Context())
	assert.True(t, businesscentral.IsAuth(err))
}

func TestForbidden(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, serveFixture(t, http.StatusForbidden, "error_403.json"))
	_, err := client.Currencies(t.Context())
	assert.True(t, businesscentral.IsForbidden(err))
	assert.False(t, businesscentral.IsAuth(err))
	assert.Equal(t, "Authorization_InsufficientPermissions", apiError(t, err).Code)
}

func TestRecordNotFound(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, serveFixture(t, http.StatusNotFound, "error_404.json"))
	_, err := client.Document(t.Context(), businesscentral.DocumentSalesInvoice, testInvoice)
	assert.True(t, businesscentral.IsNotFound(err))
	assert.False(t, businesscentral.IsValidation(err))
	assert.Contains(t, err.Error(), "Internal_RecordNotFound")
}

func TestEntityChangedIsAConflict(t *testing.T) {
	t.Parallel()

	for _, status := range []int{http.StatusConflict, http.StatusPreconditionFailed} {
		client := newAPIClient(t, serveFixture(t, status, "error_conflict.json"))
		err := client.DeleteDocument(t.Context(), businesscentral.DocumentSalesInvoice,
			testInvoice, `W/"JzIwOzE="`)
		assert.True(t, businesscentral.IsConflict(err), "status %d", status)
		assert.False(t, businesscentral.IsTransient(err))
	}

	client := newAPIClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"code":"BadRequest_InvalidToken","message":"stale"}}`))
	})
	err := client.DeleteDocument(t.Context(), businesscentral.DocumentSalesInvoice, testInvoice, "")
	assert.True(t, businesscentral.IsConflict(err))
}

func TestDuplicateKey(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, serveFixture(t, http.StatusBadRequest, "error_duplicate.json"))
	_, err := client.CreatePaymentJournal(t.Context(), businesscentral.PartyCustomer,
		&businesscentral.PaymentJournalInput{Code: "TRENOVA"})
	assert.True(t, businesscentral.IsDuplicate(err))
	assert.True(t, businesscentral.IsValidation(err))
	assert.False(t, businesscentral.IsPostingDate(err))
}

func TestPostingDateOutsideTheAllowedRange(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	client := newAPIClient(t, func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write(fixture(t, "error_posting_date.json"))
	})

	err := client.PostDocument(t.Context(), businesscentral.DocumentSalesInvoice, testInvoice)
	require.Error(t, err)
	assert.True(t, businesscentral.IsPostingDate(err))
	assert.True(t, businesscentral.IsValidation(err))
	assert.False(t, businesscentral.IsDuplicate(err))
	assert.Equal(t, businesscentral.CodeDialogException, apiError(t, err).Code)
	assert.Equal(t, int32(1), calls.Load())
}

func TestBadRequestCodeIsValidation(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, serveFixture(t, http.StatusBadRequest, "error_validation.json"))
	_, err := client.Currencies(t.Context())
	assert.True(t, businesscentral.IsValidation(err))
	assert.False(t, businesscentral.IsPostingDate(err))
}

func TestRateLimitedHonoursRetryAfter(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	client := newAPIClient(t, func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write(fixture(t, "error_429.json"))
			return
		}
		_, _ = w.Write(fixture(t, "currencies.json"))
	})

	currencies, err := client.Currencies(t.Context())
	require.NoError(t, err)
	assert.Len(t, currencies, 2)
	assert.Equal(t, int32(2), calls.Load())
}

func TestRateLimitedErrorCarriesRetryAfter(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "7")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write(fixture(t, "error_429.json"))
	}, businesscentral.WithRetry(restx.RetryConfig{}))

	_, err := client.Currencies(t.Context())
	require.Error(t, err)
	assert.True(t, businesscentral.IsRateLimited(err))
	assert.False(t, businesscentral.IsTransient(err))
	assert.Equal(t, 7*time.Second, apiError(t, err).RetryAfter)
	assert.Equal(t, 7*time.Second, businesscentral.RetryAfter(err))
}

func TestRateLimitedWithoutRetryAfterBacksOff(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	client := newAPIClient(t, func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write(fixture(t, "error_429.json"))
	})

	_, err := client.Currencies(t.Context())
	require.Error(t, err)
	assert.True(t, businesscentral.IsRateLimited(err))
	assert.Equal(t, time.Duration(0), apiError(t, err).RetryAfter)
	assert.Equal(t, int32(3), calls.Load(), "a read is retried up to the attempt limit")
}

func TestServiceUnavailableIsTransient(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	client := newAPIClient(t, func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write(fixture(t, "error_503.txt"))
	})

	_, err := client.Currencies(t.Context())
	require.Error(t, err)
	assert.True(t, businesscentral.IsTransient(err))
	assert.Equal(t, "Service Unavailable", apiError(t, err).Message)
	assert.Equal(t, int32(3), calls.Load())
}

func TestRequestTimeoutIsTransient(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusRequestTimeout)
	})
	_, err := client.Currencies(t.Context())
	assert.True(t, businesscentral.IsTransient(err))
}

func TestPredicatesOnForeignErrors(t *testing.T) {
	t.Parallel()

	err := errors.New("plain")
	assert.False(t, businesscentral.IsAuth(err))
	assert.False(t, businesscentral.IsNotFound(err))
	assert.False(t, businesscentral.IsConflict(err))
	assert.False(t, businesscentral.IsValidation(err))
	assert.False(t, businesscentral.IsPostingDate(err))
	assert.False(t, businesscentral.IsTransient(nil))
	assert.True(t, businesscentral.IsRateLimited(&restx.RateLimitedError{Key: "k"}))
}
