package jurisdictionrule

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var (
	_ validationframework.TenantedEntity = (*Override)(nil)
)

// Override lets a carrier run a tighter posture than the statutory baseline
// without editing shared reference data. Every field is optional: an unset
// field means "defer to the jurisdiction rule".
type Override struct {
	keel.Table `keel:"jurisdiction_rule_overrides"`

	ID             pulid.ID `json:"id" keel:"id,pk,autoid:jro_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	StateID        pulid.ID `json:"stateId" keel:"state_id"`

	MaxWidthFeet    *float64 `json:"maxWidthFeet" keel:"max_width_feet"`
	MaxHeightFeet   *float64 `json:"maxHeightFeet" keel:"max_height_feet"`
	MaxLengthFeet   *float64 `json:"maxLengthFeet" keel:"max_length_feet"`
	MaxWeightPounds *int64   `json:"maxWeightPounds" keel:"max_weight_pounds"`

	PermitLeadTimeDays *int16 `json:"permitLeadTimeDays" keel:"permit_lead_time_days"`

	DaylightOnly      *bool `json:"daylightOnly" keel:"daylight_only"`
	HolidayRestricted *bool `json:"holidayRestricted" keel:"holiday_restricted"`

	Reason string `json:"reason" keel:"reason"`

	Version   int64 `json:"version" keel:"version,version"`
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`

	BusinessUnit *tenant.BusinessUnit `json:"-"`
	Organization *tenant.Organization `json:"-"`
	State        *usstate.UsState     `json:"state,omitempty" keel:"belongs:state_id"`
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
