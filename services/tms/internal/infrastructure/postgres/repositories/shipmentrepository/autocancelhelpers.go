package shipmentrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/shipmentstate"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
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
		)
		if err != nil || len(entities) == 0 {
			return err
		}

		for _, entity := range entities {
			if _, err = tx.NewUpdate().
				Model((*shipment.Shipment)(nil)).
				Set("status = ?", shipment.StatusCanceled).
				Set("canceled_at = ?", currentTime).
				Set("canceled_by_id = NULL").
				Set("cancel_reason = ?", autoCancelReason).
				Set("updated_at = ?", currentTime).
				Where("sp.id = ?", entity.ID).
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

		for _, entity := range entities {
			if _, err = tx.NewUpdate().
				Model((*shipment.Shipment)(nil)).
				Set("status = ?", shipment.StatusCanceled).
				Set("canceled_at = ?", currentTime).
				Set("canceled_by_id = NULL").
				Set("cancel_reason = ?", autoCancelReason).
				Set("updated_at = ?", currentTime).
				Where("sp.id = ?", entity.ID).
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
) ([]*shipment.Shipment, error) {
	entities := make([]*shipment.Shipment, 0)
	thresholdSeconds := int64(
		shipmentstate.ResolveAutoCancelThresholdDays(thresholdDays),
	) * 24 * 60 * 60

	err := dba.NewSelect().
		Model(&entities).
		Where("sp.organization_id = ?", tenantInfo.OrgID).
		Where("sp.business_unit_id = ?", tenantInfo.BuID).
		Where("sp.status = ?", shipment.StatusNew).
		Where("sp.created_at <= ?", currentTime-thresholdSeconds).
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
