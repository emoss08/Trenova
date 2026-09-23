package agentflow

import (
	"context"
	"errors"
	"net/http"
	"time"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"go.temporal.io/sdk/temporal"
)

// retryPolicyFor tells Temporal what a failed model call deserves, the way the
// provider's own answer says it should be treated.
//
// It is the cookbook's retry-from-HTTP-response recipe. A request the provider
// rejected is rejected however often it is sent, so it is not retried, and
// neither is a model's refusal or a missing provider. Anything transient is
// retried, after however long the provider asked for when it said, so a 429
// waits out its Retry-After instead of hammering at the next backoff step.
func retryPolicyFor(err error) error {
	if err == nil {
		return nil
	}

	// A call the run's own cancellation stopped is not a failure to retry.
	if errors.Is(err, context.Canceled) {
		return err
	}

	if errors.Is(err, serviceports.ErrNoProviderConfigured) {
		return temporal.NewNonRetryableApplicationError(
			err.Error(),
			ErrTypeNoProviderConfigured,
			err,
		)
	}

	if errors.Is(err, serviceports.ErrProvidersResting) {
		return temporal.NewApplicationErrorWithOptions(err.Error(), "ProvidersResting",
			temporal.ApplicationErrorOptions{Cause: err, NextRetryDelay: restingBackoff})
	}

	// The router reports a refusal, and a provider it cannot use (a missing
	// API key, say), as business errors: something a person has to change.
	var business *errortypes.BusinessError
	if errors.As(err, &business) {
		return temporal.NewNonRetryableApplicationError(err.Error(), ErrTypeModelRejected, err)
	}

	var failure serviceports.ProviderFailure
	if errors.As(err, &failure) && !retryableStatus(failure.ProviderStatus()) {
		return temporal.NewNonRetryableApplicationError(err.Error(), ErrTypeModelRejected, err)
	}

	return temporal.NewApplicationErrorWithOptions(err.Error(), "ModelCallFailed",
		temporal.ApplicationErrorOptions{Cause: err, NextRetryDelay: providerBackoff(err)})
}

// retryableStatus reports whether a provider's HTTP status is worth asking
// again. A 4xx is the request's fault and stays wrong, except a timeout, a
// conflict and a rate limit, which are about the moment rather than the
// request. A zero status is a failure below HTTP (a dial, TLS, a dropped
// stream), and those are the failures most likely to be transient.
func retryableStatus(status int) bool {
	switch {
	case status == 0:
		return true
	case status == http.StatusRequestTimeout,
		status == http.StatusConflict,
		status == http.StatusTooManyRequests:
		return true
	case status >= 400 && status < 500:
		return false
	default:
		return true
	}
}

// providerBackoff is how long the provider asked to be left alone, capped.
// Zero leaves the wait to the retry policy's own backoff.
func providerBackoff(err error) time.Duration {
	var backoff serviceports.ProviderBackoff
	if !errors.As(err, &backoff) {
		return 0
	}

	wait := backoff.ProviderRetryAfter()
	if wait <= 0 {
		return 0
	}

	return min(wait, maxProviderBackoff)
}
