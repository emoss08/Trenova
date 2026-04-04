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

func (r *repository) GetDelayedShipments(
	ctx context.Context,
	req *repositories.GetDelayedShipmentsRequest,
	thresholdMinutes int16,
) ([]*shipment.Shipment, error) {
	return r.getDelayedShipments(
		ctx,
		r.db.DBForContext(ctx),
		req.TenantInfo,
		thresholdMinutes,
		timeutils.NowUnix(),
	)
}

func (r *repository) DelayShipments(
	ctx context.Context,
	req *repositories.DelayShipmentsRequest,
	thresholdMinutes int16,
) ([]*shipment.Shipment, error) {
	currentTime := timeutils.NowUnix()
	entities := make([]*shipment.Shipment, 0)

	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, tx bun.Tx) error {
		var err error
		entities, err = r.getDelayedShipments(
			c,
			tx,
			req.TenantInfo,
			thresholdMinutes,
			currentTime,
		)
		if err != nil || len(entities) == 0 {
			return err
		}

		cols := buncolgen.ShipmentColumns
		shipmentIDs := shipmentIDsFromEntities(entities)
		_, err = tx.NewUpdate().
			Model((*shipment.Shipment)(nil)).
			Set(cols.Status.Set(), shipment.StatusDelayed).
			Set(cols.UpdatedAt.Set(), currentTime).
			Where(cols.ID.In(), bun.List(shipmentIDs)).
			Exec(c)
		return err
	})
	if err != nil {
		return nil, err
	}

	for _, entity := range entities {
		entity.Status = shipment.StatusDelayed
		entity.UpdatedAt = currentTime
	}

	return entities, nil
}

func (r *repository) AutoDelayShipments(ctx context.Context) ([]*shipment.Shipment, error) {
	currentTime := timeutils.NowUnix()
	entities := make([]*shipment.Shipment, 0)

	err := r.db.WithTx(ctx, ports.TxOptions{}, func(c context.Context, tx bun.Tx) error {
		var err error
		entities, err = r.getAutoDelayedShipments(c, tx, currentTime)
		if err != nil || len(entities) == 0 {
			return err
		}

		cols := buncolgen.ShipmentColumns
		shipmentIDs := shipmentIDsFromEntities(entities)
		_, err = tx.NewUpdate().
			Model((*shipment.Shipment)(nil)).
			Set(cols.Status.Set(), shipment.StatusDelayed).
			Set(cols.UpdatedAt.Set(), currentTime).
			Where(cols.ID.In(), bun.List(shipmentIDs)).
			Exec(c)
		return err
	})
	if err != nil {
		return nil, err
	}

	for _, entity := range entities {
		entity.Status = shipment.StatusDelayed
		entity.UpdatedAt = currentTime
	}

	return entities, nil
}

func (r *repository) getDelayedShipments(
	ctx context.Context,
	dba bun.IDB,
	tenantInfo pagination.TenantInfo,
	thresholdMinutes int16,
	currentTime int64,
) ([]*shipment.Shipment, error) {
	entities := make([]*shipment.Shipment, 0)
	stopCte, moveCte := buildDelayedShipmentCTEs(dba, currentTime, thresholdMinutes)

	err := dba.NewSelect().
		Model(&entities).
		With("stop_cte", stopCte).
		With("move_cte", moveCte).
		Where("sp.id IN (SELECT shipment_id FROM move_cte)").
		Where("sp.organization_id = ?", tenantInfo.OrgID).
		Where("sp.business_unit_id = ?", tenantInfo.BuID).
		Where("sp.status NOT IN (?)", bun.List(nonDelayedEligibleShipmentStatuses())).
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	return entities, nil
}

func (r *repository) getAutoDelayedShipments(
	ctx context.Context,
	dba bun.IDB,
	currentTime int64,
) ([]*shipment.Shipment, error) {
	entities := make([]*shipment.Shipment, 0)
	delayedCte := dba.NewSelect().
		TableExpr("stops AS stp").
		ColumnExpr("DISTINCT sm.shipment_id").
		Join("JOIN shipment_moves AS sm ON sm.id = stp.shipment_move_id").
		Join("JOIN shipments AS sp ON sp.id = sm.shipment_id").
		Join("JOIN shipment_controls AS sc ON sc.organization_id = sp.organization_id AND sc.business_unit_id = sp.business_unit_id").
		Where("sc.auto_delay_shipments = TRUE").
		Where("stp.status NOT IN (?)", bun.List([]shipment.StopStatus{
			shipment.StopStatusCompleted,
			shipment.StopStatusCanceled,
		})).
		Where("stp.actual_departure IS NULL").
		Where("COALESCE(stp.scheduled_window_end, stp.scheduled_window_start) > 0").
		Where("COALESCE(stp.scheduled_window_end, stp.scheduled_window_start) + (COALESCE(sc.auto_delay_shipments_threshold, ?) * 60) < ?", shipmentstate.DefaultDelayThresholdMinutes, currentTime).
		Where("sm.status NOT IN (?)", bun.List([]shipment.MoveStatus{
			shipment.MoveStatusCompleted,
			shipment.MoveStatusCanceled,
		})).
		Where("sp.status NOT IN (?)", bun.List(nonDelayedEligibleShipmentStatuses()))

	err := dba.NewSelect().
		Model(&entities).
		With("delayed_cte", delayedCte).
		Where("sp.id IN (SELECT shipment_id FROM delayed_cte)").
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	return entities, nil
}

func buildDelayedShipmentCTEs(
	dba bun.IDB,
	currentTime int64,
	thresholdMinutes int16,
) (stopCte, moveCte *bun.SelectQuery) {
	thresholdSeconds := int64(shipmentstate.ResolveDelayThresholdMinutes(thresholdMinutes)) * 60

	stopCte = dba.NewSelect().
		Column("stp.shipment_move_id").
		TableExpr("stops AS stp").
		Where("stp.status NOT IN (?)", bun.List([]shipment.StopStatus{
			shipment.StopStatusCompleted,
			shipment.StopStatusCanceled,
		})).
		Where("stp.actual_departure IS NULL").
		Where("COALESCE(stp.scheduled_window_end, stp.scheduled_window_start) > 0").
		Where("COALESCE(stp.scheduled_window_end, stp.scheduled_window_start) + ? < ?", thresholdSeconds, currentTime)

	moveCte = dba.NewSelect().
		ColumnExpr("DISTINCT sm.shipment_id").
		TableExpr("shipment_moves AS sm").
		Where("sm.id IN (SELECT shipment_move_id FROM stop_cte)").
		Where("sm.status NOT IN (?)", bun.List([]shipment.MoveStatus{
			shipment.MoveStatusCompleted,
			shipment.MoveStatusCanceled,
		}))

	return stopCte, moveCte
}

func nonDelayedEligibleShipmentStatuses() []shipment.Status {
	return []shipment.Status{
		shipment.StatusDelayed,
		shipment.StatusCanceled,
		shipment.StatusCompleted,
		shipment.StatusInvoiced,
	}
}

func shipmentIDsFromEntities(entities []*shipment.Shipment) []pulid.ID {
	shipmentIDs := make([]pulid.ID, 0, len(entities))
	for _, entity := range entities {
		shipmentIDs = append(shipmentIDs, entity.ID)
	}

	return shipmentIDs
}
