package kernelports

import (
	"context"

	"go.temporal.io/sdk/client"
)

type WorkflowStarter interface {
	StartWorkflow(
		ctx context.Context,
		options client.StartWorkflowOptions,
		workflow any,
		args ...any,
	) (client.WorkflowRun, error)
	CancelWorkflow(ctx context.Context, workflowID, runID string) error
	SignalWorkflow(
		ctx context.Context,
		workflowID, runID, signalName string,
		arg any,
	) error
}

// WorkflowResults reads what a finished workflow returned. It waits for a
// workflow still running, so a caller bounds it with ctx.
type WorkflowResults interface {
	WorkflowResult(ctx context.Context, workflowID string, out any) error
}

type WorkflowSignalStarter interface {
	SignalWithStartWorkflow(
		ctx context.Context,
		workflowID, signalName string,
		signalArg any,
		options client.StartWorkflowOptions,
		workflow any,
		args ...any,
	) (client.WorkflowRun, error)
}
