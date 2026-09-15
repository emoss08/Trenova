package shipment

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/decimalutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

type ChargeAllocationKind string

const (
	ChargeAllocationKindFreight     = ChargeAllocationKind("Freight")
	ChargeAllocationKindAccessorial = ChargeAllocationKind("Accessorial")
	ChargeAllocationKindOrderCharge = ChargeAllocationKind("OrderCharge")
)

func (k ChargeAllocationKind) IsValid() bool {
	switch k {
	case ChargeAllocationKindFreight, ChargeAllocationKindAccessorial, ChargeAllocationKindOrderCharge:
		return true
	default:
		return false
	}
}

type ChargeAllocationMethod string

const (
	ChargeAllocationMethodPercent = ChargeAllocationMethod("Percent")
	ChargeAllocationMethodAmount  = ChargeAllocationMethod("Amount")
)

func (m ChargeAllocationMethod) IsValid() bool {
	switch m {
	case ChargeAllocationMethodPercent, ChargeAllocationMethodAmount:
		return true
	default:
		return false
	}
}

var _ bun.BeforeAppendModelHook = (*ChargeAllocation)(nil)

// ChargeAllocation gives one payer a share of one charge. A charge with no
// allocations belongs entirely to the shipment's payer, so rows exist only
// where billing departs from that default.
type ChargeAllocation struct {
	bun.BaseModel `json:"-" bun:"table:charge_allocations,alias:chal"`

	ID                 pulid.ID               `json:"id"                 bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID     pulid.ID               `json:"businessUnitId"     bun:"business_unit_id,pk,notnull,type:VARCHAR(100)"`
	OrganizationID     pulid.ID               `json:"organizationId"     bun:"organization_id,pk,notnull,type:VARCHAR(100)"`
	ShipmentID         *pulid.ID              `json:"shipmentId"         bun:"shipment_id,type:VARCHAR(100),nullzero"`
	AdditionalChargeID *pulid.ID              `json:"additionalChargeId" bun:"additional_charge_id,type:VARCHAR(100),nullzero"`
	OrderChargeID      *pulid.ID              `json:"orderChargeId"      bun:"order_charge_id,type:VARCHAR(100),nullzero"`
	ChargeKind         ChargeAllocationKind   `json:"chargeKind"         bun:"charge_kind,type:charge_allocation_kind_enum,notnull"`
	BillToCustomerID   pulid.ID               `json:"billToCustomerId"   bun:"bill_to_customer_id,type:VARCHAR(100),notnull"`
	Method             ChargeAllocationMethod `json:"method"             bun:"method,type:charge_allocation_method_enum,notnull"`
	Percent            decimal.NullDecimal    `json:"percent"            bun:"percent,type:NUMERIC(9,6),nullzero"`
	Amount             decimal.NullDecimal    `json:"amount"             bun:"amount,type:NUMERIC(19,4),nullzero"`
	Sequence           int16                  `json:"sequence"           bun:"sequence,type:SMALLINT,notnull"`
	InvoiceID          pulid.ID               `json:"invoiceId"          bun:"invoice_id,type:VARCHAR(100),nullzero"`
	InvoicedAt         *int64                 `json:"invoicedAt"         bun:"invoiced_at,type:BIGINT,nullzero"`
	Version            int64                  `json:"version"            bun:"version,type:BIGINT"`
	CreatedAt          int64                  `json:"createdAt"          bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt          int64                  `json:"updatedAt"          bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// AdditionalChargeIndex points at a charge in the same payload that has no
	// id yet, so a new charge and its split can arrive in one save.
	AdditionalChargeIndex *int `json:"additionalChargeIndex,omitempty" bun:"-"`

	BusinessUnit   *tenant.BusinessUnit `json:"businessUnit,omitempty"   bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization   *tenant.Organization `json:"organization,omitempty"   bun:"rel:belongs-to,join:organization_id=id"`
	Shipment       *Shipment            `json:"-"                        bun:"rel:belongs-to,join:shipment_id=id"`
	BillToCustomer *customer.Customer   `json:"billToCustomer,omitempty" bun:"rel:belongs-to,join:bill_to_customer_id=id"`
}

func (a *ChargeAllocation) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(
		a,
		validation.Field(
			&a.ChargeKind,
			validation.Required.Error("Charge kind is required"),
			validation.By(func(_ any) error {
				if !a.ChargeKind.IsValid() {
					return errortypes.NewValidationError(
						"chargeKind",
						errortypes.ErrInvalid,
						"Charge kind must be Freight, Accessorial or OrderCharge",
					)
				}
				return nil
			}),
		),
		validation.Field(
			&a.BillToCustomerID,
			validation.Required.Error("Bill-to customer is required"),
		),
		validation.Field(
			&a.Method,
			validation.Required.Error("Allocation method is required"),
			validation.By(func(_ any) error {
				if !a.Method.IsValid() {
					return errortypes.NewValidationError(
						"method",
						errortypes.ErrInvalid,
						"Allocation method must be Percent or Amount",
					)
				}
				return nil
			}),
		),
		validation.Field(
			&a.Sequence,
			validation.Min(0).Error("Sequence cannot be negative"),
		),
	))

	switch a.Method {
	case ChargeAllocationMethodPercent:
		switch {
		case !a.Percent.Valid:
			multiErr.Add("percent", errortypes.ErrRequired, "Percent is required")
		case a.Percent.Decimal.LessThanOrEqual(decimal.Zero) ||
			a.Percent.Decimal.GreaterThan(decimalutils.Percent100):
			multiErr.Add(
				"percent",
				errortypes.ErrInvalid,
				"Percent must be greater than 0 and at most 100",
			)
		}
		if a.Amount.Valid {
			multiErr.Add("amount", errortypes.ErrInvalid, "A percent allocation cannot carry an amount")
		}
	case ChargeAllocationMethodAmount:
		switch {
		case !a.Amount.Valid:
			multiErr.Add("amount", errortypes.ErrRequired, "Amount is required")
		case a.Amount.Decimal.LessThanOrEqual(decimal.Zero):
			multiErr.Add("amount", errortypes.ErrInvalid, "Amount must be greater than zero")
		}
		if a.Percent.Valid {
			multiErr.Add("percent", errortypes.ErrInvalid, "An amount allocation cannot carry a percent")
		}
	}

	switch a.ChargeKind {
	case ChargeAllocationKindFreight:
		if a.AdditionalChargeID != nil || a.OrderChargeID != nil {
			multiErr.Add("chargeKind", errortypes.ErrInvalid, "A freight allocation targets only the shipment")
		}
	case ChargeAllocationKindAccessorial:
		if a.OrderChargeID != nil {
			multiErr.Add("chargeKind", errortypes.ErrInvalid, "An accessorial allocation cannot target an order charge")
		}
		if (a.AdditionalChargeID == nil || a.AdditionalChargeID.IsNil()) && a.AdditionalChargeIndex == nil {
			multiErr.Add("additionalChargeId", errortypes.ErrRequired, "An accessorial allocation must name its charge")
		}
	case ChargeAllocationKindOrderCharge:
		if a.OrderChargeID == nil || a.OrderChargeID.IsNil() {
			multiErr.Add("orderChargeId", errortypes.ErrRequired, "An order charge allocation must name its charge")
		}
		if a.ShipmentID != nil || a.AdditionalChargeID != nil {
			multiErr.Add("chargeKind", errortypes.ErrInvalid, "An order charge allocation targets only the order charge")
		}
	}
}

// TargetID identifies the charge this allocation belongs to: the shipment for
// freight, the additional charge for an accessorial, the order charge otherwise.
func (a *ChargeAllocation) TargetID() pulid.ID {
	if a == nil {
		return pulid.Nil
	}
	switch a.ChargeKind {
	case ChargeAllocationKindFreight:
		return pulid.ConvertFromPtr(a.ShipmentID)
	case ChargeAllocationKindAccessorial:
		return pulid.ConvertFromPtr(a.AdditionalChargeID)
	case ChargeAllocationKindOrderCharge:
		return pulid.ConvertFromPtr(a.OrderChargeID)
	default:
		return pulid.Nil
	}
}

func (a *ChargeAllocation) IsInvoiced() bool {
	return a != nil && a.InvoiceID.IsNotNil()
}

func (a *ChargeAllocation) BeforeAppendModel(_ context.Context, q bun.Query) error {
	now := timeutils.NowUnix()

	switch q.(type) {
	case *bun.InsertQuery:
		if a.ID.IsNil() {
			a.ID = pulid.MustNew("chal_")
		}
		a.CreatedAt = now
	case *bun.UpdateQuery:
		a.UpdatedAt = now
	}

	return nil
}

func (a *ChargeAllocation) GetID() pulid.ID {
	return a.ID
}

func (a *ChargeAllocation) GetOrganizationID() pulid.ID {
	return a.OrganizationID
}

func (a *ChargeAllocation) GetBusinessUnitID() pulid.ID {
	return a.BusinessUnitID
}

func (a *ChargeAllocation) GetTableName() string {
	return "charge_allocations"
}
