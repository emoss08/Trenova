package order

import (
	"context"
	"errors"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*OrderCharge)(nil)

// OrderCharge is an order-level charge that is not attributable to a single leg —
// e.g. a customs brokerage fee or an order-wide fuel surcharge. Per-leg accessorials
// still live on the shipment's AdditionalCharge; these roll into the order total and
// appear as their own lines on the grouped invoice.
type OrderCharge struct {
	bun.BaseModel `json:"-" bun:"table:order_charges,alias:ordchg"`
	keel.Table    `keel:"order_charges"`

	ID             pulid.ID        `json:"id"             bun:"id,type:VARCHAR(100),pk,notnull" keel:"id,pk,autoid:ordchg_"`
	BusinessUnitID pulid.ID        `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),pk,notnull" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID        `json:"organizationId" bun:"organization_id,type:VARCHAR(100),pk,notnull" keel:"organization_id,pk,tenant"`
	OrderID        pulid.ID        `json:"orderId"        bun:"order_id,type:VARCHAR(100),notnull" keel:"order_id"`
	Description    string          `json:"description"    bun:"description,type:VARCHAR(255),notnull" keel:"description"`
	Amount         decimal.Decimal `json:"amount"         bun:"amount,type:NUMERIC(19,4),notnull,default:0" keel:"amount"`
	InvoiceID      pulid.ID        `json:"invoiceId"      bun:"invoice_id,type:VARCHAR(100),nullzero" keel:"invoice_id,nullzero"`
	InvoicedAt     int64           `json:"invoicedAt"     bun:"invoiced_at,type:BIGINT,nullzero" keel:"invoiced_at,nullzero"`
	Version        int64           `json:"version"        bun:"version,type:BIGINT" keel:"version,version"`
	CreatedAt      int64           `json:"createdAt"      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
	UpdatedAt      int64           `json:"updatedAt"      bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate"`

	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *tenant.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
	Order        *Order               `json:"order,omitempty"        bun:"rel:belongs-to,join:order_id=id" keel:"belongs:order_id"`

	Allocations []*shipment.ChargeAllocation `json:"allocations,omitempty" bun:"rel:has-many,join:id=order_charge_id" keel:"many:order_charge_id"`
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

func (c *OrderCharge) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if c.ID.IsNil() {
			c.ID = pulid.MustNew("ordchg_")
		}
		c.CreatedAt = now
	case *bun.UpdateQuery:
		c.UpdatedAt = now
	}

	return nil
}
