package accountsreceivablerepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/receivablesql"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerledger/customerledgerdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment/customerpaymentdb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice/invoicedb"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerledgerports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer/customerdb"
	"github.com/emoss08/trenova/internal/modules/parties/domain/factoringcompany/factoringcompanydb"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

type agingRowRecord struct {
	CustomerID      string
	CustomerName    string
	CurrentMinor    int64
	Days1To30Minor  int64
	Days31To60Minor int64
	Days61To90Minor int64
	DaysOver90Minor int64
	TotalOpenMinor  int64
}

type openItemRecord struct {
	InvoiceID            string
	CustomerID           string
	CustomerName         string
	InvoiceNumber        string
	BillType             string
	InvoiceDate          int64
	DueDate              int64
	CurrencyCode         string
	ShipmentProNumber    string
	ShipmentBOL          string
	TotalAmountMinor     int64
	AppliedAmountMinor   int64
	OpenAmountMinor      int64
	DaysPastDue          int
	SettlementStatus     string
	DisputeStatus        string
	HasShortPay          bool
	FactoringCompanyID   string
	FactoringCompanyName string
}

func New(p Params) customerledgerports.AccountsReceivableRepository {
	return &repository{db: p.DB, l: p.Logger.Named("postgres.accounts-receivable-repository")}
}

func NewAnalytics(p Params) customerledgerports.ARAnalyticsRepository {
	return &repository{db: p.DB, l: p.Logger.Named("postgres.ar-analytics-repository")}
}

func (r *repository) ListCustomerLedger(
	ctx context.Context,
	req customerledgerports.ListCustomerLedgerRequest,
) ([]*customerledgerports.ARLedgerEntry, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*customerledgerports.ARLedgerEntry, error) {
			cle := customerledgerdb.CustomerLedgerEntries
			rows, err := keel.Into[customerledgerports.ARLedgerEntry](ctx, r.db.Keel(ctx), cle.
				Pick(
					cle.CustomerID,
					cle.TransactionDate,
					cle.SourceEventType,
					cle.DocumentNumber.Coalesce(""),
					cle.SourceObjectType,
					cle.SourceObjectID,
					cle.AmountMinor,
					cle.RelatedInvoiceID.Coalesce(""),
				).
				Where(cle.CustomerID.Eq(req.CustomerID)).
				Order(cle.TransactionDate.Asc(), cle.LineNumber.Asc()))
			if err != nil {
				return nil, err
			}

			entries := make([]*customerledgerports.ARLedgerEntry, 0, len(rows))
			for i := range rows {
				entries = append(entries, &rows[i])
			}
			return entries, nil
		},
	)
}

func pastDueDays(inv *invoicedb.InvoicesTable, asOf int64) keel.DynExpr {
	return keel.DynLit(asOf).Sub(inv.DueDate).Div(secondsPerDay)
}

func daysPastDue(inv *invoicedb.InvoicesTable, asOf int64) keel.DynExpr {
	return keel.DynCase().
		When(keel.Or(inv.DueDate.IsNull(), inv.DueDate.Gte(asOf)), 0).
		Else(keel.Int64Of(pastDueDays(inv, asOf)).Greatest(keel.Int64(0)))
}

func amountWhen(p keel.Pred, amount any) keel.DynExpr {
	return keel.DynCase().When(p, amount).Else(0)
}

func agingBuckets(due, asOf, open keel.DynExpr) []keel.DynExpr {
	days := asOf.Sub(due).Div(secondsPerDay)
	overdue := due.Lt(asOf)
	return []keel.DynExpr{
		amountWhen(keel.Or(due.IsNull(), due.Gte(asOf)), open).Sum(),
		amountWhen(keel.And(overdue, days.Between(1, 30)), open).Sum(),
		amountWhen(keel.And(overdue, days.Between(31, 60)), open).Sum(),
		amountWhen(keel.And(overdue, days.Between(61, 90)), open).Sum(),
		amountWhen(keel.And(overdue, days.Gt(90)), open).Sum(),
	}
}

func invoiceAgingBuckets(inv *invoicedb.InvoicesTable, asOf int64) []keel.DynExpr {
	return agingBuckets(keel.DynOf(inv.DueDate), keel.DynLit(asOf), receivablesql.OpenMinor(inv))
}

func agingQuery(inv *invoicedb.InvoicesTable, asOf int64) *keel.Query[*invoice.Invoice] {
	buckets := invoiceAgingBuckets(inv, asOf)
	return inv.
		Pick(
			inv.CustomerID,
			inv.BillToName,
			buckets[0],
			buckets[1],
			buckets[2],
			buckets[3],
			buckets[4],
			receivablesql.OpenMinor(inv).Sum(),
		).
		Where(receivablesql.Open(inv)).
		Group(inv.CustomerID, inv.BillToName)
}

func postedApplications(
	cpa *customerpaymentdb.ApplicationsTable,
	cp *customerpaymentdb.PaymentsTable,
) *keel.Query[*customerpayment.Application] {
	return cpa.Join(
		cp,
		cp.ID.EqCol(cpa.CustomerPaymentID),
		cp.Status.Eq(customerpayment.StatusPosted),
	)
}

func hasShortPay(inv *invoicedb.InvoicesTable) keel.Pred {
	cpa := customerpaymentdb.Applications.As("cpa")
	cp := customerpaymentdb.Payments.As("cp")
	return keel.Exists(postedApplications(cpa, cp).
		Where(cpa.InvoiceID.EqCol(inv.ID), cpa.ShortPayAmountMinor.Gt(0)))
}

func (rec *agingRowRecord) row() *customerledgerports.ARCustomerAgingRow {
	return &customerledgerports.ARCustomerAgingRow{
		CustomerID:   pulid.ID(rec.CustomerID),
		CustomerName: rec.CustomerName,
		Buckets: customerledgerports.ARAgingBucketTotals{
			CurrentMinor:    rec.CurrentMinor,
			Days1To30Minor:  rec.Days1To30Minor,
			Days31To60Minor: rec.Days31To60Minor,
			Days61To90Minor: rec.Days61To90Minor,
			DaysOver90Minor: rec.DaysOver90Minor,
			TotalOpenMinor:  rec.TotalOpenMinor,
		},
	}
}

func (r *repository) ListARAging(
	ctx context.Context,
	req customerledgerports.ListARAgingRequest,
) ([]*customerledgerports.ARCustomerAgingRow, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*customerledgerports.ARCustomerAgingRow, error) {
			inv := invoicedb.Invoices
			records, err := keel.Into[agingRowRecord](
				ctx,
				r.db.Keel(ctx),
				agingQuery(inv, req.AsOfDate).
					Order(inv.BillToName.Asc()),
			)
			if err != nil {
				return nil, err
			}
			rows := make([]*customerledgerports.ARCustomerAgingRow, 0, len(records))
			for i := range records {
				rows = append(rows, records[i].row())
			}
			return rows, nil
		},
	)
}

func (r *repository) ListOpenItems(
	ctx context.Context,
	req customerledgerports.ListAROpenItemsRequest,
) ([]*customerledgerports.AROpenItem, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*customerledgerports.AROpenItem, error) {
			inv := invoicedb.Invoices
			fac := factoringcompanydb.FactoringCompanies
			records, err := keel.Into[openItemRecord](ctx, r.db.Keel(ctx), inv.
				Pick(
					inv.ID,
					inv.CustomerID,
					inv.BillToName,
					inv.Number,
					inv.BillType,
					inv.InvoiceDate,
					inv.DueDate.Coalesce(0),
					inv.CurrencyCode,
					inv.ShipmentProNumber.Coalesce(""),
					inv.ShipmentBOL.Coalesce(""),
					inv.TotalAmountMinor,
					inv.AppliedAmountMinor,
					receivablesql.OpenMinor(inv),
					daysPastDue(inv, req.AsOfDate),
					inv.SettlementStatus,
					inv.DisputeStatus,
					hasShortPay(inv),
					inv.FactoringCompanyID.Coalesce(""),
					fac.Name.Coalesce(""),
				).
				LeftJoin(fac, fac.ID.EqCol(inv.FactoringCompanyID)).
				Where(
					receivablesql.Open(inv),
					keel.When(!req.CustomerID.IsNil(), inv.CustomerID.Eq(req.CustomerID)),
				).
				Order(inv.DueDate.Asc().NullsFirst(), inv.InvoiceDate.Asc(), inv.Number.Asc()))
			if err != nil {
				return nil, fmt.Errorf("list ar open items: %w", err)
			}

			items := make([]*customerledgerports.AROpenItem, 0, len(records))
			for _, rec := range records {
				items = append(items, &customerledgerports.AROpenItem{
					InvoiceID:          pulid.ID(rec.InvoiceID),
					CustomerID:         pulid.ID(rec.CustomerID),
					CustomerName:       rec.CustomerName,
					InvoiceNumber:      rec.InvoiceNumber,
					BillType:           rec.BillType,
					InvoiceDate:        rec.InvoiceDate,
					DueDate:            rec.DueDate,
					CurrencyCode:       rec.CurrencyCode,
					ShipmentProNumber:  rec.ShipmentProNumber,
					ShipmentBOL:        rec.ShipmentBOL,
					TotalAmountMinor:   rec.TotalAmountMinor,
					AppliedAmountMinor: rec.AppliedAmountMinor,
					OpenAmountMinor:    rec.OpenAmountMinor,
					DaysPastDue:        rec.DaysPastDue,
					SettlementStatus:   rec.SettlementStatus,
					DisputeStatus:      rec.DisputeStatus,
					HasShortPay:        rec.HasShortPay,
					FactoringCompanyID: pulid.ID(rec.FactoringCompanyID),
					FactoringCompany:   rec.FactoringCompanyName,
				})
			}

			return items, nil
		},
	)
}

func (r *repository) GetCustomerName(
	ctx context.Context,
	req customerledgerports.GetARCustomerNameRequest,
) (string, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (string, error) {
		cus := customerdb.Customers
		name, err := keel.IntoOne[string](ctx, r.db.Keel(ctx), cus.
			Pick(cus.Name).
			Where(cus.ID.Eq(req.CustomerID)))
		if err != nil {
			return "", dberror.HandleNotFoundError(err, "Customer")
		}

		return name, nil
	})
}

func (r *repository) GetCustomerAging(
	ctx context.Context,
	req customerledgerports.GetARCustomerAgingRequest,
) (*customerledgerports.ARCustomerAgingRow, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*customerledgerports.ARCustomerAgingRow, error) {
			inv := invoicedb.Invoices
			records, err := keel.Into[agingRowRecord](
				ctx,
				r.db.Keel(ctx),
				agingQuery(inv, req.AsOfDate).
					Where(inv.CustomerID.Eq(req.CustomerID)),
			)
			if err != nil {
				return nil, fmt.Errorf("get customer ar aging: %w", err)
			}
			if len(records) == 0 {
				return &customerledgerports.ARCustomerAgingRow{CustomerID: req.CustomerID}, nil
			}

			return records[0].row(), nil
		},
	)
}
