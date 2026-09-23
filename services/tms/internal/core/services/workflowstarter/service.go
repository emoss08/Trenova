//nolint:gocritic // existing value-shaped APIs and hot-path helpers are intentionally stable
package workflowstarter

import (
	"context"
	"errors"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"go.temporal.io/api/serviceerror"
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

	run, err := s.client.ExecuteWorkflow(ctx, options, workflow, args...)
	return run, unreachable(err)
}

func (s *Service) CancelWorkflow(ctx context.Context, workflowID, runID string) error {
	if s.client == nil {
		return serviceports.ErrWorkflowStarterDisabled
	}

	return unreachable(s.client.CancelWorkflow(ctx, workflowID, runID))
}

func (s *Service) SignalWorkflow(
	ctx context.Context,
	workflowID, runID, signalName string,
	arg any,
) error {
	if s.client == nil {
		return serviceports.ErrWorkflowStarterDisabled
	}

	return unreachable(s.client.SignalWorkflow(ctx, workflowID, runID, signalName, arg))
}

func (s *Service) Enabled() bool {
	return s.client != nil
}

// unreachable turns "Temporal is down" into something a person can act on.
//
// The client connects lazily, so an outage no longer shows up once at boot as
// a nil client that every feature then treats as "background work is off"
// until the next restart. It shows up here, on the call that hit it, and the
// same call succeeds again once Temporal is back. Every other error passes
// through untouched, because only this one is about availability rather than
// the request.
func unreachable(err error) error {
	var down *serviceerror.Unavailable
	if !errors.As(err, &down) {
		return err
	}

	return errortypes.NewBusinessError(
		"Background work is temporarily unavailable. Try again in a moment.",
	).WithInternal(err)
}
