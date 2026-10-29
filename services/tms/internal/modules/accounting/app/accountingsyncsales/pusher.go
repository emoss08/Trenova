package accountingsyncsales

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingsyncpush"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerpaymentports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceadjustmentports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

type Deps struct {
	Kit         *accountingsyncpush.Kit
	Invoices    invoiceports.InvoiceRepository
	Adjustments invoiceadjustmentports.InvoiceAdjustmentRepository
	Payments    customerpaymentports.CustomerPaymentRepository
}

type Pusher struct {
	kit         *accountingsyncpush.Kit
	invoices    invoiceports.InvoiceRepository
	adjustments invoiceadjustmentports.InvoiceAdjustmentRepository
	payments    customerpaymentports.CustomerPaymentRepository
}

func New(d Deps) *Pusher {
	return &Pusher{
		kit:         d.Kit,
		invoices:    d.Invoices,
		adjustments: d.Adjustments,
		payments:    d.Payments,
	}
}

func (p *Pusher) stampInvoice(
	sess *accountingsyncpush.Session,
	id pulid.ID,
) func(context.Context, decimal.Decimal, int64) error {
	return func(ctx context.Context, rate decimal.Decimal, date int64) error {
		return p.invoices.StampExchangeRate(ctx, &kernelports.StampExchangeRateRequest{
			TenantInfo: sess.Tenant,
			ID:         id,
			Rate:       rate,
			Date:       date,
		})
	}
}

func (p *Pusher) stampPayment(
	sess *accountingsyncpush.Session,
	id pulid.ID,
) func(context.Context, decimal.Decimal, int64) error {
	return func(ctx context.Context, rate decimal.Decimal, date int64) error {
		return p.payments.StampExchangeRate(ctx, &kernelports.StampExchangeRateRequest{
			TenantInfo: sess.Tenant,
			ID:         id,
			Rate:       rate,
			Date:       date,
		})
	}
}
