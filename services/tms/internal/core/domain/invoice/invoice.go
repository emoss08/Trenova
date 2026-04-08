package invoice

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*Invoice)(nil)
	_ validationframework.TenantedEntity = (*Invoice)(nil)
	_ domaintypes.PostgresSearchable     = (*Invoice)(nil)
	_ bun.BeforeAppendModelHook          = (*Line)(nil)
)

type Invoice struct {
	bun.BaseModel `bun:"table:invoices,alias:inv" json:"-"`

	ID                 pulid.ID              `json:"id"                 bun:"id,pk,type:VARCHAR(100),notnull"`
	OrganizationID     pulid.ID              `json:"organizationId"     bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID     pulid.ID              `json:"businessUnitId"     bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	BillingQueueItemID pulid.ID              `json:"billingQueueItemId" bun:"billing_queue_item_id,type:VARCHAR(100),notnull"`
	ShipmentID         pulid.ID              `json:"shipmentId"         bun:"shipment_id,type:VARCHAR(100),notnull"`
	CustomerID         pulid.ID              `json:"customerId"         bun:"customer_id,type:VARCHAR(100),notnull"`
	Number             string                `json:"number"             bun:"number,type:VARCHAR(100),notnull"`
	BillType           billingqueue.BillType `json:"billType"           bun:"bill_type,type:VARCHAR(50),notnull"`
	Status             Status                `json:"status"             bun:"status,type:VARCHAR(50),notnull,default:'Draft'"`
	PaymentTerm        PaymentTerm           `json:"paymentTerm"        bun:"payment_term,type:VARCHAR(50),notnull"`
	CurrencyCode       string                `json:"currencyCode"       bun:"currency_code,type:VARCHAR(3),notnull,default:'USD'"`
	InvoiceDate        int64                 `json:"invoiceDate"        bun:"invoice_date,type:BIGINT,notnull"`
	DueDate            *int64                `json:"dueDate"            bun:"due_date,type:BIGINT,nullzero"`
	PostedAt           *int64                `json:"postedAt"           bun:"posted_at,type:BIGINT,nullzero"`
	ShipmentProNumber  string                `json:"shipmentProNumber"  bun:"shipment_pro_number,type:VARCHAR(100),nullzero"`
	ShipmentBOL        string                `json:"shipmentBol"        bun:"shipment_bol,type:VARCHAR(100),nullzero"`
	ServiceDate        *int64                `json:"serviceDate"        bun:"service_date,type:BIGINT,nullzero"`
	BillToName         string                `json:"billToName"         bun:"bill_to_name,type:VARCHAR(255),notnull"`
	BillToCode         string                `json:"billToCode"         bun:"bill_to_code,type:VARCHAR(50),nullzero"`
	BillToAddressLine1 string                `json:"billToAddressLine1" bun:"bill_to_address_line_1,type:VARCHAR(255),nullzero"`
	BillToAddressLine2 string                `json:"billToAddressLine2" bun:"bill_to_address_line_2,type:VARCHAR(255),nullzero"`
	BillToCity         string                `json:"billToCity"         bun:"bill_to_city,type:VARCHAR(100),nullzero"`
	BillToState        string                `json:"billToState"        bun:"bill_to_state,type:VARCHAR(100),nullzero"`
	BillToPostalCode   string                `json:"billToPostalCode"   bun:"bill_to_postal_code,type:VARCHAR(20),nullzero"`
	BillToCountry      string                `json:"billToCountry"      bun:"bill_to_country,type:VARCHAR(100),nullzero"`
	SubtotalAmount     decimal.Decimal       `json:"subtotalAmount"     bun:"subtotal_amount,type:NUMERIC(19,4),notnull,default:0"`
	OtherAmount        decimal.Decimal       `json:"otherAmount"        bun:"other_amount,type:NUMERIC(19,4),notnull,default:0"`
	TotalAmount        decimal.Decimal       `json:"totalAmount"        bun:"total_amount,type:NUMERIC(19,4),notnull,default:0"`
	Version            int64                 `json:"version"            bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt          int64                 `json:"createdAt"          bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt          int64                 `json:"updatedAt"          bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	BillingQueueItem *billingqueue.BillingQueueItem `json:"billingQueueItem,omitempty" bun:"rel:belongs-to,join:billing_queue_item_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	Shipment         *shipment.Shipment             `json:"shipment,omitempty"         bun:"rel:belongs-to,join:shipment_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	Customer         *customer.Customer             `json:"customer,omitempty"         bun:"rel:belongs-to,join:customer_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	Lines            []*Line                        `json:"lines,omitempty"            bun:"rel:has-many,join:id=invoice_id"`
}

type Line struct {
	bun.BaseModel `bun:"table:invoice_lines,alias:invl" json:"-"`

	ID             pulid.ID        `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID        `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID        `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	InvoiceID      pulid.ID        `json:"invoiceId"      bun:"invoice_id,type:VARCHAR(100),notnull"`
	LineNumber     int             `json:"lineNumber"     bun:"line_number,type:INTEGER,notnull"`
	Type           LineType        `json:"type"           bun:"type,type:VARCHAR(50),notnull"`
	Description    string          `json:"description"    bun:"description,type:TEXT,notnull"`
	Quantity       decimal.Decimal `json:"quantity"       bun:"quantity,type:NUMERIC(19,4),notnull,default:0"`
	UnitPrice      decimal.Decimal `json:"unitPrice"      bun:"unit_price,type:NUMERIC(19,4),notnull,default:0"`
	Amount         decimal.Decimal `json:"amount"         bun:"amount,type:NUMERIC(19,4),notnull,default:0"`
	Version        int64           `json:"version"        bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt      int64           `json:"createdAt"      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64           `json:"updatedAt"      bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Invoice *Invoice `json:"-" bun:"rel:belongs-to,join:invoice_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
}

func (i *Invoice) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(
		i,
		validation.Field(
			&i.OrganizationID,
			validation.Required.Error("Organization ID is required"),
		),
		validation.Field(
			&i.BusinessUnitID,
			validation.Required.Error("Business unit ID is required"),
		),
		validation.Field(
			&i.BillingQueueItemID,
			validation.Required.Error("Billing queue item ID is required"),
		),
		validation.Field(&i.ShipmentID, validation.Required.Error("Shipment ID is required")),
		validation.Field(&i.CustomerID, validation.Required.Error("Customer ID is required")),
		validation.Field(&i.Number,
			validation.Required.Error("Invoice number is required"),
			validation.Length(1, 100).Error("Invoice number must be between 1 and 100 characters"),
		),
		validation.Field(&i.BillType,
			validation.Required.Error("Bill type is required"),
			validation.In(
				billingqueue.BillTypeInvoice,
				billingqueue.BillTypeCreditMemo,
				billingqueue.BillTypeDebitMemo,
			).Error("Invalid bill type"),
		),
		validation.Field(&i.Status,
			validation.Required.Error("Invoice status is required"),
			validation.By(func(value any) error {
				status, _ := value.(Status)
				if !status.IsValid() {
					return errors.New("invalid invoice status")
				}
				return nil
			}),
		),
		validation.Field(&i.PaymentTerm,
			validation.Required.Error("Payment term is required"),
			validation.By(func(value any) error {
				term, _ := value.(PaymentTerm)
				if !term.IsValid() {
					return errors.New("invalid payment term")
				}
				return nil
			}),
		),
		validation.Field(&i.CurrencyCode,
			validation.Required.Error("Currency code is required"),
			validation.Length(3, 3).Error("Currency code must be a 3-character ISO code"),
		),
		validation.Field(&i.InvoiceDate, validation.Required.Error("Invoice date is required")),
		validation.Field(&i.BillToName,
			validation.Required.Error("Bill-to name is required"),
			validation.Length(1, 255).Error("Bill-to name must be between 1 and 255 characters"),
		),
	)
	if err != nil {
		if validationErrs, ok := errors.AsType[validation.Errors](err); ok {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}

	if i.BillType == billingqueue.BillTypeCreditMemo {
		if i.TotalAmount.GreaterThan(decimal.Zero) {
			multiErr.Add(
				"totalAmount",
				errortypes.ErrInvalid,
				"Credit memo total must be zero or negative",
			)
		}
	} else if i.TotalAmount.LessThan(decimal.Zero) {
		multiErr.Add("totalAmount", errortypes.ErrInvalid, "Invoice total must be zero or positive")
	}

	if len(i.Lines) == 0 {
		multiErr.Add("lines", errortypes.ErrRequired, "At least one invoice line is required")
	}

	for idx, line := range i.Lines {
		if line == nil {
			multiErr.Add(
				"lines",
				errortypes.ErrInvalid,
				"Invoice lines must not contain null values",
			)
			continue
		}

		line.Validate(multiErr, idx)
	}
}

func (l *Line) Validate(multiErr *errortypes.MultiError, idx int) {
	prefix := "lines[" + strconv.Itoa(idx) + "]"

	if l.Type == "" || !l.Type.IsValid() {
		multiErr.Add(prefix+".type", errortypes.ErrInvalid, "Invalid invoice line type")
	}
	if strings.TrimSpace(l.Description) == "" {
		multiErr.Add(
			prefix+".description",
			errortypes.ErrRequired,
			"Invoice line description is required",
		)
	}
	if l.Quantity.LessThanOrEqual(decimal.Zero) {
		multiErr.Add(
			prefix+".quantity",
			errortypes.ErrInvalid,
			"Invoice line quantity must be greater than zero",
		)
	}
}

func (i *Invoice) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if i.ID.IsNil() {
			i.ID = pulid.MustNew("inv_")
		}
		i.CreatedAt = now
	case *bun.UpdateQuery:
		i.UpdatedAt = now
	}

	return nil
}

func (l *Line) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if l.ID.IsNil() {
			l.ID = pulid.MustNew("invl_")
		}
		l.CreatedAt = now
	case *bun.UpdateQuery:
		l.UpdatedAt = now
	}

	return nil
}

func (i *Invoice) GetID() pulid.ID {
	return i.ID
}

func (i *Invoice) GetTableName() string {
	return "invoices"
}

func (i *Invoice) GetOrganizationID() pulid.ID {
	return i.OrganizationID
}

func (i *Invoice) GetBusinessUnitID() pulid.ID {
	return i.BusinessUnitID
}

func (i *Invoice) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "inv",
		UseSearchVector: false,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "number", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{Name: "status", Type: domaintypes.FieldTypeEnum, Weight: domaintypes.SearchWeightA},
			{
				Name:   "bill_to_name",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
			{
				Name:   "shipment_pro_number",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
			{
				Name:   "shipment_bol",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
		},
		Relationships: []*domaintypes.RelationshipDefintion{
			{
				Field:        "shipment",
				Type:         dbtype.RelationshipTypeBelongsTo,
				TargetEntity: (*shipment.Shipment)(nil),
				TargetTable:  "shipments",
				ForeignKey:   "shipment_id",
				ReferenceKey: "id",
				Alias:        "sp",
				Queryable:    true,
			},
			{
				Field:        "customer",
				Type:         dbtype.RelationshipTypeBelongsTo,
				TargetEntity: (*customer.Customer)(nil),
				TargetTable:  "customers",
				ForeignKey:   "customer_id",
				ReferenceKey: "id",
				Alias:        "cus",
				Queryable:    true,
			},
		},
	}
}
