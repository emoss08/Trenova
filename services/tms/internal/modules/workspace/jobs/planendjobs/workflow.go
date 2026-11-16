package planendjobs

import (
	"time"

	"github.com/emoss08/trenova/internal/modules/kernel/ports/managedfeaturesports"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

var reminderOptions = workflow.ActivityOptions{
	StartToCloseTimeout: time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    10 * time.Second,
		BackoffCoefficient: 2,
		MaximumInterval:    time.Minute,
		MaximumAttempts:    3,
	},
}

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{{
		Name:        ReminderWorkflowName,
		Fn:          RemindManagedPlanEndWorkflow,
		TaskQueue:   temporaltype.TaskQueueSystem.String(),
		Description: reminderDescription,
	}}
}

func RemindManagedPlanEndWorkflow(
	ctx workflow.Context,
) (*managedfeaturesports.PlanEndReminderResult, error) {
	var a *Activities
	ctx = workflow.WithActivityOptions(ctx, reminderOptions)

	result := new(managedfeaturesports.PlanEndReminderResult)
	err := workflow.ExecuteActivity(ctx, a.RemindManagedPlanEndActivity).Get(ctx, result)

	return result, err
}
