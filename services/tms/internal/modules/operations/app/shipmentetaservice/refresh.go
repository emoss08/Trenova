package shipmentetaservice

import (
	"context"
	"fmt"
	"slices"

	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/operations/app/etaprojection"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/stopeta"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchcontrolports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/stopetaports"
	"github.com/emoss08/trenova/internal/modules/places/ports/routeestimateports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
	"github.com/emoss08/trenova/shared/typeutils"
	"go.uber.org/zap"
)

const (
	RefreshBatchSize    = 100
	RealtimeResource    = "shipmentEta"
	maxTrackedShipments = 20000
)

func (r *Refresher) RefreshTenant(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*stopetaports.RefreshResult, error) {
	ids, err := r.stopEtas.ListTrackedShipmentIDs(ctx, &stopetaports.ListTrackedShipmentIDsRequest{
		TenantInfo: tenantInfo,
		Statuses:   stopeta.TrackedShipmentStatuses(),
		Limit:      maxTrackedShipments,
	})
	if err != nil {
		return nil, err
	}

	return r.RefreshShipments(ctx, tenantInfo, ids)
}

func (r *Refresher) RefreshShipments(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentIDs []pulid.ID,
) (*stopetaports.RefreshResult, error) {
	result := &stopetaports.RefreshResult{}
	ids := sliceutils.Dedupe(slices.DeleteFunc(
		slices.Clone(shipmentIDs),
		func(id pulid.ID) bool { return id.IsNil() },
	))
	if len(ids) == 0 {
		return result, nil
	}

	settings, err := r.loadSettings(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	changed := make([]pulid.ID, 0, len(ids))
	for start := 0; start < len(ids); start += RefreshBatchSize {
		batch := ids[start:min(start+RefreshBatchSize, len(ids))]
		batchChanged, bErr := r.refreshBatch(ctx, &refreshBatch{
			tenant:   tenantInfo,
			ids:      batch,
			settings: settings,
			result:   result,
		})
		if bErr != nil {
			return nil, bErr
		}
		changed = append(changed, batchChanged...)
	}

	r.announce(ctx, tenantInfo, changed)

	return result, nil
}

type refreshBatch struct {
	tenant   pagination.TenantInfo
	ids      []pulid.ID
	settings *dispatchcontrol.DispatchControl
	result   *stopetaports.RefreshResult
}

func (r *Refresher) refreshBatch(ctx context.Context, b *refreshBatch) ([]pulid.ID, error) {
	entities, err := r.shipments.ListTrackingShipments(
		ctx,
		&shipmentports.ListTrackingShipmentsRequest{TenantInfo: b.tenant, ShipmentIDs: b.ids},
	)
	if err != nil {
		return nil, err
	}

	tracked := make([]*shipment.Shipment, 0, len(entities))
	for _, entity := range entities {
		if entity != nil && stopeta.TrackedShipment(entity.Status) {
			tracked = append(tracked, entity)
		}
	}

	sources, err := r.collectForRefresh(ctx, b.tenant, tracked)
	if err != nil {
		return nil, err
	}

	now := r.now().Unix()
	plans := make([]*etaprojection.Plan, 0, len(tracked))
	legs := make([]routeestimateports.Leg, 0, len(tracked)*2)
	for _, entity := range tracked {
		plan := etaprojection.Prepare(&etaprojection.Input{
			Now:         now,
			Shipment:    entity,
			Assignments: sources.assignments,
			Positions:   sources.positions,
			HOS:         sources.hos,
			HOSExempt:   sources.exempt,
			CheckCalls:  sources.checkCalls,
			Settings:    settingsFor(b.settings, entity),
		})
		plans = append(plans, plan)
		legs = append(legs, plan.Legs()...)
	}

	estimates, err := r.estimateLegs(ctx, b.tenant, legs)
	if err != nil {
		return nil, err
	}

	rows := make([]*stopeta.StopETA, 0, len(legs)+len(plans))
	offset := 0
	for _, plan := range plans {
		count := len(plan.Legs())
		rows = append(rows, plan.Project(estimates[offset:offset+count])...)
		offset += count
	}

	previous, err := r.sources.storedRows(ctx, b.tenant, b.ids)
	if err != nil {
		return nil, err
	}

	if err = r.stopEtas.ReplaceForShipments(ctx, &stopetaports.ReplaceForShipmentsRequest{
		TenantInfo:  b.tenant,
		ShipmentIDs: b.ids,
		Rows:        rows,
	}); err != nil {
		return nil, err
	}

	b.result.Shipments += len(b.ids)
	b.result.Rows += len(rows)
	current := make(map[pulid.ID][]*stopeta.StopETA, len(tracked))
	for _, row := range rows {
		current[row.ShipmentID] = append(current[row.ShipmentID], row)
		if row.NeedsAttention() {
			b.result.Attention++
		}
	}

	changed := make([]pulid.ID, 0, len(b.ids))
	for _, id := range b.ids {
		if RowsDiffer(previous[id], current[id]) {
			changed = append(changed, id)
		}
	}

	return changed, nil
}

func (r *Refresher) estimateLegs(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	legs []routeestimateports.Leg,
) ([]routeestimateports.LegEstimate, error) {
	estimates := make([]routeestimateports.LegEstimate, 0, len(legs))
	for start := 0; start < len(legs); start += routeestimateports.MaxLegsPerEstimate {
		batch := legs[start:min(start+routeestimateports.MaxLegsPerEstimate, len(legs))]
		out, err := r.routes.EstimateLegs(ctx, tenantInfo, batch)
		if err != nil {
			return nil, err
		}
		if len(out) != len(batch) {
			return nil, fmt.Errorf(
				"route estimator returned %d estimates for %d legs",
				len(out),
				len(batch),
			)
		}
		estimates = append(estimates, out...)
	}
	return estimates, nil
}

func (r *Refresher) loadSettings(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*dispatchcontrol.DispatchControl, error) {
	control, err := r.controls.GetByOrgID(ctx, dispatchcontrolports.GetDispatchControlRequest{
		TenantInfo: tenantInfo,
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return dispatchcontrol.NewDefaultDispatchControl(tenantInfo.OrgID, tenantInfo.BuID), nil
		}
		return nil, err
	}
	return control, nil
}

func settingsFor(
	control *dispatchcontrol.DispatchControl,
	entity *shipment.Shipment,
) etaprojection.Settings {
	var customerThreshold *int16
	if entity.Customer != nil {
		customerThreshold = entity.Customer.EtaRiskThresholdMinutes
	}

	return etaprojection.Settings{
		ThresholdMinutes: int64(control.EtaThresholdMinutes(customerThreshold)),
		DwellMinutes:     int64(control.EtaDwell()),
	}
}

func (r *Refresher) announce(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	changed []pulid.ID,
) {
	if len(changed) == 0 {
		return
	}

	for _, observer := range r.observers {
		if observer != nil {
			observer.StopEtasRefreshed(ctx, tenantInfo, changed)
		}
	}

	for _, id := range changed {
		err := r.realtime.PublishResourceInvalidation(
			ctx,
			&realtimeports.PublishResourceInvalidationRequest{
				OrganizationID: tenantInfo.OrgID,
				BusinessUnitID: tenantInfo.BuID,
				Resource:       RealtimeResource,
				Action:         "updated",
				RecordID:       id,
			},
		)
		if err != nil {
			r.l.Warn("failed to publish shipment eta invalidation",
				zap.String("shipmentId", id.String()),
				zap.Error(err))
		}
	}
}

func RowsDiffer(before, after []*stopeta.StopETA) bool {
	if len(before) != len(after) {
		return true
	}
	byStop := make(map[pulid.ID]*stopeta.StopETA, len(before))
	for _, row := range before {
		byStop[row.StopID] = row
	}
	for _, row := range after {
		old, ok := byStop[row.StopID]
		if !ok || rowDiffers(old, row) {
			return true
		}
	}
	return false
}

func rowDiffers(a, b *stopeta.StopETA) bool {
	return a.Verdict != b.Verdict ||
		a.Basis != b.Basis ||
		a.Arrived != b.Arrived ||
		a.InputStale != b.InputStale ||
		!typeutils.EqualPtr(a.SlackMinutes, b.SlackMinutes) ||
		minuteOf(a.EstimatedArrival) != minuteOf(b.EstimatedArrival) ||
		minuteOf(a.WindowEnd) != minuteOf(b.WindowEnd)
}

func minuteOf(v *int64) int64 {
	if v == nil {
		return -1
	}
	return *v / 60
}
