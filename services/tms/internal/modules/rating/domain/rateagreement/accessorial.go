package rateagreement

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
)

const maxConditionLength = 20000

// The negotiated price of each accessorial under a contract
type RateAgreementAccessorial struct {
	keel.Table `keel:"rate_agreement_accessorials,report=Billing,label=Contract Accessorial"`

	ID                  pulid.ID `json:"id"                  keel:"id,pk,autoid:raga_"`
	BusinessUnitID      pulid.ID `json:"businessUnitId"      keel:"business_unit_id,pk,tenant"`
	OrganizationID      pulid.ID `json:"organizationId"      keel:"organization_id,pk,tenant"`
	RateAgreementID     pulid.ID `json:"rateAgreementId"     keel:"rate_agreement_id"`
	AccessorialChargeID pulid.ID `json:"accessorialChargeId" keel:"accessorial_charge_id"`

	Method   accessorialcharge.Method   `json:"method"   keel:"method"`
	RateUnit accessorialcharge.RateUnit `json:"rateUnit" keel:"rate_unit,nullzero"`
	Amount   decimal.Decimal            `json:"amount"   keel:"amount,format=money"`

	Waived bool `json:"waived" keel:"waived"`

	AutoApply bool `json:"autoApply" keel:"auto_apply"`

	Condition string `json:"condition" keel:"condition,nullzero"`

	FreeUnits *int16              `json:"freeUnits" keel:"free_units,format=count"`
	MaxAmount decimal.NullDecimal `json:"maxAmount" keel:"max_amount,label=Maximum Amount,format=money"`

	ServiceTypeIDs  []pulid.ID `json:"serviceTypeIds"  keel:"service_type_ids,jsonb,nullzero"`
	ShipmentTypeIDs []pulid.ID `json:"shipmentTypeIds" keel:"shipment_type_ids,jsonb,nullzero"`

	EffectiveFrom *int64 `json:"effectiveFrom" keel:"effective_from,type=epoch"`
	EffectiveTo   *int64 `json:"effectiveTo"   keel:"effective_to,type=epoch"`

	Version   int64 `json:"version"   keel:"version,version"`
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`

	Agreement         *RateAgreement                       `json:"-"                           keel:"belongs:rate_agreement_id"`
	AccessorialCharge *accessorialcharge.AccessorialCharge `json:"accessorialCharge,omitempty" keel:"belongs:accessorial_charge_id"`
}

func (raga *RateAgreementAccessorial) BeforeUpdate(_ context.Context) error {
	raga.applyDefaults()

	return nil
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

type AccessorialMeasures struct {
	Miles  decimal.Decimal
	Dwells []decimal.Decimal
	Days   decimal.Decimal
	Stops  decimal.Decimal
}

func (raga *RateAgreementAccessorial) BillableQuantity(
	measures *AccessorialMeasures,
) decimal.Decimal {
	if raga.Method != accessorialcharge.MethodPerUnit {
		return decimal.NewFromInt(1)
	}
	free := decimal.Zero
	if raga.FreeUnits != nil {
		free = decimal.NewFromInt(int64(*raga.FreeUnits))
	}
	if raga.RateUnit == accessorialcharge.RateUnitHour {
		billable := decimal.Zero
		for _, dwell := range measures.Dwells {
			billable = billable.Add(decimal.Max(dwell.Sub(free), decimal.Zero))
		}
		return billable
	}
	var measured decimal.Decimal
	switch raga.RateUnit { //nolint:exhaustive // hours are measured per stop above
	case accessorialcharge.RateUnitMile:
		measured = measures.Miles
	case accessorialcharge.RateUnitDay:
		measured = measures.Days
	case accessorialcharge.RateUnitStop:
		measured = measures.Stops
	default:
		measured = decimal.NewFromInt(1)
	}
	return decimal.Max(measured.Sub(free), decimal.Zero)
}

func (raga *RateAgreementAccessorial) Cap(amount decimal.Decimal) (decimal.Decimal, bool) {
	if raga.MaxAmount.Valid && amount.GreaterThan(raga.MaxAmount.Decimal) {
		return raga.MaxAmount.Decimal, true
	}
	return amount, false
}

func (raga *RateAgreementAccessorial) TariffRateUnit() string {
	if raga.Method != accessorialcharge.MethodPerUnit {
		return ""
	}
	return raga.RateUnit.TariffRateUnit()
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
		validation.Field(&raga.Condition,
			validation.Length(0, maxConditionLength).
				Error("Condition cannot be longer than 20000 characters"),
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
	if raga.Condition != "" && !raga.AutoApply {
		multiErr.Add(
			"condition",
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

func (raga *RateAgreementAccessorial) BeforeCreate(_ context.Context) error {
	raga.applyDefaults()
	raga.applyDefaults()

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

func (RateAgreementAccessorial) Constraints() []keel.Constraint {
	return []keel.Constraint{
		{
			Name:    "uq_rate_agreement_accessorials",
			Field:   "accessorials",
			Message: "An accessorial charge can appear only once in an agreement's schedule",
		},
	}
}
