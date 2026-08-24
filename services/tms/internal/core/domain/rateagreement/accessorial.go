package rateagreement

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*RateAgreementAccessorial)(nil)

const maxApplyConditionLength = 500

// RateAgreementAccessorial is a contract's price for one accessorial.
//
// Organizations already define accessorials once, centrally, with a default
// price. What a contract negotiates is a different price for the same charge —
// or free hours before it starts, or a cap, or a waiver. This row carries that
// negotiation, and it is why the rate confirmation and the invoice can finally
// agree: both read the accessorial's price from the same contract.
type RateAgreementAccessorial struct {
	bun.BaseModel `bun:"table:rate_agreement_accessorials,alias:raga" json:"-"`

	ID                  pulid.ID `json:"id"                  bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID      pulid.ID `json:"businessUnitId"      bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID      pulid.ID `json:"organizationId"      bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	RateAgreementID     pulid.ID `json:"rateAgreementId"     bun:"rate_agreement_id,type:VARCHAR(100),notnull"`
	AccessorialChargeID pulid.ID `json:"accessorialChargeId" bun:"accessorial_charge_id,type:VARCHAR(100),notnull"`

	Method   accessorialcharge.Method   `json:"method"   bun:"method,type:accessorial_method_enum,notnull"`
	RateUnit accessorialcharge.RateUnit `json:"rateUnit" bun:"rate_unit,type:rate_unit_enum,nullzero"`
	Amount   decimal.Decimal            `json:"amount"   bun:"amount,type:NUMERIC(19,4),notnull,default:0"`

	// Waived prices the accessorial at nothing for this contract. It is a
	// distinct state from an amount of zero, because a waiver has to survive a
	// later change to the organization's default price.
	Waived bool `json:"waived" bun:"waived,type:BOOLEAN,notnull,default:false"`

	// AutoApply adds the charge to every shipment the agreement prices, subject
	// to ApplyCondition. Left off, the accessorial is merely priced, and a user
	// still has to add it.
	AutoApply bool `json:"autoApply" bun:"auto_apply,type:BOOLEAN,notnull,default:false"`

	// ApplyCondition is an expression evaluated against the shipment, in the
	// same language and against the same schema the formula editor already
	// exposes — "totalStops > 2", say. Blank means always.
	ApplyCondition string `json:"applyCondition" bun:"apply_condition,type:TEXT,nullzero"`

	// FreeUnits are granted before the charge starts counting, which is how
	// detention and storage allowances are written.
	FreeUnits *int16              `json:"freeUnits" bun:"free_units,type:SMALLINT,nullzero"`
	MaxAmount decimal.NullDecimal `json:"maxAmount" bun:"max_amount,type:NUMERIC(19,4),nullzero"`

	FormulaTemplateID *pulid.ID `json:"formulaTemplateId" bun:"formula_template_id,type:VARCHAR(100),nullzero"`

	ServiceTypeIDs  []pulid.ID `json:"serviceTypeIds"  bun:"service_type_ids,type:JSONB,nullzero"`
	ShipmentTypeIDs []pulid.ID `json:"shipmentTypeIds" bun:"shipment_type_ids,type:JSONB,nullzero"`

	EffectiveFrom *int64 `json:"effectiveFrom" bun:"effective_from,type:BIGINT,nullzero"`
	EffectiveTo   *int64 `json:"effectiveTo"   bun:"effective_to,type:BIGINT,nullzero"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Agreement         *RateAgreement                       `json:"-"                           bun:"rel:belongs-to,join:rate_agreement_id=id"`
	AccessorialCharge *accessorialcharge.AccessorialCharge `json:"accessorialCharge,omitempty" bun:"rel:belongs-to,join:accessorial_charge_id=id"`
}

// IsEffectiveAt reports whether this price is live at a moment. Both bounds are
// optional, since most schedule rows simply run with their agreement.
func (raga *RateAgreementAccessorial) IsEffectiveAt(timestamp int64) bool {
	if raga.EffectiveFrom != nil && timestamp < *raga.EffectiveFrom {
		return false
	}

	if raga.EffectiveTo != nil && timestamp >= *raga.EffectiveTo {
		return false
	}

	return true
}

// PricedAmount is the amount to charge, which is nothing when the contract
// waived it.
func (raga *RateAgreementAccessorial) PricedAmount() decimal.Decimal {
	if raga.Waived {
		return decimal.Zero
	}

	return raga.Amount
}

// BillableUnits deducts any free allowance from a measured quantity, never
// going below zero.
func (raga *RateAgreementAccessorial) BillableUnits(units int16) int16 {
	if raga.FreeUnits == nil {
		return units
	}

	if units <= *raga.FreeUnits {
		return 0
	}

	return units - *raga.FreeUnits
}

func (raga *RateAgreementAccessorial) applyDefaults() {
	if raga.Method != accessorialcharge.MethodPerUnit {
		raga.RateUnit = ""
	}
}

func (raga *RateAgreementAccessorial) Validate(multiErr *errortypes.MultiError) {
	raga.applyDefaults()

	multiErr.AddOzzoError(validation.ValidateStruct(raga,
		validation.Field(&raga.AccessorialChargeID,
			validation.Required.Error("Accessorial charge is required"),
		),
		validation.Field(&raga.Method,
			validation.Required.Error("Method is required"),
			validation.In(
				accessorialcharge.MethodFlat,
				accessorialcharge.MethodPerUnit,
				accessorialcharge.MethodPercentage,
			).Error("Method is invalid"),
		),
		validation.Field(&raga.RateUnit,
			validation.When(
				raga.Method == accessorialcharge.MethodPerUnit,
				validation.Required.Error("Rate unit is required when method is PerUnit"),
			),
			validation.In(
				accessorialcharge.RateUnitMile,
				accessorialcharge.RateUnitHour,
				accessorialcharge.RateUnitDay,
				accessorialcharge.RateUnitStop,
			).Error("Rate unit is invalid"),
		),
		validation.Field(&raga.ApplyCondition,
			validation.Length(0, maxApplyConditionLength).
				Error("Apply condition cannot be longer than 500 characters"),
		),
	))

	raga.validateAmounts(multiErr)

	if raga.EffectiveFrom != nil && raga.EffectiveTo != nil &&
		*raga.EffectiveTo <= *raga.EffectiveFrom {
		multiErr.Add(
			"effectiveTo",
			errortypes.ErrInvalid,
			"Effective to must be after effective from",
		)
	}

	// A condition can only gate a charge that applies on its own; on a row the
	// user adds by hand it would be stored and never consulted.
	if raga.ApplyCondition != "" && !raga.AutoApply {
		multiErr.Add(
			"applyCondition",
			errortypes.ErrInvalid,
			"A condition only applies to an automatically applied accessorial",
		)
	}
}

func (raga *RateAgreementAccessorial) validateAmounts(multiErr *errortypes.MultiError) {
	if raga.Amount.IsNegative() {
		multiErr.Add("amount", errortypes.ErrInvalid, "Amount cannot be negative")
	}

	// A waived accessorial is priced at nothing by definition, so a non-zero
	// amount beside the waiver is a contradiction the user should resolve
	// rather than have the engine silently pick a side of.
	if raga.Waived && raga.Amount.GreaterThan(decimal.Zero) {
		multiErr.Add(
			"amount",
			errortypes.ErrInvalid,
			"A waived accessorial cannot also carry an amount",
		)
	}

	if raga.MaxAmount.Valid {
		if raga.MaxAmount.Decimal.IsNegative() {
			multiErr.Add("maxAmount", errortypes.ErrInvalid, "Maximum cannot be negative")
		}
		if raga.MaxAmount.Decimal.LessThan(raga.Amount) &&
			raga.Method != accessorialcharge.MethodPercentage {
			multiErr.Add(
				"maxAmount",
				errortypes.ErrInvalid,
				"Maximum cannot be less than the amount",
			)
		}
	}

	if raga.FreeUnits != nil && *raga.FreeUnits < 0 {
		multiErr.Add("freeUnits", errortypes.ErrInvalid, "Free units cannot be negative")
	}
}

func (raga *RateAgreementAccessorial) BeforeAppendModel(
	_ context.Context,
	query bun.Query,
) error {
	raga.applyDefaults()

	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if raga.ID.IsNil() {
			raga.ID = pulid.MustNew("raga_")
		}
		raga.CreatedAt = now
		raga.UpdatedAt = now
	case *bun.UpdateQuery:
		raga.UpdatedAt = now
	}

	return nil
}

func (raga *RateAgreementAccessorial) GetID() pulid.ID {
	return raga.ID
}

func (raga *RateAgreementAccessorial) GetCreatedAt() int64 {
	return raga.CreatedAt
}

func (raga *RateAgreementAccessorial) GetOrganizationID() pulid.ID {
	return raga.OrganizationID
}

func (raga *RateAgreementAccessorial) GetBusinessUnitID() pulid.ID {
	return raga.BusinessUnitID
}

func (raga *RateAgreementAccessorial) GetTableName() string {
	return "rate_agreement_accessorials"
}
