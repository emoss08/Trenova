package dispatchcoveragewatch

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttrigger"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenteventports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/dispatchconsoleports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchcontrolports"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Controls  dispatchcontrolports.DispatchControlRepository
	Requester dispatchconsoleports.CoverageRefreshRequester
	Logger    *zap.Logger
}

// Watcher asks for an organization's coverage to be planned again when the
// board gains a move nobody covers: a shipment created, or a move unassigned.
// Only an organization that keeps coverage proposed is asked for.
type Watcher struct {
	controls  dispatchcontrolports.DispatchControlRepository
	requester dispatchconsoleports.CoverageRefreshRequester
	l         *zap.Logger
}

var _ agenteventports.AgentEventObserver = (*Watcher)(nil)

func New(p Params) *Watcher {
	return &Watcher{
		controls:  p.Controls,
		requester: p.Requester,
		l:         p.Logger.Named("service.dispatch-coverage-watch"),
	}
}

func (w *Watcher) ObserveAgentEvent(ctx context.Context, event *agenteventports.AgentEvent) {
	if !uncoversWork(event.Kind) {
		return
	}

	control, err := w.controls.GetByOrgID(
		ctx,
		dispatchcontrolports.GetDispatchControlRequest{TenantInfo: event.TenantInfo},
	)
	if err != nil {
		w.l.Warn("dispatch control unavailable; coverage not re-planned",
			zap.String("orgId", event.TenantInfo.OrgID.String()),
			zap.Error(err),
		)
		return
	}
	if !control.EnableAutoAssignment || !control.AutoProposeCoverage {
		return
	}

	w.requester.RequestRefresh(ctx, event.TenantInfo)
}

func uncoversWork(kind agenttrigger.EventKind) bool {
	switch kind {
	case agenttrigger.EventShipmentCreated, agenttrigger.EventShipmentMoveUnassigned:
		return true
	default:
		return false
	}
}
