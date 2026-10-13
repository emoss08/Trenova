package shipmentrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/pkg/buncolgen/shipmentcol"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type SearchCardParams struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type searchCardReader struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewSearchCardReader(p SearchCardParams) shipmentports.ShipmentSearchCardReader {
	return &searchCardReader{db: p.DB, l: p.Logger.Named("postgres.shipment-search-card-reader")}
}

var liveShipmentStatuses = []shipment.Status{
	shipment.StatusNew,
	shipment.StatusPartiallyAssigned,
	shipment.StatusAssigned,
	shipment.StatusInTransit,
	shipment.StatusDelayed,
	shipment.StatusPartiallyCompleted,
}

var openMoveStatuses = []shipment.MoveStatus{
	shipment.MoveStatusNew,
	shipment.MoveStatusAssigned,
	shipment.MoveStatusInTransit,
}

func (r *searchCardReader) ListSearchCards(
	ctx context.Context,
	req *shipmentports.ListShipmentSearchCardsRequest,
) ([]*shipmentports.ShipmentSearchCard, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*shipmentports.ShipmentSearchCard, error) {
			if len(req.ShipmentIDs) == 0 {
				return nil, nil
			}

			cards := make([]*shipmentports.ShipmentSearchCard, 0, len(req.ShipmentIDs))
			err := searchCardsQuery(r.db.DBForContext(ctx), req).
				Scan(ctx, &cards)
			if err != nil {
				r.l.Error("failed to list shipment search cards", zap.Error(err))
				return nil, fmt.Errorf("list shipment search cards: %w", err)
			}

			return cards, nil
		},
	)
}

func searchCardsQuery(
	db bun.IDB,
	req *shipmentports.ListShipmentSearchCardsRequest,
) *bun.SelectQuery {
	sp := shipmentcol.ShipmentColumns

	return db.
		NewSelect().
		Model((*shipment.Shipment)(nil)).
		ColumnExpr("sp.id AS shipment_id").
		ColumnExpr("COALESCE(sp.pro_number, '') AS pro_number").
		ColumnExpr("COALESCE(sp.bol, '') AS bol").
		ColumnExpr("sp.status AS status").
		ColumnExpr("COALESCE(sp.customer_id, '') AS customer_id").
		ColumnExpr("COALESCE(cus.name, '') AS customer_name").
		ColumnExpr("COALESCE(cus.code, '') AS customer_code").
		ColumnExpr("COALESCE(svt.code, '') AS service_type_code").
		ColumnExpr("COALESCE(origin.city, '') AS origin_city").
		ColumnExpr("COALESCE(origin.state, '') AS origin_state").
		ColumnExpr("COALESCE(dest.city, '') AS destination_city").
		ColumnExpr("COALESCE(dest.state, '') AS destination_state").
		ColumnExpr("COALESCE(drv.id, '') AS driver_id").
		ColumnExpr("COALESCE(drv.name, '') AS driver_name").
		Join("LEFT JOIN customers AS cus ON cus.id = sp.customer_id").
		Join("AND cus.organization_id = sp.organization_id").
		Join("AND cus.business_unit_id = sp.business_unit_id").
		Join("LEFT JOIN service_types AS svt ON svt.id = sp.service_type_id").
		Join("AND svt.organization_id = sp.organization_id").
		Join("AND svt.business_unit_id = sp.business_unit_id").
		Join(`LEFT JOIN LATERAL (
	SELECT loc.city AS city, ust.abbreviation AS state
	FROM stops AS stp
	JOIN shipment_moves AS smv ON smv.id = stp.shipment_move_id
	LEFT JOIN locations AS loc ON loc.id = stp.location_id
	LEFT JOIN us_states AS ust ON ust.id = loc.state_id
	WHERE smv.shipment_id = sp.id
	  AND smv.organization_id = sp.organization_id
	  AND smv.business_unit_id = sp.business_unit_id
	  AND stp.type IN ('Pickup', 'SplitPickup')
	ORDER BY smv.sequence ASC, stp.sequence ASC
	LIMIT 1
		) AS origin ON TRUE`).
		Join(`LEFT JOIN LATERAL (
	SELECT loc.city AS city, ust.abbreviation AS state
	FROM stops AS stp
	JOIN shipment_moves AS smv ON smv.id = stp.shipment_move_id
	LEFT JOIN locations AS loc ON loc.id = stp.location_id
	LEFT JOIN us_states AS ust ON ust.id = loc.state_id
	WHERE smv.shipment_id = sp.id
	  AND smv.organization_id = sp.organization_id
	  AND smv.business_unit_id = sp.business_unit_id
	  AND stp.type IN ('Delivery', 'SplitDelivery')
	ORDER BY smv.sequence DESC, stp.sequence DESC
	LIMIT 1
		) AS dest ON TRUE`).
		Join(`LEFT JOIN LATERAL (
	SELECT wrk.id AS id, CONCAT_WS(' ', wrk.first_name, wrk.last_name) AS name
	FROM shipment_moves AS smv
	JOIN assignments AS asg ON asg.shipment_move_id = smv.id
	  AND asg.organization_id = smv.organization_id
	  AND asg.business_unit_id = smv.business_unit_id
	  AND asg.archived_at IS NULL
	  AND asg.status != ?
	JOIN workers AS wrk ON wrk.id = asg.primary_worker_id
	  AND wrk.organization_id = asg.organization_id
	  AND wrk.business_unit_id = asg.business_unit_id
	WHERE smv.shipment_id = sp.id
	  AND smv.organization_id = sp.organization_id
	  AND smv.business_unit_id = sp.business_unit_id
	  AND smv.status != ?
	ORDER BY (smv.status = ?) ASC, smv.sequence ASC
	LIMIT 1
		) AS drv ON TRUE`,
			shipment.AssignmentStatusCanceled,
			shipment.MoveStatusCanceled,
			shipment.MoveStatusCompleted,
		).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return shipmentcol.ShipmentScopeTenant(sq, req.TenantInfo).
				Where(sp.ID.In(), bun.List(req.ShipmentIDs))
		})
}

func (r *searchCardReader) ListPartyLinks(
	ctx context.Context,
	req *shipmentports.ListShipmentPartyLinksRequest,
) ([]*shipmentports.ShipmentPartyLink, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*shipmentports.ShipmentPartyLink, error) {
			links := make([]*shipmentports.ShipmentPartyLink, 0)

			if len(req.CustomerIDs) > 0 && req.CustomerLimit > 0 {
				customerLinks, err := r.customerLinks(ctx, req)
				if err != nil {
					return nil, err
				}
				links = append(links, customerLinks...)
			}

			if len(req.WorkerIDs) > 0 {
				workerLinks, err := r.workerLinks(ctx, req)
				if err != nil {
					return nil, err
				}
				links = append(links, workerLinks...)
			}

			return links, nil
		},
	)
}

func (r *searchCardReader) customerLinks(
	ctx context.Context,
	req *shipmentports.ListShipmentPartyLinksRequest,
) ([]*shipmentports.ShipmentPartyLink, error) {
	links := make([]*shipmentports.ShipmentPartyLink, 0, req.CustomerLimit)
	err := customerLinksQuery(r.db.DBForContext(ctx), req).
		Scan(ctx, &links)
	if err != nil {
		r.l.Error("failed to list customer shipments for search", zap.Error(err))
		return nil, fmt.Errorf("list customer shipments for search: %w", err)
	}

	return links, nil
}

func (r *searchCardReader) workerLinks(
	ctx context.Context,
	req *shipmentports.ListShipmentPartyLinksRequest,
) ([]*shipmentports.ShipmentPartyLink, error) {
	links := make([]*shipmentports.ShipmentPartyLink, 0, len(req.WorkerIDs))
	err := workerLinksQuery(r.db.DBForContext(ctx), req).Scan(ctx, &links)
	if err != nil {
		r.l.Error("failed to list worker loads for search", zap.Error(err))
		return nil, fmt.Errorf("list worker loads for search: %w", err)
	}

	return links, nil
}

func customerLinksQuery(
	db bun.IDB,
	req *shipmentports.ListShipmentPartyLinksRequest,
) *bun.SelectQuery {
	sp := shipmentcol.ShipmentColumns

	return db.
		NewSelect().
		Model((*shipment.Shipment)(nil)).
		ColumnExpr("sp.id AS shipment_id").
		ColumnExpr("sp.customer_id AS customer_id").
		ColumnExpr("'' AS worker_id").
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return shipmentcol.ShipmentScopeTenant(sq, req.TenantInfo).
				Where(sp.CustomerID.In(), bun.List(req.CustomerIDs)).
				Where(sp.Status.NotEq(), shipment.StatusCanceled)
		}).
		OrderExpr("(sp.status IN (?)) DESC", bun.List(liveShipmentStatuses)).
		OrderExpr("sp.updated_at DESC").
		Limit(int64(req.CustomerLimit))
}

func workerLinksQuery(
	db bun.IDB,
	req *shipmentports.ListShipmentPartyLinksRequest,
) *bun.RawQuery {
	return db.NewRaw(`
		SELECT DISTINCT ON (seat.worker_id)
			sp.id AS shipment_id,
			'' AS customer_id,
			seat.worker_id AS worker_id
		FROM assignments AS asg
		JOIN shipment_moves AS smv ON smv.id = asg.shipment_move_id
		  AND smv.organization_id = asg.organization_id
		  AND smv.business_unit_id = asg.business_unit_id
		JOIN shipments AS sp ON sp.id = smv.shipment_id
		  AND sp.organization_id = smv.organization_id
		  AND sp.business_unit_id = smv.business_unit_id
		CROSS JOIN LATERAL (VALUES (asg.primary_worker_id), (asg.secondary_worker_id)) AS seat(worker_id)
		WHERE asg.organization_id = ?
		  AND asg.business_unit_id = ?
		  AND asg.archived_at IS NULL
		  AND asg.status != ?
		  AND smv.status IN (?)
		  AND sp.status IN (?)
		  AND seat.worker_id IN (?)
		ORDER BY seat.worker_id, (smv.status = ?) DESC, smv.sequence ASC, sp.updated_at DESC
	`,
		req.TenantInfo.OrgID,
		req.TenantInfo.BuID,
		shipment.AssignmentStatusCanceled,
		bun.List(openMoveStatuses),
		bun.List(liveShipmentStatuses),
		bun.List(req.WorkerIDs),
		shipment.MoveStatusInTransit,
	)
}
