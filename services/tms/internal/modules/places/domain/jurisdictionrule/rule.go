package jurisdictionrule

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
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

var _ bun.BeforeAppendModelHook = (*JurisdictionRule)(nil)

// JurisdictionRule is global reference data rather than tenant data: the legal
// width of a load in Nebraska does not vary by carrier. Carriers express their
// own, stricter posture through JurisdictionOverride instead of editing this.
type JurisdictionRule struct {
	bun.BaseModel `bun:"table:jurisdiction_rules,alias:jr" json:"-"`
	keel.Table    `keel:"jurisdiction_rules"`

	ID      pulid.ID `json:"id"      bun:"id,pk,type:VARCHAR(100),notnull" keel:"id,pk,autoid:jr_"`
	StateID pulid.ID `json:"stateId" bun:"state_id,type:VARCHAR(100),notnull" keel:"state_id"`
	Status  Status   `json:"status"  bun:"status,type:jurisdiction_rule_status_enum,notnull,default:'Active'" keel:"status"`

	MaxWidthFeet    float64 `json:"maxWidthFeet"    bun:"max_width_feet,type:NUMERIC(10,2),notnull" keel:"max_width_feet"`
	MaxHeightFeet   float64 `json:"maxHeightFeet"   bun:"max_height_feet,type:NUMERIC(10,2),notnull" keel:"max_height_feet"`
	MaxLengthFeet   float64 `json:"maxLengthFeet"   bun:"max_length_feet,type:NUMERIC(10,2),notnull" keel:"max_length_feet"`
	MaxWeightPounds int64   `json:"maxWeightPounds" bun:"max_weight_pounds,type:BIGINT,notnull" keel:"max_weight_pounds"`

	SuperloadWidthFeet    *float64 `json:"superloadWidthFeet"    bun:"superload_width_feet,type:NUMERIC(10,2),nullzero" keel:"superload_width_feet"`
	SuperloadWeightPounds *int64   `json:"superloadWeightPounds" bun:"superload_weight_pounds,type:BIGINT,nullzero" keel:"superload_weight_pounds"`

	DaylightOnly       bool `json:"daylightOnly"       bun:"daylight_only,type:BOOLEAN,notnull,default:false" keel:"daylight_only"`
	RushHourRestricted bool `json:"rushHourRestricted" bun:"rush_hour_restricted,type:BOOLEAN,notnull,default:false" keel:"rush_hour_restricted"`
	WeekendRestricted  bool `json:"weekendRestricted"  bun:"weekend_restricted,type:BOOLEAN,notnull,default:false" keel:"weekend_restricted"`
	HolidayRestricted  bool `json:"holidayRestricted"  bun:"holiday_restricted,type:BOOLEAN,notnull" keel:"holiday_restricted"`

	PermitLeadTimeDays int16               `json:"permitLeadTimeDays" bun:"permit_lead_time_days,type:SMALLINT,notnull,default:1" keel:"permit_lead_time_days"`
	PermitValidityDays int16               `json:"permitValidityDays" bun:"permit_validity_days,type:SMALLINT,notnull,default:5" keel:"permit_validity_days"`
	PermitBaseFee      decimal.NullDecimal `json:"permitBaseFee"      bun:"permit_base_fee,type:NUMERIC(19,4),nullzero" keel:"permit_base_fee"`
	PermitPerMileFee   decimal.NullDecimal `json:"permitPerMileFee"   bun:"permit_per_mile_fee,type:NUMERIC(19,4),nullzero" keel:"permit_per_mile_fee"`

	SourceNote        string            `json:"sourceNote"        bun:"source_note,type:TEXT,nullzero" keel:"source_note,nullzero"`
	SourceURL         string            `json:"sourceUrl"         bun:"source_url,type:TEXT,nullzero" keel:"source_url,nullzero"`
	VerificationState VerificationState `json:"verificationState" bun:"verification_state,type:jurisdiction_verification_state_enum,notnull,default:'Unverified'" keel:"verification_state"`
	VerifiedAt        *int64            `json:"verifiedAt"        bun:"verified_at,type:BIGINT,nullzero" keel:"verified_at"`

	EffectiveStartDate *int64 `json:"effectiveStartDate" bun:"effective_start_date,type:BIGINT,nullzero" keel:"effective_start_date"`
	EffectiveEndDate   *int64 `json:"effectiveEndDate"   bun:"effective_end_date,type:BIGINT,nullzero" keel:"effective_end_date"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT" keel:"version,version"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate"`

	State            *usstate.UsState   `json:"state,omitempty"            bun:"rel:belongs-to,join:state_id=id" keel:"belongs:state_id"`
	EscortThresholds []*EscortThreshold `json:"escortThresholds,omitempty" bun:"rel:has-many,join:id=jurisdiction_rule_id" keel:"many:jurisdiction_rule_id"`
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

func (r *JurisdictionRule) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	if r.Status == "" {
		r.Status = StatusActive
	}
	if r.VerificationState == "" {
		r.VerificationState = VerificationUnverified
	}

	switch query.(type) {
	case *bun.InsertQuery:
		if r.ID.IsNil() {
			r.ID = pulid.MustNew("jr_")
		}
		r.CreatedAt = now
		r.UpdatedAt = now
	case *bun.UpdateQuery:
		r.UpdatedAt = now
	}

	return nil
}

func (r *JurisdictionRule) BeforeCreate(ctx context.Context) error {
	return r.BeforeAppendModel(ctx, (*bun.InsertQuery)(nil))
}
