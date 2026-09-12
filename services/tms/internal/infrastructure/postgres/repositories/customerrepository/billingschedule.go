package customerrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/zap"
)

// AdvanceBilledPeriod moves a customer's billing watermark past the period a run
// just billed.
//
// The GREATEST guard means the watermark only ever moves forward. A retry, or a
// run committed out of order, must not drag it backwards — that would re-open a
// period whose invoices already exist and bill the same freight twice.
func (r *repository) AdvanceBilledPeriod(
	ctx context.Context,
	req *repositories.AdvanceBilledPeriodRequest,
) error {
	cbp := buncolgen.CustomerBillingProfileColumns

	if _, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*customer.CustomerBillingProfile)(nil)).
		Set(
			cbp.LastBilledPeriodEnd.SetExpr("GREATEST(COALESCE({}, 0), ?)"),
			req.PeriodEnd,
		).
		Where(cbp.CustomerID.Eq(), req.CustomerID).
		Where(cbp.OrganizationID.Eq(), req.TenantInfo.OrgID).
		Where(cbp.BusinessUnitID.Eq(), req.TenantInfo.BuID).
		Exec(ctx); err != nil {
		r.l.Error("failed to advance billed period", zap.Error(err))
		return fmt.Errorf("advance billed period: %w", err)
	}

	return nil
}

// ListDueBillingSchedules is every statement-billed customer, optionally narrowed
// to one tenant.
//
// Cross-tenant when the request carries no tenant: the consolidation sweep runs
// once for the whole deployment rather than once per organization, so the tenant
// travels with each row. Whether a period has actually closed is decided in Go,
// because the answer depends on the customer's own time zone and DST, which SQL
// cannot express portably.
func (r *repository) ListDueBillingSchedules(
	ctx context.Context,
	req *repositories.ListBillingSchedulesRequest,
) ([]*repositories.DueBillingSchedule, error) {
	cbp := buncolgen.CustomerBillingProfileColumns
	cus := buncolgen.CustomerColumns

	schedules := make([]*repositories.DueBillingSchedule, 0)
	q := r.db.DBForContext(ctx).
		NewSelect().
		Model((*customer.CustomerBillingProfile)(nil)).
		ColumnExpr(cbp.OrganizationID.Qualified()).
		ColumnExpr(cbp.BusinessUnitID.Qualified()).
		ColumnExpr(cbp.CustomerID.Qualified()).
		ColumnExpr(cus.Name.Qualified() + " AS customer_name").
		ColumnExpr(cus.Code.Qualified() + " AS customer_code").
		ColumnExpr(cus.Status.Qualified() + " AS customer_status").
		ColumnExpr(cbp.BillingCycle.Qualified()).
		ColumnExpr(cbp.BillingCycleAnchorDay.Qualified()).
		ColumnExpr(cbp.BillingCycleTimezone.Qualified()).
		ColumnExpr(cbp.InvoiceDelivery.Qualified()).
		ColumnExpr(cbp.LastBilledPeriodEnd.Qualified()).
		ColumnExpr(cbp.SplitBy.Qualified()).
		ColumnExpr(cbp.SectionBy.Qualified()).
		ColumnExpr(cbp.InvoiceDetail.Qualified()).
		ColumnExpr(cbp.MinConsolidatedAmount.Qualified()).
		ColumnExpr(cbp.MaxShipmentsPerInvoice.Qualified()).
		ColumnExpr(cbp.AutoBill.Qualified()).
		ColumnExpr(cbp.BillingCurrency.Qualified()).
		Join("JOIN customers AS cus ON "+cus.ID.Qualified()+" = "+cbp.CustomerID.Qualified()).
		Join("AND "+cus.OrganizationID.Qualified()+" = "+cbp.OrganizationID.Qualified()).
		Join("AND "+cus.BusinessUnitID.Qualified()+" = "+cbp.BusinessUnitID.Qualified()).
		Where(cbp.InvoiceDelivery.Eq(), customer.InvoiceDeliveryConsolidated).
		Where(cbp.BillingCycle.Ne(), customer.BillingCycleImmediate).
		OrderExpr(cus.Name.Qualified() + " ASC")

	if !req.TenantInfo.OrgID.IsNil() {
		q = q.Where(cbp.OrganizationID.Eq(), req.TenantInfo.OrgID).
			Where(cbp.BusinessUnitID.Eq(), req.TenantInfo.BuID)
	}

	if err := q.Scan(ctx, &schedules); err != nil {
		r.l.Error("failed to list due billing schedules", zap.Error(err))
		return nil, fmt.Errorf("list due billing schedules: %w", err)
	}

	for _, schedule := range schedules {
		schedule.TenantInfo = pagination.TenantInfo{
			OrgID: schedule.OrganizationID,
			BuID:  schedule.BusinessUnitID,
		}
	}

	return schedules, nil
}
