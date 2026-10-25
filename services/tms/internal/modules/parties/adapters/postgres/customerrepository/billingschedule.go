package customerrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer/customerdb"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type dueBillingScheduleRow struct {
	OrganizationID         pulid.ID
	BusinessUnitID         pulid.ID
	CustomerID             pulid.ID
	CustomerName           string
	CustomerCode           string
	CustomerStatus         string
	BillingCycle           customer.BillingCycle
	BillingCycleAnchorDay  int16
	BillingCycleTimezone   string
	InvoiceDelivery        customer.InvoiceDelivery
	LastBilledPeriodEnd    *int64
	SplitBy                customer.InvoiceSplitKey
	SectionBy              customer.InvoiceSectionKey
	InvoiceDetail          customer.InvoiceDetail
	MinConsolidatedAmount  decimal.NullDecimal
	MaxShipmentsPerInvoice int16
	AutoBill               bool
	BillingCurrency        string
}

// AdvanceBilledPeriod moves a customer's billing watermark past the period a run
// just billed.
//
// The GREATEST guard means the watermark only ever moves forward. A retry, or a
// run committed out of order, must not drag it backwards — that would re-open a
// period whose invoices already exist and bill the same freight twice.
func (r *repository) AdvanceBilledPeriod(
	ctx context.Context,
	req *customerports.AdvanceBilledPeriodRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		cbp := customerdb.CustomerBillingProfiles
		if _, err := cbp.Where(cbp.CustomerID.Eq(req.CustomerID)).Update(
			ctx,
			r.db.Keel(ctx),
			cbp.LastBilledPeriodEnd.ToExpr(keel.DynFunc(
				"greatest",
				keel.DynOf(cbp.LastBilledPeriodEnd).Coalesce(0),
				keel.DynLit(req.PeriodEnd).Cast(keel.TypeInt64),
			)),
			cbp.Version.ToExpr(cbp.Version),
			cbp.UpdatedAt.ToExpr(cbp.UpdatedAt),
		); err != nil {
			r.l.Error("failed to advance billed period", zap.Error(err))
			return fmt.Errorf("advance billed period: %w", err)
		}

		return nil
	})
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
	req *customerports.ListBillingSchedulesRequest,
) ([]*customerports.DueBillingSchedule, error) {
	allTenants := req.TenantInfo.OrgID.IsNil()
	if allTenants {
		ctx = dbscope.WithSystem(ctx, "list billing schedules due across every organization")
	}
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*customerports.DueBillingSchedule, error) {
			rows, err := keel.Into[dueBillingScheduleRow](
				ctx,
				r.db.Keel(ctx),
				dueBillingSchedulesQuery(allTenants),
			)
			if err != nil {
				r.l.Error("failed to list due billing schedules", zap.Error(err))
				return nil, fmt.Errorf("list due billing schedules: %w", err)
			}

			schedules := make([]*customerports.DueBillingSchedule, 0, len(rows))
			for i := range rows {
				schedules = append(schedules, rows[i].schedule())
			}

			return schedules, nil
		},
	)
}

func dueBillingSchedulesQuery(allTenants bool) *keel.Query[*customer.CustomerBillingProfile] {
	cbp := customerdb.CustomerBillingProfiles
	cus := customerdb.Customers

	q := cbp.Query()
	if allTenants {
		q = q.AllTenants()
	}

	return q.Join(
		cus,
		cus.ID.EqCol(cbp.CustomerID),
		cus.OrganizationID.EqCol(cbp.OrganizationID),
		cus.BusinessUnitID.EqCol(cbp.BusinessUnitID),
	).
		Pick(
			cbp.OrganizationID,
			cbp.BusinessUnitID,
			cbp.CustomerID,
			keel.DynOf(cus.Name).Coalesce(""),
			cus.Code,
			keel.DynOf(cus.Status).Cast(keel.TypeString),
			cbp.BillingCycle,
			cbp.BillingCycleAnchorDay,
			cbp.BillingCycleTimezone,
			cbp.InvoiceDelivery,
			cbp.LastBilledPeriodEnd,
			cbp.SplitBy,
			cbp.SectionBy,
			cbp.InvoiceDetail,
			cbp.MinConsolidatedAmount,
			cbp.MaxShipmentsPerInvoice,
			cbp.AutoBill,
			cbp.BillingCurrency,
		).
		Where(
			cbp.InvoiceDelivery.Eq(customer.InvoiceDeliveryConsolidated),
			cbp.BillingCycle.Ne(customer.BillingCycleImmediate),
		).
		Order(cus.Name.Asc())
}

func (row *dueBillingScheduleRow) schedule() *customerports.DueBillingSchedule {
	return &customerports.DueBillingSchedule{
		TenantInfo: pagination.TenantInfo{
			OrgID: row.OrganizationID,
			BuID:  row.BusinessUnitID,
		},
		OrganizationID:         row.OrganizationID,
		BusinessUnitID:         row.BusinessUnitID,
		CustomerID:             row.CustomerID,
		CustomerName:           row.CustomerName,
		CustomerCode:           row.CustomerCode,
		CustomerStatus:         row.CustomerStatus,
		BillingCycle:           row.BillingCycle,
		BillingCycleAnchorDay:  row.BillingCycleAnchorDay,
		BillingCycleTimezone:   row.BillingCycleTimezone,
		InvoiceDelivery:        row.InvoiceDelivery,
		LastBilledPeriodEnd:    row.LastBilledPeriodEnd,
		SplitBy:                row.SplitBy,
		SectionBy:              row.SectionBy,
		InvoiceDetail:          row.InvoiceDetail,
		MinConsolidatedAmount:  row.MinConsolidatedAmount,
		MaxShipmentsPerInvoice: row.MaxShipmentsPerInvoice,
		AutoBill:               row.AutoBill,
		BillingCurrency:        row.BillingCurrency,
	}
}
