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
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook      = (*Order)(nil)
	_ domaintypes.PostgresSearchable = (*Order)(nil)
	_ pagination.CursorEntity        = (*Order)(nil)
)

type Order struct {
	bun.BaseModel             `json:"-" bun:"table:orders,alias:ord"`
	keel.Table                `keel:"orders"`
	pagination.CursorValueSet `json:"-" bun:",embed"`

	ID             pulid.ID            `json:"id"             bun:"id,type:VARCHAR(100),pk,notnull" keel:"id,pk,autoid:ord_"`
	BusinessUnitID pulid.ID            `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),pk,notnull" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID            `json:"organizationId" bun:"organization_id,type:VARCHAR(100),pk,notnull" keel:"organization_id,pk,tenant"`
	CustomerID     pulid.ID            `json:"customerId"     bun:"customer_id,type:VARCHAR(100),notnull" keel:"customer_id"`
	OwnerID        pulid.ID            `json:"ownerId"        bun:"owner_id,type:VARCHAR(100),nullzero" keel:"owner_id,nullzero"`
	EnteredByID    pulid.ID            `json:"enteredById"    bun:"entered_by_id,type:VARCHAR(100),nullzero" keel:"entered_by_id,nullzero"`
	Status         Status              `json:"status"         bun:"status,type:order_status_enum,notnull,default:'Draft'" keel:"status"`
	OrderNumber    string              `json:"orderNumber"    bun:"order_number,type:VARCHAR(100),notnull" keel:"order_number"`
	PONumber       string              `json:"poNumber"       bun:"po_number,type:VARCHAR(100),nullzero" keel:"po_number,nullzero"`
	BOL            string              `json:"bol"            bun:"bol,type:VARCHAR(100),nullzero" keel:"bol,nullzero"`
	CurrencyCode   string              `json:"currencyCode"   bun:"currency_code,type:VARCHAR(3),notnull,default:'USD'" keel:"currency_code"`
	QuotedAmount   decimal.NullDecimal `json:"quotedAmount"   bun:"quoted_amount,type:NUMERIC(19,4),nullzero" keel:"quoted_amount"`
	BaseAmount     decimal.NullDecimal `json:"baseAmount"     bun:"base_amount,type:NUMERIC(19,4),nullzero" keel:"base_amount"`
	TotalAmount    decimal.NullDecimal `json:"totalAmount"    bun:"total_amount,type:NUMERIC(19,4),notnull,default:0" keel:"total_amount"`
	SearchVector   string              `json:"-"              bun:"search_vector,type:TSVECTOR,scanonly" keel:"search_vector,tsvector,readonly,noselect"`
	Rank           string              `json:"-"              bun:"rank,type:VARCHAR(100),scanonly" keel:"rank,computed"`
	Version        int64               `json:"version"        bun:"version,type:BIGINT" keel:"version,version"`
	CreatedAt      int64               `json:"createdAt"      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
	UpdatedAt      int64               `json:"updatedAt"      bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate"`

	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *tenant.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
	Customer     *customer.Customer   `json:"customer,omitempty"     bun:"rel:belongs-to,join:customer_id=id" keel:"belongs:customer_id"`
	Owner        *tenant.User         `json:"owner,omitempty"        bun:"rel:belongs-to,join:owner_id=id" keel:"belongs:owner_id"`
	EnteredBy    *tenant.User         `json:"enteredBy,omitempty"    bun:"rel:belongs-to,join:entered_by_id=id" keel:"belongs:entered_by_id"`
	Shipments    []*shipment.Shipment `json:"shipments,omitempty"    bun:"rel:has-many,join:id=order_id" keel:"many:order_id"`
	Charges      []*OrderCharge       `json:"charges,omitempty"      bun:"rel:has-many,join:id=order_id" keel:"many:order_id"`
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

func (o *Order) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if o.ID.IsNil() {
			o.ID = pulid.MustNew("ord_")
		}
		o.CreatedAt = now
	case *bun.UpdateQuery:
		o.UpdatedAt = now
	}

	return nil
}

func (o *Order) BeforeCreate(ctx context.Context) error {
	return o.BeforeAppendModel(ctx, (*bun.InsertQuery)(nil))
}
