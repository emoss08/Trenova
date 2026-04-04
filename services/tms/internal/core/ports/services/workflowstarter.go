package services

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
	Enabled() bool
}
