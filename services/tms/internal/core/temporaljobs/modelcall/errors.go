// Package modelcall is how every activity that calls a model tells Temporal
// what a failure deserves, and how the kind of failure survives the trip back
// to workflow code.
package modelcall

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
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
		return fail(ErrTypeProvidersResting, false, restingBackoff)
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

	return fail(ErrTypeModelCallFailed, false, providerBackoff(err))
}

// modelFailure is what kind of failure a model call was, carried as the
// details of the error the activity returns. The error Temporal hands back to
// the workflow is its own type, so without these the saved turn could no
// longer say whether the provider refused the request or was unreachable.
type modelFailure struct {
	Status        int      `json:"status,omitempty"`
	Retryable     bool     `json:"retryable,omitempty"`
	Resting       bool     `json:"resting,omitempty"`
	TimedOut      bool     `json:"timedOut,omitempty"`
	NoProvider    bool     `json:"noProvider,omitempty"`
	SchemaInvalid bool     `json:"schemaInvalid,omitempty"`
	Refusal       *refusal `json:"refusal,omitempty"`
}

// refusal is a business error as data: the message key the error handler
// translates, its arguments, and the detail it appends.
type refusal struct {
	Message string   `json:"message"`
	Args    []string `json:"args,omitempty"`
	Details string   `json:"details,omitempty"`
}

func modelFailureOf(err error) modelFailure {
	detail := modelFailure{
		Resting:       errors.Is(err, serviceports.ErrProvidersResting),
		TimedOut:      errors.Is(err, context.DeadlineExceeded),
		NoProvider:    errors.Is(err, serviceports.ErrNoProviderConfigured),
		SchemaInvalid: errors.Is(err, serviceports.ErrModelSchemaValidation),
	}

	var failure serviceports.ProviderFailure
	if errors.As(err, &failure) {
		detail.Status = failure.ProviderStatus()
		detail.Retryable = failure.ProviderRetryable()
	}

	var business *errortypes.BusinessError
	if errors.As(err, &business) {
		args := make([]string, 0, len(business.Args))
		for _, arg := range business.Args {
			args = append(args, fmt.Sprint(arg))
		}
		detail.Refusal = &refusal{Message: business.Message, Args: args, Details: business.Details}
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
	// NoProvider says no provider is configured for the task.
	NoProvider bool `json:"noProvider,omitempty"`
	// SchemaInvalid says the model's answer did not fit the schema asked for.
	SchemaInvalid bool `json:"schemaInvalid,omitempty"`
	// Refusal is a business error, kept whole so the person is told what
	// they would have been told had the call run in their request.
	Refusal *refusal `json:"refusal,omitempty"`
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
			failure.NoProvider = detail.NoProvider
			failure.SchemaInvalid = detail.SchemaInvalid
			failure.Refusal = detail.Refusal
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
	case f.Refusal != nil:
		return f.Refusal.err()
	case f.NoProvider:
		return fmt.Errorf("%s: %w", f.Message, serviceports.ErrNoProviderConfigured)
	case f.SchemaInvalid:
		return fmt.Errorf("%s: %w", f.Message, serviceports.ErrModelSchemaValidation)
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

// Err returns the model call's failure from an error workflow code or a
// workflow's caller was handed, as the error the call itself returned. A call
// that ran out of time is context.DeadlineExceeded, as it would have been in
// the request. An error that carries neither comes back as it is.
func Err(err error) error {
	var appErr *temporal.ApplicationError
	var timeout *temporal.TimeoutError
	if (errors.As(err, &appErr) && appErr.HasDetails()) || errors.As(err, &timeout) {
		return FailureOf(err).Err()
	}

	return err
}

func (r *refusal) err() error {
	args := make([]any, 0, len(r.Args))
	for _, arg := range r.Args {
		args = append(args, arg)
	}
	business := errortypes.NewBusinessError(r.Message, args...)
	business.Details = r.Details

	return business
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
