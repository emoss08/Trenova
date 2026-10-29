package invoiceassembly

import (
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/billingcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

func basicShipmentByIDRequest(
	id pulid.ID,
	tenantInfo pagination.TenantInfo,
) *shipmentports.GetShipmentByIDRequest {
	return ShipmentByIDRequest(id, tenantInfo, false)
}

func ExpandedShipmentByIDRequest(
	id pulid.ID,
	tenantInfo pagination.TenantInfo,
) *shipmentports.GetShipmentByIDRequest {
	return ShipmentByIDRequest(id, tenantInfo, true)
}

func ShipmentByIDRequest(
	id pulid.ID,
	tenantInfo pagination.TenantInfo,
	expandShipmentDetails bool,
) *shipmentports.GetShipmentByIDRequest {
	return &shipmentports.GetShipmentByIDRequest{
		ID:         id,
		TenantInfo: tenantInfo,
		ShipmentOptions: shipmentports.ShipmentOptions{
			ExpandShipmentDetails: expandShipmentDetails,
		},
	}
}

// billingProfileOf is the customer's billing profile, or nil when it was not
// loaded with them.
func BillingProfileOf(cus *customer.Customer) *customer.CustomerBillingProfile {
	if cus == nil {
		return nil
	}

	return cus.BillingProfile
}

const UnnumberedInvoice = "Unnumbered"

// applyInvoiceDetail copies the customer's presentation preference onto the
// invoice, so how it reads is fixed at the moment it was billed.
func ApplyInvoiceDetail(entity *invoice.Invoice, cus *customer.Customer) {
	entity.Detail = customer.InvoiceDetailDetailed
	entity.SectionBy = customer.InvoiceSectionKeyShipment

	profile := BillingProfileOf(cus)
	if profile == nil {
		return
	}
	if profile.InvoiceDetail.IsValid() {
		entity.Detail = profile.InvoiceDetail
	}
	if profile.SectionBy.IsValid() {
		entity.SectionBy = profile.SectionBy
	}
}

func SyncInvoiceTotalsFromLines(entity *invoice.Invoice) {
	entity.SubtotalAmount = SumLinesByType(entity.Lines, invoice.InvoiceLineTypeFreight)
	entity.OtherAmount = SumLinesByType(entity.Lines, invoice.InvoiceLineTypeAccessorial)
	entity.TotalAmount = SumLinesByType(entity.Lines, "")
}

func SumLinesByType(
	lines []*invoice.InvoiceLine,
	lineType invoice.InvoiceLineType,
) decimal.Decimal {
	total := decimal.Zero
	for _, line := range lines {
		if line == nil {
			continue
		}
		if lineType == "" || line.Type == lineType {
			total = total.Add(line.Amount)
		}
	}
	return total
}

func paymentTermFromCustomer(cus *customer.Customer) invoice.PaymentTerm {
	if cus == nil || cus.BillingProfile == nil {
		return ""
	}

	return invoice.PaymentTerm(cus.BillingProfile.PaymentTerm)
}

func paymentTermFromBillingControl(control *billingcontrol.BillingControl) invoice.PaymentTerm {
	if control == nil {
		return ""
	}

	return invoice.PaymentTerm(control.DefaultPaymentTerm)
}

func ResolvePaymentTerm(
	cus *customer.Customer,
	control *billingcontrol.BillingControl,
) invoice.PaymentTerm {
	if term := paymentTermFromCustomer(cus); term != "" {
		return term
	}

	return paymentTermFromBillingControl(control)
}

func BillingCurrencyFromCustomer(cus *customer.Customer) string {
	return cus.BillingCurrencyCode()
}
