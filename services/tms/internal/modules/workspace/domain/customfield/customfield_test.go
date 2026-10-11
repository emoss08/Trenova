package customfield

import (
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

func TestCustomFieldDefinition_Validate_Success(t *testing.T) {
	t.Parallel()

	def := &CustomFieldDefinition{
		ResourceType: "trailer",
		Name:         "custom_field",
		Label:        "Custom Field",
		FieldType:    FieldTypeText,
	}

	multiErr := errortypes.NewMultiError()
	def.Validate(multiErr)

	assert.False(t, multiErr.HasErrors())
}

func TestCustomFieldDefinition_Validate_NameRequired(t *testing.T) {
	t.Parallel()

	def := &CustomFieldDefinition{
		ResourceType: "trailer",
		Name:         "",
		Label:        "Custom Field",
		FieldType:    FieldTypeText,
	}

	multiErr := errortypes.NewMultiError()
	def.Validate(multiErr)

	require.True(t, multiErr.HasErrors())
	assert.True(t, hasErrorForField(multiErr, "name"))
}

func TestCustomFieldDefinition_Validate_LabelRequired(t *testing.T) {
	t.Parallel()

	def := &CustomFieldDefinition{
		ResourceType: "trailer",
		Name:         "custom_field",
		Label:        "",
		FieldType:    FieldTypeText,
	}

	multiErr := errortypes.NewMultiError()
	def.Validate(multiErr)

	require.True(t, multiErr.HasErrors())
	assert.True(t, hasErrorForField(multiErr, "label"))
}

func TestCustomFieldDefinition_Validate_ResourceTypeRequired(t *testing.T) {
	t.Parallel()

	def := &CustomFieldDefinition{
		ResourceType: "",
		Name:         "custom_field",
		Label:        "Custom Field",
		FieldType:    FieldTypeText,
	}

	multiErr := errortypes.NewMultiError()
	def.Validate(multiErr)

	require.True(t, multiErr.HasErrors())
	assert.True(t, hasErrorForField(multiErr, "resourceType"))
}

func TestCustomFieldDefinition_Validate_InvalidFieldType(t *testing.T) {
	t.Parallel()

	def := &CustomFieldDefinition{
		ResourceType: "trailer",
		Name:         "custom_field",
		Label:        "Custom Field",
		FieldType:    FieldType("invalid"),
	}

	multiErr := errortypes.NewMultiError()
	def.Validate(multiErr)

	require.True(t, multiErr.HasErrors())
	assert.True(t, hasErrorForField(multiErr, "fieldType"))
}

func TestCustomFieldDefinition_Validate_SelectRequiresOptions(t *testing.T) {
	t.Parallel()

	def := &CustomFieldDefinition{
		ResourceType: "trailer",
		Name:         "select_field",
		Label:        "Select Field",
		FieldType:    FieldTypeSelect,
		Options:      []SelectOption{},
	}

	multiErr := errortypes.NewMultiError()
	def.Validate(multiErr)

	require.True(t, multiErr.HasErrors())
	assert.True(t, hasErrorForField(multiErr, "options"))
}

func TestCustomFieldDefinition_Validate_MultiSelectRequiresOptions(t *testing.T) {
	t.Parallel()

	def := &CustomFieldDefinition{
		ResourceType: "trailer",
		Name:         "multi_select_field",
		Label:        "Multi Select Field",
		FieldType:    FieldTypeMultiSelect,
		Options:      nil,
	}

	multiErr := errortypes.NewMultiError()
	def.Validate(multiErr)

	require.True(t, multiErr.HasErrors())
	assert.True(t, hasErrorForField(multiErr, "options"))
}

func TestCustomFieldDefinition_Validate_SelectWithOptions(t *testing.T) {
	t.Parallel()

	def := &CustomFieldDefinition{
		ResourceType: "trailer",
		Name:         "select_field",
		Label:        "Select Field",
		FieldType:    FieldTypeSelect,
		Options: []SelectOption{
			{Value: "option1", Label: "Option 1"},
			{Value: "option2", Label: "Option 2"},
		},
	}

	multiErr := errortypes.NewMultiError()
	def.Validate(multiErr)

	assert.False(t, multiErr.HasErrors())
}

func TestCustomFieldDefinition_Validate_NameFormat_Valid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		fieldName string
	}{
		{name: "lowercase", fieldName: "customfield"},
		{name: "with underscore", fieldName: "custom_field"},
		{name: "with numbers", fieldName: "custom123"},
		{name: "with underscore and numbers", fieldName: "custom_field_123"},
		{name: "single letter", fieldName: "a"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			def := &CustomFieldDefinition{
				ResourceType: "trailer",
				Name:         tt.fieldName,
				Label:        "Custom Field",
				FieldType:    FieldTypeText,
			}

			multiErr := errortypes.NewMultiError()
			def.Validate(multiErr)

			assert.False(t, multiErr.HasErrors(), "expected no errors for name: %s", tt.fieldName)
		})
	}
}

func TestCustomFieldDefinition_Validate_NameFormat_Invalid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		fieldName string
	}{
		{name: "starts with number", fieldName: "1custom"},
		{name: "starts with underscore", fieldName: "_custom"},
		{name: "uppercase letters", fieldName: "CustomField"},
		{name: "spaces", fieldName: "custom field"},
		{name: "dashes", fieldName: "custom-field"},
		{name: "special characters", fieldName: "custom@field"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			def := &CustomFieldDefinition{
				ResourceType: "trailer",
				Name:         tt.fieldName,
				Label:        "Custom Field",
				FieldType:    FieldTypeText,
			}

			multiErr := errortypes.NewMultiError()
			def.Validate(multiErr)

			assert.True(t, multiErr.HasErrors(), "expected error for name: %s", tt.fieldName)
		})
	}
}

func TestCustomFieldDefinition_Validate_UnsupportedResourceType(t *testing.T) {
	t.Parallel()

	def := &CustomFieldDefinition{
		ResourceType: "unsupported_type",
		Name:         "custom_field",
		Label:        "Custom Field",
		FieldType:    FieldTypeText,
	}

	multiErr := errortypes.NewMultiError()
	def.Validate(multiErr)

	require.True(t, multiErr.HasErrors())
	assert.True(t, hasErrorForField(multiErr, "resourceType"))
}

func TestCustomFieldDefinition_BeforeAppendModel_Insert(t *testing.T) {
	t.Parallel()

	def := &CustomFieldDefinition{
		ResourceType: "trailer",
		Name:         "custom_field",
		Label:        "Custom Field",
		FieldType:    FieldTypeText,
	}

	query := (*bun.InsertQuery)(nil)
	err := def.BeforeAppendModel(t.Context(), query)

	require.NoError(t, err)
	assert.False(t, def.ID.IsNil())
	assert.Equal(t, "cfd_", def.ID.Prefix())
	assert.NotZero(t, def.CreatedAt)
	assert.NotZero(t, def.UpdatedAt)
	assert.Equal(t, def.CreatedAt, def.UpdatedAt)
	assert.NotNil(t, def.Options)
}

func TestCustomFieldDefinition_BeforeAppendModel_Update(t *testing.T) {
	t.Parallel()

	def := &CustomFieldDefinition{
		ID:           pulid.MustNew("cfd_"),
		ResourceType: "trailer",
		Name:         "custom_field",
		Label:        "Custom Field",
		FieldType:    FieldTypeText,
		CreatedAt:    1000,
		UpdatedAt:    1000,
	}

	query := (*bun.UpdateQuery)(nil)
	err := def.BeforeAppendModel(t.Context(), query)

	require.NoError(t, err)
	assert.Equal(t, int64(1000), def.CreatedAt)
	assert.NotEqual(t, int64(1000), def.UpdatedAt)
}

func TestCustomFieldDefinition_GetID(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("cfd_")
	def := &CustomFieldDefinition{ID: id}

	assert.Equal(t, id, def.GetID())
}

func TestCustomFieldDefinition_GetOrganizationID(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	def := &CustomFieldDefinition{OrganizationID: orgID}

	assert.Equal(t, orgID, def.GetOrganizationID())
}

func TestCustomFieldDefinition_GetBusinessUnitID(t *testing.T) {
	t.Parallel()

	buID := pulid.MustNew("bu_")
	def := &CustomFieldDefinition{BusinessUnitID: buID}

	assert.Equal(t, buID, def.GetBusinessUnitID())
}

func TestCustomFieldDefinition_GetTableName(t *testing.T) {
	t.Parallel()

	def := &CustomFieldDefinition{}
	assert.Equal(t, "custom_field_definitions", def.GetTableName())
}

func TestCustomFieldDefinition_GetPostgresSearchConfig(t *testing.T) {
	t.Parallel()

	def := &CustomFieldDefinition{}
	config := def.GetPostgresSearchConfig()

	assert.Equal(t, "cfd", config.TableAlias)
	assert.False(t, config.UseSearchVector)
	assert.Len(t, config.SearchableFields, 4)
}

func hasErrorForField(multiErr *errortypes.MultiError, field string) bool {
	for _, e := range multiErr.Errors {
		if e.Field == field {
			return true
		}
	}
	return false
}
