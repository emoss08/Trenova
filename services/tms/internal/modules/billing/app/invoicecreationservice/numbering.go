package invoicecreationservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/app/billingcontrolpolicyservice"
	"github.com/emoss08/trenova/internal/modules/billing/app/billingnumber"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/zap"
)

func (s *Service) numberer() billingnumber.Numberer {
	return billingnumber.New(s.sequenceGenerator, s.sequenceProvider, s.customerRepo)
}

func (s *Service) billingPolicyService() *billingcontrolpolicyservice.Service {
	if s.billingPolicy != nil {
		return s.billingPolicy
	}
	return billingcontrolpolicyservice.New(billingcontrolpolicyservice.Params{Logger: zap.NewNop()})
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
