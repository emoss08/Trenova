package shipmentrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

const shipmentSummariesSQL = `
SELECT
	sp.id AS shipment_id,
	COALESCE(sp.pro_number, '') AS pro_number,
	COALESCE(sp.bol, '') AS bol,
	COALESCE(ord.po_number, '') AS po_number,
	sp.actual_delivery_date AS service_date,
	COALESCE(origin.city, '') AS origin_city,
	COALESCE(origin.state, '') AS origin_state,
	COALESCE(dest.city, '') AS destination_city,
	COALESCE(dest.state, '') AS destination_state,
	sp.total_charge_amount AS total_charge_amount
FROM shipments AS sp
LEFT JOIN orders AS ord ON ord.id = sp.order_id
	AND ord.organization_id = sp.organization_id
	AND ord.business_unit_id = sp.business_unit_id
LEFT JOIN LATERAL (
	SELECT COALESCE(loc.city, '') AS city, COALESCE(ust.abbreviation, '') AS state
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
	SELECT COALESCE(loc.city, '') AS city, COALESCE(ust.abbreviation, '') AS state
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
WHERE sp.organization_id = $1
	AND sp.business_unit_id = $2
	AND sp.id = ANY($3)
ORDER BY sp.actual_delivery_date ASC NULLS LAST, sp.pro_number ASC`

func (r *repository) ListSummariesByIDs(
	ctx context.Context,
	req *shipmentports.ListShipmentSummariesRequest,
) ([]*shipmentports.ShipmentSummary, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*shipmentports.ShipmentSummary, error) {
			if len(req.ShipmentIDs) == 0 {
				return nil, nil
			}

			summaries, err := keel.QueryAll(
				ctx,
				r.db.Keel(ctx),
				pgx.RowToAddrOfStructByName[shipmentports.ShipmentSummary],
				shipmentSummariesSQL,
				req.TenantInfo.OrgID.String(),
				req.TenantInfo.BuID.String(),
				keel.Strings(req.ShipmentIDs),
			)
			if err != nil {
				r.l.Error("failed to list shipment summaries", zap.Error(err))
				return nil, fmt.Errorf("list shipment summaries: %w", err)
			}

			return summaries, nil
		},
	)
}
