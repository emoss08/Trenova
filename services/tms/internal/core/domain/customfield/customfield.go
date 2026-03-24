package customfield

import (
	"context"
	"errors"
	"regexp"

	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook      = (*CustomFieldDefinition)(nil)
	_ domaintypes.PostgresSearchable = (*CustomFieldDefinition)(nil)
)

var namePattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

type CustomFieldDefinition struct {
	bun.BaseModel `bun:"table:custom_field_definitions,alias:cfd" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),notnull,pk"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,type:VARCHAR(100),notnull,pk"`

	ResourceType    string           `json:"resourceType"    bun:"resource_type,type:VARCHAR(100),notnull"`
	Name            string           `json:"name"            bun:"name,type:VARCHAR(100),notnull"`
	Label           string           `json:"label"           bun:"label,type:VARCHAR(150),notnull"`
	Description     string           `json:"description"     bun:"description,type:TEXT,nullzero"`
	FieldType       FieldType        `json:"fieldType"       bun:"field_type,type:custom_field_type_enum,notnull"`
	IsRequired      bool             `json:"isRequired"      bun:"is_required,default:false"`
	IsActive        bool             `json:"isActive"        bun:"is_active,default:true"`
	DisplayOrder    int              `json:"displayOrder"    bun:"display_order,default:0"`
	Color           string           `json:"color"           bun:"color,type:VARCHAR(20),nullzero"`
	Options         []SelectOption   `json:"options"         bun:"options,type:JSONB"`
	ValidationRules *ValidationRules `json:"validationRules" bun:"validation_rules,type:JSONB"`
	DefaultValue    any              `json:"defaultValue"    bun:"default_value,type:JSONB"`
	UIAttributes    *UIAttributes    `json:"uiAttributes"    bun:"ui_attributes,type:JSONB"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull"`
}

func (c *CustomFieldDefinition) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(
		c,
		validation.Field(&c.ResourceType, validation.Required.Error("Resource type is required")),
		validation.Field(
			&c.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 100).Error("Name must be between 1 and 100 characters"),
			validation.Match(namePattern).
				Error("Name must start with a lowercase letter and contain only lowercase letters, numbers, and underscores"),
		),
		validation.Field(&c.Label,
			validation.Required.Error("Label is required"),
			validation.Length(1, 150).Error("Label must be between 1 and 150 characters"),
		),
		validation.Field(&c.FieldType,
			validation.Required.Error("Field type is required"),
			validation.By(func(value any) error {
				ft, ok := value.(FieldType)
				if !ok {
					return errors.New("invalid field type")
				}
				if !ft.IsValid() {
					return errors.New("invalid field type")
				}
				return nil
			}),
		),
		validation.Field(&c.Color,
			validation.Length(0, 20).Error("Color must be at most 20 characters"),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}

	if c.FieldType.RequiresOptions() && len(c.Options) == 0 {
		multiErr.Add(
			"options",
			errortypes.ErrRequired,
			"Options are required for select and multiSelect field types",
		)
	}

	if !IsResourceTypeSupported(c.ResourceType) {
		multiErr.Add("resourceType", errortypes.ErrInvalid, "Unsupported resource type")
	}
}

func (c *CustomFieldDefinition) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if c.ID.IsNil() {
			c.ID = pulid.MustNew("cfd_")
		}
		c.CreatedAt = now
		c.UpdatedAt = now

		if c.Options == nil {
			c.Options = []SelectOption{}
		}

	case *bun.UpdateQuery:
		c.UpdatedAt = now
	}

	return nil
}

func (c *CustomFieldDefinition) GetID() pulid.ID {
	return c.ID
}

func (c *CustomFieldDefinition) GetOrganizationID() pulid.ID {
	return c.OrganizationID
}

func (c *CustomFieldDefinition) GetBusinessUnitID() pulid.ID {
	return c.BusinessUnitID
}

func (c *CustomFieldDefinition) GetTableName() string {
	return "custom_field_definitions"
}

func (c *CustomFieldDefinition) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "cfd",
		UseSearchVector: false,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "name", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{Name: "label", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{
				Name:   "description",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
			{
				Name:   "resource_type",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightC,
			},
		},
	}
}
