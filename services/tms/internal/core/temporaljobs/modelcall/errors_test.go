package modelcall

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

func TestClassify(t *testing.T) {
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
			errType:   ErrTypeModelCallFailed,
			nextDelay: 7 * time.Second,
		},
		{
			name:      "a Retry-After past the cap is capped",
			err:       providerError{status: http.StatusServiceUnavailable, wait: time.Hour},
			errType:   ErrTypeModelCallFailed,
			nextDelay: maxProviderBackoff,
		},
		{
			name:    "a timeout is retried on the policy's own backoff",
			err:     providerError{status: http.StatusRequestTimeout},
			errType: ErrTypeModelCallFailed,
		},
		{
			name:    "a failure below HTTP is retried",
			err:     providerError{status: 0},
			errType: ErrTypeModelCallFailed,
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
			errType:   ErrTypeProvidersResting,
			nextDelay: restingBackoff,
		},
		{
			name:    "an unrecognised error is retried",
			err:     errors.New("connection reset"),
			errType: ErrTypeModelCallFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			appErr := applicationError(t, Classify(tt.err))
			assert.Equal(t, tt.nonRetryable, appErr.NonRetryable())
			assert.Equal(t, tt.errType, appErr.Type())
			assert.Equal(t, tt.nextDelay, appErr.NextRetryDelay())
			assert.ErrorIs(t, appErr, tt.err, "the cause is kept for whoever reads the failure")
		})
	}
}

func TestClassifyLeavesCancellationAlone(t *testing.T) {
	t.Parallel()

	require.NoError(t, Classify(nil))

	err := fmt.Errorf("stream: %w", context.Canceled)
	assert.Same(t, err, Classify(err))
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
			crossed := converter.FailureToError(converter.ErrorToFailure(Classify(tt.err)))
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

func TestTransient(t *testing.T) {
	t.Parallel()

	assert.True(t, Transient(providerError{status: http.StatusTooManyRequests}))
	assert.True(t, Transient(errors.New("connection reset")))
	assert.True(t, Transient(serviceports.ErrProvidersResting))
	assert.False(t, Transient(providerError{status: http.StatusBadRequest}))
	assert.False(t, Transient(errortypes.NewBusinessError("the model refused")))
	assert.False(t, Transient(serviceports.ErrNoProviderConfigured))
	assert.False(t, Transient(context.Canceled))
	assert.False(t, Transient(nil))
}

func TestRetryPolicyNeverRetriesWhatCannotChange(t *testing.T) {
	t.Parallel()

	policy := RetryPolicy(4)
	assert.Equal(t, int32(4), policy.MaximumAttempts)
	assert.ElementsMatch(t,
		[]string{ErrTypeModelRejected, ErrTypeNoProviderConfigured},
		policy.NonRetryableErrorTypes,
	)
}

// A caller that waited on a workflow is handed Temporal's error. It still has
// to tell the person what they would have been told had the call run in their
// request: a refusal in its own words, a missing provider as such.
func TestErrRebuildsWhatTheCallReturned(t *testing.T) {
	t.Parallel()

	cross := func(err error) error {
		converter := temporal.GetDefaultFailureConverter()
		return converter.FailureToError(converter.ErrorToFailure(Classify(err)))
	}

	refused := errortypes.NewBusinessError("AI is turned off for %s", "Acme")
	refused.Details = "an administrator can turn it on"
	var business *errortypes.BusinessError
	require.ErrorAs(t, Err(cross(refused)), &business)
	assert.Equal(t, refused.Error(), business.Error())

	assert.ErrorIs(t, Err(cross(fmt.Errorf("route: %w", serviceports.ErrNoProviderConfigured))),
		serviceports.ErrNoProviderConfigured)
	assert.ErrorIs(t, Err(cross(fmt.Errorf("parse: %w", serviceports.ErrModelSchemaValidation))),
		serviceports.ErrModelSchemaValidation)

	plain := errors.New("not a model call")
	assert.Same(t, plain, Err(plain))
}
