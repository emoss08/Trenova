package accessjobs

import (
	"time"

	"github.com/emoss08/trenova/internal/modules/identity/app/userservice"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

var sweepOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 10 * time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    10 * time.Second,
		BackoffCoefficient: 2,
		MaximumInterval:    5 * time.Minute,
		MaximumAttempts:    3,
	},
}

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{{
		Name:      SweepWorkflowName,
		Fn:        SweepOrphanedOrganizationAccessWorkflow,
		TaskQueue: temporaltype.TaskQueueSystem.String(),
		Description: "Revoke the roles and sessions people still hold in organizations " +
			"they no longer belong to",
	}}
}

func SweepOrphanedOrganizationAccessWorkflow(
	ctx workflow.Context,
) (*userservice.OrganizationAccessSweepResult, error) {
	var a *Activities
	ctx = workflow.WithActivityOptions(ctx, sweepOptions)

	result := new(userservice.OrganizationAccessSweepResult)
	err := workflow.ExecuteActivity(ctx, a.SweepOrphanedOrganizationAccessActivity).
		Get(ctx, result)

	return result, err
}
