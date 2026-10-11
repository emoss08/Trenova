package modelcallcontract

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"go.temporal.io/sdk/temporal"
)

// ModelFailure is what kind of failure a model call was, carried as the
// details of the error the activity returns. The error Temporal hands back to
// the workflow is its own type, so without these the saved turn could no
// longer say whether the provider refused the request or was unreachable.
type ModelFailure struct {
	Status        int          `json:"status,omitempty"`
	Retryable     bool         `json:"retryable,omitempty"`
	Resting       bool         `json:"resting,omitempty"`
	TimedOut      bool         `json:"timedOut,omitempty"`
	NoProvider    bool         `json:"noProvider,omitempty"`
	SchemaInvalid bool         `json:"schemaInvalid,omitempty"`
	Refusal       *refusal     `json:"refusal,omitempty"`
	Plan          *PlanRefusal `json:"plan,omitempty"`
	// Providers are the providers asked, when every one of them failed.
	Providers []agentports.ChatProviderFailure `json:"providers,omitempty"`
}

type PlanRefusal struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	Meter      string `json:"meter,omitempty"`
	Limit      int64  `json:"limit,omitempty"`
	Used       int64  `json:"used,omitempty"`
	Capability string `json:"capability,omitempty"`
	Reason     string `json:"reason,omitempty"`
	Plan       string `json:"plan,omitempty"`
}

func planRefusalOf(err error) *PlanRefusal {
	if quota, ok := errors.AsType[*errortypes.QuotaExceededError](err); ok {
		return &PlanRefusal{
			Code:    string(errortypes.ErrQuotaExceeded),
			Message: quota.Message,
			Meter:   quota.Meter,
			Limit:   quota.Limit,
			Used:    quota.Used,
			Plan:    quota.Plan,
		}
	}
	if restriction, ok := errors.AsType[*errortypes.PlanRestrictionError](err); ok {
		return &PlanRefusal{
			Code:       string(errortypes.ErrPlanRestricted),
			Message:    restriction.Message,
			Capability: restriction.Capability,
			Reason:     restriction.Reason,
			Plan:       restriction.Plan,
		}
	}

	return nil
}

func (r *PlanRefusal) Err() error {
	if r.Code == string(errortypes.ErrQuotaExceeded) {
		return errortypes.NewQuotaExceededError(r.Meter, r.Limit, r.Used, r.Plan)
	}

	return errortypes.NewPlanRestrictionError(r.Capability, r.Reason, r.Plan)
}

// refusal is a business error as data: the message key the error handler
// translates, its arguments, and the detail it appends.
type refusal struct {
	Message string   `json:"message"`
	Args    []string `json:"args,omitempty"`
	Details string   `json:"details,omitempty"`
}

func ModelFailureOf(err error) ModelFailure {
	detail := ModelFailure{
		Resting:       errors.Is(err, agentports.ErrProvidersResting),
		TimedOut:      errors.Is(err, context.DeadlineExceeded),
		NoProvider:    errors.Is(err, agentports.ErrNoProviderConfigured),
		SchemaInvalid: errors.Is(err, agentports.ErrModelSchemaValidation),
		Plan:          planRefusalOf(err),
	}

	var failure agentports.ProviderFailure
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

	var exhausted *agentports.ChatProvidersFailedError
	if errors.As(err, &exhausted) {
		detail.Providers = exhausted.Failures
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
	Refusal *refusal     `json:"refusal,omitempty"`
	Plan    *PlanRefusal `json:"plan,omitempty"`
	// Providers are the providers asked, when every one of them failed.
	Providers []agentports.ChatProviderFailure `json:"providers,omitempty"`
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
		var detail ModelFailure
		if appErr.Details(&detail) == nil {
			failure.Status = detail.Status
			failure.Retryable = detail.Retryable
			failure.Resting = detail.Resting
			failure.TimedOut = detail.TimedOut
			failure.NoProvider = detail.NoProvider
			failure.SchemaInvalid = detail.SchemaInvalid
			failure.Refusal = detail.Refusal
			failure.Plan = detail.Plan
			failure.Providers = detail.Providers
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
	err := f.err()
	// Every model the reply was asked of failed: the error says which, so the
	// saved reply can list them the way the reader saw them.
	if f != nil && len(f.Providers) > 0 && err != nil {
		return &agentports.ChatProvidersFailedError{Failures: f.Providers, Err: err}
	}

	return err
}

func (f *Failure) err() error {
	switch {
	case f == nil:
		return nil
	case f.Stopped:
		return fmt.Errorf("%s: %w", f.Message, context.Canceled)
	case f.Plan != nil:
		return f.Plan.Err()
	case f.Refusal != nil:
		return f.Refusal.err()
	case f.NoProvider:
		return fmt.Errorf("%s: %w", f.Message, agentports.ErrNoProviderConfigured)
	case f.SchemaInvalid:
		return fmt.Errorf("%s: %w", f.Message, agentports.ErrModelSchemaValidation)
	case f.Status != 0:
		return providerFailureError{f}
	case f.Resting:
		return fmt.Errorf("%s: %w", f.Message, agentports.ErrProvidersResting)
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

type providerFailureError struct{ f *Failure }

func (p providerFailureError) Error() string           { return p.f.Message }
func (p providerFailureError) ProviderStatus() int     { return p.f.Status }
func (p providerFailureError) ProviderRetryable() bool { return p.f.Retryable }
