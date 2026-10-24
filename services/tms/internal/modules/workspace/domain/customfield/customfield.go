package customfield

import (
	"context"
	"regexp"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var (
	_ pagination.CursorEntity        = (*CustomFieldDefinition)(nil)
	_ domaintypes.PostgresSearchable = (*CustomFieldDefinition)(nil)
)

var namePattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

type CustomFieldDefinition struct {
	keel.Table `                                                        keel:"custom_field_definitions"`

	ID             pulid.ID `json:"id" keel:"id,pk,autoid:cfd_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`

	ResourceType    string           `json:"resourceType" keel:"resource_type"`
	Name            string           `json:"name" keel:"name"`
	Label           string           `json:"label" keel:"label"`
	Description     string           `json:"description" keel:"description,nullzero"`
	FieldType       FieldType        `json:"fieldType" keel:"field_type"`
	IsRequired      bool             `json:"isRequired" keel:"is_required"`
	IsActive        bool             `json:"isActive" keel:"is_active"`
	DisplayOrder    int              `json:"displayOrder" keel:"display_order"`
	Color           string           `json:"color" keel:"color,nullzero"`
	Options         []SelectOption   `json:"options" keel:"options,jsonb"`
	ValidationRules *ValidationRules `json:"validationRules" keel:"validation_rules,jsonb"`
	DefaultValue    any              `json:"defaultValue" keel:"default_value,jsonb"`
	UIAttributes    *UIAttributes    `json:"uiAttributes" keel:"ui_attributes,jsonb"`

	SearchVector string `json:"-"`
	Rank         string `json:"-" keel:"rank,computed"`
	Version      int64  `json:"version" keel:"version,version"`
	CreatedAt    int64  `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt    int64  `json:"updatedAt" keel:"updated_at,autoupdate"`
}

func (c *CustomFieldDefinition) BeforeCreate(_ context.Context) error {
	if c.Options == nil {
		c.Options = []SelectOption{}
	}

	return nil
}

func (c *CustomFieldDefinition) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(
		c,
		validation.Field(&c.ResourceType, validation.Required.Error("Resource type is required")),
		validation.Field(
			&c.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 100).Error("Name must be between 1 and 100 characters"),
			validation.Match(namePattern).
				Error("Name must start with a lowercase letter and contain only lowercase letters, numbers, and underscores"),
		),
		validation.Field(
			&c.Label,
			validation.Required.Error("Label is required"),
			validation.Length(1, 150).Error("Label must be between 1 and 150 characters"),
		),
		validation.Field(
			&c.FieldType,
			validation.Required.Error("Field type is required"),
			domainvalidation.ValidEnum[FieldType]("invalid field type"),
		),
		validation.Field(
			&c.Color,
			validation.Length(0, 20).Error("Color must be at most 20 characters"),
		),
	))

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

func (c *CustomFieldDefinition) GetID() pulid.ID {
	return c.ID
}

func (c *CustomFieldDefinition) GetCreatedAt() int64 {
	return c.CreatedAt
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
