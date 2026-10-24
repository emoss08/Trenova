package jurisdictionrule

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
)

const (
	FederalMaxWidthFeet    = 8.5
	FederalMaxHeightFeet   = 13.5
	FederalMaxLengthFeet   = 53.0
	FederalMaxWeightPounds = 80000

	MaxReasonableWidthFeet  = 60.0
	MaxReasonableHeightFeet = 40.0
	MaxReasonableLengthFeet = 300.0
	MaxLeadTimeDays         = 60
	MaxValidityDays         = 365

	// MinSourceNoteLength is what a verification has to say about itself. A
	// confirmation with no traceable note is indistinguishable from a guess to
	// the next person who reads the row.
	MinSourceNoteLength = 10
)

// JurisdictionRule is global reference data rather than tenant data: the legal
// width of a load in Nebraska does not vary by carrier. Carriers express their
// own, stricter posture through JurisdictionOverride instead of editing this.
type JurisdictionRule struct {
	keel.Table `keel:"jurisdiction_rules"`

	ID      pulid.ID `json:"id" keel:"id,pk,autoid:jr_"`
	StateID pulid.ID `json:"stateId" keel:"state_id"`
	Status  Status   `json:"status" keel:"status"`

	MaxWidthFeet    float64 `json:"maxWidthFeet" keel:"max_width_feet"`
	MaxHeightFeet   float64 `json:"maxHeightFeet" keel:"max_height_feet"`
	MaxLengthFeet   float64 `json:"maxLengthFeet" keel:"max_length_feet"`
	MaxWeightPounds int64   `json:"maxWeightPounds" keel:"max_weight_pounds"`

	SuperloadWidthFeet    *float64 `json:"superloadWidthFeet" keel:"superload_width_feet"`
	SuperloadWeightPounds *int64   `json:"superloadWeightPounds" keel:"superload_weight_pounds"`

	DaylightOnly       bool `json:"daylightOnly" keel:"daylight_only"`
	RushHourRestricted bool `json:"rushHourRestricted" keel:"rush_hour_restricted"`
	WeekendRestricted  bool `json:"weekendRestricted" keel:"weekend_restricted"`
	HolidayRestricted  bool `json:"holidayRestricted" keel:"holiday_restricted"`

	PermitLeadTimeDays int16               `json:"permitLeadTimeDays" keel:"permit_lead_time_days"`
	PermitValidityDays int16               `json:"permitValidityDays" keel:"permit_validity_days"`
	PermitBaseFee      decimal.NullDecimal `json:"permitBaseFee" keel:"permit_base_fee"`
	PermitPerMileFee   decimal.NullDecimal `json:"permitPerMileFee" keel:"permit_per_mile_fee"`

	SourceNote        string            `json:"sourceNote" keel:"source_note,nullzero"`
	SourceURL         string            `json:"sourceUrl" keel:"source_url,nullzero"`
	VerificationState VerificationState `json:"verificationState" keel:"verification_state"`
	VerifiedAt        *int64            `json:"verifiedAt" keel:"verified_at"`

	EffectiveStartDate *int64 `json:"effectiveStartDate" keel:"effective_start_date"`
	EffectiveEndDate   *int64 `json:"effectiveEndDate" keel:"effective_end_date"`

	Version   int64 `json:"version" keel:"version,version"`
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`

	State            *usstate.UsState   `json:"state,omitempty" keel:"belongs:state_id"`
	EscortThresholds []*EscortThreshold `json:"escortThresholds,omitempty" keel:"many:jurisdiction_rule_id"`
}

func (r *JurisdictionRule) BeforeUpdate(_ context.Context) error {
	if r.Status == "" {
		r.Status = StatusActive
	}
	if r.VerificationState == "" {
		r.VerificationState = VerificationUnverified
	}

	return nil
}

func (r *JurisdictionRule) GetID() pulid.ID { return r.ID }

func (r *JurisdictionRule) GetCreatedAt() int64 { return r.CreatedAt }

func (r *JurisdictionRule) GetTableName() string { return "jurisdiction_rules" }

func (r *JurisdictionRule) IsEffectiveAt(timestamp int64) bool {
	if r.EffectiveStartDate != nil && timestamp < *r.EffectiveStartDate {
		return false
	}
	if r.EffectiveEndDate != nil && timestamp > *r.EffectiveEndDate {
		return false
	}
	return true
}

func (r *JurisdictionRule) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(r,
		validation.Field(&r.StateID, validation.Required.Error("State is required")),
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[Status]("Status is invalid"),
		),
		validation.Field(&r.VerificationState,
			validation.Required.Error("Verification state is required"),
			domainvalidation.ValidEnum[VerificationState]("Verification state is invalid"),
		),
	))

	r.validateLimits(multiErr)
	r.validatePermitTerms(multiErr)
	r.validateEffectiveDates(multiErr)

	for idx, threshold := range r.EscortThresholds {
		if threshold != nil {
			threshold.Validate(multiErr.WithIndex("escortThresholds", idx))
		}
	}
}

func (r *JurisdictionRule) validateLimits(multiErr *errortypes.MultiError) {
	if r.MaxWidthFeet <= 0 || r.MaxWidthFeet > MaxReasonableWidthFeet {
		multiErr.Add("maxWidthFeet", errortypes.ErrInvalid,
			"Maximum width must be between 0 and 60 feet")
	}
	if r.MaxHeightFeet <= 0 || r.MaxHeightFeet > MaxReasonableHeightFeet {
		multiErr.Add("maxHeightFeet", errortypes.ErrInvalid,
			"Maximum height must be between 0 and 40 feet")
	}
	if r.MaxLengthFeet <= 0 || r.MaxLengthFeet > MaxReasonableLengthFeet {
		multiErr.Add("maxLengthFeet", errortypes.ErrInvalid,
			"Maximum length must be between 0 and 300 feet")
	}
	if r.MaxWeightPounds <= 0 {
		multiErr.Add("maxWeightPounds", errortypes.ErrInvalid,
			"Maximum weight must be greater than zero")
	}

	if r.SuperloadWidthFeet != nil && *r.SuperloadWidthFeet <= r.MaxWidthFeet {
		multiErr.Add("superloadWidthFeet", errortypes.ErrInvalid,
			"Superload width must be greater than the permitted maximum width")
	}
	if r.SuperloadWeightPounds != nil && *r.SuperloadWeightPounds <= r.MaxWeightPounds {
		multiErr.Add("superloadWeightPounds", errortypes.ErrInvalid,
			"Superload weight must be greater than the permitted maximum weight")
	}
}

func (r *JurisdictionRule) validatePermitTerms(multiErr *errortypes.MultiError) {
	if r.PermitLeadTimeDays < 0 || r.PermitLeadTimeDays > MaxLeadTimeDays {
		multiErr.Add("permitLeadTimeDays", errortypes.ErrInvalid,
			"Permit lead time must be between 0 and 60 days")
	}
	if r.PermitValidityDays <= 0 || r.PermitValidityDays > MaxValidityDays {
		multiErr.Add("permitValidityDays", errortypes.ErrInvalid,
			"Permit validity must be between 1 and 365 days")
	}
	if r.PermitBaseFee.Valid && r.PermitBaseFee.Decimal.IsNegative() {
		multiErr.Add("permitBaseFee", errortypes.ErrInvalid,
			"Permit base fee cannot be negative")
	}
	if r.PermitPerMileFee.Valid && r.PermitPerMileFee.Decimal.IsNegative() {
		multiErr.Add("permitPerMileFee", errortypes.ErrInvalid,
			"Permit per mile fee cannot be negative")
	}
}

func (r *JurisdictionRule) validateEffectiveDates(multiErr *errortypes.MultiError) {
	if r.EffectiveStartDate == nil || r.EffectiveEndDate == nil {
		return
	}

	if *r.EffectiveEndDate <= *r.EffectiveStartDate {
		multiErr.Add("effectiveEndDate", errortypes.ErrInvalid,
			"Effective end date must be after the effective start date")
	}
}

func (r *JurisdictionRule) BeforeCreate(ctx context.Context) error {
	if r.Status == "" {
		r.Status = StatusActive
	}
	if r.PermitLeadTimeDays == 0 {
		r.PermitLeadTimeDays = 1
	}
	if r.PermitValidityDays == 0 {
		r.PermitValidityDays = 5
	}
	if r.VerificationState == "" {
		r.VerificationState = VerificationUnverified
	}

	return nil
}
