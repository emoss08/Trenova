package paymentreminderrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/billing/domain/paymentreminder"
	"github.com/emoss08/trenova/internal/modules/billing/ports/paymentreminderports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
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

func New(p Params) paymentreminderports.Repository {
	return &repository{db: p.DB, l: p.Logger.Named("postgres.payment-reminder-repository")}
}

const overdueInvoicesSQL = `
	SELECT
		inv.id AS invoice_id,
		inv.number AS invoice_number,
		inv.customer_id,
		inv.currency_code,
		inv.invoice_date,
		inv.due_date,
		inv.total_amount_minor - inv.applied_amount_minor AS open_minor,
		inv.dispute_status = 'Disputed' AS disputed,
		COALESCE((
			SELECT MAX(ipr.step)
			FROM invoice_payment_reminders AS ipr
			WHERE ipr.organization_id = inv.organization_id
				AND ipr.business_unit_id = inv.business_unit_id
				AND ipr.invoice_id = inv.id
		), 0) AS last_step
	FROM invoices AS inv
	WHERE inv.organization_id = ?
		AND inv.business_unit_id = ?
		AND inv.status = 'Posted'
		AND inv.bill_type IN ('Invoice', 'DebitMemo')
		AND inv.total_amount_minor > inv.applied_amount_minor
		AND inv.due_date IS NOT NULL
		AND inv.due_date < ?
		%s
	ORDER BY inv.customer_id, inv.due_date, inv.number`

type overdueRecord struct {
	InvoiceID     pulid.ID `bun:"invoice_id"`
	InvoiceNumber string   `bun:"invoice_number"`
	CustomerID    pulid.ID `bun:"customer_id"`
	CurrencyCode  string   `bun:"currency_code"`
	InvoiceDate   int64    `bun:"invoice_date"`
	DueDate       int64    `bun:"due_date"`
	OpenMinor     int64    `bun:"open_minor"`
	Disputed      bool     `bun:"disputed"`
	LastStep      int      `bun:"last_step"`
}

// ListOverdueInvoices reads every posted invoice past due with money open,
// oldest first within each customer, with the scheduled step it last drew.
func (r *repository) ListOverdueInvoices(
	ctx context.Context,
	req paymentreminderports.ListOverdueInvoicesRequest,
) ([]*paymentreminderports.OverdueInvoice, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*paymentreminderports.OverdueInvoice, error) {
		args := []any{req.TenantInfo.OrgID, req.TenantInfo.BuID, req.AsOf}
		filter := ""
		if len(req.CustomerIDs) > 0 {
			filter = "AND inv.customer_id IN (?)"
			args = append(args, bun.In(req.CustomerIDs))
		}

		records := make([]*overdueRecord, 0, 64)
		if err := r.db.DBForContext(ctx).
			NewRaw(fmt.Sprintf(overdueInvoicesSQL, filter), args...).
			Scan(ctx, &records); err != nil {
			return nil, fmt.Errorf("list overdue invoices: %w", err)
		}

		invoices := make([]*paymentreminderports.OverdueInvoice, 0, len(records))
		for _, rec := range records {
			invoices = append(invoices, &paymentreminderports.OverdueInvoice{
				InvoiceID:     rec.InvoiceID,
				InvoiceNumber: rec.InvoiceNumber,
				CustomerID:    rec.CustomerID,
				CurrencyCode:  rec.CurrencyCode,
				InvoiceDate:   rec.InvoiceDate,
				DueDate:       rec.DueDate,
				OpenMinor:     rec.OpenMinor,
				LastStep:      rec.LastStep,
				Disputed:      rec.Disputed,
			})
		}
		return invoices, nil
	})
}

// InsertReminders records the invoices a reminder listed. A scheduled step
// already recorded for an invoice is left as it was, so a retried send never
// records a step twice.
func (r *repository) InsertReminders(
	ctx context.Context,
	reminders []*paymentreminder.Reminder,
) error {
	if len(reminders) == 0 {
		return nil
	}
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if _, err := r.db.DBForContext(ctx).NewInsert().
			Model(&reminders).
			On("CONFLICT (organization_id, business_unit_id, invoice_id, step) WHERE step > 0 DO NOTHING").
			Exec(ctx); err != nil {
			return fmt.Errorf("insert payment reminders: %w", err)
		}
		return nil
	})
}
