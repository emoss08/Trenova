package accountsreceivablerepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/accountsreceivable"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
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
	CustomerID      string `bun:"customer_id"`
	CustomerName    string `bun:"customer_name"`
	CurrentMinor    int64  `bun:"current_minor"`
	Days1To30Minor  int64  `bun:"days1_to30_minor"`
	Days31To60Minor int64  `bun:"days31_to60_minor"`
	Days61To90Minor int64  `bun:"days61_to90_minor"`
	DaysOver90Minor int64  `bun:"days_over90_minor"`
	TotalOpenMinor  int64  `bun:"total_open_minor"`
}

func New(p Params) repositories.AccountsReceivableRepository {
	return &repository{db: p.DB, l: p.Logger.Named("postgres.accounts-receivable-repository")}
}

func (r *repository) ListCustomerLedger(ctx context.Context, req repositories.ListCustomerLedgerRequest) ([]*accountsreceivable.LedgerEntry, error) {
	entries := make([]*accountsreceivable.LedgerEntry, 0)
	err := r.db.DBForContext(ctx).NewRaw(`
		SELECT customer_id, transaction_date, event_type, document_number, source_object_id, amount_minor
		FROM (
			SELECT inv.customer_id,
				COALESCE(inv.posted_at, inv.invoice_date) AS transaction_date,
				CASE WHEN inv.bill_type = 'DebitMemo' THEN 'DebitMemoPosted' ELSE 'InvoicePosted' END AS event_type,
				inv.number AS document_number,
				inv.id::text AS source_object_id,
				inv.total_amount_minor AS amount_minor
			FROM invoices inv
			WHERE inv.organization_id = ?
			  AND inv.business_unit_id = ?
			  AND inv.customer_id = ?
			  AND inv.status = 'Posted'
			  AND inv.bill_type IN ('Invoice', 'DebitMemo')

			UNION ALL

			SELECT cp.customer_id,
				cp.accounting_date AS transaction_date,
				'CustomerPaymentApplied' AS event_type,
				cp.reference_number AS document_number,
				cp.id::text AS source_object_id,
				-cpa.applied_amount_minor AS amount_minor
			FROM customer_payments cp
			JOIN customer_payment_applications cpa
			  ON cpa.customer_payment_id = cp.id
			 AND cpa.organization_id = cp.organization_id
			 AND cpa.business_unit_id = cp.business_unit_id
			WHERE cp.organization_id = ?
			  AND cp.business_unit_id = ?
			  AND cp.customer_id = ?
			  AND cp.status = 'Posted'

			UNION ALL

			SELECT cp.customer_id,
				COALESCE(cp.reversed_at, cp.accounting_date) AS transaction_date,
				'CustomerPaymentReversed' AS event_type,
				cp.reference_number AS document_number,
				cp.id::text AS source_object_id,
				cp.applied_amount_minor AS amount_minor
			FROM customer_payments cp
			WHERE cp.organization_id = ?
			  AND cp.business_unit_id = ?
			  AND cp.customer_id = ?
			  AND cp.status = 'Reversed'
			  AND cp.applied_amount_minor > 0
		) ledger
		ORDER BY transaction_date ASC, event_type ASC
	`, req.TenantInfo.OrgID, req.TenantInfo.BuID, req.CustomerID, req.TenantInfo.OrgID, req.TenantInfo.BuID, req.CustomerID, req.TenantInfo.OrgID, req.TenantInfo.BuID, req.CustomerID).Scan(ctx, &entries)
	if err != nil {
		return nil, err
	}
	return entries, nil
}

func (r *repository) ListARAging(ctx context.Context, req repositories.ListARAgingRequest) ([]*accountsreceivable.CustomerAgingRow, error) {
	records := make([]*agingRowRecord, 0)
	err := r.db.DBForContext(ctx).NewRaw(`
		SELECT
			inv.customer_id,
			inv.bill_to_name AS customer_name,
			SUM(CASE WHEN inv.due_date IS NULL OR inv.due_date >= ? THEN inv.total_amount_minor - inv.applied_amount_minor ELSE 0 END) AS current_minor,
			SUM(CASE WHEN inv.due_date < ? AND (? - inv.due_date) / 86400 BETWEEN 1 AND 30 THEN inv.total_amount_minor - inv.applied_amount_minor ELSE 0 END) AS days1_to30_minor,
			SUM(CASE WHEN inv.due_date < ? AND (? - inv.due_date) / 86400 BETWEEN 31 AND 60 THEN inv.total_amount_minor - inv.applied_amount_minor ELSE 0 END) AS days31_to60_minor,
			SUM(CASE WHEN inv.due_date < ? AND (? - inv.due_date) / 86400 BETWEEN 61 AND 90 THEN inv.total_amount_minor - inv.applied_amount_minor ELSE 0 END) AS days61_to90_minor,
			SUM(CASE WHEN inv.due_date < ? AND (? - inv.due_date) / 86400 > 90 THEN inv.total_amount_minor - inv.applied_amount_minor ELSE 0 END) AS days_over90_minor,
			SUM(inv.total_amount_minor - inv.applied_amount_minor) AS total_open_minor
		FROM invoices inv
		WHERE inv.organization_id = ?
		  AND inv.business_unit_id = ?
		  AND inv.status = 'Posted'
		  AND inv.bill_type IN ('Invoice', 'DebitMemo')
		  AND inv.total_amount_minor > inv.applied_amount_minor
		GROUP BY inv.customer_id, inv.bill_to_name
		ORDER BY inv.bill_to_name ASC
	`, req.AsOfDate, req.AsOfDate, req.AsOfDate, req.AsOfDate, req.AsOfDate, req.AsOfDate, req.AsOfDate, req.AsOfDate, req.AsOfDate, req.TenantInfo.OrgID, req.TenantInfo.BuID).Scan(ctx, &records)
	if err != nil {
		return nil, err
	}
	rows := make([]*accountsreceivable.CustomerAgingRow, 0, len(records))
	for _, rec := range records {
		rows = append(rows, &accountsreceivable.CustomerAgingRow{
			CustomerID:   pulid.ID(rec.CustomerID),
			CustomerName: rec.CustomerName,
			Buckets: accountsreceivable.AgingBucketTotals{
				CurrentMinor:    rec.CurrentMinor,
				Days1To30Minor:  rec.Days1To30Minor,
				Days31To60Minor: rec.Days31To60Minor,
				Days61To90Minor: rec.Days61To90Minor,
				DaysOver90Minor: rec.DaysOver90Minor,
				TotalOpenMinor:  rec.TotalOpenMinor,
			},
		})
	}
	return rows, nil
}
