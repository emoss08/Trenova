package billingqueuejobs

import (
	"errors"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

var startActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    time.Second,
		BackoffCoefficient: 2.0,
		MaximumAttempts:    5,
		MaximumInterval:    10 * time.Second,
	},
}

// itemActivityOptions retry a transient failure a few times. A refusal — the
// item is not ready, somebody held it — is recorded by the activity itself and
// never comes back as an error, so only real faults are retried.
var itemActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 2 * time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:        2 * time.Second,
		BackoffCoefficient:     2.0,
		MaximumAttempts:        3,
		MaximumInterval:        20 * time.Second,
		NonRetryableErrorTypes: []string{ErrTypeApprovalItem},
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

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        BulkApprovalWorkflowName,
			Fn:          BulkApprovalWorkflow,
			TaskQueue:   temporaltype.TaskQueueBilling.String(),
			Description: "Approve several billing queue items after an undo window",
		},
	}
}

// BulkApprovalWorkflow approves a run's items once its undo window has passed.
//
// The window is a durable timer, so a worker restart inside it neither loses
// the undo nor approves early. Undo arrives as a signal, but the run row is
// what decides: StartApprovalRunActivity moves the run out of the window only
// if no undo was written, in the same conditional update the undo endpoint
// races against, so an undo either stops every write or none.
//
// Each item is approved by its own activity, which re-reads the item and
// approves only if every check still passes. One item failing does not stop
// the others; it is recorded against that item.
func BulkApprovalWorkflow(
	ctx workflow.Context,
	payload *ApprovalRunPayload,
) (*ApprovalRunResult, error) {
	var a *Activities

	undoCh := workflow.GetSignalChannel(ctx, UndoSignalName)
	timerCtx, cancelTimer := workflow.WithCancel(ctx)
	timer := workflow.NewTimer(timerCtx, billingqueue.ApprovalUndoWindowSeconds*time.Second)

	undone := false
	selector := workflow.NewSelector(ctx)
	selector.AddFuture(timer, func(workflow.Future) {})
	selector.AddReceive(undoCh, func(c workflow.ReceiveChannel, _ bool) {
		c.Receive(ctx, nil)
		undone = true
	})
	selector.Select(ctx)
	cancelTimer()

	if undone {
		return finalize(ctx, payload, billingqueue.ApprovalRunUndone, "", billingqueue.ApprovalFailureUndone)
	}

	startCtx := workflow.WithActivityOptions(ctx, startActivityOptions)
	var started bool
	if err := workflow.ExecuteActivity(startCtx, a.StartApprovalRunActivity, payload).
		Get(startCtx, &started); err != nil {
		return finalizeFailure(ctx, payload, err)
	}
	if !started {
		return finalize(ctx, payload, billingqueue.ApprovalRunUndone, "", billingqueue.ApprovalFailureUndone)
	}

	var itemIDs []pulid.ID
	if err := workflow.ExecuteActivity(startCtx, a.ListApprovalRunItemsActivity, payload).
		Get(startCtx, &itemIDs); err != nil {
		return finalizeFailure(ctx, payload, err)
	}

	itemCtx := workflow.WithActivityOptions(ctx, itemActivityOptions)
	for _, itemID := range itemIDs {
		itemPayload := &ApproveItemPayload{
			BasePayload:    payload.BasePayload,
			RunID:          payload.RunID,
			ItemID:         itemID,
			AssignApprover: payload.AssignApprover,
		}
		var result *ApproveItemResult
		err := workflow.ExecuteActivity(itemCtx, a.ApproveQueueItemActivity, itemPayload).
			Get(itemCtx, &result)
		if err == nil {
			continue
		}
		if errors.Is(ctx.Err(), workflow.ErrCanceled) {
			return finalizeFailure(ctx, payload, err)
		}
		// Out of retries: the item is recorded as failed and the run goes on.
		if recErr := workflow.ExecuteActivity(startCtx, a.RecordApprovalItemFailureActivity,
			&RecordItemFailurePayload{
				BasePayload: payload.BasePayload,
				RunID:       payload.RunID,
				ItemID:      itemID,
				Message:     failureMessageFrom(err),
			}).Get(startCtx, nil); recErr != nil {
			return finalizeFailure(ctx, payload, recErr)
		}
	}

	return finalize(ctx, payload, billingqueue.ApprovalRunCompleted, "", billingqueue.ApprovalFailureUnexpected)
}

func finalize(
	ctx workflow.Context,
	payload *ApprovalRunPayload,
	status billingqueue.ApprovalRunStatus,
	message string,
	leftover billingqueue.ApprovalFailureCode,
) (*ApprovalRunResult, error) {
	finalizeCtx := workflow.WithActivityOptions(ctx, finalizeActivityOptions)
	var result *ApprovalRunResult
	if err := workflow.ExecuteActivity(finalizeCtx, (*Activities)(nil).FinalizeApprovalRunActivity,
		&FinalizeApprovalRunPayload{
			BasePayload:    payload.BasePayload,
			RunID:          payload.RunID,
			Status:         status,
			FailureMessage: message,
			LeftoverCode:   leftover,
		}).Get(finalizeCtx, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// finalizeFailure records a terminal state on every exit path, including the
// workflow itself being canceled, so a run never sits in Scheduled or Running
// with nobody coming back for it.
func finalizeFailure(
	ctx workflow.Context,
	payload *ApprovalRunPayload,
	cause error,
) (*ApprovalRunResult, error) {
	disconnected, cancel := workflow.NewDisconnectedContext(ctx)
	defer cancel()

	result, err := finalize(disconnected, payload, billingqueue.ApprovalRunFailed,
		failureMessageFrom(cause), billingqueue.ApprovalFailureUnexpected)
	if err != nil {
		return nil, err
	}

	return result, cause
}

func failureMessageFrom(err error) string {
	var appErr *temporal.ApplicationError
	if errors.As(err, &appErr) {
		return appErr.Message()
	}
	var timeoutErr *temporal.TimeoutError
	if errors.As(err, &timeoutErr) {
		return "Approving took longer than allowed"
	}

	return "Approving stopped because of an unexpected error"
}
