package ratesimjobs

import (
	"time"

	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const RunRateSimulationWorkflowName = "RunRateSimulationWorkflow"

// A simulation is not retried on failure the way an integration poll is.
//
// It walks tens of thousands of shipments and writes as it goes, so a second
// attempt would duplicate every result the first one produced. A failed run is
// left failed with what it managed to price, which somebody can read, and
// re-running is a new simulation.
var runRateSimulationRetryPolicy = &temporal.RetryPolicy{
	MaximumAttempts: 1,
}

var runRateSimulationActivityOptions = workflow.ActivityOptions{
	// A year of a large carrier's freight is the upper bound this is sized for.
	StartToCloseTimeout: 2 * time.Hour,
	// Short enough that a stuck run is noticed, long enough that a slow page
	// of shipments is not mistaken for one.
	HeartbeatTimeout: 2 * time.Minute,
	RetryPolicy:      runRateSimulationRetryPolicy,
}

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:      RunRateSimulationWorkflowName,
			Fn:        RunRateSimulationWorkflow,
			TaskQueue: temporaltype.IntegrationTaskQueue,
			Description: "Replay a rate agreement against historical shipments and " +
				"report what it would have charged",
		},
	}
}

func RunRateSimulationWorkflow(
	ctx workflow.Context,
	payload *RunSimulationPayload,
) (*RunSimulationResult, error) {
	activityCtx := workflow.WithActivityOptions(ctx, runRateSimulationActivityOptions)

	var a *Activities
	var result *RunSimulationResult

	if err := workflow.ExecuteActivity(
		activityCtx,
		a.RunSimulationActivity,
		payload,
	).Get(ctx, &result); err != nil {
		return nil, err
	}

	return result, nil
}
