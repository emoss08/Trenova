package formulatemplate

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/core/types/formula"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/rotisserie/eris"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook = (*FormulaTemplate)(nil)
	_ domain.Validatable        = (*FormulaTemplate)(nil)
)

// TemplateVariable represents a variable used in a formula template
type TemplateVariable struct {
	Name         string            `json:"name"`
	Type         formula.ValueType `json:"type"`
	Description  string            `json:"description"`
	Required     bool              `json:"required"`
	DefaultValue any               `json:"defaultValue,omitempty"`
	Source       string            `json:"source"` // e.g., "shipment.weight", "shipment.distance"
}

// TemplateParameter represents a configurable parameter in a template
type TemplateParameter struct {
	Name         string            `json:"name"`
	Type         formula.ValueType `json:"type"`
	Description  string            `json:"description"`
	DefaultValue any               `json:"defaultValue"`
	Required     bool              `json:"required"`
	MinValue     *float64          `json:"minValue,omitempty"`
	MaxValue     *float64          `json:"maxValue,omitempty"`
	Options      []ParameterOption `json:"options,omitempty"`
}

// ParameterOption represents an option for a parameter
type ParameterOption struct {
	Value       any    `json:"value"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
}

// TemplateExample shows how to use a template for rate calculation
type TemplateExample struct {
	Name         string         `json:"name"`
	Description  string         `json:"description"`
	Parameters   map[string]any `json:"parameters"`
	ShipmentData map[string]any `json:"shipmentData"` // Example shipment data
	ExpectedRate float64        `json:"expectedRate"`
}

// TemplateRequirement specifies what's needed for a template
type TemplateRequirement struct {
	Type        string `json:"type"` // "variable", "field", "function"
	Name        string `json:"name"`
	Description string `json:"description"`
}

// FormulaTemplate represents a formula template for shipment rate calculations
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
	BusinessUnit *businessunit.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *organization.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
}

func (ft *FormulaTemplate) Validate(ctx context.Context, multiErr *errors.MultiError) {
	err := validation.ValidateStructWithContext(ctx, ft,
		// Name is required and must be between 3 and 255 characters
		validation.Field(&ft.Name,
			validation.Required.Error("Name is required"),
			validation.Length(3, 255).Error("Name must be between 3 and 255 characters"),
		),

		// Expression is required
		validation.Field(&ft.Expression,
			validation.Required.Error("Expression is required"),
			validation.Length(1, 0).Error("Expression cannot be empty"),
		),

		// Category is required and must be valid
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

		// OutputUnit defaults to USD but can be customized
		validation.Field(&ft.OutputUnit,
			validation.Length(0, 50).Error("Output unit must be less than 50 characters"),
		),

		// MinRate must be less than MaxRate if both are specified
		validation.Field(&ft.MaxRate,
			validation.When(
				ft.MinRate != nil && ft.MaxRate != nil,
				validation.By(func(value interface{}) error {
					if *ft.MinRate > *ft.MaxRate {
						return validation.NewError(
							"validation_max_rate",
							"Maximum rate must be greater than minimum rate",
						)
					}
					return nil
				}),
			),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if eris.As(err, &validationErrs) {
			errors.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (ft *FormulaTemplate) GetID() string {
	return ft.ID.String()
}

func (ft *FormulaTemplate) GetTableName() string {
	return "formula_templates"
}

func (ft *FormulaTemplate) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

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

// HasParameters returns true if the template has configurable parameters
func (ft *FormulaTemplate) HasParameters() bool {
	return len(ft.Parameters) > 0
}

// HasRequirements returns true if the template has requirements
func (ft *FormulaTemplate) HasRequirements() bool {
	return len(ft.Requirements) > 0
}

// HasExamples returns true if the template has examples
func (ft *FormulaTemplate) HasExamples() bool {
	return len(ft.Examples) > 0
}

// IsValid checks if the template is valid for use
func (ft *FormulaTemplate) IsValid() bool {
	return ft.IsActive
}

// GetRequiredVariables returns all required variables
func (ft *FormulaTemplate) GetRequiredVariables() []TemplateVariable {
	var required []TemplateVariable
	for _, v := range ft.Variables {
		if v.Required {
			required = append(required, v)
		}
	}
	return required
}
