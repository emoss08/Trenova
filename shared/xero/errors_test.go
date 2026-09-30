package xero_test

import (
	"errors"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/emoss08/trenova/shared/restx"
	"github.com/emoss08/trenova/shared/xero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func noRetry() xero.Option {
	return xero.WithRetry(restx.RetryConfig{})
}

func apiError(t *testing.T, err error) *xero.APIError {
	t.Helper()
	var apiErr *xero.APIError
	require.ErrorAs(t, err, &apiErr)
	return apiErr
}

func TestValidationErrorCollectsEveryMessageOnce(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	client := newAPIClient(t, func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.Header().Set("Xero-Correlation-Id", "6f1c3a2e-correlation")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write(fixture(t, "error_validation.json"))
	})

	_, err := client.UpdateInvoice(t.Context(), testKey, testInvoice, freightInvoice())
	require.Error(t, err)
	apiErr := apiError(t, err)
	assert.Equal(t, http.StatusBadRequest, apiErr.Status)
	assert.Equal(t, xero.TypeValidation, apiErr.Type)
	assert.Equal(t, "A validation exception occurred", apiErr.Message)
	assert.Equal(t, []string{
		"Invoice not of valid status for modification",
		"Account code '999' is not a valid code for this document.",
	}, apiErr.ValidationMessages)
	assert.Equal(t, "6f1c3a2e-correlation", apiErr.CorrelationID)
	assert.True(t, xero.IsValidation(err))
	assert.False(t, xero.IsDuplicateNumber(err))
	assert.False(t, xero.IsLockDate(err))
	assert.False(t, xero.IsTransient(err))
	assert.Equal(t, int32(1), calls.Load(), "a validation failure is not retried")
	assert.Contains(t, err.Error(), "Invoice not of valid status for modification")
	assert.Contains(t, err.Error(), "6f1c3a2e-correlation")
}

func TestLockedPeriodWordingIsALockDate(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"ErrorNumber":10,"Type":"ValidationException","Message":"A validation exception occurred",` +
			`"Elements":[{"ValidationErrors":[{"Message":"The period is locked for this date"}]}]}`))
	})

	_, err := client.CreatePayment(t.Context(), testKey, invoicePayment())
	assert.True(t, xero.IsLockDate(err))
}

func TestExpiredTokenIsAuthAndNeverLeaksTheToken(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	client := newAPIClient(t, func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write(fixture(t, "error_401.json"))
	})

	_, err := client.Organisation(t.Context())
	require.Error(t, err)
	apiErr := apiError(t, err)
	assert.Empty(t, apiErr.Type)
	assert.Contains(t, apiErr.Message, "TokenExpired")
	assert.True(t, xero.IsAuth(err))
	assert.False(t, xero.IsInsufficientScope(err))
	assert.False(t, xero.IsForbidden(err))
	assert.False(t, xero.IsTransient(err))
	assert.Equal(t, int32(1), calls.Load())
	assert.NotContains(t, err.Error(), testAccessToken)
	assert.NotContains(t, err.Error(), "super-secret-access")
}

func TestEchoedTokenIsRedacted(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"Title":"Unauthorized","Detail":"bad header ` + r.Header.Get("Authorization") + `"}`))
	})

	_, err := client.Organisation(t.Context())
	require.Error(t, err)
	assert.NotContains(t, err.Error(), testAccessToken)
	assert.NotContains(t, apiError(t, err).Message, testAccessToken)
	assert.Contains(t, err.Error(), "REDACTED")
}

func TestInsufficientScopeIsReadFromTheChallenge(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("WWW-Authenticate", `Bearer error="insufficent_scope"`)
		w.WriteHeader(http.StatusUnauthorized)
	})

	_, err := client.TrialBalance(t.Context(), time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC))
	require.Error(t, err)
	assert.True(t, xero.IsAuth(err))
	assert.True(t, xero.IsInsufficientScope(err))
	assert.Equal(t, "Unauthorized", apiError(t, err).Message)
}

func TestDisconnectedTenantIsForbidden(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write(fixture(t, "error_403.json"))
	})

	_, err := client.Accounts(t.Context(), nil)
	require.Error(t, err)
	apiErr := apiError(t, err)
	assert.Equal(t, xero.TypeAuthUnsuccessful, apiErr.Type)
	assert.Equal(t, "AuthenticationUnsuccessful", apiErr.Message)
	assert.True(t, xero.IsForbidden(err))
	assert.False(t, xero.IsAuth(err))
	assert.False(t, xero.IsTransient(err))
}

func TestMissingDocumentIsNotFound(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write(fixture(t, "error_404.txt"))
	})

	_, err := client.VoidInvoice(t.Context(), testKey, testInvoice)
	require.Error(t, err)
	assert.True(t, xero.IsNotFound(err))
	assert.Equal(t, "The resource you're looking for cannot be found", apiError(t, err).Message)
	assert.False(t, xero.IsValidation(err))
}

func TestRateLimitCarriesRetryAfterAndProblem(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "37")
		w.Header().Set("X-Rate-Limit-Problem", "Minute")
		w.Header().Set("X-MinLimit-Remaining", "0")
		w.WriteHeader(http.StatusTooManyRequests)
	}, noRetry())

	_, err := client.Contacts(t.Context(), 1, nil)
	require.Error(t, err)
	apiErr := apiError(t, err)
	assert.Equal(t, http.StatusTooManyRequests, apiErr.Status)
	assert.Equal(t, 37*time.Second, apiErr.RetryAfter)
	assert.Equal(t, xero.RateLimitMinute, apiErr.RateLimitProblem)
	assert.True(t, xero.IsRateLimited(err))
	assert.True(t, xero.IsTransient(err))
	assert.Equal(t, 37*time.Second, restx.RetryAfterOf(err), "restx sees the retry hint too")
}

func TestRateLimitIsRetriedAfterTheHint(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	client := newAPIClient(t, func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) == 1 {
			w.Header().Set("Retry-After", "0")
			w.Header().Set("X-Rate-Limit-Problem", "concurrent")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		_, _ = w.Write(fixture(t, "organisation.json"))
	})

	_, err := client.Organisation(t.Context())
	require.NoError(t, err)
	assert.Equal(t, int32(2), calls.Load())
}

func TestOfflineOrganisation(t *testing.T) {
	t.Parallel()

	client := newAPIClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write(fixture(t, "error_503.txt"))
	}, noRetry())

	_, err := client.Payments(t.Context(), 1, nil)
	require.Error(t, err)
	assert.True(t, xero.IsOrganisationOffline(err))
	assert.True(t, xero.IsTransient(err))
	assert.Equal(t, "The Organisation is offline", apiError(t, err).Message)
}

func TestServerErrorsAreRetriedAndTransient(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	client := newAPIClient(t, func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`<html><body>Internal error</body></html>`))
	})

	_, err := client.Items(t.Context(), nil)
	require.Error(t, err)
	assert.True(t, xero.IsTransient(err))
	assert.False(t, xero.IsOrganisationOffline(err))
	assert.Equal(t, "Internal Server Error", apiError(t, err).Message, "an html page is not a message")
	assert.Equal(t, int32(3), calls.Load())
}

func TestClassifiersAreFalseForOtherErrors(t *testing.T) {
	t.Parallel()

	other := errors.New("boom")
	assert.False(t, xero.IsAuth(other))
	assert.False(t, xero.IsForbidden(other))
	assert.False(t, xero.IsNotFound(other))
	assert.False(t, xero.IsRateLimited(other))
	assert.False(t, xero.IsValidation(other))
	assert.False(t, xero.IsDuplicateNumber(other))
	assert.False(t, xero.IsLockDate(other))
	assert.False(t, xero.IsOrganisationOffline(other))
	assert.False(t, xero.IsTransient(other))
	assert.False(t, xero.IsTransient(nil))
	assert.True(t, xero.IsRateLimited(&restx.RateLimitedError{Key: "xero:tenant:x"}),
		"the local limiter counts as rate limited")
}
