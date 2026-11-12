package etacontract

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/workqueueports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/stopetaports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	// DrainKind queues a shipment whose arrival at each open stop needs
	// projecting again; one drain per tenant works them all.
	DrainKind         = "shipment-eta"
	DrainWorkflowName = "ShipmentEtaDrainWorkflow"
	DrainSignalName   = "shipment-eta-drain"
	SweepWorkflowName = "ShipmentEtaSweepWorkflow"
)

type RequesterParams struct {
	fx.In

	Queue   workqueueports.Queue
	Signals kernelports.WorkflowSignalStarter
	Logger  *zap.Logger
}

// Requester queues a shipment for its tenant's ETA drain.
type Requester struct {
	drain *temporaljobs.DrainRequester
	l     *zap.Logger
}

var _ stopetaports.RefreshRequester = (*Requester)(nil)

func NewRequester(p RequesterParams) *Requester {
	l := p.Logger.Named("shipment-eta-requester")

	return &Requester{
		drain: &temporaljobs.DrainRequester{
			Queue:     p.Queue,
			Signals:   p.Signals,
			Kind:      DrainKind,
			Workflow:  DrainWorkflowName,
			TaskQueue: temporaltype.TaskQueueDispatch.String(),
			Signal:    DrainSignalName,
			Summary:   "Project the arrival at each open stop of the tenant's changed shipments",
			Logger:    l,
		},
		l: l,
	}
}

func (r *Requester) RequestRefresh(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentID pulid.ID,
) {
	if err := r.drain.Request(ctx, tenantInfo, shipmentID); err != nil {
		r.l.Warn("could not queue a shipment eta refresh; the sweep projects it",
			zap.String("shipmentId", shipmentID.String()),
			zap.Error(err))
	}
}

// WakeDue wakes every tenant with shipments waiting for a refresh.
func (r *Requester) WakeDue(ctx context.Context, limit int) (int, error) {
	return r.drain.WakeDue(ctx, limit)
}
