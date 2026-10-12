// Package modelcall is how every activity that calls a model tells Temporal
// what a failure deserves, and how the kind of failure survives the trip back
// to workflow code.
package modelcall

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/jobs/modelcall/modelcallcontract"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
)

// Error types a failed model call is reported as. The first two fail the same
// way however many times they are asked, so a retry policy lists them as
// non-retryable.
const (
	ErrTypeModelRejected        = "ModelRejected"
	ErrTypeNoProviderConfigured = "NoProviderConfigured"
	ErrTypeModelCallFailed      = "ModelCallFailed"
	ErrTypeProvidersResting     = "ProvidersResting"
)

const (
	// maxProviderBackoff caps how long a provider's own Retry-After is
	// honoured. A provider asking for longer is treated as unavailable rather
	// than waited on.
	maxProviderBackoff = time.Minute

	// restingBackoff is how long to wait when every provider is resting after
	// repeated failures, which is the length of the breaker's rest.
	restingBackoff = time.Minute
)

// RetryPolicy is the policy for an activity whose failures went through
// Classify: exponential from a second, capped at thirty, and never retrying
// what the provider rejected or what no provider can serve.
func RetryPolicy(attempts int32) *temporal.RetryPolicy {
	return &temporal.RetryPolicy{
		InitialInterval:        time.Second,
		BackoffCoefficient:     2,
		MaximumInterval:        30 * time.Second,
		MaximumAttempts:        attempts,
		NonRetryableErrorTypes: []string{ErrTypeModelRejected, ErrTypeNoProviderConfigured},
	}
}

// Transient reports whether a failed model call is worth asking again.
func Transient(err error) bool {
	if err == nil || errors.Is(err, context.Canceled) {
		return false
	}

	var appErr *temporal.ApplicationError

	return errors.As(Classify(err), &appErr) && !appErr.NonRetryable()
}

// FinalAttempt reports whether this attempt of the running activity is its
// last under a policy of attempts. An activity that falls back to a
// deterministic answer when the model fails does so only here: before the
// last attempt, a transient failure is Temporal's to retry.
//
// Outside an activity nothing retries the call, so every attempt is the last.
func FinalAttempt(ctx context.Context, attempts int32) bool {
	if !activity.IsActivity(ctx) {
		return true
	}

	return attempts > 0 && activity.GetInfo(ctx).Attempt >= attempts
}

// Classify tells Temporal what a failed model call deserves, the way the
// provider's own answer says it should be treated.
//
// It is the cookbook's retry-from-HTTP-response recipe. A request the provider
// rejected is rejected however often it is sent, so it is not retried, and
// neither is a model's refusal or a missing provider. Anything transient is
// retried, after however long the provider asked for when it said, so a 429
// waits out its Retry-After instead of hammering at the next backoff step.
func Classify(err error) error {
	if err == nil {
		return nil
	}

	// A call the run's own cancellation stopped is not a failure to retry.
	if errors.Is(err, context.Canceled) {
		return err
	}

	detail := modelcallcontract.ModelFailureOf(err)
	fail := func(errType string, nonRetryable bool, nextDelay time.Duration) error {
		return temporal.NewApplicationErrorWithOptions(err.Error(), errType,
			temporal.ApplicationErrorOptions{
				NonRetryable:   nonRetryable,
				Cause:          err,
				Details:        []any{detail},
				NextRetryDelay: nextDelay,
			})
	}

	if errors.Is(err, agentports.ErrNoProviderConfigured) {
		return fail(ErrTypeNoProviderConfigured, true, 0)
	}

	if errors.Is(err, agentports.ErrProvidersResting) {
		return fail(ErrTypeProvidersResting, false, restingBackoff)
	}

	if detail.Plan != nil {
		return fail(ErrTypeModelRejected, true, 0)
	}

	// The router reports a refusal, and a provider it cannot use (a missing
	// API key, say), as business errors: something a person has to change.
	var business *errortypes.BusinessError
	if errors.As(err, &business) {
		return fail(ErrTypeModelRejected, true, 0)
	}

	var failure agentports.ProviderFailure
	if errors.As(err, &failure) && !retryableStatus(failure.ProviderStatus()) {
		return fail(ErrTypeModelRejected, true, 0)
	}

	return fail(ErrTypeModelCallFailed, false, providerBackoff(err))
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
	var backoff agentports.ProviderBackoff
	if !errors.As(err, &backoff) {
		return 0
	}

	wait := backoff.ProviderRetryAfter()
	if wait <= 0 {
		return 0
	}

	return min(wait, maxProviderBackoff)
}
