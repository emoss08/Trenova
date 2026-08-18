package rateagreement

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/fuelsurcharge"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*RateAgreementFuelBinding)(nil)

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
	bun.BaseModel `bun:"table:rate_agreement_fuel_bindings,alias:ragf" json:"-"`

	ID                     pulid.ID `json:"id"                     bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID         pulid.ID `json:"businessUnitId"         bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID         pulid.ID `json:"organizationId"         bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	RateAgreementID        pulid.ID `json:"rateAgreementId"        bun:"rate_agreement_id,type:VARCHAR(100),notnull,unique"`
	FuelSurchargeProgramID pulid.ID `json:"fuelSurchargeProgramId" bun:"fuel_surcharge_program_id,type:VARCHAR(100),notnull"`

	// Waived suppresses fuel entirely for this contract. All-in rates are
	// common enough that this needs to be a stated term rather than an omission
	// somebody has to notice.
	Waived bool `json:"waived" bun:"waived,type:BOOLEAN,notnull,default:false"`

	PegPriceOverride      decimal.NullDecimal `json:"pegPriceOverride"      bun:"peg_price_override,type:NUMERIC(19,4),nullzero"`
	IncrementRateOverride decimal.NullDecimal `json:"incrementRateOverride" bun:"increment_rate_override,type:NUMERIC(19,4),nullzero"`
	CapAmount             decimal.NullDecimal `json:"capAmount"             bun:"cap_amount,type:NUMERIC(19,4),nullzero"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Agreement *RateAgreement                      `json:"-"                 bun:"rel:belongs-to,join:rate_agreement_id=id"`
	Program   *fuelsurcharge.FuelSurchargeProgram `json:"program,omitempty" bun:"rel:belongs-to,join:fuel_surcharge_program_id=id"`
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
		multiErr.Add(field, errortypes.ErrInvalid, label+" cannot be negative")
	}
}

func (ragf *RateAgreementFuelBinding) BeforeAppendModel(
	_ context.Context,
	query bun.Query,
) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if ragf.ID.IsNil() {
			ragf.ID = pulid.MustNew("ragf_")
		}
		ragf.CreatedAt = now
		ragf.UpdatedAt = now
	case *bun.UpdateQuery:
		ragf.UpdatedAt = now
	}

	return nil
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
