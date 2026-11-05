package mocks

import (
	"context"

	"go.temporal.io/sdk/client"
)

type NoopWorkflowStarter struct{}

func (s *NoopWorkflowStarter) StartWorkflow(
	_ context.Context,
	_ client.StartWorkflowOptions,
	_ any,
	_ ...any,
) (client.WorkflowRun, error) {
	return nil, nil
}

func (s *NoopWorkflowStarter) CancelWorkflow(_ context.Context, _, _ string) error {
	return nil
}

func (s *NoopWorkflowStarter) SignalWorkflow(
	_ context.Context,
	_, _, _ string,
	_ any,
) error {
	return nil
}
