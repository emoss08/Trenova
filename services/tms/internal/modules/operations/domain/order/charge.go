package order

import (
	"errors"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
)

// OrderCharge is an order-level charge that is not attributable to a single leg —
// e.g. a customs brokerage fee or an order-wide fuel surcharge. Per-leg accessorials
// still live on the shipment's AdditionalCharge; these roll into the order total and
// appear as their own lines on the grouped invoice.
type OrderCharge struct {
	keel.Table `keel:"order_charges"`

	ID             pulid.ID        `json:"id"             keel:"id,pk,autoid:ordchg_"`
	BusinessUnitID pulid.ID        `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID        `json:"organizationId" keel:"organization_id,pk,tenant"`
	OrderID        pulid.ID        `json:"orderId"        keel:"order_id"`
	Description    string          `json:"description"    keel:"description"`
	Amount         decimal.Decimal `json:"amount"         keel:"amount"`
	InvoiceID      pulid.ID        `json:"invoiceId"      keel:"invoice_id,nullzero"`
	InvoicedAt     int64           `json:"invoicedAt"     keel:"invoiced_at,nullzero"`
	Version        int64           `json:"version"        keel:"version,version"`
	CreatedAt      int64           `json:"createdAt"      keel:"created_at,autocreate"`
	UpdatedAt      int64           `json:"updatedAt"      keel:"updated_at,autoupdate"`

	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty"`
	Organization *tenant.Organization `json:"organization,omitempty"`
	Order        *Order               `json:"order,omitempty"        keel:"belongs:order_id"`

	Allocations []*shipment.ChargeAllocation `json:"allocations,omitempty" keel:"many:order_charge_id"`
}

// IsFullyInvoiced reports whether every share of this charge has been billed.
// InvoicedAt is stamped only once the last unbilled share is invoiced, so a
// split charge stays open until each payer's invoice exists.
func (c *OrderCharge) IsFullyInvoiced() bool {
	return c != nil && c.InvoicedAt != 0
}

func (c *OrderCharge) HasAllocations() bool {
	return c != nil && len(c.Allocations) > 0
}

func (c *OrderCharge) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(
		c,
		validation.Field(&c.OrderID, validation.Required.Error("Order is required")),
		validation.Field(&c.Description,
			validation.Required.Error("Description is required"),
			validation.Length(1, 255).Error("Description must be between 1 and 255 characters"),
		),
		validation.Field(&c.Amount, validation.By(func(value any) error {
			amount, ok := value.(decimal.Decimal)
			if !ok || amount.LessThanOrEqual(decimal.Zero) {
				return errors.New("amount must be greater than zero")
			}
			return nil
		})),
	))
}

func (c *OrderCharge) GetID() pulid.ID {
	return c.ID
}

func (c *OrderCharge) GetOrganizationID() pulid.ID {
	return c.OrganizationID
}

func (c *OrderCharge) GetBusinessUnitID() pulid.ID {
	return c.BusinessUnitID
}
