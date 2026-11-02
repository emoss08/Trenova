package invoicecreationservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/app/billingnumber"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingvocab"
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
		BillType:   billingvocab.BillTypeInvoice,
		Profile:    profile,
	})
}
