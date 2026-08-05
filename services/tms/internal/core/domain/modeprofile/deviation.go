package modeprofile

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

const MinAcknowledgementReasonLength = 10

var (
	_ bun.BeforeAppendModelHook          = (*Deviation)(nil)
	_ validationframework.TenantedEntity = (*Deviation)(nil)
	_ domaintypes.PostgresSearchable     = (*Deviation)(nil)
)

type Deviation struct {
	bun.BaseModel             `bun:"table:mode_profile_deviations,alias:mpd" json:"-"`
	pagination.CursorValueSet `bun:",embed"                                 json:"-"`

	ID               pulid.ID  `json:"id"               bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID   pulid.ID  `json:"businessUnitId"   bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID   pulid.ID  `json:"organizationId"   bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	ModeProfileID    pulid.ID  `json:"modeProfileId"    bun:"mode_profile_id,type:VARCHAR(100),notnull"`
	CapabilityRuleID *pulid.ID `json:"capabilityRuleId" bun:"capability_rule_id,type:VARCHAR(100),nullzero"`

	RuleKey     RuleKey                 `json:"ruleKey"     bun:"rule_key,type:VARCHAR(100),notnull"`
	Capability  Capability              `json:"capability"  bun:"capability,type:VARCHAR(100),notnull"`
	Enforcement tenant.EnforcementLevel `json:"enforcement" bun:"enforcement,type:enforcement_level_enum,notnull"`

	ResourceType string   `json:"resourceType" bun:"resource_type,type:VARCHAR(100),notnull"`
	ResourceID   pulid.ID `json:"resourceId"   bun:"resource_id,type:VARCHAR(100),notnull"`

	Field   string `json:"field"   bun:"field,type:VARCHAR(200),nullzero"`
	Message string `json:"message" bun:"message,type:TEXT,notnull"`

	State      DeviationState `json:"state"      bun:"state,type:mode_profile_deviation_state_enum,notnull,default:'Open'"`
	OccurredAt int64          `json:"occurredAt" bun:"occurred_at,type:BIGINT,notnull"`

	AcknowledgedByID      *pulid.ID `json:"acknowledgedById"      bun:"acknowledged_by_id,type:VARCHAR(100),nullzero"`
	AcknowledgedAt        *int64    `json:"acknowledgedAt"        bun:"acknowledged_at,type:BIGINT,nullzero"`
	AcknowledgementReason string    `json:"acknowledgementReason" bun:"acknowledgement_reason,type:TEXT,nullzero"`

	Provenance *Provenance `json:"provenance" bun:"provenance,type:JSONB,nullzero"`

	Version      int64  `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt    int64  `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt    int64  `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	SearchVector string `json:"-"         bun:"search_vector,type:TSVECTOR,scanonly"`
	Rank         string `json:"-"         bun:"rank,type:VARCHAR(100),scanonly"`

	BusinessUnit   *tenant.BusinessUnit `json:"-"                        bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization   *tenant.Organization `json:"-"                        bun:"rel:belongs-to,join:organization_id=id"`
	Profile        *Profile             `json:"profile,omitempty"        bun:"rel:belongs-to,join:mode_profile_id=id"`
	AcknowledgedBy *tenant.User         `json:"acknowledgedBy,omitempty" bun:"rel:belongs-to,join:acknowledged_by_id=id"`
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

func (d *Deviation) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	if d.State == "" {
		d.State = DeviationStateOpen
	}
	if d.OccurredAt == 0 {
		d.OccurredAt = now
	}
	if d.Capability == "" {
		if def, err := RuleDefinitionFor(d.RuleKey); err == nil {
			d.Capability = def.Capability
		}
	}

	switch query.(type) {
	case *bun.InsertQuery:
		if d.ID.IsNil() {
			d.ID = pulid.MustNew("mpd_")
		}
		d.CreatedAt = now
		d.UpdatedAt = now
	case *bun.UpdateQuery:
		d.UpdatedAt = now
	}

	return nil
}
