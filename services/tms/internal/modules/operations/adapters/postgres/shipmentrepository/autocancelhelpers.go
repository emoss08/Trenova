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

const (
	autoCancelReason = "Automatically canceled by shipment control"
	secondsPerDay    = int64(24 * 60 * 60)
)

func (r *repository) GetAutoCancelableShipments(
	ctx context.Context,
	req *shipmentports.GetAutoCancelableShipmentsRequest,
	thresholdDays int16,
) ([]*shipment.Shipment, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*shipment.Shipment, error) {
		return autoCancelableShipments(thresholdDays, timeutils.NowUnix(), req.Limit).
			List(ctx, r.db.Keel(ctx))
	})
}

func (r *repository) AutoCancelShipments(
	ctx context.Context,
	req *shipmentports.AutoCancelShipmentsRequest,
	thresholdDays int16,
) ([]*shipment.Shipment, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) ([]*shipment.Shipment, error) {
		currentTime := timeutils.NowUnix()
		entities, err := autoCancelableShipments(thresholdDays, currentTime, req.Limit).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, err
		}

		return r.autoCancel(ctx, entities, currentTime, scopedTenant)
	})
}

func (r *repository) RunAutoCancelShipments(ctx context.Context) ([]*shipment.Shipment, error) {
	ctx = dbscope.WithSystem(ctx, "cancel expired shipments across every organization")
	return dbtx.Write(ctx, r.db, func(ctx context.Context) ([]*shipment.Shipment, error) {
		currentTime := timeutils.NowUnix()
		sp := shipmentdb.Shipments
		entities, err := sp.AllTenants().
			Where(sp.ID.InQuery(controlledAutoCancelIDs(currentTime, everyTenant))).
			All(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, err
		}

		return r.autoCancel(ctx, entities, currentTime, everyTenant)
	})
}

func (r *repository) ListAutoCancelShipmentTenants(
	ctx context.Context,
	limit int,
) ([]pagination.TenantInfo, error) {
	ctx = dbscope.WithSystem(ctx, "list organizations with automatic shipment cancellation on")
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]pagination.TenantInfo, error) {
		sc := tenantdb.ShipmentControls
		return r.controlTenants(ctx, sc.AutoCancelShipments.IsTrue(), limit)
	})
}

func (r *repository) RunAutoCancelShipmentsForTenant(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	limit int,
) ([]*shipment.Shipment, error) {
	ctx = tenantScope(ctx, tenantInfo)
	return dbtx.Write(ctx, r.db, func(ctx context.Context) ([]*shipment.Shipment, error) {
		if limit <= 0 {
			limit = defaultSweepLimit
		}

		currentTime := timeutils.NowUnix()
		sp := shipmentdb.Shipments
		candidates := controlledAutoCancelIDs(currentTime, scopedTenant).Limit(limit)
		entities, err := sp.Where(sp.ID.InQuery(candidates)).
			Order(sp.ID.Asc()).
			Limit(limit).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, err
		}

		return r.autoCancel(ctx, entities, currentTime, scopedTenant)
	})
}

func (r *repository) autoCancel(
	ctx context.Context,
	entities []*shipment.Shipment,
	currentTime int64,
	t tenancy,
) ([]*shipment.Shipment, error) {
	if len(entities) == 0 {
		return entities, nil
	}

	ids := shipmentIDsFromEntities(entities)
	sp := shipmentdb.Shipments
	if _, err := scoped(sp.Where(sp.ID.In(ids...)), t).Update(
		ctx,
		r.db.Keel(ctx),
		sp.Status.To(shipment.StatusCanceled),
		sp.CanceledAt.To(currentTime),
		sp.CanceledByID.ToNull(),
		sp.CancelReason.To(autoCancelReason),
		sp.UpdatedAt.To(currentTime),
		sp.Version.ToExpr(sp.Version),
	); err != nil {
		return nil, err
	}

	if err := r.cancelShipmentComponents(ctx, ids, t); err != nil {
		return nil, err
	}

	for _, entity := range entities {
		entity.Status = shipment.StatusCanceled
		entity.CanceledAt = &currentTime
		entity.CanceledByID = pulid.Nil
		entity.CancelReason = autoCancelReason
		entity.UpdatedAt = currentTime
	}

	return entities, nil
}

func autoCancelableShipments(
	thresholdDays int16,
	currentTime int64,
	limit int,
) *keel.Query[*shipment.Shipment] {
	if limit <= 0 {
		limit = defaultSweepLimit
	}

	thresholdSeconds := int64(shipmentstate.ResolveAutoCancelThresholdDays(thresholdDays)) *
		secondsPerDay
	sp := shipmentdb.Shipments

	return sp.Where(
		sp.Status.Eq(shipment.StatusNew),
		sp.CreatedAt.Lte(currentTime-thresholdSeconds),
	).Order(sp.ID.Asc()).Limit(limit)
}

func controlledAutoCancelIDs(currentTime int64, t tenancy) *keel.Query[*shipment.Shipment] {
	sp := shipmentdb.Shipments.As("cancel_sp")
	sc := tenantdb.ShipmentControls.As("cancel_sc")

	q := sp.Pick(sp.ID).
		Join(sc, sc.OrganizationID.EqCol(sp.OrganizationID), sc.BusinessUnitID.EqCol(sp.BusinessUnitID)).
		Where(
			sc.AutoCancelShipments.IsTrue(),
			sp.Status.Eq(shipment.StatusNew),
			keel.DynOf(sp.CreatedAt).Lte(
				keel.DynLit(currentTime).Sub(
					keel.DynOf(sc.AutoCancelShipmentsThreshold).
						Coalesce(shipmentstate.DefaultAutoCancelThresholdDays).
						Mul(keel.DynLit(secondsPerDay).Cast(keel.TypeInt64)),
				),
			),
		).
		Order(sp.ID.Asc())

	return scoped(q, t)
}
