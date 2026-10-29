package shipment

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/decimalutils"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
)

// ChargeAllocation gives one payer a share of one charge. A charge with no
// allocations belongs entirely to the shipment's payer, so rows exist only
// where billing departs from that default.
type ChargeAllocation struct {
	keel.Table `keel:"charge_allocations"`

	ID                 pulid.ID                             `json:"id"                 keel:"id,pk,autoid:chal_"`
	BusinessUnitID     pulid.ID                             `json:"businessUnitId"     keel:"business_unit_id,pk,tenant"`
	OrganizationID     pulid.ID                             `json:"organizationId"     keel:"organization_id,pk,tenant"`
	ShipmentID         *pulid.ID                            `json:"shipmentId"         keel:"shipment_id"`
	AdditionalChargeID *pulid.ID                            `json:"additionalChargeId" keel:"additional_charge_id"`
	OrderChargeID      *pulid.ID                            `json:"orderChargeId"      keel:"order_charge_id"`
	ChargeKind         shipmentvocab.ChargeAllocationKind   `json:"chargeKind"         keel:"charge_kind"`
	BillToCustomerID   pulid.ID                             `json:"billToCustomerId"   keel:"bill_to_customer_id"`
	Method             shipmentvocab.ChargeAllocationMethod `json:"method"             keel:"method"`
	Percent            decimal.NullDecimal                  `json:"percent"            keel:"percent"`
	Amount             decimal.NullDecimal                  `json:"amount"             keel:"amount"`
	Sequence           int16                                `json:"sequence"           keel:"sequence"`
	InvoiceID          pulid.ID                             `json:"invoiceId"          keel:"invoice_id,nullzero"`
	InvoicedAt         *int64                               `json:"invoicedAt"         keel:"invoiced_at"`
	Version            int64                                `json:"version"            keel:"version,version"`
	CreatedAt          int64                                `json:"createdAt"          keel:"created_at,autocreate"`
	UpdatedAt          int64                                `json:"updatedAt"          keel:"updated_at,autoupdate"`

	// AdditionalChargeIndex points at a charge in the same payload that has no
	// id yet, so a new charge and its split can arrive in one save.
	AdditionalChargeIndex *int `json:"additionalChargeIndex,omitempty"`

	BusinessUnit   *tenant.BusinessUnit `json:"businessUnit,omitempty"`
	Organization   *tenant.Organization `json:"organization,omitempty"`
	Shipment       *Shipment            `json:"-"                        keel:"belongs:shipment_id"`
	BillToCustomer *customer.Customer   `json:"billToCustomer,omitempty" keel:"belongs:bill_to_customer_id"`
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
	case shipmentvocab.ChargeAllocationMethodPercent:
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
			multiErr.Add(
				"amount",
				errortypes.ErrInvalid,
				"A percent allocation cannot carry an amount",
			)
		}
	case shipmentvocab.ChargeAllocationMethodAmount:
		switch {
		case !a.Amount.Valid:
			multiErr.Add("amount", errortypes.ErrRequired, "Amount is required")
		case a.Amount.Decimal.LessThanOrEqual(decimal.Zero):
			multiErr.Add("amount", errortypes.ErrInvalid, "Amount must be greater than zero")
		}
		if a.Percent.Valid {
			multiErr.Add(
				"percent",
				errortypes.ErrInvalid,
				"An amount allocation cannot carry a percent",
			)
		}
	}

	switch a.ChargeKind {
	case shipmentvocab.ChargeAllocationKindFreight:
		if a.AdditionalChargeID != nil || a.OrderChargeID != nil {
			multiErr.Add(
				"chargeKind",
				errortypes.ErrInvalid,
				"A freight allocation targets only the shipment",
			)
		}
	case shipmentvocab.ChargeAllocationKindAccessorial:
		if a.OrderChargeID != nil {
			multiErr.Add(
				"chargeKind",
				errortypes.ErrInvalid,
				"An accessorial allocation cannot target an order charge",
			)
		}
		if (a.AdditionalChargeID == nil || a.AdditionalChargeID.IsNil()) &&
			a.AdditionalChargeIndex == nil {
			multiErr.Add(
				"additionalChargeId",
				errortypes.ErrRequired,
				"An accessorial allocation must name its charge",
			)
		}
	case shipmentvocab.ChargeAllocationKindOrderCharge:
		if a.OrderChargeID == nil || a.OrderChargeID.IsNil() {
			multiErr.Add(
				"orderChargeId",
				errortypes.ErrRequired,
				"An order charge allocation must name its charge",
			)
		}
		if a.ShipmentID != nil || a.AdditionalChargeID != nil {
			multiErr.Add(
				"chargeKind",
				errortypes.ErrInvalid,
				"An order charge allocation targets only the order charge",
			)
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
	case shipmentvocab.ChargeAllocationKindFreight:
		return pulid.ConvertFromPtr(a.ShipmentID)
	case shipmentvocab.ChargeAllocationKindAccessorial:
		return pulid.ConvertFromPtr(a.AdditionalChargeID)
	case shipmentvocab.ChargeAllocationKindOrderCharge:
		return pulid.ConvertFromPtr(a.OrderChargeID)
	default:
		return pulid.Nil
	}
}

func (a *ChargeAllocation) IsInvoiced() bool {
	return a != nil && a.InvoiceID.IsNotNil()
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
