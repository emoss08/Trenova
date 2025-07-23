// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package email

import (
	"context"
	"regexp"
	"time"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/core/ports/infra"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/rotisserie/eris"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook = (*Template)(nil)
	_ domain.Validatable        = (*Template)(nil)
)

// Template represents an email template
type Template struct {
	bun.BaseModel `bun:"table:email_templates,alias:et" json:"-"`

	ID              pulid.ID         `json:"id"              bun:"id,type:varchar(255),pk,notnull"`
	BusinessUnitID  pulid.ID         `json:"businessUnitId"  bun:"business_unit_id,type:varchar(255),pk,notnull"`
	OrganizationID  pulid.ID         `json:"organizationId"  bun:"organization_id,type:varchar(255),pk,notnull"`
	Name            string           `json:"name"            bun:"name,type:varchar(255),notnull"`
	Slug            string           `json:"slug"            bun:"slug,type:varchar(255),notnull"`
	Description     string           `json:"description"     bun:"description,type:text"`
	Category        TemplateCategory `json:"category"        bun:"category,type:email_template_category_enum,notnull"`
	IsSystem        bool             `json:"isSystem"        bun:"is_system,type:boolean,default:false"`
	IsActive        bool             `json:"isActive"        bun:"is_active,type:boolean,default:true"`
	Status          domain.Status    `json:"status"          bun:"status,type:status_enum,default:'Active'"`
	SubjectTemplate string           `json:"subjectTemplate" bun:"subject_template,type:text,notnull"`
	HTMLTemplate    string           `json:"htmlTemplate"    bun:"html_template,type:text,notnull"`
	TextTemplate    string           `json:"textTemplate"    bun:"text_template,type:text"`
	VariablesSchema map[string]any   `json:"variablesSchema" bun:"variables_schema,type:jsonb"`
	Metadata        map[string]any   `json:"metadata"        bun:"metadata,type:jsonb"`
	SearchVector    string           `json:"-"               bun:"search_vector,type:tsvector,scanonly"`
	Rank            string           `json:"-"               bun:"rank,type:varchar(100),scanonly"`
	Version         int64            `json:"version"         bun:"version,type:BIGINT"`
	CreatedAt       int64            `json:"createdAt"       bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt       int64            `json:"updatedAt"       bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Relationships
	BusinessUnit *businessunit.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *organization.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
}

// Validate implements the Validatable interface
func (t *Template) Validate(ctx context.Context, multiErr *errors.MultiError) {
	err := validation.ValidateStructWithContext(ctx, t,
		// Basic fields validation
		validation.Field(&t.BusinessUnitID,
			validation.Required.Error("Business Unit is required"),
		),
		validation.Field(&t.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&t.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 255).Error("Name must be between 1 and 255 characters"),
		),
		validation.Field(
			&t.Slug,
			validation.Required.Error("Slug is required"),
			validation.Length(1, 255).Error("Slug must be between 1 and 255 characters"),
			validation.Match(regexp.MustCompile(`^[a-z0-9-]+$`)).
				Error("Slug must contain only lowercase letters, numbers, and hyphens"),
		),
		validation.Field(&t.Description,
			validation.Length(0, 1000).Error("Description must not exceed 1000 characters"),
		),
		validation.Field(&t.Category,
			validation.Required.Error("Category is required"),
			validation.In(
				TemplateCategoryNotification,
				TemplateCategoryMarketing,
				TemplateCategorySystem,
				TemplateCategoryCustom,
			).Error("Category must be a valid template category"),
		),
		validation.Field(
			&t.Status,
			validation.In(domain.StatusActive, domain.StatusInactive).
				Error("Status must be Active or Inactive"),
		),

		// Template content validation
		validation.Field(
			&t.SubjectTemplate,
			validation.Required.Error("Subject Template is required"),
			validation.Length(1, 500).
				Error("Subject Template must be between 1 and 500 characters"),
		),
		validation.Field(&t.HTMLTemplate,
			validation.Required.Error("HTML Template is required"),
			validation.Length(1, 1048576).Error("HTML Template must not exceed 1MB"),
		),
		validation.Field(&t.TextTemplate,
			validation.Length(0, 524288).Error("Text Template must not exceed 512KB"),
		),

		// System templates cannot be modified
		// TODO(wolfred): Move into validator
		// validation.Field(&t.IsSystem,
		// 	validation.When(
		// 		t.IsSystem && t.ID != "",
		// 		validation.By(func(value any) error {
		// 			// This would be checked in the service layer with database access
		// 			// to ensure system templates are not being modified
		// 			return nil
		// 		}),
		// 	),
		// ),
	)

	if err != nil {
		var validationErrs validation.Errors
		if eris.As(err, &validationErrs) {
			errors.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (t *Template) GetTableName() string {
	return "email_templates"
}

// GetPostgresSearchConfig implements infra.PostgresSearchable
func (t *Template) GetPostgresSearchConfig() infra.PostgresSearchConfig {
	return infra.PostgresSearchConfig{
		TableAlias: "et",
		Fields: []infra.PostgresSearchableField{
			{
				Name:   "name",
				Weight: "A",
				Type:   infra.PostgresSearchTypeText,
			},
			{
				Name:   "slug",
				Weight: "A",
				Type:   infra.PostgresSearchTypeText,
			},
			{
				Name:   "description",
				Weight: "B",
				Type:   infra.PostgresSearchTypeText,
			},
			{
				Name:       "category",
				Weight:     "C",
				Type:       infra.PostgresSearchTypeEnum,
				Dictionary: "english",
			},
		},
		MinLength:       2,
		MaxTerms:        6,
		UsePartialMatch: true,
	}
}

// BeforeAppendModel implements the bun.BeforeAppendModelHook interface
func (t *Template) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if t.ID.IsNil() {
			t.ID = pulid.MustNew("emt_")
		}
		t.CreatedAt = now
	case *bun.UpdateQuery:
		t.UpdatedAt = now
	}

	return nil
}

// GetSampleVariables returns sample variables for testing the template
func (t *Template) GetSampleVariables() map[string]any {
	if t.VariablesSchema == nil {
		return make(map[string]any)
	}

	// Generate sample data based on the schema
	samples := make(map[string]any)
	for key, schema := range t.VariablesSchema {
		if schemaMap, ok := schema.(map[string]any); ok {
			if varType, ok := schemaMap["type"].(string); ok {
				switch varType {
				case "string":
					samples[key] = "Sample " + key
				case "number":
					samples[key] = 123
				case "boolean":
					samples[key] = true
				case "date":
					samples[key] = time.Now().Format(time.RFC3339)
				case "array":
					samples[key] = []string{"item1", "item2", "item3"}
				case "object":
					samples[key] = map[string]any{"field1": "value1", "field2": "value2"}
				default:
					samples[key] = "Sample " + key
				}
			}
		}
	}

	return samples
}

// IsEditable returns true if the template can be edited
func (t *Template) IsEditable() bool {
	return !t.IsSystem && t.Status == domain.StatusActive
}
