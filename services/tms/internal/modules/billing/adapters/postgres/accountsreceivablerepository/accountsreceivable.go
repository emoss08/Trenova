package accountsreceivablerepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/receivablesql"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerledger"
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

type ledgerEntryRecord struct {
	ID                  pulid.ID
	CustomerID          pulid.ID
	TransactionDate     int64
	LineNumber          int64
	EventType           string
	DocumentNumber      string
	SourceObjectType    string
	SourceObjectID      string
	AmountMinor         int64
	RelatedInvoiceID    pulid.ID
	SupersededInvoiceID pulid.ID
}

func (rec *ledgerEntryRecord) entry() *customerledgerports.ARLedgerEntry {
	return &customerledgerports.ARLedgerEntry{
		CustomerID:          rec.CustomerID,
		TransactionDate:     rec.TransactionDate,
		EventType:           rec.EventType,
		DocumentNumber:      rec.DocumentNumber,
		SourceObjectType:    rec.SourceObjectType,
		SourceObjectID:      rec.SourceObjectID,
		AmountMinor:         rec.AmountMinor,
		RelatedInvoiceID:    rec.RelatedInvoiceID,
		SupersededInvoiceID: rec.SupersededInvoiceID,
		Position: customerledgerports.ARLedgerPosition{
			TransactionDate: rec.TransactionDate,
			LineNumber:      rec.LineNumber,
			ID:              rec.ID,
		},
	}
}

func (r *repository) GetCustomerLedgerBalance(
	ctx context.Context,
	req customerledgerports.GetCustomerLedgerBalanceRequest,
) (int64, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int64, error) {
		parts, err := keel.Into[int64](
			ctx,
			r.db.Keel(ctx),
			ledgerBalanceQuery(req.CustomerID, req.Before),
		)
		if err != nil {
			return 0, fmt.Errorf("get customer ledger balance: %w", err)
		}

		var balance int64
		for _, part := range parts {
			balance += part
		}
		return balance, nil
	})
}

func ledgerBalanceQuery(customerID pulid.ID, before int64) keel.Compound {
	month := customerledger.MonthStart(before)
	return keel.UnionAll(
		closedMonthsQuery(customerID, month),
		openingMonthQuery(customerID, month, before),
	)
}

func closedMonthsQuery(
	customerID pulid.ID,
	month int64,
) keel.Query[*customerledger.MonthlyBalance] {
	mb := customerledgerdb.MonthlyBalances
	return mb.
		Pick(keel.NumericOf(mb.AmountMinor.Sum()).Coalesce(0)).
		Where(mb.CustomerID.Eq(customerID), mb.MonthStart.Lt(month))
}

func openingMonthQuery(
	customerID pulid.ID,
	month, before int64,
) keel.Query[*customerledger.CustomerLedgerEntry] {
	cle := customerledgerdb.CustomerLedgerEntries
	return cle.
		Pick(keel.NumericOf(cle.AmountMinor.Sum()).Coalesce(0)).
		Where(
			cle.CustomerID.Eq(customerID),
			cle.TransactionDate.Gte(month),
			cle.TransactionDate.Lt(before),
		)
}

func (r *repository) ListCustomerLedgerWindow(
	ctx context.Context,
	req customerledgerports.ListCustomerLedgerWindowRequest,
) ([]*customerledgerports.ARLedgerEntry, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*customerledgerports.ARLedgerEntry, error) {
			records, err := keel.Into[ledgerEntryRecord](
				ctx,
				r.db.Keel(ctx),
				ledgerWindowQuery(&req),
			)
			if err != nil {
				return nil, fmt.Errorf("list customer ledger window: %w", err)
			}

			entries := make([]*customerledgerports.ARLedgerEntry, 0, len(records))
			for i := range records {
				entries = append(entries, records[i].entry())
			}
			return entries, nil
		},
	)
}

func ledgerWindowQuery(
	req *customerledgerports.ListCustomerLedgerWindowRequest,
) keel.Query[*customerledger.CustomerLedgerEntry] {
	cle := customerledgerdb.CustomerLedgerEntries
	replaced := invoicedb.Invoices.As("replaced")
	return cle.
		Pick(
			cle.ID,
			cle.CustomerID,
			cle.TransactionDate,
			cle.LineNumber,
			cle.SourceEventType,
			cle.DocumentNumber.Coalesce(""),
			cle.SourceObjectType,
			cle.SourceObjectID,
			cle.AmountMinor,
			cle.RelatedInvoiceID.Coalesce(""),
			replaced.ID.Coalesce(""),
		).
		LeftJoin(
			replaced,
			replaced.ID.EqCol(receivablesql.LedgerInvoiceID(cle)),
			replaced.SupersededByInvoiceID.IsNotNull(),
		).
		Where(
			cle.CustomerID.Eq(req.CustomerID),
			keel.When(req.From > 0, cle.TransactionDate.Gte(req.From)),
			cle.TransactionDate.Lte(req.Through),
			afterLedgerPosition(cle, req.After),
		).
		Order(cle.TransactionDate.Asc(), cle.LineNumber.Asc(), cle.ID.Asc()).
		Limit(req.Limit)
}

func afterLedgerPosition(
	cle *customerledgerdb.CustomerLedgerEntriesTable,
	after *customerledgerports.ARLedgerPosition,
) keel.Pred {
	if after == nil {
		return nil
	}
	return keel.Tuple(cle.TransactionDate, cle.LineNumber, cle.ID).
		Gt(keel.Tuple(after.TransactionDate, after.LineNumber, after.ID.String()))
}

func pastDueDays(inv *invoicedb.InvoicesTable, asOf int64) keel.Int64Expr {
	return keel.Int64(asOf).Sub(inv.DueDate).Div(keel.Int64(secondsPerDay))
}

func daysPastDue(inv *invoicedb.InvoicesTable, asOf int64) keel.Int64Expr {
	return keel.CaseInt64().
		When(keel.Or(inv.DueDate.IsNull(), inv.DueDate.Gte(asOf)), keel.Int64(0)).
		Else(pastDueDays(inv, asOf).Greatest(keel.Int64(0)))
}

func amountWhen(p keel.Pred, amount keel.Int64Value) keel.Int64Expr {
	return keel.CaseInt64().When(p, amount).Else(keel.Int64(0))
}

func agingBuckets(dueDate, asOf, open keel.Int64Value) []keel.Agg {
	due := keel.Int64Of(dueDate)
	days := keel.Int64Of(asOf).Sub(due).Div(keel.Int64(secondsPerDay))
	overdue := due.LtExpr(asOf)
	return []keel.Agg{
		amountWhen(keel.Or(due.IsNull(), due.GteExpr(asOf)), open).Sum(),
		amountWhen(keel.And(overdue, days.Between(1, 30)), open).Sum(),
		amountWhen(keel.And(overdue, days.Between(31, 60)), open).Sum(),
		amountWhen(keel.And(overdue, days.Between(61, 90)), open).Sum(),
		amountWhen(keel.And(overdue, days.Gt(90)), open).Sum(),
	}
}

func invoiceAgingBuckets(inv *invoicedb.InvoicesTable, asOf int64) []keel.Agg {
	return agingBuckets(inv.DueDate, keel.Int64(asOf), receivablesql.OpenMinor(inv))
}

func agingQuery(inv *invoicedb.InvoicesTable, asOf int64) keel.Query[*invoice.Invoice] {
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
) keel.Query[*customerpayment.Application] {
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
		Where(
			cpa.InvoiceID.EqCol(inv.ID),
			keel.Int64Of(cpa.ShortPayAmountMinor).GtExpr(keel.ConstInt64(0)),
		))
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
