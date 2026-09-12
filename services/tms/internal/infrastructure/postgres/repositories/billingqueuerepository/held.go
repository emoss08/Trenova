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
// would miss the invoice. This is the same window and the same eligibility as
// ListConsolidationCandidates, inverted on status — anything not yet Approved,
// and not already dead.
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
		Join("LEFT JOIN customer_billing_profiles AS cbp ON cbp.customer_id = sp.customer_id").
		Join("AND cbp.organization_id = sp.organization_id").
		Join("AND cbp.business_unit_id = sp.business_unit_id").
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
		Where(
			"COALESCE(sp.actual_delivery_date, bqi.created_at) >= ? - COALESCE(cbp.consolidation_lookback_days, 30) * 86400",
			req.PeriodStart,
		).
		Where("COALESCE(sp.actual_delivery_date, bqi.created_at) < ?", req.PeriodEnd).
		GroupExpr("sp.customer_id").
		Scan(ctx, &held)
	if err != nil {
		r.l.Error("failed to count held billing queue items", zap.Error(err))
		return nil, fmt.Errorf("count held for period: %w", err)
	}

	return held, nil
}
