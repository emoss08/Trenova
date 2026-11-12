package watchtowershipment

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/stopeta"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchcontrolports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/stopetaports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type EtaAtRiskParams struct {
	fx.In

	StopEtas stopetaports.Repository
	Controls dispatchcontrolports.DispatchControlRepository
}

type EtaAtRiskSource struct {
	stopEtas stopetaports.Repository
	controls dispatchcontrolports.DispatchControlRepository
	now      func() int64
}

func NewEtaAtRiskSource(p EtaAtRiskParams) watchtowerports.WatchtowerSource {
	return &EtaAtRiskSource{stopEtas: p.StopEtas, controls: p.Controls, now: timeutils.NowUnix}
}

func (s *EtaAtRiskSource) Kind() watchtower.SourceKind {
	return watchtower.SourceEtaAtRisk
}

func (s *EtaAtRiskSource) Snapshot(
	ctx context.Context,
	tenant pagination.TenantInfo,
) ([]watchtowerports.WatchtowerItemInput, error) {
	alerts, err := etaAlertsOf(ctx, s.controls, tenant)
	if err != nil {
		return nil, err
	}
	if !alerts.enabled {
		return []watchtowerports.WatchtowerItemInput{}, nil
	}

	stops, err := s.stopEtas.ListAttention(ctx, &stopetaports.ListAttentionRequest{
		TenantInfo: tenant,
		Statuses:   stopeta.TrackedShipmentStatuses(),
	})
	if err != nil {
		return nil, err
	}

	return DescribeEtaRisks(tenant, stops, alerts.missedBefore(s.now())), nil
}

// etaAlerts is what the organization's dispatch settings say about arrival
// alerts: whether they are raised, and how long past its window a stop waits
// before it is a missed stop.
type etaAlerts struct {
	enabled     bool
	missedGrace int64
}

// missedBefore is the window close before which a stop with no arrival is
// already a missed stop.
func (a etaAlerts) missedBefore(now int64) int64 {
	return now - a.missedGrace
}

func etaAlertsOf(
	ctx context.Context,
	controls dispatchcontrolports.DispatchControlRepository,
	tenant pagination.TenantInfo,
) (etaAlerts, error) {
	control, err := controls.GetByOrgID(ctx, dispatchcontrolports.GetDispatchControlRequest{
		TenantInfo: tenant,
	})
	if err != nil {
		if !errortypes.IsNotFoundError(err) {
			return etaAlerts{}, err
		}
		control = nil
	}

	return etaAlerts{
		enabled:     control.EtaAlertsEnabled(),
		missedGrace: int64(control.ExceptionPolicy().MissedStopGrace) * secondsPerMinute,
	}, nil
}

// DescribeEtaRisks raises one item per shipment, for its worst stop still
// ahead. A stop whose window closed before missedBefore with no arrival is no
// longer a projection: it is the missed stop item's, so it is left out here
// and one late stop is not two items.
func DescribeEtaRisks(
	tenant pagination.TenantInfo,
	stops []*stopetaports.AttentionStop,
	missedBefore int64,
) []watchtowerports.WatchtowerItemInput {
	order := make([]pulid.ID, 0, len(stops))
	byShipment := make(map[pulid.ID][]*stopetaports.AttentionStop, len(stops))
	for _, stop := range stops {
		if stop == nil || stop.ETA == nil || !stop.ETA.NeedsAttention() ||
			alreadyMissed(stop.ETA, missedBefore) {
			continue
		}
		id := stop.ETA.ShipmentID
		if _, seen := byShipment[id]; !seen {
			order = append(order, id)
		}
		byShipment[id] = append(byShipment[id], stop)
	}

	items := make([]watchtowerports.WatchtowerItemInput, 0, len(order))
	for _, id := range order {
		if stop := governingStop(byShipment[id]); stop != nil {
			items = append(items, DescribeEtaRisk(tenant, stop))
		}
	}
	return items
}

func alreadyMissed(eta *stopeta.StopETA, missedBefore int64) bool {
	return eta.WindowEnd != nil && *eta.WindowEnd < missedBefore
}

func governingStop(stops []*stopetaports.AttentionStop) *stopetaports.AttentionStop {
	rows := make([]*stopeta.StopETA, 0, len(stops))
	for _, stop := range stops {
		rows = append(rows, stop.ETA)
	}
	governing := stopeta.Governing(rows)
	for _, stop := range stops {
		if stop.ETA == governing {
			return stop
		}
	}
	return nil
}

type EtaAtRiskObserverParams struct {
	fx.In

	StopEtas  stopetaports.Repository
	Controls  dispatchcontrolports.DispatchControlRepository
	Projector watchtowerports.WatchtowerProjector
	Logger    *zap.Logger
}

type EtaAtRiskObserver struct {
	stopEtas  stopetaports.Repository
	controls  dispatchcontrolports.DispatchControlRepository
	projector watchtowerports.WatchtowerProjector
	now       func() int64
	l         *zap.Logger
}

var _ stopetaports.Observer = (*EtaAtRiskObserver)(nil)

func NewEtaAtRiskObserver(p EtaAtRiskObserverParams) stopetaports.Observer {
	return &EtaAtRiskObserver{
		stopEtas:  p.StopEtas,
		controls:  p.Controls,
		projector: p.Projector,
		now:       timeutils.NowUnix,
		l:         p.Logger.Named("watchtower.eta-at-risk"),
	}
}

func (o *EtaAtRiskObserver) StopEtasRefreshed(
	ctx context.Context,
	tenant pagination.TenantInfo,
	shipmentIDs []pulid.ID,
) {
	if len(shipmentIDs) == 0 {
		return
	}

	raised := make(map[string]struct{}, len(shipmentIDs))
	alerts, err := etaAlertsOf(ctx, o.controls, tenant)
	if err != nil {
		o.l.Warn("could not read eta alert settings", zap.Error(err))
		return
	}
	if alerts.enabled {
		stops, lErr := o.stopEtas.ListAttention(ctx, &stopetaports.ListAttentionRequest{
			TenantInfo:  tenant,
			Statuses:    stopeta.TrackedShipmentStatuses(),
			ShipmentIDs: shipmentIDs,
		})
		if lErr != nil {
			o.l.Warn("could not read stop etas needing attention", zap.Error(lErr))
			return
		}
		items := DescribeEtaRisks(tenant, stops, alerts.missedBefore(o.now()))
		for i := range items {
			o.projector.Upsert(ctx, items[i])
			raised[items[i].SourceID] = struct{}{}
		}
	}

	for _, id := range shipmentIDs {
		if _, ok := raised[id.String()]; !ok {
			o.projector.Resolve(ctx, tenant, watchtower.SourceEtaAtRisk, id.String())
		}
	}
}
