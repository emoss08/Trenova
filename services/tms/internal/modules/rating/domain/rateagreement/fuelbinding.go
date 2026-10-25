package rateagreement

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/rating/domain/fuelsurcharge"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
)

// RateAgreementFuelBinding attaches a fuel surcharge program to a contract, and
// lets that contract bend the program's terms.
//
// Fuel programs are organization-wide, and until now the only way to point a
// customer at one was through their billing profile — one program per customer,
// no matter how many contracts they hold. Real contracts negotiate fuel
// separately: a different peg, a capped surcharge, or none at all. Those are
// the three overrides here, and the resolution order the engine follows is
// agreement, then customer billing profile, then organization default.
type RateAgreementFuelBinding struct {
	keel.Table `keel:"rate_agreement_fuel_bindings"`

	ID                     pulid.ID `json:"id"                     keel:"id,pk,autoid:ragf_"`
	BusinessUnitID         pulid.ID `json:"businessUnitId"         keel:"business_unit_id,pk,tenant"`
	OrganizationID         pulid.ID `json:"organizationId"         keel:"organization_id,pk,tenant"`
	RateAgreementID        pulid.ID `json:"rateAgreementId"        keel:"rate_agreement_id"`
	FuelSurchargeProgramID pulid.ID `json:"fuelSurchargeProgramId" keel:"fuel_surcharge_program_id"`

	// Waived suppresses fuel entirely for this contract. All-in rates are
	// common enough that this needs to be a stated term rather than an omission
	// somebody has to notice.
	Waived bool `json:"waived" keel:"waived"`

	PegPriceOverride      decimal.NullDecimal `json:"pegPriceOverride"      keel:"peg_price_override"`
	IncrementRateOverride decimal.NullDecimal `json:"incrementRateOverride" keel:"increment_rate_override"`
	CapAmount             decimal.NullDecimal `json:"capAmount"             keel:"cap_amount"`

	Version   int64 `json:"version"   keel:"version,version"`
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`

	Agreement *RateAgreement                      `json:"-"                 keel:"belongs:rate_agreement_id"`
	Program   *fuelsurcharge.FuelSurchargeProgram `json:"program,omitempty" keel:"belongs:fuel_surcharge_program_id"`
}

func (ragf *RateAgreementFuelBinding) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(ragf,
		validation.Field(&ragf.FuelSurchargeProgramID,
			validation.Required.Error("Fuel surcharge program is required"),
		),
	))

	validateNonNegative(multiErr, "pegPriceOverride", ragf.PegPriceOverride, "Peg price")
	validateNonNegative(
		multiErr,
		"incrementRateOverride",
		ragf.IncrementRateOverride,
		"Increment rate",
	)
	validateNonNegative(multiErr, "capAmount", ragf.CapAmount, "Cap amount")

	// A waiver and an override describe opposite intentions, and leaving both
	// set makes the effective terms depend on which the engine reads first.
	if ragf.Waived &&
		(ragf.PegPriceOverride.Valid || ragf.IncrementRateOverride.Valid ||
			ragf.CapAmount.Valid) {
		multiErr.Add(
			"waived",
			errortypes.ErrInvalid,
			"A waived fuel binding cannot also override the program's terms",
		)
	}
}

func validateNonNegative(
	multiErr *errortypes.MultiError,
	field string,
	value decimal.NullDecimal,
	label string,
) {
	if value.Valid && value.Decimal.IsNegative() {
		multiErr.Add(field, errortypes.ErrInvalid, "{0} cannot be negative", label)
	}
}

func (ragf *RateAgreementFuelBinding) GetID() pulid.ID {
	return ragf.ID
}

func (ragf *RateAgreementFuelBinding) GetCreatedAt() int64 {
	return ragf.CreatedAt
}

func (ragf *RateAgreementFuelBinding) GetOrganizationID() pulid.ID {
	return ragf.OrganizationID
}

func (ragf *RateAgreementFuelBinding) GetBusinessUnitID() pulid.ID {
	return ragf.BusinessUnitID
}

func (ragf *RateAgreementFuelBinding) GetTableName() string {
	return "rate_agreement_fuel_bindings"
}

func (RateAgreementFuelBinding) Constraints() []keel.Constraint {
	return []keel.Constraint{
		{
			Name:    "uq_rate_agreement_fuel_bindings",
			Field:   "fuelBinding",
			Message: "A rate agreement can bind only one fuel surcharge program",
		},
	}
}
