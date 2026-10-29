package factoringjobs

import (
	"context"
	"errors"
	"time"

	"github.com/emoss08/trenova/internal/modules/billing/jobs/factoringjobs/factoringcontract"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringsubmissionports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	ErrTypeSendRefused  = "FactoringSendRefused"
	sendMaxAttempts     = 6
	prepareMaxAttempts  = 3
	sendDescription     = "Build a factoring schedule's documents and deliver them to the factor, retrying before giving up"
	defaultFailureCause = "The schedule could not be delivered"
)

var prepareSendOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 20 * time.Minute,
	HeartbeatTimeout:    2 * time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:        10 * time.Second,
		BackoffCoefficient:     2.0,
		MaximumInterval:        2 * time.Minute,
		MaximumAttempts:        prepareMaxAttempts,
		NonRetryableErrorTypes: []string{ErrTypeSendRefused},
	},
}

var deliverOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 15 * time.Minute,
	HeartbeatTimeout:    2 * time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:        30 * time.Second,
		BackoffCoefficient:     2.0,
		MaximumInterval:        10 * time.Minute,
		MaximumAttempts:        sendMaxAttempts,
		NonRetryableErrorTypes: []string{ErrTypeSendRefused},
	},
}

var markFailedOptions = workflow.ActivityOptions{
	StartToCloseTimeout: time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    5 * time.Second,
		BackoffCoefficient: 2.0,
		MaximumInterval:    time.Minute,
		MaximumAttempts:    10,
	},
}

type FailPayload struct {
	Send     factoringcontract.SendPayload `json:"send"`
	Attempts int                           `json:"attempts"`
	Cause    string                        `json:"cause"`
}

func FactoringSubmissionSendWorkflow(
	ctx workflow.Context,
	payload *factoringcontract.SendPayload,
) (*factoringsubmissionports.DeliverSubmissionResult, error) {
	var a *Activities

	prepareCtx := workflow.WithActivityOptions(ctx, prepareSendOptions)
	if err := workflow.ExecuteActivity(prepareCtx, a.PrepareFactoringSubmissionActivity, payload).
		Get(prepareCtx, nil); err != nil {
		return nil, fail(ctx, payload, err, prepareMaxAttempts)
	}

	deliverCtx := workflow.WithActivityOptions(ctx, deliverOptions)
	var result factoringsubmissionports.DeliverSubmissionResult
	if err := workflow.ExecuteActivity(deliverCtx, a.DeliverFactoringSubmissionActivity, payload).
		Get(deliverCtx, &result); err != nil {
		return nil, fail(ctx, payload, err, sendMaxAttempts)
	}
	return &result, nil
}

func fail(
	ctx workflow.Context,
	payload *factoringcontract.SendPayload,
	cause error,
	maxAttempts int,
) error {
	var a *Activities
	failCtx := workflow.WithActivityOptions(ctx, markFailedOptions)
	failure := &FailPayload{
		Send:     *payload,
		Attempts: maxAttempts,
		Cause:    failureMessage(cause),
	}
	var appErr *temporal.ApplicationError
	if errors.As(cause, &appErr) && appErr.NonRetryable() {
		failure.Attempts = 1
	}
	if err := workflow.ExecuteActivity(failCtx, a.MarkFactoringSubmissionFailedActivity, failure).
		Get(failCtx, nil); err != nil {
		return err
	}
	return cause
}

func failureMessage(err error) string {
	var appErr *temporal.ApplicationError
	if errors.As(err, &appErr) && appErr.Message() != "" {
		return appErr.Message()
	}
	if err == nil {
		return defaultFailureCause
	}
	return err.Error()
}

func (a *Activities) PrepareFactoringSubmissionActivity(
	ctx context.Context,
	payload *factoringcontract.SendPayload,
) error {
	tenant := payload.TenantInfo()
	ctx = dbscope.WithTenant(ctx, tenant.DBTenant())
	heartbeat(ctx, tenant)
	return refusal(
		a.submissions.PrepareDocuments(ctx, &factoringsubmissionports.DeliverSubmissionRequest{
			TenantInfo:   tenant,
			SubmissionID: payload.SubmissionID,
			Group:        payload.Group,
		}),
	)
}

func (a *Activities) DeliverFactoringSubmissionActivity(
	ctx context.Context,
	payload *factoringcontract.SendPayload,
) (*factoringsubmissionports.DeliverSubmissionResult, error) {
	tenant := payload.TenantInfo()
	ctx = dbscope.WithTenant(ctx, tenant.DBTenant())
	heartbeat(ctx, tenant)
	result, err := a.submissions.Deliver(ctx, &factoringsubmissionports.DeliverSubmissionRequest{
		TenantInfo:   tenant,
		SubmissionID: payload.SubmissionID,
		Group:        payload.Group,
		Attempt:      int(activity.GetInfo(ctx).Attempt),
	})
	if err != nil {
		return nil, refusal(err)
	}
	return result, nil
}

func (a *Activities) MarkFactoringSubmissionFailedActivity(
	ctx context.Context,
	payload *FailPayload,
) error {
	tenant := payload.Send.TenantInfo()
	ctx = dbscope.WithTenant(ctx, tenant.DBTenant())
	return a.submissions.MarkFailed(ctx, &factoringsubmissionports.FailSubmissionRequest{
		TenantInfo:   tenant,
		SubmissionID: payload.Send.SubmissionID,
		Group:        payload.Send.Group,
		Attempts:     payload.Attempts,
		Cause:        payload.Cause,
	})
}

func refusal(err error) error {
	if err == nil {
		return nil
	}
	if errortypes.IsRefusal(err) {
		return temporal.NewNonRetryableApplicationError(
			errortypes.Summary(err),
			ErrTypeSendRefused,
			err,
		)
	}
	return err
}
