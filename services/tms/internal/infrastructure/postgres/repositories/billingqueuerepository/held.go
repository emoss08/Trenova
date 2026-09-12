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

// CountHeldForPeriod is the freight that belongs to a customer's open period but
// is still waiting on a biller.
//
// A statement that showed only approved freight understated the period: a biller
// looking at 197 shipments had no way to know 12 more were sitting in review and
// would miss the invoice. This is the same eligibility as
// ListConsolidationCandidates, inverted on status — anything not yet Approved, and
// not already dead or billed — including freight from earlier periods, which is
// just as owed.
//
// Grouped by customer so one query answers for every statement on the list.
func (r *repository) CountHeldForPeriod(
	ctx context.Context,
	req *repositories.CountHeldForPeriodRequest,
) ([]*repositories.HeldForPeriod, error) {
	if len(req.CustomerIDs) == 0 {
		return nil, nil
	}

	held := make([]*repositories.HeldForPeriod, 0, len(req.CustomerIDs))
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*billingqueue.BillingQueueItem)(nil)).
		ColumnExpr("sp.customer_id AS customer_id").
		ColumnExpr("COUNT(*) AS shipment_count").
		ColumnExpr("COALESCE(SUM(sp.total_charge_amount), 0) AS total_amount").
		Join("JOIN shipments AS sp ON sp.id = bqi.shipment_id").
		Join("AND sp.organization_id = bqi.organization_id").
		Join("AND sp.business_unit_id = bqi.business_unit_id").
		Where("bqi.organization_id = ?", req.TenantInfo.OrgID).
		Where("bqi.business_unit_id = ?", req.TenantInfo.BuID).
		// Everything a biller could still act on. Canceled is gone for good, and
		// Approved or Posted is already counted on the statement itself.
		Where("bqi.status IN (?)", bun.List([]billingqueue.Status{
			billingqueue.StatusReadyForReview,
			billingqueue.StatusInReview,
			billingqueue.StatusOnHold,
			billingqueue.StatusSentBackToOps,
			billingqueue.StatusException,
		})).
		Where("bqi.bill_type = ?", billingqueue.BillTypeInvoice).
		Where("bqi.invoice_id IS NULL").
		Where("bqi.is_adjustment_origin = FALSE").
		Where("sp.customer_id IN (?)", bun.List(req.CustomerIDs)).
		Where("sp.status <> ?", shipment.StatusCanceled).
		// Already billed. The invoice_id back-link is the primary double-bill guard,
		// but it was backfilled conservatively for legacy order-grouped invoices, and
		// a leg on a Draft invoice stays Approved until posting. This reads the truth
		// straight from the invoice lines, so freight that is on any invoice is never
		// swept onto a statement however it got there.
		Where(`NOT EXISTS (
			SELECT 1 FROM invoice_lines AS il
			WHERE il.shipment_id = sp.id
			  AND il.organization_id = sp.organization_id
			  AND il.business_unit_id = sp.business_unit_id
		)`).
		Where("COALESCE(sp.actual_delivery_date, bqi.created_at) < ?", req.PeriodEnd).
		GroupExpr("sp.customer_id").
		Scan(ctx, &held)
	if err != nil {
		r.l.Error("failed to count held billing queue items", zap.Error(err))
		return nil, fmt.Errorf("count held for period: %w", err)
	}

	return held, nil
}
