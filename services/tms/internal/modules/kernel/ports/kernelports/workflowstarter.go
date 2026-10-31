package kernelports

import (
	"context"
	"errors"

	"go.temporal.io/sdk/client"
)

var ErrWorkflowStarterDisabled = errors.New("workflow starter is disabled")

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
	Enabled() bool
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
