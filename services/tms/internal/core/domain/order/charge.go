package order

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
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

	ID             pulid.ID        `json:"id"             bun:"id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID pulid.ID        `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID pulid.ID        `json:"organizationId" bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	OrderID        pulid.ID        `json:"orderId"        bun:"order_id,type:VARCHAR(100),notnull"`
	Description    string          `json:"description"    bun:"description,type:VARCHAR(255),notnull"`
	Amount         decimal.Decimal `json:"amount"         bun:"amount,type:NUMERIC(19,4),notnull,default:0"`
	InvoiceID      pulid.ID        `json:"invoiceId"      bun:"invoice_id,type:VARCHAR(100),nullzero"`
	InvoicedAt     int64           `json:"invoicedAt"     bun:"invoiced_at,type:BIGINT,nullzero"`
	Version        int64           `json:"version"        bun:"version,type:BIGINT"`
	CreatedAt      int64           `json:"createdAt"      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64           `json:"updatedAt"      bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *tenant.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
	Order        *Order               `json:"order,omitempty"        bun:"rel:belongs-to,join:order_id=id"`
}

func (c *OrderCharge) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(
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
	)
	if err != nil {
		if validationErrs, ok := errors.AsType[validation.Errors](err); ok {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
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
