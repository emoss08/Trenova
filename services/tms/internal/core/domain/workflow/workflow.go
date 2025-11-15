package workflow

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/emoss08/trenova/pkg/validator/framework"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook      = (*Workflow)(nil)
	_ domaintypes.PostgresSearchable = (*Workflow)(nil)
	_ domain.Validatable             = (*Workflow)(nil)
	_ framework.TenantedEntity       = (*Workflow)(nil)
)

// Workflow represents a workflow definition
type Workflow struct {
	bun.BaseModel `bun:"table:workflows,alias:wf" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,notnull,pk,type:VARCHAR(100)"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,notnull,pk,type:VARCHAR(100)"`

	// Basic Info
	Name        string         `json:"name"        bun:"name,notnull,type:VARCHAR(255)"`
	Description string         `json:"description" bun:"description,type:TEXT"`
	Status      WorkflowStatus `json:"status"      bun:"status,type:workflow_status_enum,default:'draft'"`

	// Trigger Configuration
	TriggerType   TriggerType    `json:"triggerType"   bun:"trigger_type,type:workflow_trigger_type_enum,notnull"`
	TriggerConfig map[string]any `json:"triggerConfig" bun:"trigger_config,type:jsonb,default:'{}'"`

	// Versioning
	CurrentVersionID   *pulid.ID `json:"currentVersionId"   bun:"current_version_id,type:VARCHAR(100),nullzero"`
	PublishedVersionID *pulid.ID `json:"publishedVersionId" bun:"published_version_id,type:VARCHAR(100),nullzero"`

	// Execution Settings
	TimeoutSeconds      int  `json:"timeoutSeconds"      bun:"timeout_seconds,default:300"`
	MaxRetries          int  `json:"maxRetries"          bun:"max_retries,default:3"`
	RetryDelaySeconds   int  `json:"retryDelaySeconds"   bun:"retry_delay_seconds,default:60"`
	EnableLogging       bool `json:"enableLogging"       bun:"enable_logging,type:BOOLEAN,default:true"`
	EnableNotifications bool `json:"enableNotifications" bun:"enable_notifications,type:BOOLEAN,default:false"`

	// Permissions
	CreatedBy pulid.ID  `json:"createdBy" bun:"created_by,notnull,type:VARCHAR(100)"`
	UpdatedBy *pulid.ID `json:"updatedBy" bun:"updated_by,type:VARCHAR(100),nullzero"`

	// Tags and Categories
	Tags     []string `json:"tags"     bun:"tags,type:TEXT[],array,default:'{}'"`
	Category string   `json:"category" bun:"category,type:VARCHAR(100)"`

	// Metadata
	Version   int64 `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Relationships
	BusinessUnit     *tenant.BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id"     json:"-"`
	Organization     *tenant.Organization `bun:"rel:belongs-to,join:organization_id=id"      json:"-"`
	CurrentVersion   *WorkflowVersion     `bun:"rel:belongs-to,join:current_version_id=id"   json:"currentVersion,omitempty"`
	PublishedVersion *WorkflowVersion     `bun:"rel:belongs-to,join:published_version_id=id" json:"publishedVersion,omitempty"`
	Versions         []*WorkflowVersion   `bun:"rel:has-many,join:id=workflow_id"            json:"versions,omitempty"`
}

func (w *Workflow) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(w,
		validation.Field(&w.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 255).Error("Name must be between 1 and 255 characters"),
		),
		validation.Field(&w.TriggerType,
			validation.Required.Error("Trigger type is required"),
			validation.In(
				TriggerTypeManual,
				TriggerTypeScheduled,
				TriggerTypeShipmentStatus,
				TriggerTypeDocumentUploaded,
				TriggerTypeEntityCreated,
				TriggerTypeEntityUpdated,
				TriggerTypeWebhook,
			).Error("Invalid trigger type"),
		),
		validation.Field(&w.TimeoutSeconds,
			validation.Min(1).Error("Timeout must be at least 1 second"),
			validation.Max(86400).Error("Timeout cannot exceed 24 hours"),
		),
		validation.Field(&w.MaxRetries,
			validation.Min(0).Error("Max retries cannot be negative"),
			validation.Max(10).Error("Max retries cannot exceed 10"),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (w *Workflow) GetID() string {
	return w.ID.String()
}

func (w *Workflow) GetTableName() string {
	return "workflows"
}

func (w *Workflow) GetOrganizationID() pulid.ID {
	return w.OrganizationID
}

func (w *Workflow) GetBusinessUnitID() pulid.ID {
	return w.BusinessUnitID
}

func (w *Workflow) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "wf",
		UseSearchVector: false,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "name", Weight: domaintypes.SearchWeightA, Type: domaintypes.FieldTypeText},
			{
				Name:   "description",
				Weight: domaintypes.SearchWeightB,
				Type:   domaintypes.FieldTypeText,
			},
			{Name: "category", Weight: domaintypes.SearchWeightC, Type: domaintypes.FieldTypeText},
		},
	}
}

func (w *Workflow) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := utils.NowUnix()
	switch query.(type) {
	case *bun.InsertQuery:
		if w.ID.IsNil() {
			w.ID = pulid.MustNew("wf_")
		}
		w.CreatedAt = now
		w.UpdatedAt = now
	case *bun.UpdateQuery:
		w.UpdatedAt = now
	}
	return nil
}

// IsPublished checks if the workflow has a published version
func (w *Workflow) IsPublished() bool {
	return w.PublishedVersionID != nil && !w.PublishedVersionID.IsNil()
}

// CanExecute checks if the workflow can be executed
func (w *Workflow) CanExecute() bool {
	return w.Status == WorkflowStatusActive && w.IsPublished()
}

// IsEditable checks if the workflow can be edited
func (w *Workflow) IsEditable() bool {
	return w.Status != WorkflowStatusArchived
}
