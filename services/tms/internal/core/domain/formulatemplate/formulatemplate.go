package formulatemplate

import (
	"context"
	"errors"
	"reflect"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/formulatypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*FormulaTemplate)(nil)
	_ validationframework.TenantedEntity = (*FormulaTemplate)(nil)
	_ domaintypes.PostgresSearchable     = (*FormulaTemplate)(nil)
)

type FormulaTemplate struct {
	bun.BaseModel `bun:"table:formula_templates,alias:ft" json:"-"`

	pagination.CursorValueSet `json:"-" bun:",embed"`

	ID                   pulid.ID                            `json:"id"                   bun:"id,pk,type:VARCHAR(100)"`
	OrganizationID       pulid.ID                            `json:"organizationId"       bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID       pulid.ID                            `json:"businessUnitId"       bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	Name                 string                              `json:"name"                 bun:"name,type:VARCHAR(100),notnull"`
	Description          string                              `json:"description"          bun:"description,type:TEXT"`
	Type                 TemplateType                        `json:"type"                 bun:"type,type:formula_template_type_enum,notnull,default:'FreightCharge'"`
	Expression           string                              `json:"expression"           bun:"expression,type:TEXT,notnull"`
	Status               Status                              `json:"status"               bun:"status,type:formula_template_status_enum,notnull,default:'Draft'"`
	SchemaID             string                              `json:"schemaId"             bun:"schema_id,type:VARCHAR(100),notnull,default:'shipment'"`
	VariableDefinitions  []*formulatypes.VariableDefinition  `json:"variableDefinitions"  bun:"variable_definitions,type:JSONB,notnull,default:'[]'"`
	BreakdownDefinitions []*formulatypes.BreakdownDefinition `json:"breakdownDefinitions" bun:"breakdown_definitions,type:JSONB,notnull,default:'[]'"`
	MinCharge            decimal.NullDecimal                 `json:"minCharge"            bun:"min_charge,type:NUMERIC(19,4)"`
	MaxCharge            decimal.NullDecimal                 `json:"maxCharge"            bun:"max_charge,type:NUMERIC(19,4)"`
	SubmittedByID        *pulid.ID                           `json:"submittedById"        bun:"submitted_by_id,type:VARCHAR(100)"`
	SubmittedAt          *int64                              `json:"submittedAt"          bun:"submitted_at,type:BIGINT"`
	ApprovedByID         *pulid.ID                           `json:"approvedById"         bun:"approved_by_id,type:VARCHAR(100)"`
	ApprovedAt           *int64                              `json:"approvedAt"           bun:"approved_at,type:BIGINT"`
	ReviewComment        string                              `json:"reviewComment"        bun:"review_comment,type:TEXT"`
	SearchVector         string                              `json:"-"                    bun:"search_vector,type:TSVECTOR,scanonly"`
	Rank                 string                              `json:"-"                    bun:"rank,type:VARCHAR(100),scanonly"`
	Metadata             map[string]any                      `json:"metadata"             bun:"metadata,type:JSONB"`
	Version              int64                               `json:"version"              bun:"version,type:BIGINT"`
	SourceTemplateID     *pulid.ID                           `json:"sourceTemplateId"     bun:"source_template_id,type:VARCHAR(100)"`
	SourceVersionNumber  *int64                              `json:"sourceVersionNumber"  bun:"source_version_number,type:BIGINT"`
	CurrentVersionNumber int64                               `json:"currentVersionNumber" bun:"current_version_number,type:BIGINT,notnull,default:1"`
	CreatedAt            int64                               `json:"createdAt"            bun:"created_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt            int64                               `json:"updatedAt"            bun:"updated_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Organization   *tenant.Organization      `json:"organization,omitempty"   bun:"rel:belongs-to,join:organization_id=id"`
	BusinessUnit   *tenant.BusinessUnit      `json:"businessUnit,omitempty"   bun:"rel:belongs-to,join:business_unit_id=id"`
	SourceTemplate *FormulaTemplate          `json:"sourceTemplate,omitempty" bun:"rel:belongs-to,join:source_template_id=id"`
	Versions       []*FormulaTemplateVersion `json:"versions,omitempty"       bun:"rel:has-many,join:id=template_id"`
}

func (ft *FormulaTemplate) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(ft,
		validation.Field(&ft.Name, validation.Required, validation.Length(1, 100)),
		validation.Field(&ft.Expression, validation.Required),
		validation.Field(&ft.Type, validation.Required, validation.In(
			TemplateTypeFreightCharge,
			TemplateTypeAccessorialCharge,
		)),
		validation.Field(&ft.Status, validation.Required, validation.In(
			StatusActive,
			StatusInactive,
			StatusDraft,
			StatusInReview,
		)),
	)
	if err != nil {
		var validationErrs validation.Errors
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}

	ft.validateGuardrails(multiErr)
	formulatypes.ValidateBreakdownDefinitions(ft.BreakdownDefinitions, multiErr)
}

func (ft *FormulaTemplate) validateGuardrails(multiErr *errortypes.MultiError) {
	if ft.MinCharge.Valid && ft.MinCharge.Decimal.IsNegative() {
		multiErr.Add("minCharge", errortypes.ErrInvalid, "Minimum charge cannot be negative")
	}

	if ft.MaxCharge.Valid && ft.MaxCharge.Decimal.IsNegative() {
		multiErr.Add("maxCharge", errortypes.ErrInvalid, "Maximum charge cannot be negative")
	}

	if ft.MinCharge.Valid && ft.MaxCharge.Valid &&
		ft.MinCharge.Decimal.GreaterThan(ft.MaxCharge.Decimal) {
		multiErr.Add(
			"minCharge",
			errortypes.ErrInvalid,
			"Minimum charge cannot exceed maximum charge",
		)
	}
}

func (ft *FormulaTemplate) ApplyVersion(version *FormulaTemplateVersion) *FormulaTemplate {
	resolved := *ft
	resolved.Expression = version.Expression
	resolved.SchemaID = version.SchemaID
	resolved.Type = version.Type
	resolved.VariableDefinitions = version.VariableDefinitions
	resolved.BreakdownDefinitions = version.BreakdownDefinitions
	resolved.MinCharge = version.MinCharge
	resolved.MaxCharge = version.MaxCharge
	resolved.CurrentVersionNumber = version.VersionNumber
	return &resolved
}

func (ft *FormulaTemplate) HasMaterialChange(other *FormulaTemplate) bool {
	if other == nil {
		return true
	}

	return ft.Expression != other.Expression ||
		ft.SchemaID != other.SchemaID ||
		ft.Type != other.Type ||
		!nullDecimalEqual(ft.MinCharge, other.MinCharge) ||
		!nullDecimalEqual(ft.MaxCharge, other.MaxCharge) ||
		!variableDefinitionsEqual(ft.VariableDefinitions, other.VariableDefinitions) ||
		!breakdownDefinitionsEqual(ft.BreakdownDefinitions, other.BreakdownDefinitions)
}

func nullDecimalEqual(a, b decimal.NullDecimal) bool {
	if a.Valid != b.Valid {
		return false
	}
	if !a.Valid {
		return true
	}
	return a.Decimal.Equal(b.Decimal)
}

func variableDefinitionsEqual(a, b []*formulatypes.VariableDefinition) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if (a[i] == nil) != (b[i] == nil) {
			return false
		}
		if a[i] != nil && !reflect.DeepEqual(*a[i], *b[i]) {
			return false
		}
	}
	return true
}

func breakdownDefinitionsEqual(a, b []*formulatypes.BreakdownDefinition) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if (a[i] == nil) != (b[i] == nil) {
			return false
		}
		if a[i] != nil && *a[i] != *b[i] {
			return false
		}
	}
	return true
}

func (ft *FormulaTemplate) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if ft.ID.IsNil() {
			ft.ID = pulid.MustNew("ft_")
		}
		ft.CreatedAt = now
	case *bun.UpdateQuery:
		ft.UpdatedAt = now
	}

	return nil
}

func (ft *FormulaTemplate) GetID() pulid.ID {
	return ft.ID
}

func (ft *FormulaTemplate) GetCreatedAt() int64 {
	return ft.CreatedAt
}

func (ft *FormulaTemplate) GetOrganizationID() pulid.ID {
	return ft.OrganizationID
}

func (ft *FormulaTemplate) GetBusinessUnitID() pulid.ID {
	return ft.BusinessUnitID
}

func (ft *FormulaTemplate) GetTableName() string {
	return "formula_templates"
}

func (ft *FormulaTemplate) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "ft",
		UseSearchVector: true,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "name", Type: domaintypes.FieldTypeText},
			{Name: "description", Type: domaintypes.FieldTypeText},
			{Name: "type", Type: domaintypes.FieldTypeEnum},
			{Name: "status", Type: domaintypes.FieldTypeEnum},
		},
	}
}
