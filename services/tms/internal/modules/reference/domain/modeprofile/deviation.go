package modeprofile

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

const MinAcknowledgementReasonLength = 10

var (
	_ validationframework.TenantedEntity = (*Deviation)(nil)
	_ domaintypes.PostgresSearchable     = (*Deviation)(nil)
)

type Deviation struct {
	keel.Table       `                                                       keel:"mode_profile_deviations"`
	ID               pulid.ID  `json:"id" keel:"id,pk,autoid:mpd_"`
	BusinessUnitID   pulid.ID  `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID   pulid.ID  `json:"organizationId" keel:"organization_id,pk,tenant"`
	ModeProfileID    pulid.ID  `json:"modeProfileId" keel:"mode_profile_id"`
	CapabilityRuleID *pulid.ID `json:"capabilityRuleId" keel:"capability_rule_id"`

	RuleKey     RuleKey                 `json:"ruleKey" keel:"rule_key"`
	Capability  Capability              `json:"capability" keel:"capability"`
	Enforcement tenant.EnforcementLevel `json:"enforcement" keel:"enforcement"`

	ResourceType string   `json:"resourceType" keel:"resource_type"`
	ResourceID   pulid.ID `json:"resourceId" keel:"resource_id"`

	Field   string `json:"field" keel:"field,nullzero"`
	Message string `json:"message" keel:"message"`

	State      DeviationState `json:"state" keel:"state"`
	OccurredAt int64          `json:"occurredAt" keel:"occurred_at"`

	AcknowledgedByID      *pulid.ID `json:"acknowledgedById" keel:"acknowledged_by_id"`
	AcknowledgedAt        *int64    `json:"acknowledgedAt" keel:"acknowledged_at"`
	AcknowledgementReason string    `json:"acknowledgementReason" keel:"acknowledgement_reason,nullzero"`

	Provenance *Provenance `json:"provenance" keel:"provenance,jsonb"`

	Version      int64  `json:"version" keel:"version,version"`
	CreatedAt    int64  `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt    int64  `json:"updatedAt" keel:"updated_at,autoupdate"`
	SearchVector string `json:"-" keel:"search_vector,tsvector,readonly,noselect"`
	Rank         string `json:"-" keel:"rank,computed"`

	BusinessUnit   *tenant.BusinessUnit `json:"-"`
	Organization   *tenant.Organization `json:"-"`
	Profile        *Profile             `json:"profile,omitempty" keel:"belongs:mode_profile_id"`
	AcknowledgedBy *tenant.User         `json:"acknowledgedBy,omitempty" keel:"belongs:acknowledged_by_id"`
}

func (d *Deviation) BeforeUpdate(_ context.Context) error {
	d.applyDefaults()

	return nil
}

func (d *Deviation) BeforeCreate(_ context.Context) error {
	d.applyDefaults()
	if d.State == "" {
		d.State = DeviationStateOpen
	}

	return nil
}

func (d *Deviation) GetID() pulid.ID { return d.ID }

func (d *Deviation) GetCreatedAt() int64 { return d.CreatedAt }

func (d *Deviation) GetOrganizationID() pulid.ID { return d.OrganizationID }

func (d *Deviation) GetBusinessUnitID() pulid.ID { return d.BusinessUnitID }

func (d *Deviation) GetTableName() string { return "mode_profile_deviations" }

func (d *Deviation) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "mpd",
		UseSearchVector: true,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "rule_key", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{Name: "message", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightB},
			{
				Name:   "acknowledgement_reason",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightC,
			},
		},
	}
}

func (d *Deviation) RequiresAcknowledgement() bool {
	return d.State == DeviationStateOpen
}

func (d *Deviation) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(d,
		validation.Field(&d.RuleKey,
			validation.Required.Error("Rule key is required"),
		),
		validation.Field(&d.ResourceType,
			validation.Required.Error("Resource type is required"),
		),
		validation.Field(&d.ResourceID,
			validation.Required.Error("Resource is required"),
		),
		validation.Field(&d.Message,
			validation.Required.Error("Message is required"),
		),
		validation.Field(&d.State,
			validation.Required.Error("State is required"),
			domainvalidation.ValidEnum[DeviationState]("State is invalid"),
		),
		validation.Field(&d.Enforcement,
			validation.Required.Error("Enforcement level is required"),
			validation.In(
				tenant.EnforcementLevelWarn,
				tenant.EnforcementLevelRequireReview,
			).Error("Only warn and review level results are recorded as deviations"),
		),
	))

	d.validateAcknowledgement(multiErr)
}

func (d *Deviation) validateAcknowledgement(multiErr *errortypes.MultiError) {
	if d.State == DeviationStateOpen {
		return
	}

	if d.AcknowledgedByID == nil || d.AcknowledgedByID.IsNil() {
		multiErr.Add("acknowledgedById", errortypes.ErrRequired,
			"An acknowledging user is required once a deviation leaves the open state")
	}

	if len(d.AcknowledgementReason) < MinAcknowledgementReasonLength {
		multiErr.Add("acknowledgementReason", errortypes.ErrInvalid,
			"Provide a reason of at least 10 characters explaining why this deviation is accepted")
	}
}

func (d *Deviation) applyDefaults() {
	if d.State == "" {
		d.State = DeviationStateOpen
	}
	if d.OccurredAt == 0 {
		d.OccurredAt = timeutils.NowUnix()
	}
	if d.Capability == "" {
		if def, err := RuleDefinitionFor(d.RuleKey); err == nil {
			d.Capability = def.Capability
		}
	}
}

func (d *Deviation) BeforeSave(_ context.Context) error {
	d.applyDefaults()
	return nil
}
