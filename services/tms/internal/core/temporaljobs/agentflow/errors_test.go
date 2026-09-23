package agentflow

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
)

type providerError struct {
	status int
	wait   time.Duration
}

func (e providerError) Error() string { return fmt.Sprintf("provider answered %d", e.status) }

func (e providerError) ProviderStatus() int { return e.status }

func (e providerError) ProviderRetryable() bool { return retryableStatus(e.status) }

func (e providerError) ProviderRetryAfter() time.Duration { return e.wait }

func applicationError(t *testing.T, err error) *temporal.ApplicationError {
	t.Helper()

	var appErr *temporal.ApplicationError
	require.ErrorAs(t, err, &appErr)

	return appErr
}

func TestRetryPolicyFor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		err          error
		nonRetryable bool
		errType      string
		nextDelay    time.Duration
	}{
		{
			name:         "a rejected request is not asked again",
			err:          providerError{status: http.StatusBadRequest},
			nonRetryable: true,
			errType:      ErrTypeModelRejected,
		},
		{
			name:         "a refused key is not asked again",
			err:          fmt.Errorf("call: %w", providerError{status: http.StatusUnauthorized}),
			nonRetryable: true,
			errType:      ErrTypeModelRejected,
		},
		{
			name:      "a rate limit waits out the provider's Retry-After",
			err:       providerError{status: http.StatusTooManyRequests, wait: 7 * time.Second},
			errType:   "ModelCallFailed",
			nextDelay: 7 * time.Second,
		},
		{
			name:      "a Retry-After past the cap is capped",
			err:       providerError{status: http.StatusServiceUnavailable, wait: time.Hour},
			errType:   "ModelCallFailed",
			nextDelay: maxProviderBackoff,
		},
		{
			name:    "a timeout is retried on the policy's own backoff",
			err:     providerError{status: http.StatusRequestTimeout},
			errType: "ModelCallFailed",
		},
		{
			name:    "a failure below HTTP is retried",
			err:     providerError{status: 0},
			errType: "ModelCallFailed",
		},
		{
			name:         "a business error is a person's to fix",
			err:          errortypes.NewBusinessError("the model refused"),
			nonRetryable: true,
			errType:      ErrTypeModelRejected,
		},
		{
			name:         "no provider is not retried",
			err:          fmt.Errorf("route: %w", serviceports.ErrNoProviderConfigured),
			nonRetryable: true,
			errType:      ErrTypeNoProviderConfigured,
		},
		{
			name:      "resting providers are asked again after their rest",
			err:       serviceports.ErrProvidersResting,
			errType:   "ProvidersResting",
			nextDelay: restingBackoff,
		},
		{
			name:    "an unrecognised error is retried",
			err:     errors.New("connection reset"),
			errType: "ModelCallFailed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			appErr := applicationError(t, retryPolicyFor(tt.err))
			assert.Equal(t, tt.nonRetryable, appErr.NonRetryable())
			assert.Equal(t, tt.errType, appErr.Type())
			assert.Equal(t, tt.nextDelay, appErr.NextRetryDelay())
			assert.ErrorIs(t, appErr, tt.err, "the cause is kept for whoever reads the failure")
		})
	}
}

func TestRetryPolicyForLeavesCancellationAlone(t *testing.T) {
	t.Parallel()

	require.NoError(t, retryPolicyFor(nil))

	err := fmt.Errorf("stream: %w", context.Canceled)
	assert.Same(t, err, retryPolicyFor(err))
}

// The error a workflow is handed is Temporal's own type. The saved turn still
// has to say whether the provider refused the request or could not be
// reached, so the kind of failure travels in the error's details.
func TestFailureSurvivesTheActivityBoundary(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		err   error
		check func(t *testing.T, err error)
	}{
		{
			name: "a provider's refusal",
			err:  providerError{status: http.StatusBadRequest},
			check: func(t *testing.T, err error) {
				var failure serviceports.ProviderFailure
				require.ErrorAs(t, err, &failure)
				assert.Equal(t, http.StatusBadRequest, failure.ProviderStatus())
				assert.False(t, failure.ProviderRetryable())
			},
		},
		{
			name: "every provider resting",
			err:  serviceports.ErrProvidersResting,
			check: func(t *testing.T, err error) {
				assert.ErrorIs(t, err, serviceports.ErrProvidersResting)
			},
		},
		{
			name: "a provider that never answered in time",
			err:  fmt.Errorf("stream: %w", context.DeadlineExceeded),
			check: func(t *testing.T, err error) {
				assert.ErrorIs(t, err, context.DeadlineExceeded)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			converter := temporal.GetDefaultFailureConverter()
			crossed := converter.FailureToError(converter.ErrorToFailure(retryPolicyFor(tt.err)))
			failure := FailureOf(crossed)
			require.NotNil(t, failure)
			assert.False(t, failure.Stopped)
			tt.check(t, failure.Err())
		})
	}
}

func TestFailureOfAStopIsAStop(t *testing.T) {
	t.Parallel()

	failure := FailureOf(temporal.NewCanceledError())
	require.NotNil(t, failure)
	assert.True(t, failure.Stopped)
	assert.ErrorIs(t, failure.Err(), context.Canceled)

	assert.Nil(t, FailureOf(nil))
	assert.NoError(t, (*Failure)(nil).Err())
}
