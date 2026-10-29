package documentextractionjobs

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"go.temporal.io/sdk/client"
)

type noopWorkflowStarter struct{}

func (noopWorkflowStarter) StartWorkflow(
	context.Context,
	client.StartWorkflowOptions,
	any,
	...any,
) (client.WorkflowRun, error) {
	return nil, kernelports.ErrWorkflowStarterDisabled
}

func (noopWorkflowStarter) CancelWorkflow(context.Context, string, string) error {
	return nil
}

func (noopWorkflowStarter) SignalWorkflow(context.Context, string, string, string, any) error {
	return nil
}

func (noopWorkflowStarter) Enabled() bool {
	return false
}
