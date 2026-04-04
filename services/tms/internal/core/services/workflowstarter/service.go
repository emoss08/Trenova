package workflowstarter

import (
	"context"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"go.temporal.io/sdk/client"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	TemporalClient client.Client `optional:"true"`
}

type Service struct {
	client client.Client
}

var _ serviceports.WorkflowStarter = (*Service)(nil)

func New(p Params) serviceports.WorkflowStarter {
	return &Service{client: p.TemporalClient}
}

func (s *Service) StartWorkflow(
	ctx context.Context,
	options client.StartWorkflowOptions,
	workflow any,
	args ...any,
) (client.WorkflowRun, error) {
	if s.client == nil {
		return nil, serviceports.ErrWorkflowStarterDisabled
	}

	return s.client.ExecuteWorkflow(ctx, options, workflow, args...)
}

func (s *Service) Enabled() bool {
	return s.client != nil
}
