package billingqueuerepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

const defaultCandidateLimit = 5000

const consolidationCandidatesSQL = `SELECT
	bqi.id AS billing_queue_item_id,
	bqi.shipment_id AS shipment_id,
	COALESCE(sp.order_id, '') AS order_id,
	bqi.bill_to_customer_id AS customer_id,
	COALESCE(cus.name, '') AS customer_name,
	COALESCE(sp.pro_number, '') AS pro_number,
	COALESCE(sp.bol, '') AS shipment_bol,
	COALESCE(ord.order_number, '') AS order_number,
	COALESCE(ord.po_number, '') AS order_po_number,
	COALESCE(st.code, '') AS service_type_code,
	COALESCE(origin.location_key, '') AS origin_key,
	COALESCE(dest.location_key, '') AS destination_key,
	sp.actual_delivery_date AS service_date,
	bqi.allocated_total_amount AS total_charge_amount,
	COALESCE(cbp.billing_currency, 'USD') AS currency_code,
	COALESCE(cbp.split_by, 'Customer') AS split_by,
	COALESCE(cbp.section_by, 'Shipment') AS section_by,
	COALESCE(cbp.invoice_detail, 'Detailed') AS invoice_detail,
	COALESCE(cbp.max_shipments_per_invoice, 0) AS max_shipments_per_invoice,
	cbp.min_consolidated_amount AS min_consolidated_amount,
	COALESCE(cbp.auto_bill, FALSE) AS auto_bill,
	COUNT(*) OVER (PARTITION BY sp.order_id, bqi.bill_to_customer_id) AS order_eligible_legs,
	COALESCE(legs.total, 0) AS order_total_legs
FROM billing_queue_items AS bqi
JOIN shipments AS sp ON sp.id = bqi.shipment_id
	AND sp.organization_id = bqi.organization_id
	AND sp.business_unit_id = bqi.business_unit_id
JOIN customers AS cus ON cus.id = bqi.bill_to_customer_id
	AND cus.organization_id = bqi.organization_id
	AND cus.business_unit_id = bqi.business_unit_id
LEFT JOIN customer_billing_profiles AS cbp ON cbp.customer_id = cus.id
	AND cbp.organization_id = cus.organization_id
	AND cbp.business_unit_id = cus.business_unit_id
LEFT JOIN orders AS ord ON ord.id = sp.order_id
	AND ord.organization_id = sp.organization_id
	AND ord.business_unit_id = sp.business_unit_id
LEFT JOIN service_types AS st ON st.id = sp.service_type_id
	AND st.organization_id = sp.organization_id
LEFT JOIN LATERAL (
	SELECT COALESCE(stp.location_id::text, '') AS location_key
	FROM stops AS stp
	JOIN shipment_moves AS smv ON smv.id = stp.shipment_move_id
	WHERE smv.shipment_id = sp.id
	  AND stp.type IN ('Pickup', 'SplitPickup')
	ORDER BY smv.sequence ASC, stp.sequence ASC
	LIMIT 1
) AS origin ON TRUE
LEFT JOIN LATERAL (
	SELECT COALESCE(stp.location_id::text, '') AS location_key
	FROM stops AS stp
	JOIN shipment_moves AS smv ON smv.id = stp.shipment_move_id
	WHERE smv.shipment_id = sp.id
	  AND stp.type IN ('Delivery', 'SplitDelivery')
	ORDER BY smv.sequence DESC, stp.sequence DESC
	LIMIT 1
) AS dest ON TRUE
LEFT JOIN LATERAL (
	SELECT COUNT(*) AS total
	FROM shipments AS sib
	WHERE sib.order_id = sp.order_id
	  AND sib.organization_id = sp.organization_id
	  AND sib.business_unit_id = sp.business_unit_id
	  AND sib.status <> $3
) AS legs ON sp.order_id IS NOT NULL
WHERE bqi.organization_id = $1
	AND bqi.business_unit_id = $2
	AND bqi.status = $4
	AND bqi.bill_type = $5
	AND bqi.invoice_id IS NULL
	AND bqi.is_adjustment_origin = FALSE
	AND bqi.bill_to_customer_id = ANY($6)
	AND sp.status = ANY($7)
	AND NOT EXISTS (
		SELECT 1 FROM invoice_lines AS il
		JOIN invoices AS inv ON inv.id = il.invoice_id
		  AND inv.organization_id = il.organization_id
		  AND inv.business_unit_id = il.business_unit_id
		WHERE il.shipment_id = sp.id
		  AND il.organization_id = sp.organization_id
		  AND il.business_unit_id = sp.business_unit_id
		  AND inv.customer_id = bqi.bill_to_customer_id
	)
	AND COALESCE(sp.actual_delivery_date, bqi.created_at) < $8
ORDER BY bqi.bill_to_customer_id ASC, sp.actual_delivery_date ASC NULLS LAST, sp.pro_number ASC
LIMIT $9`

func (r *repository) ListConsolidationCandidates(
	ctx context.Context,
	req *billingqueueports.ListConsolidationCandidatesRequest,
) ([]*billingqueueports.ConsolidationCandidate, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*billingqueueports.ConsolidationCandidate, error) {
			if len(req.CustomerIDs) == 0 {
				return nil, nil
			}

			limit := req.Limit
			if limit <= 0 {
				limit = defaultCandidateLimit
			}

			candidates, err := keel.QueryAll(
				ctx,
				r.db.Keel(ctx),
				pgx.RowToAddrOfStructByName[billingqueueports.ConsolidationCandidate],
				consolidationCandidatesSQL,
				req.TenantInfo.OrgID.String(),
				req.TenantInfo.BuID.String(),
				string(shipment.StatusCanceled),
				string(billingqueue.StatusApproved),
				string(billingqueue.BillTypeInvoice),
				keel.Strings(req.CustomerIDs),
				[]string{string(shipment.StatusReadyToInvoice), string(shipment.StatusCompleted)},
				req.PeriodEnd,
				limit,
			)
			if err != nil {
				r.l.Error("failed to list consolidation candidates", zap.Error(err))
				return nil, err
			}

			return candidates, nil
		},
	)
}
