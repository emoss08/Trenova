package bulkeditjobs

import (
	"errors"
	"time"

	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

var prepareOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 5 * time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:        2 * time.Second,
		BackoffCoefficient:     2.0,
		MaximumAttempts:        3,
		MaximumInterval:        15 * time.Second,
		NonRetryableErrorTypes: []string{ErrTypeNotRunnable},
	},
}

var batchOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 10 * time.Minute,
	HeartbeatTimeout:    2 * time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:        5 * time.Second,
		BackoffCoefficient:     2.0,
		MaximumAttempts:        3,
		MaximumInterval:        time.Minute,
		NonRetryableErrorTypes: []string{ErrTypeNotRunnable},
	},
}

var finalizeOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 2 * time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    2 * time.Second,
		BackoffCoefficient: 2.0,
		MaximumAttempts:    10,
		MaximumInterval:    30 * time.Second,
	},
}

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{{
		Name:        BulkEditWorkflowName,
		Fn:          BulkEditWorkflow,
		TaskQueue:   temporaltype.TaskQueueSystem.String(),
		Description: "Change one field on many rows, batch by batch, or put them back",
	}}
}

func BulkEditWorkflow(ctx workflow.Context, payload *EditPayload) (*EditResult, error) {
	var a *Activities

	var prepared *PreparedEdit
	prepareCtx := workflow.WithActivityOptions(ctx, prepareOptions)
	if err := workflow.ExecuteActivity(prepareCtx, a.PrepareEditActivity, payload).
		Get(prepareCtx, &prepared); err != nil {
		return finalize(ctx, payload, failureMessage(err), err)
	}

	batchCtx := workflow.WithActivityOptions(ctx, batchOptions)
	maxBatches := prepared.TotalCount/BatchSize + 2
	for range maxBatches {
		var processed int
		if err := workflow.ExecuteActivity(batchCtx, a.ProcessBatchActivity, &BatchPayload{
			BasePayload: payload.BasePayload,
			EditID:      payload.EditID,
			Limit:       BatchSize,
		}).Get(batchCtx, &processed); err != nil {
			return finalize(ctx, payload, failureMessage(err), err)
		}
		if processed == 0 {
			break
		}
	}

	return finalize(ctx, payload, "", nil)
}

func finalize(
	ctx workflow.Context,
	payload *EditPayload,
	failure string,
	cause error,
) (*EditResult, error) {
	finalizeCtx, cancel := workflow.NewDisconnectedContext(ctx)
	defer cancel()
	finalizeCtx = workflow.WithActivityOptions(finalizeCtx, finalizeOptions)

	var result *EditResult
	if err := workflow.ExecuteActivity(
		finalizeCtx,
		(*Activities)(nil).FinalizeEditActivity,
		&FinalizePayload{
			BasePayload: payload.BasePayload,
			EditID:      payload.EditID,
			Failure:     failure,
		},
	).Get(finalizeCtx, &result); err != nil {
		return nil, err
	}
	return result, cause
}

func failureMessage(err error) string {
	var appErr *temporal.ApplicationError
	if errors.As(err, &appErr) {
		return appErr.Message()
	}
	if temporal.IsCanceledError(err) {
		return "The bulk edit was stopped before every row was reached"
	}
	return "The bulk edit stopped because of an unexpected error"
}
