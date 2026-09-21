package briefingjobs

import (
	"time"

	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

var briefingRetryPolicy = &temporal.RetryPolicy{
	InitialInterval:    time.Second,
	BackoffCoefficient: 2.0,
	MaximumAttempts:    3,
	MaximumInterval:    30 * time.Second,
}

// The sweep walks every tenant, and each due one runs a handful of
// aggregates plus a model call per role, so the timeout is generous and
// the heartbeat is what tells Temporal it is alive. It must finish well
// inside the hour, because the next firing is the next hour's tenants.
var briefingActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 45 * time.Minute,
	HeartbeatTimeout:    3 * time.Minute,
	RetryPolicy:         briefingRetryPolicy,
}

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        DailyBriefingWorkflowName,
			Fn:          DailyBriefingWorkflow,
			TaskQueue:   temporaltype.TaskQueueSystem.String(),
			Description: "Write the morning briefing for every organization whose hour has come",
		},
		{
			Name:        BriefingRetentionWorkflowName,
			Fn:          BriefingRetentionWorkflow,
			TaskQueue:   temporaltype.TaskQueueSystem.String(),
			Description: "Remove briefings older than the window a reader can page back through",
		},
	}
}

func DailyBriefingWorkflow(ctx workflow.Context) (*DailyBriefingResult, error) {
	ctx = workflow.WithActivityOptions(ctx, briefingActivityOptions)

	var a *Activities
	result := new(DailyBriefingResult)
	if err := workflow.ExecuteActivity(ctx, a.WriteDueBriefingsActivity).Get(ctx, result); err != nil {
		workflow.GetLogger(ctx).Error("Daily briefing workflow failed", "error", err)

		return nil, err
	}

	return result, nil
}

func BriefingRetentionWorkflow(ctx workflow.Context) (*BriefingRetentionResult, error) {
	ctx = workflow.WithActivityOptions(ctx, briefingActivityOptions)

	var a *Activities
	result := new(BriefingRetentionResult)
	if err := workflow.ExecuteActivity(ctx, a.RetentionActivity).Get(ctx, result); err != nil {
		workflow.GetLogger(ctx).Error("Briefing retention workflow failed", "error", err)

		return nil, err
	}

	return result, nil
}
