package invoicepostingservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/app/invoicedocument"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueuereview"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/pkg/pagination"
)

// EmailPlan is whether posting emails the invoice to the customer without
// anyone sending it, and to whom. An agent that posted an invoice could not
// tell whether it had gone out, so it offered to send one that may already
// have been sent.
type EmailPlan struct {
	// Allowed is the customer taking invoices by email at all.
	Allowed bool
	// AutoSend is posting emailing it once its PDF is made.
	AutoSend   bool
	Recipients []string
}

func (s *Service) emailPlanFor(
	ctx context.Context,
	posted *invoice.Invoice,
	tenant pagination.TenantInfo,
) (*EmailPlan, error) {
	cus, err := s.customerRepo.GetByID(ctx, customerports.GetCustomerByIDRequest{
		ID:         posted.CustomerID,
		TenantInfo: tenant,
		CustomerFilterOptions: customerports.CustomerFilterOptions{
			IncludeBillingProfile: true,
			IncludeEmailProfile:   true,
		},
	})
	if err != nil {
		return nil, err
	}

	plan := &EmailPlan{
		AutoSend:   invoicedocument.AutoSendsOnGeneration(cus, posted),
		Recipients: []string{},
	}
	if cus.BillingProfile != nil {
		plan.Allowed = cus.BillingProfile.EmailInvoiceEnabled
	}
	if cus.EmailProfile != nil {
		plan.Recipients = billingqueuereview.ParseRecipients(cus.EmailProfile.ToRecipients)
	}

	return plan, nil
}
