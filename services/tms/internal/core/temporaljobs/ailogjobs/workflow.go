package ailogjobs

import (
	"time"

	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	AILogTaskQueue = "ailog-queue"
)

func RegisterWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        "InsertAILogWorkflow",
			Fn:          InsertAILogWorkflow,
			TaskQueue:   AILogTaskQueue,
			Description: "Insert AI operation log to database",
		},
	}
}

func InsertAILogWorkflow(
	ctx workflow.Context,
	payload *InsertAILogPayload,
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
		ExecuteActivity(sessionCtx, a.InsertAILogActivity, &payload).
		Get(sessionCtx, nil)
	if err != nil {
		return err
	}

	return nil
}
