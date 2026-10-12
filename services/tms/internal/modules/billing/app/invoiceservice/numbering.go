package invoiceservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/app/billingnumber"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/pkg/pagination"
)

func (s *Service) numberer() billingnumber.Numberer {
	return billingnumber.New(s.sequenceGenerator, s.sequenceProvider, s.customerRepo)
}

// generateInvoiceNumber issues the one number an invoice carries, honouring the
// customer's own numbering preference.
func (s *Service) generateInvoiceNumber(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	profile *customer.CustomerBillingProfile,
) (string, error) {
	return s.numberer().Next(ctx, &billingnumber.Request{
		TenantInfo: tenantInfo,
		BillType:   billingqueue.BillTypeInvoice,
		Profile:    profile,
	})
}

// billingProfileOf is the customer's billing profile, or nil when it was not
// loaded with them.
func billingProfileOf(cus *customer.Customer) *customer.CustomerBillingProfile {
	if cus == nil {
		return nil
	}

	return cus.BillingProfile
}
