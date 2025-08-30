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
	// Configure activity options for bulk deletion operations
	// This operation may process large amounts of data across multiple organizations
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Minute, // Allow up to 5 minutes for bulk deletion
		HeartbeatTimeout:    30 * time.Second, // Heartbeat every 30 seconds for long operations
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