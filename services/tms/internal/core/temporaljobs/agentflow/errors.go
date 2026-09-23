package agentflow

import (
	"context"
	"errors"
	"fmt"
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

	detail := modelFailureOf(err)
	fail := func(errType string, nonRetryable bool, nextDelay time.Duration) error {
		return temporal.NewApplicationErrorWithOptions(err.Error(), errType,
			temporal.ApplicationErrorOptions{
				NonRetryable:   nonRetryable,
				Cause:          err,
				Details:        []any{detail},
				NextRetryDelay: nextDelay,
			})
	}

	if errors.Is(err, serviceports.ErrNoProviderConfigured) {
		return fail(ErrTypeNoProviderConfigured, true, 0)
	}

	if errors.Is(err, serviceports.ErrProvidersResting) {
		return fail("ProvidersResting", false, restingBackoff)
	}

	// The router reports a refusal, and a provider it cannot use (a missing
	// API key, say), as business errors: something a person has to change.
	var business *errortypes.BusinessError
	if errors.As(err, &business) {
		return fail(ErrTypeModelRejected, true, 0)
	}

	var failure serviceports.ProviderFailure
	if errors.As(err, &failure) && !retryableStatus(failure.ProviderStatus()) {
		return fail(ErrTypeModelRejected, true, 0)
	}

	return fail("ModelCallFailed", false, providerBackoff(err))
}

// modelFailure is what kind of failure a model call was, carried as the
// details of the error the activity returns. The error Temporal hands back to
// the workflow is its own type, so without these the saved turn could no
// longer say whether the provider refused the request or was unreachable.
type modelFailure struct {
	Status    int  `json:"status,omitempty"`
	Retryable bool `json:"retryable,omitempty"`
	Resting   bool `json:"resting,omitempty"`
	TimedOut  bool `json:"timedOut,omitempty"`
}

func modelFailureOf(err error) modelFailure {
	detail := modelFailure{
		Resting:  errors.Is(err, serviceports.ErrProvidersResting),
		TimedOut: errors.Is(err, context.DeadlineExceeded),
	}

	var failure serviceports.ProviderFailure
	if errors.As(err, &failure) {
		detail.Status = failure.ProviderStatus()
		detail.Retryable = failure.ProviderRetryable()
	}

	return detail
}

// Failure is why a turn ended before it finished, as data. It crosses from
// workflow code into whichever activity saves the turn, and still says what
// kind of failure it was when it gets there.
type Failure struct {
	Message string `json:"message"`
	// Stopped says somebody stopped the run.
	Stopped bool `json:"stopped,omitempty"`
	// Status is the provider's HTTP status, when a provider answered at all.
	Status    int  `json:"status,omitempty"`
	Retryable bool `json:"retryable,omitempty"`
	Resting   bool `json:"resting,omitempty"`
	TimedOut  bool `json:"timedOut,omitempty"`
}

// FailureOf reads why a run ended from the error workflow code was handed.
func FailureOf(err error) *Failure {
	if err == nil {
		return nil
	}

	failure := &Failure{Message: err.Error()}
	if temporal.IsCanceledError(err) || errors.Is(err, context.Canceled) {
		failure.Stopped = true

		return failure
	}

	var appErr *temporal.ApplicationError
	if errors.As(err, &appErr) && appErr.HasDetails() {
		var detail modelFailure
		if appErr.Details(&detail) == nil {
			failure.Status = detail.Status
			failure.Retryable = detail.Retryable
			failure.Resting = detail.Resting
			failure.TimedOut = detail.TimedOut
		}
	}

	var timeout *temporal.TimeoutError
	if errors.As(err, &timeout) {
		failure.TimedOut = true
	}

	return failure
}

// Err is the failure as an error the services that describe a failure already
// understand: a stop is context.Canceled, a provider's answer is a
// ProviderFailure, and so on.
func (f *Failure) Err() error {
	switch {
	case f == nil:
		return nil
	case f.Stopped:
		return fmt.Errorf("%s: %w", f.Message, context.Canceled)
	case f.Status != 0:
		return providerFailure{f}
	case f.Resting:
		return fmt.Errorf("%s: %w", f.Message, serviceports.ErrProvidersResting)
	case f.TimedOut:
		return fmt.Errorf("%s: %w", f.Message, context.DeadlineExceeded)
	default:
		return errors.New(f.Message)
	}
}

type providerFailure struct{ f *Failure }

func (p providerFailure) Error() string           { return p.f.Message }
func (p providerFailure) ProviderStatus() int     { return p.f.Status }
func (p providerFailure) ProviderRetryable() bool { return p.f.Retryable }

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
