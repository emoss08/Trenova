package jurisdictionrule

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*Override)(nil)
	_ validationframework.TenantedEntity = (*Override)(nil)
)

// Override lets a carrier run a tighter posture than the statutory baseline
// without editing shared reference data. Every field is optional: an unset
// field means "defer to the jurisdiction rule".
type Override struct {
	bun.BaseModel `bun:"table:jurisdiction_rule_overrides,alias:jro" json:"-"`
	keel.Table    `keel:"jurisdiction_rule_overrides"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull" keel:"id,pk,autoid:jro_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull" keel:"organization_id,pk,tenant"`
	StateID        pulid.ID `json:"stateId"        bun:"state_id,type:VARCHAR(100),notnull" keel:"state_id"`

	MaxWidthFeet    *float64 `json:"maxWidthFeet"    bun:"max_width_feet,type:NUMERIC(10,2),nullzero" keel:"max_width_feet"`
	MaxHeightFeet   *float64 `json:"maxHeightFeet"   bun:"max_height_feet,type:NUMERIC(10,2),nullzero" keel:"max_height_feet"`
	MaxLengthFeet   *float64 `json:"maxLengthFeet"   bun:"max_length_feet,type:NUMERIC(10,2),nullzero" keel:"max_length_feet"`
	MaxWeightPounds *int64   `json:"maxWeightPounds" bun:"max_weight_pounds,type:BIGINT,nullzero" keel:"max_weight_pounds"`

	PermitLeadTimeDays *int16 `json:"permitLeadTimeDays" bun:"permit_lead_time_days,type:SMALLINT,nullzero" keel:"permit_lead_time_days"`

	DaylightOnly      *bool `json:"daylightOnly"      bun:"daylight_only,type:BOOLEAN,nullzero" keel:"daylight_only"`
	HolidayRestricted *bool `json:"holidayRestricted" bun:"holiday_restricted,type:BOOLEAN,nullzero" keel:"holiday_restricted"`

	Reason string `json:"reason" bun:"reason,type:TEXT,notnull" keel:"reason"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT" keel:"version,version"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate"`

	BusinessUnit *tenant.BusinessUnit `json:"-"               bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *tenant.Organization `json:"-"               bun:"rel:belongs-to,join:organization_id=id"`
	State        *usstate.UsState     `json:"state,omitempty" bun:"rel:belongs-to,join:state_id=id" keel:"belongs:state_id"`
}

func (o *Override) GetID() pulid.ID { return o.ID }

func (o *Override) GetCreatedAt() int64 { return o.CreatedAt }

func (o *Override) GetOrganizationID() pulid.ID { return o.OrganizationID }

func (o *Override) GetBusinessUnitID() pulid.ID { return o.BusinessUnitID }

func (o *Override) GetTableName() string { return "jurisdiction_rule_overrides" }

func (o *Override) HasAnyOverride() bool {
	return o.MaxWidthFeet != nil ||
		o.MaxHeightFeet != nil ||
		o.MaxLengthFeet != nil ||
		o.MaxWeightPounds != nil ||
		o.PermitLeadTimeDays != nil ||
		o.DaylightOnly != nil ||
		o.HolidayRestricted != nil
}

func (o *Override) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(o,
		validation.Field(&o.StateID, validation.Required.Error("State is required")),
		validation.Field(&o.Reason,
			validation.Required.Error("Explain why this jurisdiction is overridden"),
			validation.Length(10, 0).
				Error("Provide at least 10 characters explaining the override"),
		),
	))

	if !o.HasAnyOverride() {
		multiErr.Add("stateId", errortypes.ErrInvalid,
			"An override must change at least one limit or restriction")
	}

	o.validatePositive(multiErr)
}

func (o *Override) validatePositive(multiErr *errortypes.MultiError) {
	check := func(field string, value *float64) {
		if value != nil && *value <= 0 {
			multiErr.Add(field, errortypes.ErrInvalid, "Value must be greater than zero")
		}
	}

	check("maxWidthFeet", o.MaxWidthFeet)
	check("maxHeightFeet", o.MaxHeightFeet)
	check("maxLengthFeet", o.MaxLengthFeet)

	if o.MaxWeightPounds != nil && *o.MaxWeightPounds <= 0 {
		multiErr.Add("maxWeightPounds", errortypes.ErrInvalid,
			"Value must be greater than zero")
	}

	if o.PermitLeadTimeDays != nil &&
		(*o.PermitLeadTimeDays < 0 || *o.PermitLeadTimeDays > MaxLeadTimeDays) {
		multiErr.Add("permitLeadTimeDays", errortypes.ErrInvalid,
			"Permit lead time must be between 0 and 60 days")
	}
}

func (o *Override) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if o.ID.IsNil() {
			o.ID = pulid.MustNew("jro_")
		}
		o.CreatedAt = now
		o.UpdatedAt = now
	case *bun.UpdateQuery:
		o.UpdatedAt = now
	}

	return nil
}
