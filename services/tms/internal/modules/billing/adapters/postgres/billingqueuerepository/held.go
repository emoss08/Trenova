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

const heldForPeriodSQL = `SELECT
	bqi.bill_to_customer_id AS customer_id,
	COUNT(*) AS shipment_count,
	COALESCE(SUM(bqi.allocated_total_amount), 0) AS total_amount
FROM billing_queue_items AS bqi
JOIN shipments AS sp ON sp.id = bqi.shipment_id
	AND sp.organization_id = bqi.organization_id
	AND sp.business_unit_id = bqi.business_unit_id
WHERE bqi.organization_id = $1
	AND bqi.business_unit_id = $2
	AND bqi.status = ANY($3)
	AND bqi.bill_type = $4
	AND bqi.invoice_id IS NULL
	AND bqi.is_adjustment_origin = FALSE
	AND bqi.bill_to_customer_id = ANY($5)
	AND sp.status <> $6
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
	AND COALESCE(sp.actual_delivery_date, bqi.created_at) < $7
GROUP BY bqi.bill_to_customer_id`

var heldStatuses = []string{
	string(billingqueue.StatusReadyForReview),
	string(billingqueue.StatusInReview),
	string(billingqueue.StatusOnHold),
	string(billingqueue.StatusSentBackToOps),
	string(billingqueue.StatusException),
}

func (r *repository) CountHeldForPeriod(
	ctx context.Context,
	req *billingqueueports.CountHeldForPeriodRequest,
) ([]*billingqueueports.HeldForPeriod, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*billingqueueports.HeldForPeriod, error) {
			if len(req.CustomerIDs) == 0 {
				return nil, nil
			}

			held, err := keel.QueryAll(
				ctx,
				r.db.Keel(ctx),
				pgx.RowToAddrOfStructByName[billingqueueports.HeldForPeriod],
				heldForPeriodSQL,
				req.TenantInfo.OrgID.String(),
				req.TenantInfo.BuID.String(),
				heldStatuses,
				string(billingqueue.BillTypeInvoice),
				keel.Strings(req.CustomerIDs),
				string(shipment.StatusCanceled),
				req.PeriodEnd,
			)
			if err != nil {
				r.l.Error("failed to count held billing queue items", zap.Error(err))
				return nil, err
			}

			return held, nil
		},
	)
}
