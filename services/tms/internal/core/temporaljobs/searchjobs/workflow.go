package searchjobs

import (
	"time"

	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	SearchTaskQueue = "search-queue"
)

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        "IndexEntityWorkflow",
			Fn:          IndexEntityWorkflow,
			TaskQueue:   SearchTaskQueue,
			Description: "Index an entity in search",
		},
	}
}

func IndexEntityWorkflow(
	ctx workflow.Context,
	payload *IndexEntityPayload,
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
	so := &workflow.SessionOptions{
		CreationTimeout:  10 * time.Second,
		ExecutionTimeout: 10 * time.Second,
	}

	sessionCtx, err := workflow.CreateSession(ctx, so)
	if err != nil {
		return err
	}
	defer workflow.CompleteSession(sessionCtx)

	var a *Activities
	err = workflow.
		ExecuteActivity(sessionCtx, a.IndexEntityActivity, &payload).
		Get(sessionCtx, nil)
	if err != nil {
		return err
	}

	return nil
}
