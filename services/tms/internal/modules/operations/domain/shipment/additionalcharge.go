package shipment

import (
	"errors"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
)

type AdditionalCharge struct {
	keel.Table `keel:"additional_charges"`

	ID                     pulid.ID                 `json:"id"                         keel:"id,pk,autoid:ac_"`
	BusinessUnitID         pulid.ID                 `json:"businessUnitId"             keel:"business_unit_id,pk,tenant"`
	OrganizationID         pulid.ID                 `json:"organizationId"             keel:"organization_id,pk,tenant"`
	ShipmentID             pulid.ID                 `json:"shipmentId"                 keel:"shipment_id,pk"`
	AccessorialChargeID    pulid.ID                 `json:"accessorialChargeId"        keel:"accessorial_charge_id,pk"`
	IsSystemGenerated      bool                     `json:"isSystemGenerated"          keel:"is_system_generated"`
	Method                 accessorialcharge.Method `json:"method"                     keel:"method"`
	Amount                 decimal.Decimal          `json:"amount"                     keel:"amount"`
	Unit                   int16                    `json:"unit"                       keel:"unit"`
	FuelSurchargeProgramID *pulid.ID                `json:"fuelSurchargeProgramId"     keel:"fuel_surcharge_program_id"`
	FuelSurchargeDetail    *FuelSurchargeDetail     `json:"fuelSurchargeDetail"        keel:"fuel_surcharge_detail,jsonb"`
	IsDetention            bool                     `json:"isDetention"                keel:"is_detention"`
	IsTariff               bool                     `json:"isTariff"                   keel:"is_tariff"`
	Label                  string                   `json:"label"                      keel:"label,nullzero"`
	Quantity               decimal.NullDecimal      `json:"quantity"                   keel:"quantity"`
	Rate                   decimal.NullDecimal      `json:"rate"                       keel:"rate"`
	RateUnit               string                   `json:"rateUnit"                   keel:"rate_unit,nullzero"`
	// RateAgreementAccessorialID marks a charge the contract's own accessorial
	// schedule produced, and is what its reconciliation pass matches on.
	RateAgreementAccessorialID *pulid.ID `json:"rateAgreementAccessorialId" keel:"rate_agreement_accessorial_id"`
	RateQuoteID                *pulid.ID `json:"rateQuoteId"                keel:"rate_quote_id"`
	Version                    int64     `json:"version"                    keel:"version,version"`
	CreatedAt                  int64     `json:"createdAt"                  keel:"created_at,autocreate"`
	UpdatedAt                  int64     `json:"updatedAt"                  keel:"updated_at,autoupdate"`

	DetentionOccurrenceIDs []pulid.ID `json:"-"`

	BusinessUnit      *tenant.BusinessUnit                 `json:"businessUnit,omitempty"`
	Organization      *tenant.Organization                 `json:"organization,omitempty"`
	Shipment          *Shipment                            `json:"shipment,omitempty"          keel:"belongs:shipment_id"`
	AccessorialCharge *accessorialcharge.AccessorialCharge `json:"accessorialCharge,omitempty" keel:"belongs:accessorial_charge_id"`
}

func (a *AdditionalCharge) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(
		a,
		validation.Field(
			&a.AccessorialChargeID,
			validation.Required.Error("Accessorial charge is required"),
		),
		validation.Field(
			&a.Unit,
			validation.Min(1).Error("Unit must be greater than or equal to 1"),
		),
		validation.Field(
			&a.Method,
			validation.Required.Error("Method is required"),
			validation.In(
				accessorialcharge.MethodFlat,
				accessorialcharge.MethodPerUnit,
				accessorialcharge.MethodPercentage,
			).Error("Invalid method"),
		),
		validation.Field(
			&a.Amount,
			validation.Required.Error("Amount is required"),
			validation.By(func(_ any) error {
				if a.Amount.LessThanOrEqual(decimal.Zero) {
					return errors.New("amount must be greater than zero")
				}
				return nil
			}),
		),
	))
}

// SystemOwner names the engine that owns a machine generated charge, which is
// how the reconciliation passes tell their own rows apart from each other's.
type SystemOwner string

const (
	SystemOwnerNone      = SystemOwner("")
	SystemOwnerFuel      = SystemOwner("FuelSurcharge")
	SystemOwnerDetention = SystemOwner("Detention")
	SystemOwnerTariff    = SystemOwner("Tariff")
	SystemOwnerAgreement = SystemOwner("RateAgreement")
)

// Owner reports which engine, if any, owns this charge.
func (a *AdditionalCharge) Owner() SystemOwner {
	switch {
	case !a.IsSystemGenerated:
		return SystemOwnerNone
	case a.FuelSurchargeProgramID != nil:
		return SystemOwnerFuel
	case a.IsDetention:
		return SystemOwnerDetention
	case a.IsTariff:
		return SystemOwnerTariff
	case a.RateAgreementAccessorialID != nil:
		return SystemOwnerAgreement
	default:
		return SystemOwnerNone
	}
}

func (AdditionalCharge) Constraints() []keel.Constraint {
	return []keel.Constraint{
		{
			Name:    "fk_additional_charges_accessorial_charge",
			Message: "Accessorial charge does not exist in your organization",
		},
		{
			Name:    "fk_additional_charges_shipment",
			Message: "Shipment does not exist in your organization",
		},
	}
}
