package dispatchjobs

import (
	"time"

	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

var dispatchRetryPolicy = &temporal.RetryPolicy{
	InitialInterval:    time.Second,
	BackoffCoefficient: 2.0,
	MaximumAttempts:    3,
	MaximumInterval:    30 * time.Second,
}

var dispatchActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 15 * time.Minute,
	HeartbeatTimeout:    time.Minute,
	RetryPolicy:         dispatchRetryPolicy,
}

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        HorizonPlanSweepWorkflowName,
			Fn:          HorizonPlanSweepWorkflow,
			TaskQueue:   temporaltype.TaskQueueSystem.String(),
			Description: "Re-plan the dispatch horizon for organizations running horizon planning",
		},
	}
}

func HorizonPlanSweepWorkflow(ctx workflow.Context) (*HorizonPlanSweepResult, error) {
	ctx = workflow.WithActivityOptions(ctx, dispatchActivityOptions)

	var a *Activities
	result := new(HorizonPlanSweepResult)
	if err := workflow.ExecuteActivity(
		ctx,
		a.HorizonPlanSweepActivity,
	).Get(ctx, result); err != nil {
		workflow.GetLogger(ctx).Error("Horizon plan sweep workflow failed", "error", err)
		return nil, err
	}

	workflow.GetLogger(ctx).Info("Horizon plan sweep workflow completed",
		"tenantsScanned", result.TenantsScanned,
		"tenantsPlanned", result.TenantsPlanned,
		"tenantsFailed", result.TenantsFailed,
		"movesPlanned", result.MovesPlanned,
		"movesUncovered", result.MovesUncovered,
		"toursBuilt", result.ToursBuilt,
		"chainedMoves", result.ChainedMoves,
	)

	return result, nil
}
