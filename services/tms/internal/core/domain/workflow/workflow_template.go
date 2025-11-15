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
	_ bun.BeforeAppendModelHook = (*WorkflowTemplate)(nil)
	_ domain.Validatable        = (*WorkflowTemplate)(nil)
	_ framework.TenantedEntity  = (*WorkflowTemplate)(nil)
)

// WorkflowTemplate represents a reusable workflow template
type WorkflowTemplate struct {
	bun.BaseModel `bun:"table:workflow_templates,alias:wft" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,notnull,pk,type:VARCHAR(100)"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,notnull,pk,type:VARCHAR(100)"`

	// Template Info
	Name        string   `json:"name"        bun:"name,notnull,type:VARCHAR(255)"`
	Description string   `json:"description" bun:"description,type:TEXT"`
	Category    string   `json:"category"    bun:"category,type:VARCHAR(100)"`
	Tags        []string `json:"tags"        bun:"tags,type:TEXT[],array,default:'{}'"`

	// Template Definition
	TemplateDefinition utils.JSONB `json:"templateDefinition" bun:"template_definition,type:jsonb,notnull"`

	// Visibility
	IsSystemTemplate bool `json:"isSystemTemplate" bun:"is_system_template,type:BOOLEAN,default:false"` // System-wide templates
	IsPublic         bool `json:"isPublic"         bun:"is_public,type:BOOLEAN,default:false"`

	// Usage Stats
	UsageCount int `json:"usageCount" bun:"usage_count,default:0"`

	// Metadata
	CreatedBy pulid.ID `json:"createdBy" bun:"created_by,notnull,type:VARCHAR(100)"`
	Version   int64    `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt int64    `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64    `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Relationships
	BusinessUnit *tenant.BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization *tenant.Organization `bun:"rel:belongs-to,join:organization_id=id"  json:"-"`
}

func (wt *WorkflowTemplate) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(wt,
		validation.Field(&wt.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 255).Error("Name must be between 1 and 255 characters"),
		),
		validation.Field(&wt.TemplateDefinition,
			validation.Required.Error("Template definition is required"),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (wt *WorkflowTemplate) GetID() string {
	return wt.ID.String()
}

func (wt *WorkflowTemplate) GetTableName() string {
	return "workflow_templates"
}

func (wt *WorkflowTemplate) GetOrganizationID() pulid.ID {
	return wt.OrganizationID
}

func (wt *WorkflowTemplate) GetBusinessUnitID() pulid.ID {
	return wt.BusinessUnitID
}

func (wt *WorkflowTemplate) BeforeAppendModel(_ context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		if wt.ID.IsNil() {
			wt.ID = pulid.MustNew("wft_")
		}
	case *bun.UpdateQuery:
		wt.Version++
	}
	return nil
}

// IncrementUsage increments the usage count of the template
func (wt *WorkflowTemplate) IncrementUsage() {
	wt.UsageCount++
}
