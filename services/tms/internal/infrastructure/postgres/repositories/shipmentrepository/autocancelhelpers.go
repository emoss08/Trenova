package shipmentrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/shipmentstate"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

const autoCancelReason = "Automatically canceled by shipment control"

func (r *repository) GetAutoCancelableShipments(
	ctx context.Context,
	req *repositories.GetAutoCancelableShipmentsRequest,
	thresholdDays int8,
) ([]*shipment.Shipment, error) {
	return r.getAutoCancelableShipments(
		ctx,
		r.db.DBForContext(ctx),
		req.TenantInfo,
		thresholdDays,
		timeutils.NowUnix(),
		req.Limit,
	)
}

func (r *repository) AutoCancelShipments(
	ctx context.Context,
	req *repositories.AutoCancelShipmentsRequest,
	thresholdDays int8,
) ([]*shipment.Shipment, error) {
	currentTime := timeutils.NowUnix()
	entities := make([]*shipment.Shipment, 0)

	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, tx bun.Tx) error {
		var err error
		entities, err = r.getAutoCancelableShipments(
			c,
			tx,
			req.TenantInfo,
			thresholdDays,
			currentTime,
			req.Limit,
		)
		if err != nil || len(entities) == 0 {
			return err
		}

		cols := buncolgen.ShipmentColumns
		for _, entity := range entities {
			if _, err = tx.NewUpdate().
				Model((*shipment.Shipment)(nil)).
				Set(cols.Status.Set(), shipment.StatusCanceled).
				Set(cols.CanceledAt.Set(), currentTime).
				Set(cols.CanceledByID.Set(), pulid.Nil).
				Set(cols.CancelReason.Set(), autoCancelReason).
				Set(cols.UpdatedAt.Set(), currentTime).
				Where(cols.ID.Eq(), entity.ID).
				Exec(c); err != nil {
				return err
			}

			if err = r.cancelShipmentComponents(c, tx, entity.ID); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
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

func (r *repository) RunAutoCancelShipments(ctx context.Context) ([]*shipment.Shipment, error) {
	currentTime := timeutils.NowUnix()
	entities := make([]*shipment.Shipment, 0)

	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, tx bun.Tx) error {
		var err error
		entities, err = r.getGloballyAutoCancelableShipments(c, tx, currentTime)
		if err != nil || len(entities) == 0 {
			return err
		}

		cols := buncolgen.ShipmentColumns

		for _, entity := range entities {
			if _, err = tx.NewUpdate().
				Model((*shipment.Shipment)(nil)).
				Set(cols.Status.Set(), shipment.StatusCanceled).
				Set(cols.CanceledAt.Set(), currentTime).
				Set(cols.CanceledByID.SetNull()).
				Set(cols.CancelReason.Set(), autoCancelReason).
				Set(cols.UpdatedAt.Set(), currentTime).
				Where(cols.ID.Eq(), entity.ID).
				Exec(c); err != nil {
				return err
			}

			if err = r.cancelShipmentComponents(c, tx, entity.ID); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
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

func (r *repository) ListAutoCancelShipmentTenants(
	ctx context.Context,
	limit int,
) ([]pagination.TenantInfo, error) {
	if limit <= 0 {
		limit = 100
	}

	type tenantRow struct {
		OrganizationID pulid.ID `bun:"organization_id"`
		BusinessUnitID pulid.ID `bun:"business_unit_id"`
	}

	rows := make([]tenantRow, 0, limit)
	cols := buncolgen.ShipmentControlColumns
	err := r.db.DBForContext(ctx).
		NewSelect().
		TableExpr(buncolgen.ShipmentControlTable.Name+" AS sc").
		Column(cols.OrganizationID.Name, cols.BusinessUnitID.Name).
		Where(cols.AutoCancelShipments.Eq(), true).
		Order(cols.OrganizationID.OrderAsc()).
		Order(cols.BusinessUnitID.OrderAsc()).
		Limit(limit).
		Scan(ctx, &rows)
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

func (r *repository) RunAutoCancelShipmentsForTenant(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	limit int,
) ([]*shipment.Shipment, error) {
	currentTime := timeutils.NowUnix()
	entities := make([]*shipment.Shipment, 0)

	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, tx bun.Tx) error {
		var err error
		entities, err = r.getAutoCancelableShipmentsForTenant(
			c,
			tx,
			tenantInfo,
			currentTime,
			limit,
		)
		if err != nil || len(entities) == 0 {
			return err
		}

		cols := buncolgen.ShipmentColumns
		for _, entity := range entities {
			if _, err = tx.NewUpdate().
				Model((*shipment.Shipment)(nil)).
				Set(cols.Status.Set(), shipment.StatusCanceled).
				Set(cols.CanceledAt.Set(), currentTime).
				Set(cols.CanceledByID.SetNull()).
				Set(cols.CancelReason.Set(), autoCancelReason).
				Set(cols.UpdatedAt.Set(), currentTime).
				Where(cols.ID.Eq(), entity.ID).
				Where(cols.OrganizationID.Eq(), tenantInfo.OrgID).
				Where(cols.BusinessUnitID.Eq(), tenantInfo.BuID).
				Exec(c); err != nil {
				return err
			}

			if err = r.cancelShipmentComponents(c, tx, entity.ID); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
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

func (r *repository) getAutoCancelableShipments(
	ctx context.Context,
	dba bun.IDB,
	tenantInfo pagination.TenantInfo,
	thresholdDays int8,
	currentTime int64,
	limit int,
) ([]*shipment.Shipment, error) {
	if limit <= 0 {
		limit = 100
	}

	entities := make([]*shipment.Shipment, 0, limit)
	thresholdSeconds := int64(
		shipmentstate.ResolveAutoCancelThresholdDays(thresholdDays),
	) * 24 * 60 * 60

	cols := buncolgen.ShipmentColumns
	err := dba.NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.ShipmentScopeTenant(sq, tenantInfo)
		}).
		Where(cols.Status.Eq(), shipment.StatusNew).
		Where(cols.CreatedAt.Lte(), currentTime-thresholdSeconds).
		Order(cols.ID.OrderAsc()).
		Limit(limit).
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	return entities, nil
}

func (r *repository) getAutoCancelableShipmentsForTenant(
	ctx context.Context,
	dba bun.IDB,
	tenantInfo pagination.TenantInfo,
	currentTime int64,
	limit int,
) ([]*shipment.Shipment, error) {
	if limit <= 0 {
		limit = 100
	}

	entities := make([]*shipment.Shipment, 0, limit)
	autoCancelableCte := dba.NewSelect().
		TableExpr("shipments AS sp").
		ColumnExpr("sp.id").
		Join("JOIN shipment_controls AS sc ON sc.organization_id = sp.organization_id AND sc.business_unit_id = sp.business_unit_id").
		Where("sc.auto_cancel_shipments = TRUE").
		Where("sp.organization_id = ?", tenantInfo.OrgID).
		Where("sp.business_unit_id = ?", tenantInfo.BuID).
		Where("sp.status = ?", shipment.StatusNew).
		Where(
			"sp.created_at <= ? - (COALESCE(sc.auto_cancel_shipments_threshold, ?) * 86400)",
			currentTime,
			shipmentstate.DefaultAutoCancelThresholdDays,
		).
		OrderExpr("sp.id ASC").
		Limit(limit)

	err := dba.NewSelect().
		Model(&entities).
		With("auto_cancelable_cte", autoCancelableCte).
		Where("sp.id IN (SELECT id FROM auto_cancelable_cte)").
		Where("sp.organization_id = ?", tenantInfo.OrgID).
		Where("sp.business_unit_id = ?", tenantInfo.BuID).
		Order(buncolgen.ShipmentColumns.ID.OrderAsc()).
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	return entities, nil
}

func (r *repository) getGloballyAutoCancelableShipments(
	ctx context.Context,
	dba bun.IDB,
	currentTime int64,
) ([]*shipment.Shipment, error) {
	entities := make([]*shipment.Shipment, 0)

	autoCancelableCte := dba.NewSelect().
		TableExpr("shipments AS sp").
		ColumnExpr("sp.id").
		Join("JOIN shipment_controls AS sc ON sc.organization_id = sp.organization_id AND sc.business_unit_id = sp.business_unit_id").
		Where("sc.auto_cancel_shipments = TRUE").
		Where("sp.status = ?", shipment.StatusNew).
		Where("sp.created_at <= ? - (COALESCE(sc.auto_cancel_shipments_threshold, ?) * 86400)", currentTime, shipmentstate.DefaultAutoCancelThresholdDays)

	err := dba.NewSelect().
		Model(&entities).
		With("auto_cancelable_cte", autoCancelableCte).
		Where("sp.id IN (SELECT id FROM auto_cancelable_cte)").
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	return entities, nil
}
