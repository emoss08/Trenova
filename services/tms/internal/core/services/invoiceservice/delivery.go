package invoiceservice

import (
	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
)

type invoiceDeliveryProfile struct {
	Customer     *customer.Customer
	Email        *customer.CustomerEmailProfile
	Organization *tenant.Organization

	// Shipment is the one shipment this invoice is about, and is nil for an
	// invoice that covers several. Every consumer that reads it — the email
	// context, the Shipper and Consignee blocks, the commodity table — is asking
	// a question that only has an answer for a single shipment.
	Shipment *shipment.Shipment

	// Shipments is the group of shipments the invoice bills, one flat row each.
	// Populated only when there is more than one, which is what a document keys
	// off to print a manifest instead of freight detail.
	Shipments []*repositories.ShipmentSummary

	BillingControl *tenant.BillingControl
}

type resolveDeliveryProfileParams struct {
	Entity                        *invoice.Invoice
	TenantInfo                    pagination.TenantInfo
	IncludeShipmentDetails        bool
	IncludeCustomer               bool
	IncludeCustomerState          bool
	IncludeCustomerBillingProfile bool
	IncludeCustomerEmailProfile   bool
	IncludeBillingControl         bool
}

type invoiceTemplateResult struct {
	Value   string
	Unknown []string
}
