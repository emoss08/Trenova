package shipmentrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentstate"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

const defaultSweepLimit = 100

func (r *repository) GetDelayedShipments(
	ctx context.Context,
	req *shipmentports.GetDelayedShipmentsRequest,
	thresholdMinutes int16,
) ([]*shipment.Shipment, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*shipment.Shipment, error) {
		return delayedShipments(thresholdMinutes, timeutils.NowUnix(), req.Limit).
			List(ctx, r.db.Keel(ctx))
	})
}

func (r *repository) DelayShipments(
	ctx context.Context,
	req *shipmentports.DelayShipmentsRequest,
	thresholdMinutes int16,
) ([]*shipment.Shipment, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) ([]*shipment.Shipment, error) {
		currentTime := timeutils.NowUnix()
		entities, err := delayedShipments(thresholdMinutes, currentTime, req.Limit).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, err
		}

		return r.markDelayed(ctx, entities, scopedTenant)
	})
}

func (r *repository) AutoDelayShipments(ctx context.Context) ([]*shipment.Shipment, error) {
	ctx = dbscope.WithSystem(ctx, "mark late shipments delayed across every organization")
	return dbtx.Write(ctx, r.db, func(ctx context.Context) ([]*shipment.Shipment, error) {
		sp := shipmentdb.Shipments
		entities, err := sp.AllTenants().
			Where(sp.ID.InQuery(autoDelayedShipmentIDs(timeutils.NowUnix(), everyTenant))).
			All(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, err
		}

		return r.markDelayed(ctx, entities, everyTenant)
	})
}

func (r *repository) ListAutoDelayShipmentTenants(
	ctx context.Context,
	limit int,
) ([]pagination.TenantInfo, error) {
	ctx = dbscope.WithSystem(ctx, "list organizations with automatic shipment delay on")
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]pagination.TenantInfo, error) {
		sc := tenantdb.ShipmentControls
		return r.controlTenants(ctx, sc.AutoDelayShipments.IsTrue(), limit)
	})
}

type controlTenant struct {
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
}

func (r *repository) controlTenants(
	ctx context.Context,
	enabled keel.Pred,
	limit int,
) ([]pagination.TenantInfo, error) {
	if limit <= 0 {
		limit = defaultSweepLimit
	}

	sc := tenantdb.ShipmentControls
	rows, err := keel.Into[controlTenant](ctx, r.db.Keel(ctx), sc.AllTenants().
		Pick(sc.OrganizationID, sc.BusinessUnitID).
		Where(enabled).
		Order(sc.OrganizationID.Asc(), sc.BusinessUnitID.Asc()).
		Limit(limit))
	if err != nil {
		return nil, err
	}

	tenants := make([]pagination.TenantInfo, 0, len(rows))
	for _, row := range rows {
		tenants = append(tenants, pagination.TenantInfo{
			OrgID: row.OrganizationID,
			BuID:  row.BusinessUnitID,
		})
	}

	return tenants, nil
}

func (r *repository) RunAutoDelayShipmentsForTenant(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	limit int,
) ([]*shipment.Shipment, error) {
	ctx = tenantScope(ctx, tenantInfo)
	return dbtx.Write(ctx, r.db, func(ctx context.Context) ([]*shipment.Shipment, error) {
		if limit <= 0 {
			limit = defaultSweepLimit
		}

		sp := shipmentdb.Shipments
		candidates := autoDelayedShipmentIDs(timeutils.NowUnix(), scopedTenant).Limit(limit)
		entities, err := sp.Where(sp.ID.InQuery(candidates)).
			Order(sp.ID.Asc()).
			Limit(limit).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, err
		}

		return r.markDelayed(ctx, entities, scopedTenant)
	})
}

func (r *repository) markDelayed(
	ctx context.Context,
	entities []*shipment.Shipment,
	t tenancy,
) ([]*shipment.Shipment, error) {
	if len(entities) == 0 {
		return entities, nil
	}

	currentTime := timeutils.NowUnix()
	sp := shipmentdb.Shipments
	if _, err := scoped(sp.Where(sp.ID.In(shipmentIDsFromEntities(entities)...)), t).Update(
		ctx,
		r.db.Keel(ctx),
		sp.Status.To(shipment.StatusDelayed),
		sp.UpdatedAt.To(currentTime),
		sp.Version.ToExpr(sp.Version),
	); err != nil {
		return nil, err
	}

	for _, entity := range entities {
		entity.Status = shipment.StatusDelayed
		entity.UpdatedAt = currentTime
	}

	return entities, nil
}

func delayedShipments(
	thresholdMinutes int16,
	currentTime int64,
	limit int,
) *keel.Query[*shipment.Shipment] {
	if limit <= 0 {
		limit = defaultSweepLimit
	}

	thresholdSeconds := int64(shipmentstate.ResolveDelayThresholdMinutes(thresholdMinutes)) * 60
	st := shipmentdb.Stops.As("late_stp")
	sm := shipmentdb.ShipmentMoves.As("late_sm")
	sp := shipmentdb.Shipments
	windowEnd := keel.DynOf(st.ScheduledWindowEnd).Coalesce(st.ScheduledWindowStart)

	lateStops := st.Pick(st.ShipmentMoveID).Where(
		st.Status.NotIn(shipmentstate.DelayedExcludedStopStatuses()...),
		st.ActualDeparture.IsNull(),
		keel.DynOf(windowEnd).Gt(0),
		windowEnd.Add(thresholdSeconds).Lt(currentTime),
	)
	lateMoves := sm.Pick(sm.ShipmentID).Where(
		sm.ID.InQuery(lateStops),
		sm.Status.NotIn(shipmentstate.DelayedExcludedMoveStatuses()...),
	)

	return sp.Where(
		sp.ID.InQuery(lateMoves),
		sp.Status.NotIn(shipmentstate.DelayedExcludedShipmentStatuses()...),
	).Order(sp.ID.Asc()).Limit(limit)
}

func autoDelayedShipmentIDs(currentTime int64, t tenancy) *keel.Query[*shipment.Stop] {
	st := shipmentdb.Stops.As("auto_stp")
	sm := shipmentdb.ShipmentMoves.As("auto_sm")
	sp := shipmentdb.Shipments.As("auto_sp")
	sc := tenantdb.ShipmentControls.As("auto_sc")
	windowEnd := keel.DynOf(st.ScheduledWindowEnd).Coalesce(st.ScheduledWindowStart)

	q := st.Pick(sm.ShipmentID).
		Join(sm, sm.ID.EqCol(st.ShipmentMoveID), sm.OrganizationID.EqCol(st.OrganizationID),
			sm.BusinessUnitID.EqCol(st.BusinessUnitID)).
		Join(sp, sp.ID.EqCol(sm.ShipmentID), sp.OrganizationID.EqCol(sm.OrganizationID),
			sp.BusinessUnitID.EqCol(sm.BusinessUnitID)).
		Join(sc, sc.OrganizationID.EqCol(sp.OrganizationID), sc.BusinessUnitID.EqCol(sp.BusinessUnitID)).
		Where(
			sc.AutoDelayShipments.IsTrue(),
			st.Status.NotIn(shipmentstate.DelayedExcludedStopStatuses()...),
			st.ActualDeparture.IsNull(),
			keel.DynOf(windowEnd).Gt(0),
			windowEnd.Add(
				keel.DynOf(sc.AutoDelayShipmentsThreshold).
					Coalesce(shipmentstate.DefaultDelayThresholdMinutes).
					Mul(keel.DynLit(60).Cast(keel.TypeInt64)),
			).Lt(currentTime),
			sm.Status.NotIn(shipmentstate.DelayedExcludedMoveStatuses()...),
			sp.Status.NotIn(shipmentstate.DelayedExcludedShipmentStatuses()...),
		)

	return scoped(q, t)
}

func tenantScope(ctx context.Context, tenantInfo pagination.TenantInfo) context.Context {
	return dbscope.EnsureTenant(ctx, dbscope.Tenant{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		UserID:         tenantInfo.UserID,
	})
}

func shipmentIDsFromEntities(entities []*shipment.Shipment) []pulid.ID {
	shipmentIDs := make([]pulid.ID, 0, len(entities))
	for _, entity := range entities {
		shipmentIDs = append(shipmentIDs, entity.ID)
	}

	return shipmentIDs
}
