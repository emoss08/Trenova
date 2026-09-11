package billingqueuerepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

const defaultCandidateLimit = 5000

// ListConsolidationCandidates flattens every approved queue item a run may bill,
// together with each key a split rule can group on.
//
// One query rather than one per shipment: a month of freight for a large customer
// is thousands of rows, and resolving the origin and destination per shipment
// would be thousands of round trips. The stop lookups are lateral joins for the
// same reason.
func (r *repository) ListConsolidationCandidates(
	ctx context.Context,
	req *repositories.ListConsolidationCandidatesRequest,
) ([]*repositories.ConsolidationCandidate, error) {
	if len(req.CustomerIDs) == 0 {
		return nil, nil
	}

	limit := req.Limit
	if limit <= 0 {
		limit = defaultCandidateLimit
	}

	candidates := make([]*repositories.ConsolidationCandidate, 0, limit)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*billingqueue.BillingQueueItem)(nil)).
		ColumnExpr("bqi.id AS billing_queue_item_id").
		ColumnExpr("bqi.shipment_id AS shipment_id").
		ColumnExpr("COALESCE(sp.order_id, '') AS order_id").
		ColumnExpr("sp.customer_id AS customer_id").
		ColumnExpr("COALESCE(cus.name, '') AS customer_name").
		ColumnExpr("COALESCE(sp.pro_number, '') AS pro_number").
		ColumnExpr("COALESCE(sp.bol, '') AS shipment_bol").
		ColumnExpr("COALESCE(ord.order_number, '') AS order_number").
		ColumnExpr("COALESCE(ord.po_number, '') AS order_po_number").
		ColumnExpr("COALESCE(st.code, '') AS service_type_code").
		ColumnExpr("COALESCE(origin.location_key, '') AS origin_key").
		ColumnExpr("COALESCE(dest.location_key, '') AS destination_key").
		ColumnExpr("sp.actual_delivery_date AS service_date").
		ColumnExpr("sp.total_charge_amount AS total_charge_amount").
		ColumnExpr("COALESCE(cbp.billing_currency, 'USD') AS currency_code").
		ColumnExpr("COALESCE(cbp.split_by, 'Customer') AS split_by").
		ColumnExpr("COALESCE(cbp.section_by, 'Shipment') AS section_by").
		ColumnExpr("COALESCE(cbp.invoice_detail, 'Detailed') AS invoice_detail").
		ColumnExpr("COALESCE(cbp.max_shipments_per_invoice, 0) AS max_shipments_per_invoice").
		ColumnExpr("cbp.min_consolidated_amount AS min_consolidated_amount").
		ColumnExpr(
			"COUNT(*) FILTER (WHERE TRUE) OVER (PARTITION BY sp.order_id) AS order_eligible_legs",
		).
		ColumnExpr("COALESCE(legs.total, 0) AS order_total_legs").
		Join("JOIN shipments AS sp ON sp.id = bqi.shipment_id").
		Join("AND sp.organization_id = bqi.organization_id").
		Join("AND sp.business_unit_id = bqi.business_unit_id").
		Join("JOIN customers AS cus ON cus.id = sp.customer_id").
		Join("AND cus.organization_id = sp.organization_id").
		Join("AND cus.business_unit_id = sp.business_unit_id").
		Join("LEFT JOIN customer_billing_profiles AS cbp ON cbp.customer_id = cus.id").
		Join("AND cbp.organization_id = cus.organization_id").
		Join("AND cbp.business_unit_id = cus.business_unit_id").
		Join("LEFT JOIN orders AS ord ON ord.id = sp.order_id").
		Join("AND ord.organization_id = sp.organization_id").
		Join("AND ord.business_unit_id = sp.business_unit_id").
		Join("LEFT JOIN service_types AS st ON st.id = sp.service_type_id").
		Join("AND st.organization_id = sp.organization_id").
		// The first pickup and the final delivery are what "group by location"
		// means; a shipment with no stop at all falls into the unknown bucket
		// rather than being dropped.
		Join(`LEFT JOIN LATERAL (
			SELECT COALESCE(stp.location_id::text, '') AS location_key
			FROM stops AS stp
			JOIN shipment_moves AS smv ON smv.id = stp.shipment_move_id
			WHERE smv.shipment_id = sp.id
			  AND stp.type IN ('Pickup', 'SplitPickup')
			ORDER BY smv.sequence ASC, stp.sequence ASC
			LIMIT 1
		) AS origin ON TRUE`).
		Join(`LEFT JOIN LATERAL (
			SELECT COALESCE(stp.location_id::text, '') AS location_key
			FROM stops AS stp
			JOIN shipment_moves AS smv ON smv.id = stp.shipment_move_id
			WHERE smv.shipment_id = sp.id
			  AND stp.type IN ('Delivery', 'SplitDelivery')
			ORDER BY smv.sequence DESC, stp.sequence DESC
			LIMIT 1
		) AS dest ON TRUE`).
		// How many legs the order has that could ever be billed, so the caller can
		// tell a complete order from a partial one without a second query.
		Join(`LEFT JOIN LATERAL (
			SELECT COUNT(*) AS total
			FROM shipments AS sib
			WHERE sib.order_id = sp.order_id
			  AND sib.organization_id = sp.organization_id
			  AND sib.business_unit_id = sp.business_unit_id
			  AND sib.status <> ?
		) AS legs ON sp.order_id IS NOT NULL`, shipment.StatusCanceled).
		Where("bqi.organization_id = ?", req.TenantInfo.OrgID).
		Where("bqi.business_unit_id = ?", req.TenantInfo.BuID).
		Where("bqi.status = ?", billingqueue.StatusApproved).
		Where("bqi.bill_type = ?", billingqueue.BillTypeInvoice).
		// The durable double-bill guard: an item already carried by an invoice is
		// never a candidate again.
		Where("bqi.invoice_id IS NULL").
		Where("bqi.is_adjustment_origin = FALSE").
		Where("sp.customer_id IN (?)", bun.In(req.CustomerIDs)).
		Where("sp.status IN (?)", bun.In([]shipment.Status{
			shipment.StatusReadyToInvoice,
			shipment.StatusCompleted,
		})).
		Where(
			"COALESCE(sp.actual_delivery_date, bqi.created_at) >= ? - COALESCE(cbp.consolidation_lookback_days, 30) * 86400",
			req.PeriodStart,
		).
		Where("COALESCE(sp.actual_delivery_date, bqi.created_at) < ?", req.PeriodEnd).
		OrderExpr("sp.customer_id ASC, sp.actual_delivery_date ASC NULLS LAST, sp.pro_number ASC").
		Limit(limit).
		Scan(ctx, &candidates)
	if err != nil {
		r.l.Error("failed to list consolidation candidates", zap.Error(err))
		return nil, fmt.Errorf("list consolidation candidates: %w", err)
	}

	return candidates, nil
}
