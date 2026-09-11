package invoiceservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/seqgen"
)

// generateInvoiceNumber issues the one number an invoice carries, honouring the
// customer's own numbering preference.
//
// Every call site used to pass empty location and business-unit codes and take
// the tenant default, which meant CustomerInvoicePrefix and InvoiceNumberFormat
// were configuration that did nothing. A consolidated invoice makes that visible:
// it is one number standing for a month of freight, and a customer who asked for
// their own prefix expects to see it.
//
// The format is copied before being handed to the generator because resolveFormat
// writes the location and business-unit codes into whatever pointer it is given —
// mutating the provider's value would leak one tenant's codes into another's.
func (s *Service) generateInvoiceNumber(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	profile *customer.CustomerBillingProfile,
) (string, error) {
	req := &seqgen.GenerateRequest{
		Type:  tenant.SequenceTypeInvoice,
		OrgID: tenantInfo.OrgID,
		BuID:  tenantInfo.BuID,
	}

	if format := customerInvoiceFormat(ctx, s, tenantInfo, profile); format != nil {
		req.Format = format
	}

	return s.sequenceGenerator.Generate(ctx, req)
}

// customerInvoiceFormat is the tenant format with the customer's prefix applied,
// or nil when the customer has no preference of their own.
func customerInvoiceFormat(
	ctx context.Context,
	s *Service,
	tenantInfo pagination.TenantInfo,
	profile *customer.CustomerBillingProfile,
) *tenant.SequenceFormat {
	if profile == nil ||
		profile.InvoiceNumberFormat != customer.InvoiceNumberFormatCustomPrefix ||
		profile.CustomerInvoicePrefix == "" {
		return nil
	}

	base, err := s.sequenceProvider.GetFormat(
		ctx,
		tenant.SequenceTypeInvoice,
		tenantInfo.OrgID,
		tenantInfo.BuID,
	)
	if err != nil || base == nil {
		// A missing tenant format is not worth failing an invoice over; the
		// generator's own default still produces a valid, unique number.
		return nil
	}

	//nolint:govet // a shallow copy is the point: the generator mutates what it is given.
	copied := *base
	copied.Prefix = profile.CustomerInvoicePrefix

	return &copied
}

// billingProfileOf is the customer's billing profile, or nil when it was not
// loaded with them.
func billingProfileOf(cus *customer.Customer) *customer.CustomerBillingProfile {
	if cus == nil {
		return nil
	}

	return cus.BillingProfile
}
