package accountsreceivablerepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
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
	CustomerID      string `bun:"customer_id"`
	CustomerName    string `bun:"customer_name"`
	CurrentMinor    int64  `bun:"current_minor"`
	Days1To30Minor  int64  `bun:"days1_to30_minor"`
	Days31To60Minor int64  `bun:"days31_to60_minor"`
	Days61To90Minor int64  `bun:"days61_to90_minor"`
	DaysOver90Minor int64  `bun:"days_over90_minor"`
	TotalOpenMinor  int64  `bun:"total_open_minor"`
}

type openItemRecord struct {
	InvoiceID          string `bun:"invoice_id"`
	CustomerID         string `bun:"customer_id"`
	CustomerName       string `bun:"customer_name"`
	InvoiceNumber      string `bun:"invoice_number"`
	BillType           string `bun:"bill_type"`
	InvoiceDate        int64  `bun:"invoice_date"`
	DueDate            int64  `bun:"due_date"`
	CurrencyCode       string `bun:"currency_code"`
	ShipmentProNumber  string `bun:"shipment_pro_number"`
	ShipmentBOL        string `bun:"shipment_bol"`
	TotalAmountMinor   int64  `bun:"total_amount_minor"`
	AppliedAmountMinor int64  `bun:"applied_amount_minor"`
	OpenAmountMinor    int64  `bun:"open_amount_minor"`
	DaysPastDue        int    `bun:"days_past_due"`
}

func New(p Params) repositories.AccountsReceivableRepository {
	return &repository{db: p.DB, l: p.Logger.Named("postgres.accounts-receivable-repository")}
}

func (r *repository) ListCustomerLedger(
	ctx context.Context,
	req repositories.ListCustomerLedgerRequest,
) ([]*repositories.ARLedgerEntry, error) {
	entries := make([]*repositories.ARLedgerEntry, 0)
	err := r.db.DBForContext(ctx).NewRaw(`
		SELECT customer_id, transaction_date, source_event_type AS event_type, document_number, source_object_id, amount_minor
		FROM customer_ledger_entries
		WHERE organization_id = ?
		  AND business_unit_id = ?
		  AND customer_id = ?
		ORDER BY transaction_date ASC, line_number ASC
	`, req.TenantInfo.OrgID, req.TenantInfo.BuID, req.CustomerID).Scan(ctx, &entries)
	if err != nil {
		return nil, err
	}
	return entries, nil
}

func (r *repository) ListARAging(
	ctx context.Context,
	req repositories.ListARAgingRequest,
) ([]*repositories.ARCustomerAgingRow, error) {
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
	rows := make([]*repositories.ARCustomerAgingRow, 0, len(records))
	for _, rec := range records {
		rows = append(rows, &repositories.ARCustomerAgingRow{
			CustomerID:   pulid.ID(rec.CustomerID),
			CustomerName: rec.CustomerName,
			Buckets: repositories.ARAgingBucketTotals{
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

func (r *repository) ListOpenItems(
	ctx context.Context,
	req repositories.ListAROpenItemsRequest,
) ([]*repositories.AROpenItem, error) {
	records := make([]*openItemRecord, 0)
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
				WHEN inv.due_date IS NULL OR inv.due_date >= ? THEN 0
				ELSE GREATEST(((? - inv.due_date) / 86400)::INT, 0)
			END AS days_past_due
		FROM invoices inv
		WHERE inv.organization_id = ?
		  AND inv.business_unit_id = ?
		  AND inv.status = 'Posted'
		  AND inv.bill_type IN ('Invoice', 'DebitMemo')
		  AND inv.total_amount_minor > inv.applied_amount_minor`
	args := []any{req.AsOfDate, req.AsOfDate, req.TenantInfo.OrgID, req.TenantInfo.BuID}
	if !req.CustomerID.IsNil() {
		query += `
		  AND inv.customer_id = ?`
		args = append(args, req.CustomerID)
	}
	query += `
		ORDER BY inv.due_date ASC NULLS FIRST, inv.invoice_date ASC, inv.number ASC`

	err := r.db.DBForContext(ctx).NewRaw(query, args...).Scan(ctx, &records)
	if err != nil {
		return nil, fmt.Errorf("list ar open items: %w", err)
	}

	items := make([]*repositories.AROpenItem, 0, len(records))
	for _, rec := range records {
		items = append(items, &repositories.AROpenItem{
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
		})
	}

	return items, nil
}

func (r *repository) GetCustomerName(
	ctx context.Context,
	req repositories.GetARCustomerNameRequest,
) (string, error) {
	var row struct {
		Name string `bun:"name"`
	}
	err := r.db.DBForContext(ctx).NewRaw(`
		SELECT cus.name
		FROM customers cus
		WHERE cus.organization_id = ?
		  AND cus.business_unit_id = ?
		  AND cus.id = ?
		LIMIT 1
	`, req.TenantInfo.OrgID, req.TenantInfo.BuID, req.CustomerID).Scan(ctx, &row)
	if err != nil {
		return "", dberror.HandleNotFoundError(err, "Customer")
	}

	return row.Name, nil
}

func (r *repository) GetCustomerAging(
	ctx context.Context,
	req repositories.GetARCustomerAgingRequest,
) (*repositories.ARCustomerAgingRow, error) {
	records := make([]*agingRowRecord, 0, 1)
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
		  AND inv.customer_id = ?
		  AND inv.status = 'Posted'
		  AND inv.bill_type IN ('Invoice', 'DebitMemo')
		  AND inv.total_amount_minor > inv.applied_amount_minor
		GROUP BY inv.customer_id, inv.bill_to_name
	`, req.AsOfDate, req.AsOfDate, req.AsOfDate, req.AsOfDate, req.AsOfDate, req.AsOfDate, req.AsOfDate, req.AsOfDate, req.AsOfDate, req.TenantInfo.OrgID, req.TenantInfo.BuID, req.CustomerID).Scan(ctx, &records)
	if err != nil {
		return nil, fmt.Errorf("get customer ar aging: %w", err)
	}
	if len(records) == 0 {
		return &repositories.ARCustomerAgingRow{CustomerID: req.CustomerID}, nil
	}

	rec := records[0]
	return &repositories.ARCustomerAgingRow{
		CustomerID:   pulid.ID(rec.CustomerID),
		CustomerName: rec.CustomerName,
		Buckets: repositories.ARAgingBucketTotals{
			CurrentMinor:    rec.CurrentMinor,
			Days1To30Minor:  rec.Days1To30Minor,
			Days31To60Minor: rec.Days31To60Minor,
			Days61To90Minor: rec.Days61To90Minor,
			DaysOver90Minor: rec.DaysOver90Minor,
			TotalOpenMinor:  rec.TotalOpenMinor,
		},
	}, nil
}
