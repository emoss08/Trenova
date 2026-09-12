package shipmentrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

// ListSummariesByIDs flattens many shipments into the one line a document names
// for each.
//
// One query with lateral joins rather than loading whole shipments: a
// consolidated invoice can list hundreds of shipments, and fetching each one
// with its moves, stops and locations to render a lane and a date would be
// hundreds of round trips for data the document discards. The lateral joins
// mirror ListConsolidationCandidates, so the origin and destination a biller
// sees while the statement accrues are the ones the invoice prints.
func (r *repository) ListSummariesByIDs(
	ctx context.Context,
	req *repositories.ListShipmentSummariesRequest,
) ([]*repositories.ShipmentSummary, error) {
	if len(req.ShipmentIDs) == 0 {
		return nil, nil
	}

	sp := buncolgen.ShipmentColumns

	summaries := make([]*repositories.ShipmentSummary, 0, len(req.ShipmentIDs))
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*shipment.Shipment)(nil)).
		ColumnExpr("sp.id AS shipment_id").
		ColumnExpr("COALESCE(sp.pro_number, '') AS pro_number").
		ColumnExpr("COALESCE(sp.bol, '') AS bol").
		ColumnExpr("COALESCE(ord.po_number, '') AS po_number").
		ColumnExpr("sp.actual_delivery_date AS service_date").
		ColumnExpr("sp.total_charge_amount AS total_charge_amount").
		ColumnExpr("COALESCE(origin.city, '') AS origin_city").
		ColumnExpr("COALESCE(origin.state, '') AS origin_state").
		ColumnExpr("COALESCE(dest.city, '') AS destination_city").
		ColumnExpr("COALESCE(dest.state, '') AS destination_state").
		Join("LEFT JOIN orders AS ord ON ord.id = sp.order_id").
		Join("AND ord.organization_id = sp.organization_id").
		Join("AND ord.business_unit_id = sp.business_unit_id").
		// A shipment with no stop at all still gets a row, with a blank lane,
		// rather than dropping off the invoice that bills it.
		Join(`LEFT JOIN LATERAL (
			SELECT COALESCE(loc.city, '') AS city, COALESCE(ust.abbreviation, '') AS state
			FROM stops AS stp
			JOIN shipment_moves AS smv ON smv.id = stp.shipment_move_id
			LEFT JOIN locations AS loc ON loc.id = stp.location_id
			LEFT JOIN us_states AS ust ON ust.id = loc.state_id
			WHERE smv.shipment_id = sp.id
			  AND stp.type IN ('Pickup', 'SplitPickup')
			ORDER BY smv.sequence ASC, stp.sequence ASC
			LIMIT 1
		) AS origin ON TRUE`).
		Join(`LEFT JOIN LATERAL (
			SELECT COALESCE(loc.city, '') AS city, COALESCE(ust.abbreviation, '') AS state
			FROM stops AS stp
			JOIN shipment_moves AS smv ON smv.id = stp.shipment_move_id
			LEFT JOIN locations AS loc ON loc.id = stp.location_id
			LEFT JOIN us_states AS ust ON ust.id = loc.state_id
			WHERE smv.shipment_id = sp.id
			  AND stp.type IN ('Delivery', 'SplitDelivery')
			ORDER BY smv.sequence DESC, stp.sequence DESC
			LIMIT 1
		) AS dest ON TRUE`).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.ShipmentScopeTenant(sq, req.TenantInfo).
				Where(sp.ID.In(), bun.List(req.ShipmentIDs))
		}).
		OrderExpr("sp.actual_delivery_date ASC NULLS LAST, sp.pro_number ASC").
		Scan(ctx, &summaries)
	if err != nil {
		r.l.Error("failed to list shipment summaries", zap.Error(err))
		return nil, fmt.Errorf("list shipment summaries: %w", err)
	}

	return summaries, nil
}
