// Package billingnumber issues the number a bill carries, honouring the
// payer's own invoice numbering preference.
//
// A billing queue item is numbered when freight is transferred to billing, and
// the invoice later built from it takes that number. Numbering the item with
// the tenant default therefore decided the invoice's number too, so a
// customer's CustomPrefix only reached invoices created outside the queue. Every
// path that numbers a bill (transfer, reassignment, a void releasing freight to
// be billed again, an adjustment's replacement) goes through here instead.
package billingnumber

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/seqgen"
	"github.com/emoss08/trenova/shared/pulid"
)

type Numberer struct {
	generator seqgen.Generator
	formats   seqgen.FormatProvider
	customers customerports.CustomerRepository
}

// New builds a Numberer. Formats and customers may be nil: without them every
// bill takes the tenant's default format.
func New(
	generator seqgen.Generator,
	formats seqgen.FormatProvider,
	customers customerports.CustomerRepository,
) Numberer {
	return Numberer{generator: generator, formats: formats, customers: customers}
}

type Request struct {
	TenantInfo pagination.TenantInfo
	BillType   billingqueue.BillType
	// Profile is the payer's billing profile when the caller already holds it;
	// otherwise PayerID is looked up.
	Profile *customer.CustomerBillingProfile
	PayerID pulid.ID
}

// Next issues the next number for a bill of the requested type.
func (n Numberer) Next(ctx context.Context, req *Request) (string, error) {
	orgID, buID := req.TenantInfo.OrgID, req.TenantInfo.BuID
	switch req.BillType {
	case billingqueue.BillTypeCreditMemo:
		return n.generator.GenerateCreditMemoNumber(ctx, orgID, buID, "", "")
	case billingqueue.BillTypeDebitMemo:
		return n.generator.GenerateDebitMemoNumber(ctx, orgID, buID, "", "")
	case billingqueue.BillTypeInvoice:
		return n.invoiceNumber(ctx, req)
	default:
		return "", errortypes.NewValidationError(
			"billType",
			errortypes.ErrInvalid,
			"Unsupported bill type for number generation",
		)
	}
}

// ForItem renumbers an item being released to be billed again, for its payer.
func (n Numberer) ForItem(
	ctx context.Context,
	item *billingqueue.BillingQueueItem,
) (string, error) {
	return n.Next(ctx, &Request{
		TenantInfo: pagination.TenantInfo{OrgID: item.OrganizationID, BuID: item.BusinessUnitID},
		BillType:   item.BillType,
		PayerID:    item.BillToCustomerID,
	})
}

func (n Numberer) invoiceNumber(ctx context.Context, req *Request) (string, error) {
	profile, err := n.profile(ctx, req)
	if err != nil {
		return "", err
	}

	format := n.customerFormat(ctx, req.TenantInfo, profile)
	if format == nil {
		return n.generator.GenerateInvoiceNumber(
			ctx,
			req.TenantInfo.OrgID,
			req.TenantInfo.BuID,
			"",
			"",
		)
	}

	return n.generator.Generate(ctx, &seqgen.GenerateRequest{
		Type:   tenant.SequenceTypeInvoice,
		OrgID:  req.TenantInfo.OrgID,
		BuID:   req.TenantInfo.BuID,
		Format: format,
	})
}

func (n Numberer) profile(
	ctx context.Context,
	req *Request,
) (*customer.CustomerBillingProfile, error) {
	if req.Profile != nil || req.PayerID.IsNil() || n.customers == nil || n.formats == nil {
		return req.Profile, nil
	}

	payer, err := n.customers.GetByID(ctx, customerports.GetCustomerByIDRequest{
		ID:         req.PayerID,
		TenantInfo: req.TenantInfo,
		CustomerFilterOptions: customerports.CustomerFilterOptions{
			IncludeBillingProfile: true,
		},
	})
	if errortypes.IsNotFoundError(err) {
		return nil, nil //nolint:nilnil // a missing customer has no billing profile prefix
	}
	if err != nil {
		return nil, err
	}

	return payer.BillingProfile, nil
}

// customerFormat is the tenant format with the customer's prefix applied, or
// nil when the customer has no preference of their own.
//
// The format is copied before being handed to the generator because the
// generator writes the location and business-unit codes into whatever pointer
// it is given; mutating the provider's value would leak one tenant's codes into
// another's.
func (n Numberer) customerFormat(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	profile *customer.CustomerBillingProfile,
) *tenant.SequenceFormat {
	if n.formats == nil || profile == nil ||
		profile.InvoiceNumberFormat != customer.InvoiceNumberFormatCustomPrefix ||
		profile.CustomerInvoicePrefix == "" {
		return nil
	}

	base, err := n.formats.GetFormat(
		ctx,
		tenant.SequenceTypeInvoice,
		tenantInfo.OrgID,
		tenantInfo.BuID,
	)
	if err != nil || base == nil {
		// A missing tenant format is not worth failing a bill over; the
		// generator's own default still produces a valid, unique number.
		return nil
	}

	copied := *base
	copied.Prefix = profile.CustomerInvoicePrefix

	return &copied
}
