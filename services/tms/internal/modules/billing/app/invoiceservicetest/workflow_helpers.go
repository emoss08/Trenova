package invoiceservicetest

import (
	"context"

	"go.temporal.io/sdk/client"
)

type WorkflowRun struct {
	ID    string
	RunID string
}

func (f WorkflowRun) GetID() string {
	return f.ID
}

func (f WorkflowRun) GetRunID() string {
	return f.RunID
}

func (f WorkflowRun) Get(context.Context, any) error {
	return nil
}

func (f WorkflowRun) GetWithOptions(
	context.Context,
	any,
	client.WorkflowRunGetOptions,
) error {
	return nil
}

// GetFirstExecutionRunID identifies the run a continue-as-new chain started
// from. Nothing here continues as new, so the first run is this one.
func (f WorkflowRun) GetFirstExecutionRunID() string {
	return f.RunID
}
