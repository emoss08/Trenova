package systemjobs

import (
	"time"

	"github.com/emoss08/trenova/pkg/types/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        "DeleteAuditEntriesWorkflow",
			Fn:          DeleteAuditEntriesWorkflow,
			TaskQueue:   temporaltype.SystemTaskQueue,
			Description: "Delete audit entries older than 120 days",
		},
	}
}

func DeleteAuditEntriesWorkflow(
	ctx workflow.Context,
) error {
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
		HeartbeatTimeout:    2 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumAttempts:    3,
			MaximumInterval:    time.Minute,
		},
	}

	ctx = workflow.WithActivityOptions(ctx, ao)
	var a *Activities

	err := workflow.ExecuteActivity(ctx, a.DeleteAuditEntriesActivity).Get(ctx, nil)
	if err != nil {
		return err
	}

	return nil
}