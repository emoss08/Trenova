package weatheralertjobs

import (
	"time"

	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const PollNWSAlertsWorkflowName = "PollNWSAlertsWorkflow"

var pollNWSAlertsRetryPolicy = &temporal.RetryPolicy{
	InitialInterval:    2 * time.Second,
	BackoffCoefficient: 2.0,
	MaximumAttempts:    3,
	MaximumInterval:    30 * time.Second,
}

var pollNWSAlertsActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 5 * time.Minute,
	HeartbeatTimeout:    30 * time.Second,
	RetryPolicy:         pollNWSAlertsRetryPolicy,
}

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        PollNWSAlertsWorkflowName,
			Fn:          PollNWSAlertsWorkflow,
			TaskQueue:   temporaltype.WeatherAlertTaskQueue,
			Description: "Poll active NWS weather alerts and persist them for all tenants",
		},
	}
}

func PollNWSAlertsWorkflow(ctx workflow.Context) (*PollNWSAlertsResult, error) {
	activityCtx := workflow.WithActivityOptions(ctx, pollNWSAlertsActivityOptions)

	var a *Activities
	var result *PollNWSAlertsResult
	if err := workflow.ExecuteActivity(
		activityCtx,
		a.PollNWSAlertsActivity,
	).Get(ctx, &result); err != nil {
		return nil, err
	}

	if err := workflow.ExecuteActivity(
		activityCtx,
		a.ExpireStaleWeatherAlertsActivity,
	).Get(ctx, nil); err != nil {
		return nil, err
	}

	return result, nil
}
