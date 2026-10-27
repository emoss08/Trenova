package paymentreminderrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/factoringsql"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/receivablesql"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice/invoicedb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/paymentreminder"
	"github.com/emoss08/trenova/internal/modules/billing/domain/paymentreminder/paymentreminderdb"
	"github.com/emoss08/trenova/internal/modules/billing/ports/paymentreminderports"
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

func New(p Params) paymentreminderports.Repository {
	return &repository{db: p.DB, l: p.Logger.Named("postgres.payment-reminder-repository")}
}

type overdueRecord struct {
	InvoiceID     pulid.ID
	InvoiceNumber string
	CustomerID    pulid.ID
	CurrencyCode  string
	InvoiceDate   int64
	DueDate       int64
	OpenMinor     int64
	Disputed      bool
	LastStep      int
}

//nolint:gocritic // Repository request structs follow the existing value-parameter port contracts.
func (r *repository) ListOverdueInvoices(
	ctx context.Context,
	req paymentreminderports.ListOverdueInvoicesRequest,
) ([]*paymentreminderports.OverdueInvoice, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*paymentreminderports.OverdueInvoice, error) {
			inv := invoicedb.Invoices
			ipr := paymentreminderdb.Reminders.As("ipr")
			lastStep := keel.Scalar(ipr.Pick(ipr.Step.Max()).Where(ipr.InvoiceID.EqCol(inv.ID)))

			records, err := keel.Into[overdueRecord](ctx, r.db.Keel(ctx), inv.
				Pick(
					inv.ID,
					inv.Number,
					inv.CustomerID,
					inv.CurrencyCode,
					inv.InvoiceDate,
					inv.DueDate,
					receivablesql.OpenMinor(inv),
					inv.DisputeStatus.Eq(invoice.DisputeStatusDisputed),
					keel.Int32Of(lastStep).Coalesce(0),
				).
				Where(
					receivablesql.Open(inv),
					inv.DueDate.IsNotNull(),
					inv.DueDate.Lt(req.AsOf),
					keel.Or(inv.FactoringCompanyID.IsNull(), factoringsql.ChargedBackInvoicePred(inv)),
					keel.When(len(req.CustomerIDs) > 0, inv.CustomerID.In(req.CustomerIDs...)),
				).
				Order(inv.CustomerID.Asc(), inv.DueDate.Asc(), inv.Number.Asc()))
			if err != nil {
				return nil, fmt.Errorf("list overdue invoices: %w", err)
			}

			invoices := make([]*paymentreminderports.OverdueInvoice, 0, len(records))
			for i := range records {
				rec := &records[i]
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
		},
	)
}

func (r *repository) InsertReminders(
	ctx context.Context,
	reminders []*paymentreminder.Reminder,
) error {
	if len(reminders) == 0 {
		return nil
	}
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if _, err := paymentreminderdb.Reminders.CreateOrSkip(
			ctx,
			r.db.Keel(ctx),
			reminders,
		); err != nil {
			return fmt.Errorf("insert payment reminders: %w", err)
		}
		return nil
	})
}
