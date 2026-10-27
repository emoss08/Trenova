package order

import (
	"context"
	"regexp"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
)

var (
	_ domaintypes.PostgresSearchable = (*Order)(nil)
	_ pagination.CursorEntity        = (*Order)(nil)
)

// Commercial orders grouping one or more shipments
type Order struct {
	keel.Table `keel:"orders,report=Operations"`

	ID             pulid.ID            `json:"id"             keel:"id,pk,autoid:ord_"`
	BusinessUnitID pulid.ID            `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID            `json:"organizationId" keel:"organization_id,pk,tenant"`
	CustomerID     pulid.ID            `json:"customerId"     keel:"customer_id"`
	OwnerID        pulid.ID            `json:"ownerId"        keel:"owner_id,nullzero"`
	EnteredByID    pulid.ID            `json:"enteredById"    keel:"entered_by_id,nullzero"`
	Status         Status              `json:"status"         keel:"status"`
	OrderNumber    string              `json:"orderNumber"    keel:"order_number"`
	PONumber       string              `json:"poNumber"       keel:"po_number,nullzero"`
	BOL            string              `json:"bol"            keel:"bol,nullzero"`
	CurrencyCode   string              `json:"currencyCode"   keel:"currency_code"`
	QuotedAmount   decimal.NullDecimal `json:"quotedAmount"   keel:"quoted_amount,format=money"`
	BaseAmount     decimal.NullDecimal `json:"baseAmount"     keel:"base_amount,format=money"`
	TotalAmount    decimal.NullDecimal `json:"totalAmount"    keel:"total_amount,format=money"`
	SearchVector   string              `json:"-"              keel:"search_vector,tsvector,readonly,noselect"`
	Rank           string              `json:"-"              keel:"rank,computed"`
	Version        int64               `json:"version"        keel:"version,version"`
	CreatedAt      int64               `json:"createdAt"      keel:"created_at,autocreate"`
	UpdatedAt      int64               `json:"updatedAt"      keel:"updated_at,autoupdate"`

	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty"`
	Organization *tenant.Organization `json:"organization,omitempty"`
	Customer     *customer.Customer   `json:"customer,omitempty"     keel:"belongs:customer_id"`
	Owner        *tenant.User         `json:"owner,omitempty"        keel:"belongs:owner_id,noreport"`
	EnteredBy    *tenant.User         `json:"enteredBy,omitempty"    keel:"belongs:entered_by_id,noreport"`
	Shipments    []*shipment.Shipment `json:"shipments,omitempty"    keel:"many:order_id"`
	Charges      []*OrderCharge       `json:"charges,omitempty"      keel:"many:order_id,noreport"`
}

func (o *Order) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(
		o,
		validation.Field(&o.CustomerID, validation.Required.Error("Customer is required")),
		validation.Field(&o.OrderNumber,
			validation.Required.Error("Order number is required"),
			validation.Length(1, 100).Error("Order number must be between 1 and 100 characters"),
		),
		validation.Field(&o.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[Status]("invalid order status"),
		),
		validation.Field(&o.CurrencyCode,
			validation.Required.Error("Currency code is required"),
			validation.Match(currencyCodePattern).
				Error("Currency code must be a 3-letter ISO 4217 code"),
		),
		validation.Field(&o.PONumber,
			validation.Length(0, 100).Error("PO number must be at most 100 characters"),
		),
		validation.Field(&o.BOL,
			validation.Length(0, 100).Error("BOL must be at most 100 characters"),
		),
		validation.Field(&o.QuotedAmount,
			domainvalidation.DecimalAtLeast(decimal.Zero, "Quoted amount must not be negative"),
		),
		validation.Field(&o.BaseAmount,
			domainvalidation.DecimalAtLeast(decimal.Zero, "Base amount must not be negative"),
		),
	))
}

var currencyCodePattern = regexp.MustCompile(`^[A-Z]{3}$`)

func (o *Order) GetID() pulid.ID {
	return o.ID
}

func (o *Order) GetCreatedAt() int64 {
	return o.CreatedAt
}

func (o *Order) GetTableName() string {
	return "orders"
}

func (o *Order) GetOrganizationID() pulid.ID {
	return o.OrganizationID
}

func (o *Order) GetBusinessUnitID() pulid.ID {
	return o.BusinessUnitID
}

func (o *Order) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias: "ord",
		SearchableFields: []domaintypes.SearchableField{
			{
				Name:   "order_number",
				Weight: domaintypes.SearchWeightA,
				Type:   domaintypes.FieldTypeText,
			},
			{Name: "po_number", Weight: domaintypes.SearchWeightB, Type: domaintypes.FieldTypeText},
			{Name: "bol", Weight: domaintypes.SearchWeightB, Type: domaintypes.FieldTypeText},
			{Name: "status", Weight: domaintypes.SearchWeightA, Type: domaintypes.FieldTypeEnum},
		},
	}
}

func (o *Order) BeforeCreate(ctx context.Context) error {
	if o.Status == "" {
		o.Status = StatusDraft
	}
	if o.CurrencyCode == "" {
		o.CurrencyCode = "USD"
	}
	if !o.TotalAmount.Valid {
		o.TotalAmount = decimal.NewNullDecimal(decimal.Zero)
	}

	return nil
}

func (Order) Constraints() []keel.Constraint {
	return []keel.Constraint{
		{
			Name:    "idx_orders_order_number",
			Message: "Order with this order number already exists in your organization",
		},
		{
			Name:    "fk_orders_customer",
			Message: "Customer does not exist in your organization",
		},
	}
}
