package billingtransferjobs

import (
	"errors"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/billingtransfer"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

var prepareActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 5 * time.Minute,
	HeartbeatTimeout:    time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:        2 * time.Second,
		BackoffCoefficient:     2.0,
		MaximumAttempts:        3,
		MaximumInterval:        15 * time.Second,
		NonRetryableErrorTypes: []string{ErrTypeTransferValidation},
	},
}

var batchActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 10 * time.Minute,
	HeartbeatTimeout:    90 * time.Second,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:        5 * time.Second,
		BackoffCoefficient:     2.0,
		MaximumAttempts:        3,
		MaximumInterval:        time.Minute,
		NonRetryableErrorTypes: []string{ErrTypeTransferValidation},
	},
}

var finalizeActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 2 * time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    2 * time.Second,
		BackoffCoefficient: 2.0,
		MaximumAttempts:    10,
		MaximumInterval:    30 * time.Second,
	},
}

var reconcileActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 10 * time.Minute,
	HeartbeatTimeout:    time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    5 * time.Second,
		BackoffCoefficient: 2.0,
		MaximumAttempts:    3,
		MaximumInterval:    time.Minute,
	},
}

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        BulkBillingTransferWorkflowName,
			Fn:          BulkBillingTransferWorkflow,
			TaskQueue:   temporaltype.TaskQueueBilling.String(),
			Description: "Move a run's shipments into the billing queue, batch by batch",
		},
		{
			Name:        ReconcileZombieRunsWorkflowName,
			Fn:          ReconcileZombieBillingTransferRunsWorkflow,
			TaskQueue:   temporaltype.TaskQueueBilling.String(),
			Description: "Fail billing transfer runs whose workflow is no longer running",
		},
	}
}

// BulkBillingTransferWorkflow walks a run's shipments in batches.
//
// The loop is driven by what is still Pending in the database rather than by an
// index the workflow carries, so a retried activity, a worker restart and a
// resumed run all converge on the same place: whatever has not been answered
// for yet. A batch that answers for nothing means the run is done.
func BulkBillingTransferWorkflow(
	ctx workflow.Context,
	payload *TransferRunPayload,
) (*TransferRunResult, error) {
	var a *Activities

	cancelCh := workflow.GetSignalChannel(ctx, CancelSignalName)

	var prepared *PreparedRun
	prepareCtx := workflow.WithActivityOptions(ctx, prepareActivityOptions)
	if err := workflow.ExecuteActivity(
		prepareCtx, a.PrepareRunActivity, payload,
	).Get(prepareCtx, &prepared); err != nil {
		return finalizeFailure(ctx, payload, err)
	}

	canceled := false
	batchCtx := workflow.WithActivityOptions(ctx, batchActivityOptions)

	// A batch that answers for nothing ends the loop, but a bound keeps a
	// misbehaving activity from spinning forever against Temporal's history
	// limit rather than failing where somebody can see it.
	maxBatches := prepared.TotalCount/prepared.BatchSize + 2
	for range maxBatches {
		var signal any
		if cancelCh.ReceiveAsync(&signal) {
			canceled = true
			break
		}

		var batch *ProcessBatchResult
		if err := workflow.ExecuteActivity(
			batchCtx, a.ProcessBatchActivity, &ProcessBatchPayload{
				BasePayload: payload.BasePayload,
				RunID:       payload.RunID,
				Limit:       prepared.BatchSize,
			},
		).Get(batchCtx, &batch); err != nil {
			return finalizeFailure(ctx, payload, err)
		}

		if batch.CancelRequested {
			canceled = true
			break
		}
		if batch.Processed == 0 {
			break
		}
	}

	status := billingtransfer.RunStatusCompleted
	message := ""
	if canceled {
		status = billingtransfer.RunStatusCanceled
		message = "The transfer was stopped before every shipment was checked"
	}

	var result *TransferRunResult
	finalizeCtx := workflow.WithActivityOptions(ctx, finalizeActivityOptions)
	if err := workflow.ExecuteActivity(
		finalizeCtx, a.FinalizeRunActivity, &FinalizeRunPayload{
			BasePayload:    payload.BasePayload,
			RunID:          payload.RunID,
			Status:         status,
			FailureMessage: message,
		},
	).Get(finalizeCtx, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// finalizeFailure records a terminal state for the run on every exit path,
// including cancellation of the workflow itself, so a run is never left sitting
// in Queued or Running with nobody coming back for it.
func finalizeFailure(
	ctx workflow.Context,
	payload *TransferRunPayload,
	cause error,
) (*TransferRunResult, error) {
	status := billingtransfer.RunStatusFailed
	if temporal.IsCanceledError(cause) || errors.Is(ctx.Err(), workflow.ErrCanceled) {
		status = billingtransfer.RunStatusCanceled
	}

	// Finalization has to run even when the workflow itself was canceled.
	finalizeCtx, cancel := workflow.NewDisconnectedContext(ctx)
	defer cancel()
	finalizeCtx = workflow.WithActivityOptions(finalizeCtx, finalizeActivityOptions)

	var result *TransferRunResult
	if err := workflow.ExecuteActivity(
		finalizeCtx, (*Activities)(nil).FinalizeRunActivity, &FinalizeRunPayload{
			BasePayload:    payload.BasePayload,
			RunID:          payload.RunID,
			Status:         status,
			FailureMessage: failureMessageFrom(cause),
		},
	).Get(finalizeCtx, &result); err != nil {
		return nil, err
	}

	return result, cause
}

// failureMessageFrom turns a workflow error into something a biller can act on.
// The shipments themselves already carry their own reasons on their item rows;
// this is only about why the run as a whole stopped.
func failureMessageFrom(err error) string {
	if temporal.IsCanceledError(err) {
		return "The transfer was stopped before every shipment was checked"
	}

	var timeoutErr *temporal.TimeoutError
	if errors.As(err, &timeoutErr) {
		return "The transfer took longer than allowed and was stopped — retry the shipments it did not reach"
	}

	var appErr *temporal.ApplicationError
	if errors.As(err, &appErr) {
		return appErr.Message()
	}

	return "The transfer stopped because of an unexpected error — retry the shipments it did not reach"
}

func ReconcileZombieBillingTransferRunsWorkflow(
	ctx workflow.Context,
) (*ReconcileZombieRunsResult, error) {
	ctx = workflow.WithActivityOptions(ctx, reconcileActivityOptions)

	var a *Activities
	var result *ReconcileZombieRunsResult
	if err := workflow.ExecuteActivity(
		ctx, a.ReconcileZombieRunsActivity,
	).Get(ctx, &result); err != nil {
		return nil, err
	}

	return result, nil
}
