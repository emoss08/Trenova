package shipmentrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/jackc/pgx/v5"
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

var liveShipmentStatuses = []string{
	string(shipment.StatusNew),
	string(shipment.StatusPartiallyAssigned),
	string(shipment.StatusAssigned),
	string(shipment.StatusInTransit),
	string(shipment.StatusDelayed),
	string(shipment.StatusPartiallyCompleted),
}

var openMoveStatuses = []string{
	string(shipment.MoveStatusNew),
	string(shipment.MoveStatusAssigned),
	string(shipment.MoveStatusInTransit),
}

const searchCardsSQL = `
SELECT
	sp.id AS shipment_id,
	COALESCE(sp.pro_number, '') AS pro_number,
	COALESCE(sp.bol, '') AS bol,
	sp.status AS status,
	COALESCE(sp.customer_id, '') AS customer_id,
	COALESCE(cus.name, '') AS customer_name,
	COALESCE(cus.code, '') AS customer_code,
	COALESCE(svt.code, '') AS service_type_code,
	COALESCE(origin.city, '') AS origin_city,
	COALESCE(origin.state, '') AS origin_state,
	COALESCE(dest.city, '') AS destination_city,
	COALESCE(dest.state, '') AS destination_state,
	COALESCE(drv.id, '') AS driver_id,
	COALESCE(drv.name, '') AS driver_name
FROM shipments AS sp
LEFT JOIN customers AS cus ON cus.id = sp.customer_id
	AND cus.organization_id = sp.organization_id
	AND cus.business_unit_id = sp.business_unit_id
LEFT JOIN service_types AS svt ON svt.id = sp.service_type_id
	AND svt.organization_id = sp.organization_id
	AND svt.business_unit_id = sp.business_unit_id
LEFT JOIN LATERAL (
	SELECT loc.city AS city, ust.abbreviation AS state
	FROM stops AS stp
	JOIN shipment_moves AS smv ON smv.id = stp.shipment_move_id
		AND smv.organization_id = stp.organization_id
		AND smv.business_unit_id = stp.business_unit_id
	LEFT JOIN locations AS loc ON loc.id = stp.location_id
		AND loc.organization_id = stp.organization_id
		AND loc.business_unit_id = stp.business_unit_id
	LEFT JOIN us_states AS ust ON ust.id = loc.state_id
	WHERE smv.shipment_id = sp.id
		AND smv.organization_id = sp.organization_id
		AND smv.business_unit_id = sp.business_unit_id
		AND stp.type IN ('Pickup', 'SplitPickup')
	ORDER BY smv.sequence ASC, stp.sequence ASC
	LIMIT 1
) AS origin ON TRUE
LEFT JOIN LATERAL (
	SELECT loc.city AS city, ust.abbreviation AS state
	FROM stops AS stp
	JOIN shipment_moves AS smv ON smv.id = stp.shipment_move_id
		AND smv.organization_id = stp.organization_id
		AND smv.business_unit_id = stp.business_unit_id
	LEFT JOIN locations AS loc ON loc.id = stp.location_id
		AND loc.organization_id = stp.organization_id
		AND loc.business_unit_id = stp.business_unit_id
	LEFT JOIN us_states AS ust ON ust.id = loc.state_id
	WHERE smv.shipment_id = sp.id
		AND smv.organization_id = sp.organization_id
		AND smv.business_unit_id = sp.business_unit_id
		AND stp.type IN ('Delivery', 'SplitDelivery')
	ORDER BY smv.sequence DESC, stp.sequence DESC
	LIMIT 1
) AS dest ON TRUE
LEFT JOIN LATERAL (
	SELECT wrk.id AS id, CONCAT_WS(' ', wrk.first_name, wrk.last_name) AS name
	FROM shipment_moves AS smv
	JOIN assignments AS asg ON asg.shipment_move_id = smv.id
		AND asg.organization_id = smv.organization_id
		AND asg.business_unit_id = smv.business_unit_id
		AND asg.archived_at IS NULL
		AND asg.status != $4
	JOIN workers AS wrk ON wrk.id = asg.primary_worker_id
		AND wrk.organization_id = asg.organization_id
		AND wrk.business_unit_id = asg.business_unit_id
	WHERE smv.shipment_id = sp.id
		AND smv.organization_id = sp.organization_id
		AND smv.business_unit_id = sp.business_unit_id
		AND smv.status != $5
	ORDER BY (smv.status = $6) ASC, smv.sequence ASC
	LIMIT 1
) AS drv ON TRUE
WHERE sp.organization_id = $1
	AND sp.business_unit_id = $2
	AND sp.id = ANY($3)`

const customerLinksSQL = `
SELECT
	sp.id AS shipment_id,
	sp.customer_id AS customer_id,
	'' AS worker_id
FROM shipments AS sp
WHERE sp.organization_id = $1
	AND sp.business_unit_id = $2
	AND sp.customer_id = ANY($3)
	AND sp.status != $4
ORDER BY (sp.status::text = ANY($5)) DESC, sp.updated_at DESC
LIMIT $6`

const workerLinksSQL = `
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
WHERE asg.organization_id = $1
	AND asg.business_unit_id = $2
	AND asg.archived_at IS NULL
	AND asg.status != $3
	AND smv.status::text = ANY($4)
	AND sp.status::text = ANY($5)
	AND seat.worker_id = ANY($6)
ORDER BY seat.worker_id, (smv.status = $7) DESC, smv.sequence ASC, sp.updated_at DESC`

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

			cards, err := keel.QueryAll(
				ctx,
				r.db.Keel(ctx),
				pgx.RowToAddrOfStructByName[shipmentports.ShipmentSearchCard],
				searchCardsSQL,
				req.TenantInfo.OrgID.String(),
				req.TenantInfo.BuID.String(),
				keel.Strings(req.ShipmentIDs),
				string(shipment.AssignmentStatusCanceled),
				string(shipment.MoveStatusCanceled),
				string(shipment.MoveStatusCompleted),
			)
			if err != nil {
				r.l.Error("failed to list shipment search cards", zap.Error(err))
				return nil, fmt.Errorf("list shipment search cards: %w", err)
			}

			return cards, nil
		},
	)
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
	links, err := keel.QueryAll(
		ctx,
		r.db.Keel(ctx),
		pgx.RowToAddrOfStructByName[shipmentports.ShipmentPartyLink],
		customerLinksSQL,
		req.TenantInfo.OrgID.String(),
		req.TenantInfo.BuID.String(),
		keel.Strings(req.CustomerIDs),
		string(shipment.StatusCanceled),
		liveShipmentStatuses,
		req.CustomerLimit,
	)
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
	links, err := keel.QueryAll(
		ctx,
		r.db.Keel(ctx),
		pgx.RowToAddrOfStructByName[shipmentports.ShipmentPartyLink],
		workerLinksSQL,
		req.TenantInfo.OrgID.String(),
		req.TenantInfo.BuID.String(),
		string(shipment.AssignmentStatusCanceled),
		openMoveStatuses,
		liveShipmentStatuses,
		keel.Strings(req.WorkerIDs),
		string(shipment.MoveStatusInTransit),
	)
	if err != nil {
		r.l.Error("failed to list worker loads for search", zap.Error(err))
		return nil, fmt.Errorf("list worker loads for search: %w", err)
	}

	return links, nil
}
