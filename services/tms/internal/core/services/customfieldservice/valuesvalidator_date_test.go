package customfieldservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/customfield"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestValidateDate_NegativeFloat64Timestamp(t *testing.T) {
	t.Parallel()

	repo := new(mockCustomFieldRepository)
	validator := newTestValuesValidator(repo)
	tenantInfo := newTenantInfo()

	defID := pulid.MustNew("cfd_")
	definitions := []*customfield.CustomFieldDefinition{
		{
			ID:        defID,
			Name:      "date_field",
			Label:     "Date Field",
			FieldType: customfield.FieldTypeDate,
			IsActive:  true,
		},
	}

	repo.On("GetActiveByResourceType", mock.Anything, mock.Anything).Return(definitions, nil)

	customFields := map[string]any{
		defID.String(): float64(-100),
	}

	multiErr := errortypes.NewMultiError()
	validator.Validate(t.Context(), tenantInfo, "trailer", customFields, multiErr)

	require.True(t, multiErr.HasErrors())
	assert.Contains(t, multiErr.Errors[0].Message, "Invalid timestamp")
}

func TestValidateDate_ZeroFloat64Timestamp(t *testing.T) {
	t.Parallel()

	repo := new(mockCustomFieldRepository)
	validator := newTestValuesValidator(repo)
	tenantInfo := newTenantInfo()

	defID := pulid.MustNew("cfd_")
	definitions := []*customfield.CustomFieldDefinition{
		{
			ID:        defID,
			Name:      "date_field",
			Label:     "Date Field",
			FieldType: customfield.FieldTypeDate,
			IsActive:  true,
		},
	}

	repo.On("GetActiveByResourceType", mock.Anything, mock.Anything).Return(definitions, nil)

	customFields := map[string]any{
		defID.String(): float64(0),
	}

	multiErr := errortypes.NewMultiError()
	validator.Validate(t.Context(), tenantInfo, "trailer", customFields, multiErr)

	assert.False(t, multiErr.HasErrors())
}

func TestValidateDate_Int64Timestamp(t *testing.T) {
	t.Parallel()

	repo := new(mockCustomFieldRepository)
	validator := newTestValuesValidator(repo)
	tenantInfo := newTenantInfo()

	defID := pulid.MustNew("cfd_")
	definitions := []*customfield.CustomFieldDefinition{
		{
			ID:        defID,
			Name:      "date_field",
			Label:     "Date Field",
			FieldType: customfield.FieldTypeDate,
			IsActive:  true,
		},
	}

	repo.On("GetActiveByResourceType", mock.Anything, mock.Anything).Return(definitions, nil)

	customFields := map[string]any{
		defID.String(): int64(1705312200),
	}

	multiErr := errortypes.NewMultiError()
	validator.Validate(t.Context(), tenantInfo, "trailer", customFields, multiErr)

	assert.False(t, multiErr.HasErrors())
}

func TestValidateDate_NegativeInt64Timestamp(t *testing.T) {
	t.Parallel()

	repo := new(mockCustomFieldRepository)
	validator := newTestValuesValidator(repo)
	tenantInfo := newTenantInfo()

	defID := pulid.MustNew("cfd_")
	definitions := []*customfield.CustomFieldDefinition{
		{
			ID:        defID,
			Name:      "date_field",
			Label:     "Date Field",
			FieldType: customfield.FieldTypeDate,
			IsActive:  true,
		},
	}

	repo.On("GetActiveByResourceType", mock.Anything, mock.Anything).Return(definitions, nil)

	customFields := map[string]any{
		defID.String(): int64(-1),
	}

	multiErr := errortypes.NewMultiError()
	validator.Validate(t.Context(), tenantInfo, "trailer", customFields, multiErr)

	require.True(t, multiErr.HasErrors())
	assert.Contains(t, multiErr.Errors[0].Message, "Invalid timestamp")
}

func TestValidateDate_ZeroInt64Timestamp(t *testing.T) {
	t.Parallel()

	repo := new(mockCustomFieldRepository)
	validator := newTestValuesValidator(repo)
	tenantInfo := newTenantInfo()

	defID := pulid.MustNew("cfd_")
	definitions := []*customfield.CustomFieldDefinition{
		{
			ID:        defID,
			Name:      "date_field",
			Label:     "Date Field",
			FieldType: customfield.FieldTypeDate,
			IsActive:  true,
		},
	}

	repo.On("GetActiveByResourceType", mock.Anything, mock.Anything).Return(definitions, nil)

	customFields := map[string]any{
		defID.String(): int64(0),
	}

	multiErr := errortypes.NewMultiError()
	validator.Validate(t.Context(), tenantInfo, "trailer", customFields, multiErr)

	assert.False(t, multiErr.HasErrors())
}

func TestValidateDate_InvalidType_Bool(t *testing.T) {
	t.Parallel()

	repo := new(mockCustomFieldRepository)
	validator := newTestValuesValidator(repo)
	tenantInfo := newTenantInfo()

	defID := pulid.MustNew("cfd_")
	definitions := []*customfield.CustomFieldDefinition{
		{
			ID:        defID,
			Name:      "date_field",
			Label:     "Date Field",
			FieldType: customfield.FieldTypeDate,
			IsActive:  true,
		},
	}

	repo.On("GetActiveByResourceType", mock.Anything, mock.Anything).Return(definitions, nil)

	customFields := map[string]any{
		defID.String(): true,
	}

	multiErr := errortypes.NewMultiError()
	validator.Validate(t.Context(), tenantInfo, "trailer", customFields, multiErr)

	require.True(t, multiErr.HasErrors())
	assert.Contains(t, multiErr.Errors[0].Message, "Date must be a string")
}

func TestValidateDate_InvalidType_Int(t *testing.T) {
	t.Parallel()

	repo := new(mockCustomFieldRepository)
	validator := newTestValuesValidator(repo)
	tenantInfo := newTenantInfo()

	defID := pulid.MustNew("cfd_")
	definitions := []*customfield.CustomFieldDefinition{
		{
			ID:        defID,
			Name:      "date_field",
			Label:     "Date Field",
			FieldType: customfield.FieldTypeDate,
			IsActive:  true,
		},
	}

	repo.On("GetActiveByResourceType", mock.Anything, mock.Anything).Return(definitions, nil)

	customFields := map[string]any{
		defID.String(): 42,
	}

	multiErr := errortypes.NewMultiError()
	validator.Validate(t.Context(), tenantInfo, "trailer", customFields, multiErr)

	require.True(t, multiErr.HasErrors())
	assert.Contains(t, multiErr.Errors[0].Message, "Date must be a string")
}

func TestValidateDate_InvalidType_Slice(t *testing.T) {
	t.Parallel()

	repo := new(mockCustomFieldRepository)
	validator := newTestValuesValidator(repo)
	tenantInfo := newTenantInfo()

	defID := pulid.MustNew("cfd_")
	definitions := []*customfield.CustomFieldDefinition{
		{
			ID:        defID,
			Name:      "date_field",
			Label:     "Date Field",
			FieldType: customfield.FieldTypeDate,
			IsActive:  true,
		},
	}

	repo.On("GetActiveByResourceType", mock.Anything, mock.Anything).Return(definitions, nil)

	customFields := map[string]any{
		defID.String(): []any{"2024-01-01"},
	}

	multiErr := errortypes.NewMultiError()
	validator.Validate(t.Context(), tenantInfo, "trailer", customFields, multiErr)

	require.True(t, multiErr.HasErrors())
	assert.Contains(t, multiErr.Errors[0].Message, "Date must be a string")
}

func TestValidateDate_InvalidType_Map(t *testing.T) {
	t.Parallel()

	repo := new(mockCustomFieldRepository)
	validator := newTestValuesValidator(repo)
	tenantInfo := newTenantInfo()

	defID := pulid.MustNew("cfd_")
	definitions := []*customfield.CustomFieldDefinition{
		{
			ID:        defID,
			Name:      "date_field",
			Label:     "Date Field",
			FieldType: customfield.FieldTypeDate,
			IsActive:  true,
		},
	}

	repo.On("GetActiveByResourceType", mock.Anything, mock.Anything).Return(definitions, nil)

	customFields := map[string]any{
		defID.String(): map[string]any{"date": "2024-01-01"},
	}

	multiErr := errortypes.NewMultiError()
	validator.Validate(t.Context(), tenantInfo, "trailer", customFields, multiErr)

	require.True(t, multiErr.HasErrors())
	assert.Contains(t, multiErr.Errors[0].Message, "Date must be a string")
}

func TestValidateDate_RFC3339WithTimezone(t *testing.T) {
	t.Parallel()

	repo := new(mockCustomFieldRepository)
	validator := newTestValuesValidator(repo)
	tenantInfo := newTenantInfo()

	defID := pulid.MustNew("cfd_")
	definitions := []*customfield.CustomFieldDefinition{
		{
			ID:        defID,
			Name:      "date_field",
			Label:     "Date Field",
			FieldType: customfield.FieldTypeDate,
			IsActive:  true,
		},
	}

	repo.On("GetActiveByResourceType", mock.Anything, mock.Anything).Return(definitions, nil)

	customFields := map[string]any{
		defID.String(): "2024-01-15T10:30:00+05:00",
	}

	multiErr := errortypes.NewMultiError()
	validator.Validate(t.Context(), tenantInfo, "trailer", customFields, multiErr)

	assert.False(t, multiErr.HasErrors())
}

func TestValidateDate_StringInvalidFormats(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value string
	}{
		{"empty string", ""},
		{"random text", "not-a-date"},
		{"partial date", "2024-01"},
		{"wrong separator", "2024/01/15"},
		{"US format", "01-15-2024"},
		{"time only", "10:30:00"},
		{"epoch string", "1705312200"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := new(mockCustomFieldRepository)
			validator := newTestValuesValidator(repo)
			tenantInfo := newTenantInfo()

			defID := pulid.MustNew("cfd_")
			definitions := []*customfield.CustomFieldDefinition{
				{
					ID:        defID,
					Name:      "date_field",
					Label:     "Date Field",
					FieldType: customfield.FieldTypeDate,
					IsActive:  true,
				},
			}

			repo.On("GetActiveByResourceType", mock.Anything, mock.Anything).
				Return(definitions, nil)

			customFields := map[string]any{
				defID.String(): tt.value,
			}

			multiErr := errortypes.NewMultiError()
			validator.Validate(t.Context(), tenantInfo, "trailer", customFields, multiErr)

			require.True(t, multiErr.HasErrors())
			assert.Contains(t, multiErr.Errors[0].Message, "Invalid date format")
		})
	}
}

func TestValidateDate_LargePositiveFloat64(t *testing.T) {
	t.Parallel()

	repo := new(mockCustomFieldRepository)
	validator := newTestValuesValidator(repo)
	tenantInfo := newTenantInfo()

	defID := pulid.MustNew("cfd_")
	definitions := []*customfield.CustomFieldDefinition{
		{
			ID:        defID,
			Name:      "date_field",
			Label:     "Date Field",
			FieldType: customfield.FieldTypeDate,
			IsActive:  true,
		},
	}

	repo.On("GetActiveByResourceType", mock.Anything, mock.Anything).Return(definitions, nil)

	customFields := map[string]any{
		defID.String(): float64(99999999999),
	}

	multiErr := errortypes.NewMultiError()
	validator.Validate(t.Context(), tenantInfo, "trailer", customFields, multiErr)

	assert.False(t, multiErr.HasErrors())
}

func TestValidateMultiSelect_NonStringItem(t *testing.T) {
	t.Parallel()

	repo := new(mockCustomFieldRepository)
	validator := newTestValuesValidator(repo)
	tenantInfo := newTenantInfo()

	defID := pulid.MustNew("cfd_")
	definitions := []*customfield.CustomFieldDefinition{
		{
			ID:        defID,
			Name:      "multi_field",
			Label:     "Multi Field",
			FieldType: customfield.FieldTypeMultiSelect,
			IsActive:  true,
			Options: []customfield.SelectOption{
				{Value: "opt1", Label: "Option 1"},
			},
		},
	}

	repo.On("GetActiveByResourceType", mock.Anything, mock.Anything).Return(definitions, nil)

	customFields := map[string]any{
		defID.String(): []any{123, true},
	}

	multiErr := errortypes.NewMultiError()
	validator.Validate(t.Context(), tenantInfo, "trailer", customFields, multiErr)

	require.True(t, multiErr.HasErrors())
	assert.Equal(t, 2, len(multiErr.Errors))
}

func TestValidateNumber_Int64Type(t *testing.T) {
	t.Parallel()

	repo := new(mockCustomFieldRepository)
	validator := newTestValuesValidator(repo)
	tenantInfo := newTenantInfo()

	defID := pulid.MustNew("cfd_")
	definitions := []*customfield.CustomFieldDefinition{
		{
			ID:        defID,
			Name:      "number_field",
			Label:     "Number Field",
			FieldType: customfield.FieldTypeNumber,
			IsActive:  true,
		},
	}

	repo.On("GetActiveByResourceType", mock.Anything, mock.Anything).Return(definitions, nil)

	customFields := map[string]any{
		defID.String(): int64(42),
	}

	multiErr := errortypes.NewMultiError()
	validator.Validate(t.Context(), tenantInfo, "trailer", customFields, multiErr)

	assert.False(t, multiErr.HasErrors())
}

func TestValidateText_PatternMatchSuccess(t *testing.T) {
	t.Parallel()

	repo := new(mockCustomFieldRepository)
	validator := newTestValuesValidator(repo)
	tenantInfo := newTenantInfo()

	pattern := `^[A-Z]{3}$`
	defID := pulid.MustNew("cfd_")
	definitions := []*customfield.CustomFieldDefinition{
		{
			ID:              defID,
			Name:            "code_field",
			Label:           "Code Field",
			FieldType:       customfield.FieldTypeText,
			IsActive:        true,
			ValidationRules: &customfield.ValidationRules{Pattern: &pattern},
		},
	}

	repo.On("GetActiveByResourceType", mock.Anything, mock.Anything).Return(definitions, nil)

	customFields := map[string]any{
		defID.String(): "ABC",
	}

	multiErr := errortypes.NewMultiError()
	validator.Validate(t.Context(), tenantInfo, "trailer", customFields, multiErr)

	assert.False(t, multiErr.HasErrors())
}

func TestValidateText_InvalidPattern(t *testing.T) {
	t.Parallel()

	repo := new(mockCustomFieldRepository)
	validator := newTestValuesValidator(repo)
	tenantInfo := newTenantInfo()

	pattern := `[invalid`
	defID := pulid.MustNew("cfd_")
	definitions := []*customfield.CustomFieldDefinition{
		{
			ID:              defID,
			Name:            "text_field",
			Label:           "Text Field",
			FieldType:       customfield.FieldTypeText,
			IsActive:        true,
			ValidationRules: &customfield.ValidationRules{Pattern: &pattern},
		},
	}

	repo.On("GetActiveByResourceType", mock.Anything, mock.Anything).Return(definitions, nil)

	customFields := map[string]any{
		defID.String(): "some-value",
	}

	multiErr := errortypes.NewMultiError()
	validator.Validate(t.Context(), tenantInfo, "trailer", customFields, multiErr)

	require.True(t, multiErr.HasErrors())
	assert.Contains(t, multiErr.Errors[0].Message, "pattern")
}

func TestValidateText_EmptyPatternSkipsValidation(t *testing.T) {
	t.Parallel()

	repo := new(mockCustomFieldRepository)
	validator := newTestValuesValidator(repo)
	tenantInfo := newTenantInfo()

	pattern := ""
	defID := pulid.MustNew("cfd_")
	definitions := []*customfield.CustomFieldDefinition{
		{
			ID:              defID,
			Name:            "text_field",
			Label:           "Text Field",
			FieldType:       customfield.FieldTypeText,
			IsActive:        true,
			ValidationRules: &customfield.ValidationRules{Pattern: &pattern},
		},
	}

	repo.On("GetActiveByResourceType", mock.Anything, mock.Anything).Return(definitions, nil)

	customFields := map[string]any{
		defID.String(): "anything",
	}

	multiErr := errortypes.NewMultiError()
	validator.Validate(t.Context(), tenantInfo, "trailer", customFields, multiErr)

	assert.False(t, multiErr.HasErrors())
}

func TestValidateDate_Required_Nil(t *testing.T) {
	t.Parallel()

	repo := new(mockCustomFieldRepository)
	validator := newTestValuesValidator(repo)
	tenantInfo := newTenantInfo()

	defID := pulid.MustNew("cfd_")
	definitions := []*customfield.CustomFieldDefinition{
		{
			ID:         defID,
			Name:       "date_field",
			Label:      "Date Field",
			FieldType:  customfield.FieldTypeDate,
			IsRequired: true,
			IsActive:   true,
		},
	}

	repo.On("GetActiveByResourceType", mock.Anything, mock.Anything).Return(definitions, nil)

	customFields := map[string]any{
		defID.String(): nil,
	}

	multiErr := errortypes.NewMultiError()
	validator.Validate(t.Context(), tenantInfo, "trailer", customFields, multiErr)

	require.True(t, multiErr.HasErrors())
	assert.Equal(t, errortypes.ErrRequired, multiErr.Errors[0].Code)
}

func TestValidate_NoDefinitions_WithCustomFields(t *testing.T) {
	t.Parallel()

	repo := new(mockCustomFieldRepository)
	validator := newTestValuesValidator(repo)
	tenantInfo := newTenantInfo()

	repo.On("GetActiveByResourceType", mock.Anything, repositories.GetActiveByResourceTypeRequest{
		TenantInfo:   tenantInfo,
		ResourceType: "trailer",
	}).Return([]*customfield.CustomFieldDefinition{}, nil)

	customFields := map[string]any{
		"some_id": "value",
	}

	multiErr := errortypes.NewMultiError()
	validator.Validate(t.Context(), tenantInfo, "trailer", customFields, multiErr)

	require.True(t, multiErr.HasErrors())
	assert.Equal(t, errortypes.ErrInvalid, multiErr.Errors[0].Code)
}
