package workflow

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/emoss08/trenova/pkg/validator/framework"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook = (*Version)(nil)
	_ domain.Validatable        = (*Version)(nil)
	_ framework.TenantedEntity  = (*Version)(nil)
)

type Version struct {
	bun.BaseModel `bun:"table:workflow_versions,alias:wfv" json:"-"`

	ID                 pulid.ID       `json:"id"                 bun:"id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID     pulid.ID       `json:"businessUnitId"     bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID     pulid.ID       `json:"organizationId"     bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	WorkflowTemplateID pulid.ID       `json:"workflowTemplateId" bun:"workflow_template_id,type:VARCHAR(100),notnull"`
	VersionNumber      int            `json:"versionNumber"      bun:"version_number,type:INTEGER,notnull"`
	Name               string         `json:"name"               bun:"name,type:VARCHAR(255),notnull"`
	Description        string         `json:"description"        bun:"description,type:TEXT,nullzero"`
	TriggerType        TriggerType    `json:"triggerType"        bun:"trigger_type,type:workflow_trigger_type_enum,notnull,default:'Manual'"`
	Status             Status         `json:"status"             bun:"status,type:workflow_status_enum,notnull,default:'Draft'"`
	VersionStatus      VersionStatus  `json:"versionStatus"      bun:"version_status,type:workflow_version_status_enum,notnull,default:'Draft'"`
	ScheduleConfig     map[string]any `json:"scheduleConfig"     bun:"schedule_config,type:JSONB,default:'{}'"`
	TriggerConfig      map[string]any `json:"triggerConfig"      bun:"trigger_config,type:JSONB,default:'{}'"`
	ChangeDescription  string         `json:"changeDescription"  bun:"change_description,type:TEXT,nullzero"`
	Version            int64          `json:"version"            bun:"version,type:BIGINT,notnull,default:0"`
	CreatedByID        pulid.ID       `json:"createdById"        bun:"created_by_id,type:VARCHAR(100),notnull"`
	CreatedAt          int64          `json:"createdAt"          bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt          int64          `json:"updatedAt"          bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Relations
	BusinessUnit     *tenant.BusinessUnit `json:"businessUnit,omitempty"     bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization     *tenant.Organization `json:"organization,omitempty"     bun:"rel:belongs-to,join:organization_id=id"`
	WorkflowTemplate *Template            `json:"workflowTemplate,omitempty" bun:"rel:belongs-to,join:workflow_template_id=id"`
	CreatedBy        *tenant.User         `json:"createdBy,omitempty"        bun:"rel:belongs-to,join:created_by_id=id"`
	Nodes            []*Node              `json:"nodes,omitempty"            bun:"rel:has-many,join:id=workflow_version_id"`
	Connections      []*Connection        `json:"connections,omitempty"      bun:"rel:has-many,join:id=workflow_version_id"`
	Instances        []*Instance          `json:"instances,omitempty"        bun:"rel:has-many,join:id=workflow_version_id"`
}

func (v *Version) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(v,
		validation.Field(&v.WorkflowTemplateID,
			validation.Required.Error("Workflow Template ID is required"),
		),
		validation.Field(&v.VersionNumber,
			validation.Required.Error("Version Number is required"),
			validation.Min(1).Error("Version Number must be at least 1"),
		),
		validation.Field(&v.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 255).Error("Name must be between 1 and 255 characters"),
		),
		validation.Field(&v.TriggerType,
			validation.Required.Error("Trigger Type is required"),
			validation.In(TriggerTypeManual, TriggerTypeScheduled, TriggerTypeEvent).
				Error("Trigger Type must be a valid trigger type"),
		),
		validation.Field(&v.Status,
			validation.Required.Error("Status is required"),
			validation.In(StatusActive, StatusInactive, StatusDraft).
				Error("Status must be a valid workflow status"),
		),
		validation.Field(&v.VersionStatus,
			validation.Required.Error("Version Status is required"),
			validation.In(VersionStatusDraft, VersionStatusPublished, VersionStatusArchived).
				Error("Version Status must be a valid version status"),
		),
		validation.Field(&v.ScheduleConfig,
			validation.When(
				v.TriggerType == TriggerTypeScheduled,
				validation.Required.Error(
					"Schedule Config is required when trigger type is Scheduled",
				),
			),
		),
		validation.Field(&v.TriggerConfig,
			validation.When(
				v.TriggerType == TriggerTypeEvent,
				validation.Required.Error("Trigger Config is required when trigger type is Event"),
			),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (v *Version) GetID() string {
	return v.ID.String()
}

func (v *Version) GetOrganizationID() pulid.ID {
	return v.OrganizationID
}

func (v *Version) GetBusinessUnitID() pulid.ID {
	return v.BusinessUnitID
}

func (v *Version) GetTableName() string {
	return "workflow_versions"
}

func (v *Version) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := utils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if v.ID.IsNil() {
			v.ID = pulid.MustNew("wfv_")
		}
		v.CreatedAt = now
		v.UpdatedAt = now
	case *bun.UpdateQuery:
		v.UpdatedAt = now
	}

	return nil
}

func (v *Version) IsActive() bool {
	return v.Status == StatusActive
}

func (v *Version) IsDraft() bool {
	return v.Status == StatusDraft
}

func (v *Version) IsScheduled() bool {
	return v.TriggerType == TriggerTypeScheduled
}

func (v *Version) IsEventTriggered() bool {
	return v.TriggerType == TriggerTypeEvent
}

func (v *Version) IsPublished() bool {
	return v.VersionStatus == VersionStatusPublished
}

func (v *Version) IsArchived() bool {
	return v.VersionStatus == VersionStatusArchived
}

func (v *Version) CanEdit() bool {
	return v.VersionStatus == VersionStatusDraft
}

func (v *Version) HasNodes() bool {
	return len(v.Nodes) > 0
}

func (v *Version) HasConnections() bool {
	return len(v.Connections) > 0
}
