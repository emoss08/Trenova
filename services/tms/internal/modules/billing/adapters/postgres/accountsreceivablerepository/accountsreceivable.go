package accountsreceivablerepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/receivablesql"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerledger/customerledgerdb"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerledgerports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer/customerdb"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/jackc/pgx/v5"
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

const agingColumnsSQL = `
			inv.customer_id,
			inv.bill_to_name AS customer_name,
			SUM(CASE WHEN inv.due_date IS NULL OR inv.due_date >= $1 THEN inv.total_amount_minor - inv.applied_amount_minor ELSE 0 END) AS current_minor,
			SUM(CASE WHEN inv.due_date < $1 AND ($1 - inv.due_date) / 86400 BETWEEN 1 AND 30 THEN inv.total_amount_minor - inv.applied_amount_minor ELSE 0 END) AS days1_to30_minor,
			SUM(CASE WHEN inv.due_date < $1 AND ($1 - inv.due_date) / 86400 BETWEEN 31 AND 60 THEN inv.total_amount_minor - inv.applied_amount_minor ELSE 0 END) AS days31_to60_minor,
			SUM(CASE WHEN inv.due_date < $1 AND ($1 - inv.due_date) / 86400 BETWEEN 61 AND 90 THEN inv.total_amount_minor - inv.applied_amount_minor ELSE 0 END) AS days61_to90_minor,
			SUM(CASE WHEN inv.due_date < $1 AND ($1 - inv.due_date) / 86400 > 90 THEN inv.total_amount_minor - inv.applied_amount_minor ELSE 0 END) AS days_over90_minor,
			SUM(inv.total_amount_minor - inv.applied_amount_minor) AS total_open_minor`

const agingSQL = `
		SELECT` + agingColumnsSQL + `
		FROM invoices inv
		WHERE inv.organization_id = $2
		  AND inv.business_unit_id = $3
		  AND ` + receivablesql.OpenInvoice + `
		GROUP BY inv.customer_id, inv.bill_to_name
		ORDER BY inv.bill_to_name ASC`

const customerAgingSQL = `
		SELECT` + agingColumnsSQL + `
		FROM invoices inv
		WHERE inv.organization_id = $2
		  AND inv.business_unit_id = $3
		  AND inv.customer_id = $4
		  AND ` + receivablesql.OpenInvoice + `
		GROUP BY inv.customer_id, inv.bill_to_name`

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

func tenantArgs(tenantInfo pagination.TenantInfo) (orgID, buID string) {
	return tenantInfo.OrgID.String(), tenantInfo.BuID.String()
}

func (r *repository) ListARAging(
	ctx context.Context,
	req customerledgerports.ListARAgingRequest,
) ([]*customerledgerports.ARCustomerAgingRow, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*customerledgerports.ARCustomerAgingRow, error) {
			org, bu := tenantArgs(req.TenantInfo)
			records, err := keel.QueryAll(
				ctx,
				r.db.Keel(ctx),
				pgx.RowToAddrOfStructByName[agingRowRecord],
				agingSQL,
				req.AsOfDate, org, bu,
			)
			if err != nil {
				return nil, err
			}
			rows := make([]*customerledgerports.ARCustomerAgingRow, 0, len(records))
			for _, rec := range records {
				rows = append(rows, rec.row())
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
			query := `
		SELECT
			inv.id AS invoice_id,
			inv.customer_id,
			inv.bill_to_name AS customer_name,
			inv.number AS invoice_number,
			inv.bill_type,
			inv.invoice_date,
			COALESCE(inv.due_date, 0) AS due_date,
			inv.currency_code,
			COALESCE(inv.shipment_pro_number, '') AS shipment_pro_number,
			COALESCE(inv.shipment_bol, '') AS shipment_bol,
			inv.total_amount_minor,
			inv.applied_amount_minor,
			(inv.total_amount_minor - inv.applied_amount_minor) AS open_amount_minor,
			CASE
				WHEN inv.due_date IS NULL OR inv.due_date >= $1 THEN 0
				ELSE GREATEST((($1 - inv.due_date) / 86400)::INT, 0)
			END AS days_past_due,
			inv.settlement_status,
			inv.dispute_status,
			EXISTS (
				SELECT 1
				FROM customer_payment_applications cpa
				JOIN customer_payments cp
				  ON cp.id = cpa.customer_payment_id
				 AND cp.organization_id = cpa.organization_id
				 AND cp.business_unit_id = cpa.business_unit_id
				 AND cp.status = 'Posted'
				WHERE cpa.invoice_id = inv.id
				  AND cpa.organization_id = inv.organization_id
				  AND cpa.business_unit_id = inv.business_unit_id
				  AND cpa.short_pay_amount_minor > 0
			) AS has_short_pay,
			COALESCE(inv.factoring_company_id, '') AS factoring_company_id,
			COALESCE(fac.name, '') AS factoring_company_name
		FROM invoices inv
		LEFT JOIN factoring_companies fac
		  ON fac.id = inv.factoring_company_id
		 AND fac.organization_id = inv.organization_id
		 AND fac.business_unit_id = inv.business_unit_id
		WHERE inv.organization_id = $2
		  AND inv.business_unit_id = $3
		  AND ` + receivablesql.OpenInvoice
			org, bu := tenantArgs(req.TenantInfo)
			args := []any{req.AsOfDate, org, bu}
			if !req.CustomerID.IsNil() {
				query += `
		  AND inv.customer_id = $4`
				args = append(args, req.CustomerID.String())
			}
			query += `
		ORDER BY inv.due_date ASC NULLS FIRST, inv.invoice_date ASC, inv.number ASC`

			records, err := keel.QueryAll(
				ctx,
				r.db.Keel(ctx),
				pgx.RowToAddrOfStructByName[openItemRecord],
				query,
				args...,
			)
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
			org, bu := tenantArgs(req.TenantInfo)
			records, err := keel.QueryAll(
				ctx,
				r.db.Keel(ctx),
				pgx.RowToAddrOfStructByName[agingRowRecord],
				customerAgingSQL,
				req.AsOfDate, org, bu, req.CustomerID.String(),
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
