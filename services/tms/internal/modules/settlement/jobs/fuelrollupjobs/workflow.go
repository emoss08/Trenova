package fuelrollupjobs

import (
	"time"

	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

var refreshOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 10 * time.Minute,
	Summary:             "Rebuild the trailing days of fuel and miles by tractor",
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    5 * time.Second,
		BackoffCoefficient: 2,
		MaximumInterval:    time.Minute,
		MaximumAttempts:    3,
	},
}

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{{
		Name:        RefreshWorkflowName,
		Fn:          RefreshFuelTractorDaysWorkflow,
		TaskQueue:   temporaltype.TaskQueueSystem.String(),
		Description: "Rebuild the trailing days of fuel and miles by tractor",
	}}
}

func RefreshFuelTractorDaysWorkflow(
	ctx workflow.Context,
	input *RefreshInput,
) (*RefreshResult, error) {
	var a *Activities
	var result RefreshResult
	err := workflow.ExecuteActivity(
		workflow.WithActivityOptions(ctx, refreshOptions),
		a.RefreshFuelTractorDaysActivity,
		input,
	).Get(ctx, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}
