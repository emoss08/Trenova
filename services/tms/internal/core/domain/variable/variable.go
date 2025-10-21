package variable

import (
	"context"
	"errors"
	"regexp"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook      = (*Variable)(nil)
	_ domain.Validatable             = (*Variable)(nil)
	_ domaintypes.PostgresSearchable = (*Variable)(nil)

	variableKeyRegex = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*$`)
)

type Variable struct {
	bun.BaseModel `bun:"table:variables,alias:var" json:"-"`

	ID             pulid.ID  `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID  `json:"businessUnitId" bun:"business_unit_id,pk,notnull,type:VARCHAR(100)"`
	OrganizationID pulid.ID  `json:"organizationId" bun:"organization_id,pk,notnull,type:VARCHAR(100)"`
	Key            string    `json:"key"            bun:"key,type:VARCHAR(100),notnull"`
	DisplayName    string    `json:"displayName"    bun:"display_name,type:VARCHAR(255),notnull"`
	Description    string    `json:"description"    bun:"description,type:TEXT"`
	Category       string    `json:"category"       bun:"category,type:VARCHAR(100)"`
	Query          string    `json:"query"          bun:"query,type:TEXT,notnull"`
	AppliesTo      Context   `json:"appliesTo"      bun:"applies_to,type:variable_context_enum,notnull"`
	RequiredParams []string  `json:"requiredParams" bun:"required_params,type:JSONB"`
	DefaultValue   string    `json:"defaultValue"   bun:"default_value,type:TEXT"`
	FormatID       *pulid.ID `json:"formatId"       bun:"format_id,type:VARCHAR(100)"`
	ValueType      ValueType `json:"valueType"      bun:"value_type,type:variable_value_type_enum,default:'String'"`
	IsActive       bool      `json:"isActive"       bun:"is_active,type:BOOLEAN,default:true"`
	IsSystem       bool      `json:"isSystem"       bun:"is_system,type:BOOLEAN,default:false"`
	IsValidated    bool      `json:"isValidated"    bun:"is_validated,type:BOOLEAN,default:false"`
	Tags           []string  `json:"tags"           bun:"tags,type:TEXT[]"`
	Version        int64     `json:"version"        bun:"version,type:BIGINT"`
	CreatedAt      int64     `json:"createdAt"      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64     `json:"updatedAt"      bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	BusinessUnit *tenant.BusinessUnit `json:"-"               bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *tenant.Organization `json:"-"               bun:"rel:belongs-to,join:organization_id=id"`
	Format       *VariableFormat      `json:"format,omitzero" bun:"rel:belongs-to,join:format_id=id"`
}

func (v *Variable) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(v,
		validation.Field(
			&v.Key,
			validation.Required.Error("Key is required"),
			validation.Length(2, 100).Error("Key must be between 2 and 100 characters"),
			validation.Match(variableKeyRegex).
				Error("Key must start with a letter and contain only letters, numbers, and underscores"),
		),
		validation.Field(&v.DisplayName,
			validation.Required.Error("Display name is required"),
			validation.Length(1, 255).Error("Display name must be between 1 and 255 characters"),
		),
		validation.Field(&v.Query,
			validation.Required.Error("Query is required"),
			validation.Length(1, 0).Error("Query cannot be empty"),
		),
		validation.Field(&v.AppliesTo,
			validation.Required.Error("Context is required"),
			validation.By(validateContext),
		),
		validation.Field(&v.ValueType,
			validation.By(validateValueType),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func validateContext(value any) error {
	ctx, ok := value.(Context)
	if !ok {
		return errors.New("invalid variable context type")
	}
	if !ctx.IsValid() {
		return errors.New("invalid variable context value")
	}
	return nil
}

func validateValueType(value any) error {
	vt, ok := value.(ValueType)
	if !ok {
		return nil
	}
	if !vt.IsValid() {
		return errors.New("invalid variable value type")
	}
	return nil
}

func (v *Variable) GetID() string {
	return v.ID.String()
}

func (v *Variable) GetTableName() string {
	return "variables"
}

func (v *Variable) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := utils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if v.ID.IsNil() {
			v.ID = pulid.MustNew("var_")
		}
		v.CreatedAt = now
	case *bun.UpdateQuery:
		v.UpdatedAt = now
	}

	return nil
}

func (v *Variable) CanEdit() bool {
	return !v.IsSystem
}

func (v *Variable) CanDelete() bool {
	return !v.IsSystem
}

func (v *Variable) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "var",
		UseSearchVector: true,
		SearchableFields: []domaintypes.SearchableField{
			{
				Name:   "key",
				Weight: domaintypes.SearchWeightA,
				Type:   domaintypes.FieldTypeText,
			},
			{
				Name:   "display_name",
				Weight: domaintypes.SearchWeightB,
				Type:   domaintypes.FieldTypeText,
			},
			{
				Name:   "description",
				Weight: domaintypes.SearchWeightC,
				Type:   domaintypes.FieldTypeText,
			},
			{
				Name: "category",
				Type: domaintypes.FieldTypeText,
			},
			{
				Name: "applies_to",
				Type: domaintypes.FieldTypeEnum,
			},
			{
				Name: "is_active",
				Type: domaintypes.FieldTypeBoolean,
			},
			{
				Name: "is_system",
				Type: domaintypes.FieldTypeBoolean,
			},
		},
	}
}
