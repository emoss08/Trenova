package dispatchcoveragewatch

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttrigger"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenteventports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/dispatchconsoleports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/platformcatalogports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchcontrolports"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Controls  dispatchcontrolports.DispatchControlRepository
	Requester dispatchconsoleports.CoverageRefreshRequester
	Features  platformcatalogports.FeatureGate
	Logger    *zap.Logger
}

// Watcher asks for an organization's coverage to be planned again when the
// board gains a move nobody covers: a shipment created, or a move unassigned.
// Only an organization that keeps coverage proposed is asked for.
type Watcher struct {
	controls  dispatchcontrolports.DispatchControlRepository
	requester dispatchconsoleports.CoverageRefreshRequester
	features  platformcatalogports.FeatureGate
	l         *zap.Logger
}

var _ agenteventports.AgentEventObserver = (*Watcher)(nil)

func New(p Params) *Watcher {
	return &Watcher{
		controls:  p.Controls,
		requester: p.Requester,
		features:  p.Features,
		l:         p.Logger.Named("service.dispatch-coverage-watch"),
	}
}

func (w *Watcher) ObserveAgentEvent(ctx context.Context, event *agenteventports.AgentEvent) {
	if !uncoversWork(event.Kind) || !w.plannerOnPlan(ctx) {
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

func (w *Watcher) plannerOnPlan(ctx context.Context) bool {
	for _, feature := range []platformcatalog.FeatureKey{
		platformcatalog.FeatureDispatchIntelligence,
		platformcatalog.FeatureAIDesk,
	} {
		on, err := w.features.Enabled(ctx, feature)
		if err != nil {
			w.l.Warn("could not ask whether the planner is on the plan; coverage not re-planned",
				zap.String("feature", string(feature)),
				zap.Error(err),
			)
			return false
		}
		if !on {
			return false
		}
	}
	return true
}

func uncoversWork(kind agenttrigger.EventKind) bool {
	switch kind {
	case agenttrigger.EventShipmentCreated, agenttrigger.EventShipmentMoveUnassigned:
		return true
	default:
		return false
	}
}
