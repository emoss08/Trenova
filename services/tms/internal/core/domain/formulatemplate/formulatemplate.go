package formulatemplate

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/formulatypes"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/emoss08/trenova/pkg/validator/framework"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook      = (*FormulaTemplate)(nil)
	_ domaintypes.PostgresSearchable = (*FormulaTemplate)(nil)
	_ domain.Validatable             = (*FormulaTemplate)(nil)
	_ framework.TenantedEntity       = (*FormulaTemplate)(nil)
)

type TemplateVariable struct {
	Name         string                 `json:"name"`
	Type         formulatypes.ValueType `json:"type"`
	Description  string                 `json:"description"`
	Required     bool                   `json:"required"`
	DefaultValue any                    `json:"defaultValue,omitempty"`
	Source       string                 `json:"source"` // e.g., "shipment.weight", "shipment.distance"
}

type TemplateParameter struct {
	Name         string                 `json:"name"`
	Type         formulatypes.ValueType `json:"type"`
	Description  string                 `json:"description"`
	DefaultValue any                    `json:"defaultValue"`
	Required     bool                   `json:"required"`
	MinValue     *float64               `json:"minValue,omitempty"`
	MaxValue     *float64               `json:"maxValue,omitempty"`
	Options      []ParameterOption      `json:"options,omitempty"`
}

type ParameterOption struct {
	Value       any    `json:"value"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
}

type TemplateExample struct {
	Name         string         `json:"name"`
	Description  string         `json:"description"`
	Parameters   map[string]any `json:"parameters"`
	ShipmentData map[string]any `json:"shipmentData"` // Example shipment data
	ExpectedRate float64        `json:"expectedRate"`
}

type TemplateRequirement struct {
	Type        string `json:"type"` // "variable", "field", "function"
	Name        string `json:"name"`
	Description string `json:"description"`
}

type FormulaTemplate struct {
	bun.BaseModel `bun:"table:formula_templates,alias:ft" json:"-"`

	ID             pulid.ID              `json:"id"             bun:"id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID pulid.ID              `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID pulid.ID              `json:"organizationId" bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	Name           string                `json:"name"           bun:"name,type:VARCHAR(255),notnull"`
	Description    string                `json:"description"    bun:"description,type:TEXT,nullzero"`
	Category       Category              `json:"category"       bun:"category,type:formula_template_category_enum,notnull"`
	Expression     string                `json:"expression"     bun:"expression,type:TEXT,notnull"`
	Variables      []TemplateVariable    `json:"variables"      bun:"variables,type:JSONB,nullzero"`
	Parameters     []TemplateParameter   `json:"parameters"     bun:"parameters,type:JSONB,nullzero"`
	Tags           []string              `json:"tags"           bun:"tags,type:TEXT[],nullzero,array"`
	Examples       []TemplateExample     `json:"examples"       bun:"examples,type:JSONB,nullzero"`
	Requirements   []TemplateRequirement `json:"requirements"   bun:"requirements,type:JSONB,nullzero"`
	MinRate        *float64              `json:"minRate"        bun:"min_rate,type:NUMERIC(19,4),nullzero"`
	MaxRate        *float64              `json:"maxRate"        bun:"max_rate,type:NUMERIC(19,4),nullzero"`
	OutputUnit     string                `json:"outputUnit"     bun:"output_unit,type:VARCHAR(50),nullzero,default:'USD'"`
	IsActive       bool                  `json:"isActive"       bun:"is_active,type:BOOLEAN,notnull,default:true"`
	IsDefault      bool                  `json:"isDefault"      bun:"is_default,type:BOOLEAN,notnull,default:false"`
	Version        int64                 `json:"version"        bun:"version,type:BIGINT"`
	CreatedAt      int64                 `json:"createdAt"      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64                 `json:"updatedAt"      bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Relationships
	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *tenant.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
}

func (ft *FormulaTemplate) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(ft,
		validation.Field(&ft.Name,
			validation.Required.Error("Name is required"),
			validation.Length(3, 255).Error("Name must be between 3 and 255 characters"),
		),
		validation.Field(&ft.Expression,
			validation.Required.Error("Expression is required"),
			validation.Length(1, 0).Error("Expression cannot be empty"),
		),
		validation.Field(&ft.Category,
			validation.Required.Error("Category is required"),
			validation.In(
				CategoryBaseRate,
				CategoryDistanceBased,
				CategoryWeightBased,
				CategoryDimensionalWeight,
				CategoryFuelSurcharge,
				CategoryAccessorial,
				CategoryTimeBasedRate,
				CategoryZoneBased,
				CategoryCustom,
			).Error("Category must be valid"),
		),
		validation.Field(&ft.OutputUnit,
			validation.Length(0, 50).Error("Output unit must be less than 50 characters"),
		),
		validation.Field(&ft.MaxRate,
			validation.When(
				ft.MinRate != nil && ft.MaxRate != nil,
				validation.Min(ft.MaxRate).Error("Maximum rate must be greater than minimum rate"),
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

func (ft *FormulaTemplate) GetID() string {
	return ft.ID.String()
}

func (ft *FormulaTemplate) GetTableName() string {
	return "formula_templates"
}

func (ft *FormulaTemplate) GetOrganizationID() pulid.ID {
	return ft.OrganizationID
}

func (ft *FormulaTemplate) GetBusinessUnitID() pulid.ID {
	return ft.BusinessUnitID
}

func (ft *FormulaTemplate) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "ft",
		UseSearchVector: true,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "name", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{
				Name:   "description",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
			{Name: "category", Type: domaintypes.FieldTypeEnum, Weight: domaintypes.SearchWeightB},
		},
	}
}

func (ft *FormulaTemplate) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := utils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if ft.ID.IsNil() {
			ft.ID = pulid.MustNew("fmt_")
		}
		ft.CreatedAt = now
		ft.UpdatedAt = now
	case *bun.UpdateQuery:
		ft.UpdatedAt = now
	}

	return nil
}

func (ft *FormulaTemplate) HasParameters() bool {
	return len(ft.Parameters) > 0
}

func (ft *FormulaTemplate) HasRequirements() bool {
	return len(ft.Requirements) > 0
}

func (ft *FormulaTemplate) HasExamples() bool {
	return len(ft.Examples) > 0
}

func (ft *FormulaTemplate) IsValid() bool {
	return ft.IsActive
}

func (ft *FormulaTemplate) GetRequiredVariables() []TemplateVariable {
	var required []TemplateVariable
	for _, v := range ft.Variables {
		if v.Required {
			required = append(required, v)
		}
	}
	return required
}
