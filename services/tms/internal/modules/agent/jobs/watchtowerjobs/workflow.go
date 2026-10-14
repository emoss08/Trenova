package watchtowerjobs

import (
	"time"

	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

var watchtowerRetryPolicy = &temporal.RetryPolicy{
	InitialInterval:    time.Second,
	BackoffCoefficient: 2.0,
	MaximumAttempts:    3,
	MaximumInterval:    30 * time.Second,
}

// A sweep walks every tenant and asks each source what is open, so the
// timeout is generous; the heartbeat is what actually tells Temporal the
// activity is alive.
var watchtowerActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 30 * time.Minute,
	HeartbeatTimeout:    2 * time.Minute,
	RetryPolicy:         watchtowerRetryPolicy,
}

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        WatchtowerReconcileWorkflowName,
			Fn:          WatchtowerReconcileWorkflow,
			TaskQueue:   temporaltype.TaskQueueSystem.String(),
			Description: "Correct every organization's watchtower against its sources",
		},
		{
			Name:        WatchtowerBackfillWorkflowName,
			Fn:          WatchtowerBackfillWorkflow,
			TaskQueue:   temporaltype.TaskQueueSystem.String(),
			Description: "Fill a watchtower from what the sources report open",
		},
		{
			Name:        NoCheckCallSweepWorkflowName,
			Fn:          NoCheckCallSweepWorkflow,
			TaskQueue:   temporaltype.TaskQueueSystem.String(),
			Description: "Raise and clear watchtower items for brokered loads overdue a check call",
		},
		{
			Name:        ExceptionAttentionWorkflowName,
			Fn:          ExceptionAttentionWorkflow,
			TaskQueue:   temporaltype.TaskQueueSystem.String(),
			Description: "Wake snoozed exceptions and escalate overdue ones",
		},
		{
			Name:        ShipmentExceptionSweepWorkflowName,
			Fn:          ShipmentExceptionSweepWorkflow,
			TaskQueue:   temporaltype.TaskQueueSystem.String(),
			Description: "Raise and clear missed stops, missing PODs and exhausted tenders",
		},
		{
			Name:        FactoringSweepWorkflowName,
			Fn:          FactoringSweepWorkflow,
			TaskQueue:   temporaltype.TaskQueueSystem.String(),
			Description: "Raise and clear factoring items: late ready invoices and notices of assignment",
		},
		{
			Name:        WatchtowerRetentionWorkflowName,
			Fn:          WatchtowerRetentionWorkflow,
			TaskQueue:   temporaltype.TaskQueueSystem.String(),
			Description: "Remove watchtower items resolved more than a month ago",
		},
	}
}

func WatchtowerReconcileWorkflow(ctx workflow.Context) (*WatchtowerSweepResult, error) {
	ctx = workflow.WithActivityOptions(ctx, watchtowerActivityOptions)

	var a *Activities
	result := new(WatchtowerSweepResult)
	if err := workflow.ExecuteActivity(ctx, a.ReconcileActivity).Get(ctx, result); err != nil {
		workflow.GetLogger(ctx).Error("Watchtower reconcile workflow failed", "error", err)

		return nil, err
	}

	return result, nil
}

func WatchtowerBackfillWorkflow(
	ctx workflow.Context,
	input *WatchtowerBackfillInput,
) (*WatchtowerSweepResult, error) {
	ctx = workflow.WithActivityOptions(ctx, watchtowerActivityOptions)

	var a *Activities
	result := new(WatchtowerSweepResult)
	if err := workflow.ExecuteActivity(ctx, a.BackfillActivity, input).
		Get(ctx, result); err != nil {
		workflow.GetLogger(ctx).Error("Watchtower backfill workflow failed", "error", err)

		return nil, err
	}

	return result, nil
}

func WatchtowerRetentionWorkflow(ctx workflow.Context) (*WatchtowerRetentionResult, error) {
	ctx = workflow.WithActivityOptions(ctx, watchtowerActivityOptions)

	var a *Activities
	result := new(WatchtowerRetentionResult)
	if err := workflow.ExecuteActivity(ctx, a.WatchtowerRetentionActivity).
		Get(ctx, result); err != nil {
		workflow.GetLogger(ctx).Error("Watchtower retention workflow failed", "error", err)

		return nil, err
	}

	return result, nil
}

func NoCheckCallSweepWorkflow(ctx workflow.Context) (*WatchtowerSweepResult, error) {
	ctx = workflow.WithActivityOptions(ctx, watchtowerActivityOptions)

	var a *Activities
	result := new(WatchtowerSweepResult)
	if err := workflow.ExecuteActivity(ctx, a.NoCheckCallSweepActivity).
		Get(ctx, result); err != nil {
		workflow.GetLogger(ctx).Error("No check call sweep workflow failed", "error", err)

		return nil, err
	}

	return result, nil
}

func ExceptionAttentionWorkflow(ctx workflow.Context) (*ExceptionAttentionResult, error) {
	ctx = workflow.WithActivityOptions(ctx, watchtowerActivityOptions)

	var a *Activities
	result := new(ExceptionAttentionResult)
	if err := workflow.ExecuteActivity(ctx, a.ExceptionAttentionActivity).
		Get(ctx, result); err != nil {
		workflow.GetLogger(ctx).Error("Exception attention workflow failed", "error", err)

		return nil, err
	}

	return result, nil
}

func ShipmentExceptionSweepWorkflow(ctx workflow.Context) (*WatchtowerSweepResult, error) {
	ctx = workflow.WithActivityOptions(ctx, watchtowerActivityOptions)

	var a *Activities
	result := new(WatchtowerSweepResult)
	if err := workflow.ExecuteActivity(ctx, a.ShipmentExceptionSweepActivity).
		Get(ctx, result); err != nil {
		workflow.GetLogger(ctx).Error("Shipment exception sweep workflow failed", "error", err)

		return nil, err
	}

	return result, nil
}

func FactoringSweepWorkflow(ctx workflow.Context) (*WatchtowerSweepResult, error) {
	ctx = workflow.WithActivityOptions(ctx, watchtowerActivityOptions)

	var a *Activities
	result := new(WatchtowerSweepResult)
	if err := workflow.ExecuteActivity(ctx, a.FactoringSweepActivity).
		Get(ctx, result); err != nil {
		workflow.GetLogger(ctx).Error("Factoring sweep workflow failed", "error", err)

		return nil, err
	}

	return result, nil
}
