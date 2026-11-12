package customertrackingcontract

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/workqueueports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/customertrackingports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	// DrainKind queues a shipment whose customer status emails need
	// evaluating; one drain per tenant works them all.
	DrainKind         = "customer-status"
	DrainWorkflowName = "CustomerStatusDrainWorkflow"
	DrainSignalName   = "customer-status-drain"
	SweepWorkflowName = "CustomerStatusSweepWorkflow"
	PurgeWorkflowName = "CustomerTrackingLinkPurgeWorkflow"
)

type RequesterParams struct {
	fx.In

	Queue   workqueueports.Queue
	Signals kernelports.WorkflowSignalStarter
	Logger  *zap.Logger
}

// Requester queues a shipment for its tenant's customer status drain, so a
// burst of changes to many shipments is evaluated by one workflow.
type Requester struct {
	drain *temporaljobs.DrainRequester
	l     *zap.Logger
}

var _ customertrackingports.EvaluationRequester = (*Requester)(nil)

func NewRequester(p RequesterParams) *Requester {
	l := p.Logger.Named("customer-status-requester")

	return &Requester{
		drain: &temporaljobs.DrainRequester{
			Queue:     p.Queue,
			Signals:   p.Signals,
			Kind:      DrainKind,
			Workflow:  DrainWorkflowName,
			TaskQueue: temporaltype.TaskQueueDispatch.String(),
			Signal:    DrainSignalName,
			Summary:   "Send the customer status emails the tenant's rules ask for",
			Logger:    l,
		},
		l: l,
	}
}

func (r *Requester) RequestEvaluation(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentID pulid.ID,
) {
	if err := r.drain.Request(ctx, tenantInfo, shipmentID); err != nil {
		r.l.Warn("could not queue a customer status evaluation; the sweep evaluates it",
			zap.String("shipmentId", shipmentID.String()),
			zap.Error(err))
	}
}

// WakeDue wakes every tenant with shipments waiting for an evaluation.
func (r *Requester) WakeDue(ctx context.Context, limit int) (int, error) {
	return r.drain.WakeDue(ctx, limit)
}
