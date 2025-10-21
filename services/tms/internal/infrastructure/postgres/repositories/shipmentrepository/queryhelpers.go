package shipmentrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils/querybuilder"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

func (r *repository) addOptions(
	q *bun.SelectQuery,
	opts repositories.ShipmentOptions,
) *bun.SelectQuery {
	if opts.ExpandShipmentDetails {
		q = q.Relation("Customer")

		q = q.RelationWithOpts("Moves", bun.RelationOpts{
			Apply: func(sq *bun.SelectQuery) *bun.SelectQuery {
				return sq.Order("sm.sequence ASC").
					Relation("Assignment").
					Relation("Assignment.Tractor").
					Relation("Assignment.Trailer").
					Relation("Assignment.PrimaryWorker").
					Relation("Assignment.SecondaryWorker")
			},
		})

		q = q.RelationWithOpts("Moves.Stops", bun.RelationOpts{
			Apply: func(sq *bun.SelectQuery) *bun.SelectQuery {
				return sq.Order("stp.sequence ASC").
					Relation("Location").
					Relation("Location.State")
			},
		})

		q = q.RelationWithOpts("Commodities", bun.RelationOpts{
			Apply: func(sq *bun.SelectQuery) *bun.SelectQuery {
				return sq.Relation("Commodity")
			},
		})

		q = q.RelationWithOpts("AdditionalCharges", bun.RelationOpts{
			Apply: func(sq *bun.SelectQuery) *bun.SelectQuery {
				return sq.Relation("AccessorialCharge")
			},
		})

		q = q.Relation("ServiceType")
		q = q.Relation("ShipmentType")
		q = q.Relation("TractorType")
		q = q.Relation("TrailerType")
		q = q.Relation("CanceledBy")
		q = q.RelationWithOpts("Holds", bun.RelationOpts{
			Apply: func(sq *bun.SelectQuery) *bun.SelectQuery {
				return sq.
					Relation("ReleasedBy").
					Relation("CreatedBy")
			},
		})
		q = q.Relation("FormulaTemplate")
	}

	if opts.Status != "" {
		status, err := shipment.StatusFromString(opts.Status)
		if err != nil {
			r.l.Error("invalid status", zap.Error(err), zap.String("status", opts.Status))
			return q
		}

		q = q.Where("sp.status = ?", status)
	}

	return q
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListShipmentRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFilters(
		q,
		"sp",
		req.Filter,
		(*shipment.Shipment)(nil),
	)

	q = q.Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
		return r.addOptions(sq, req.ShipmentOptions)
	})

	return q.Limit(req.Filter.Limit).Offset(req.Filter.Offset)
}

func (r *repository) buildPreviousRatesCTEs(
	db bun.IDB,
	originLocID, destLocID pulid.ID,
) (originCTE, destCTE *bun.SelectQuery) {
	originCTE = db.NewSelect().
		Column("first_move.shipment_id").
		TableExpr("shipment_moves first_move").
		Join("JOIN stops origin_stop ON origin_stop.shipment_move_id = first_move.id").
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("first_move.sequence = 0").
				Where("origin_stop.sequence = 0").
				Where("origin_stop.type IN (?)", bun.In([]shipment.StopType{
					shipment.StopTypePickup,
					shipment.StopTypeSplitPickup,
				})).
				Where("origin_stop.location_id = ?", originLocID)
		})

	destCTE = db.NewSelect().
		Column("last_move.shipment_id").
		TableExpr("shipment_moves last_move").
		Join("JOIN stops delivery_stop ON delivery_stop.shipment_move_id = last_move.id").
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("last_move.sequence = (SELECT MAX(sm3.sequence) FROM shipment_moves sm3 WHERE sm3.shipment_id = last_move.shipment_id)").
				Where("delivery_stop.sequence = (SELECT MAX(stp3.sequence) FROM stops stp3 WHERE stp3.shipment_move_id = last_move.id)").
				Where("delivery_stop.location_id = ?", destLocID).
				Where("delivery_stop.type IN (?)", bun.In([]shipment.StopType{
					shipment.StopTypeDelivery,
					shipment.StopTypeSplitDelivery,
				}))
		})

	return originCTE, destCTE
}

func (r *repository) buildDelayedShipmentsCTEs(
	dba bun.IDB,
	currentTime int64,
) (stopCte, moveCte *bun.SelectQuery) {
	stopCte = dba.NewSelect().
		Column("stp.shipment_move_id").
		TableExpr("stops stp").
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("stp.status NOT IN (?)", bun.In([]shipment.StopStatus{
					shipment.StopStatusCompleted,
					shipment.StopStatusCanceled,
				})).
				Where("stp.actual_departure IS NULL").
				Where("stp.planned_departure < ?", currentTime)
		})

	moveCte = dba.NewSelect().
		ColumnExpr("DISTINCT sm.shipment_id").
		TableExpr("shipment_moves sm").
		Where("sm.id IN (SELECT shipment_move_id FROM stop_cte)").
		Where("sm.status NOT IN (?)", bun.In([]shipment.MoveStatus{
			shipment.MoveStatusCompleted,
			shipment.MoveStatusCanceled,
		}))

	return stopCte, moveCte
}

func (r *repository) getDelayedShipments(
	ctx context.Context,
	tx bun.IDB,
	ct int64,
) ([]*shipment.Shipment, error) {
	log := r.l.With(
		zap.String("operation", "getDelayedShipments"),
	)

	entities := make([]*shipment.Shipment, 0)

	stopCte, moveCte := r.buildDelayedShipmentsCTEs(tx, ct)

	err := tx.NewSelect().
		Model(&entities).
		With("stop_cte", stopCte).
		With("move_cte", moveCte).
		Where("sp.id IN (SELECT shipment_id FROM move_cte)").
		Where("sp.status NOT IN (?)", bun.In([]shipment.Status{
			shipment.StatusDelayed,
			shipment.StatusCanceled,
			shipment.StatusCompleted,
			shipment.StatusBilled,
		})).Scan(ctx)
	if err != nil {
		log.Error("failed to scan delayed shipments", zap.Error(err))
		return nil, err
	}

	log.Debug("found delayed shipments", zap.Int("count", len(entities)))

	return entities, nil
}
