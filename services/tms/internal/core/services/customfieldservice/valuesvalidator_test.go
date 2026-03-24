package customfieldservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/customfield"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type mockCustomFieldRepository struct {
	mock.Mock
}

func (m *mockCustomFieldRepository) List(
	ctx context.Context,
	req *repositories.ListCustomFieldDefinitionsRequest,
) (*pagination.ListResult[*customfield.CustomFieldDefinition], error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*pagination.ListResult[*customfield.CustomFieldDefinition]), args.Error(1)
}

func (m *mockCustomFieldRepository) GetByID(
	ctx context.Context,
	req repositories.GetCustomFieldDefinitionByIDRequest,
) (*customfield.CustomFieldDefinition, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*customfield.CustomFieldDefinition), args.Error(1)
}

func (m *mockCustomFieldRepository) GetActiveByResourceType(
	ctx context.Context,
	req repositories.GetActiveByResourceTypeRequest,
) ([]*customfield.CustomFieldDefinition, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*customfield.CustomFieldDefinition), args.Error(1)
}

func (m *mockCustomFieldRepository) Create(
	ctx context.Context,
	entity *customfield.CustomFieldDefinition,
) (*customfield.CustomFieldDefinition, error) {
	args := m.Called(ctx, entity)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*customfield.CustomFieldDefinition), args.Error(1)
}

func (m *mockCustomFieldRepository) Update(
	ctx context.Context,
	entity *customfield.CustomFieldDefinition,
) (*customfield.CustomFieldDefinition, error) {
	args := m.Called(ctx, entity)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*customfield.CustomFieldDefinition), args.Error(1)
}

func (m *mockCustomFieldRepository) Delete(
	ctx context.Context,
	req repositories.GetCustomFieldDefinitionByIDRequest,
) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

func (m *mockCustomFieldRepository) CountByResourceType(
	ctx context.Context,
	req repositories.CountByResourceTypeRequest,
) (int, error) {
	args := m.Called(ctx, req)
	return args.Int(0), args.Error(1)
}

func newTestValuesValidator(repo repositories.CustomFieldDefinitionRepository) *ValuesValidator {
	return &ValuesValidator{
		l:    zap.NewNop(),
		repo: repo,
	}
}

func newTenantInfo() pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}
}

func TestValidate_NoCustomFields_NoDefinitions(t *testing.T) {
	t.Parallel()

	repo := new(mockCustomFieldRepository)
	validator := newTestValuesValidator(repo)
	tenantInfo := newTenantInfo()

	repo.On("GetActiveByResourceType", mock.Anything, repositories.GetActiveByResourceTypeRequest{
		TenantInfo:   tenantInfo,
		ResourceType: "trailer",
	}).Return([]*customfield.CustomFieldDefinition{}, nil)

	customFields := map[string]any{}

	multiErr := errortypes.NewMultiError()
	validator.Validate(t.Context(), tenantInfo, "trailer", customFields, multiErr)

	assert.False(t, multiErr.HasErrors())
	repo.AssertExpectations(t)
}

func TestValidate_RequiredFieldMissing(t *testing.T) {
	t.Parallel()

	repo := new(mockCustomFieldRepository)
	validator := newTestValuesValidator(repo)
	tenantInfo := newTenantInfo()

	defID := pulid.MustNew("cfd_")
	definitions := []*customfield.CustomFieldDefinition{
		{
			ID:         defID,
			Name:       "required_field",
			Label:      "Required Field",
			FieldType:  customfield.FieldTypeText,
			IsRequired: true,
			IsActive:   true,
		},
	}

	repo.On("GetActiveByResourceType", mock.Anything, repositories.GetActiveByResourceTypeRequest{
		TenantInfo:   tenantInfo,
		ResourceType: "trailer",
	}).Return(definitions, nil)

	customFields := map[string]any{}

	multiErr := errortypes.NewMultiError()
	validator.Validate(t.Context(), tenantInfo, "trailer", customFields, multiErr)

	require.True(t, multiErr.HasErrors())
	assert.Len(t, multiErr.Errors, 1)
	assert.Equal(t, errortypes.ErrRequired, multiErr.Errors[0].Code)
	repo.AssertExpectations(t)
}

func TestValidate_RequiredFieldPresent(t *testing.T) {
	t.Parallel()

	repo := new(mockCustomFieldRepository)
	validator := newTestValuesValidator(repo)
	tenantInfo := newTenantInfo()

	defID := pulid.MustNew("cfd_")
	definitions := []*customfield.CustomFieldDefinition{
		{
			ID:         defID,
			Name:       "required_field",
			Label:      "Required Field",
			FieldType:  customfield.FieldTypeText,
			IsRequired: true,
			IsActive:   true,
		},
	}

	repo.On("GetActiveByResourceType", mock.Anything, repositories.GetActiveByResourceTypeRequest{
		TenantInfo:   tenantInfo,
		ResourceType: "trailer",
	}).Return(definitions, nil)

	customFields := map[string]any{
		defID.String(): "some value",
	}

	multiErr := errortypes.NewMultiError()
	validator.Validate(t.Context(), tenantInfo, "trailer", customFields, multiErr)

	assert.False(t, multiErr.HasErrors())
	repo.AssertExpectations(t)
}

func TestValidate_UnknownFieldID(t *testing.T) {
	t.Parallel()

	repo := new(mockCustomFieldRepository)
	validator := newTestValuesValidator(repo)
	tenantInfo := newTenantInfo()

	repo.On("GetActiveByResourceType", mock.Anything, repositories.GetActiveByResourceTypeRequest{
		TenantInfo:   tenantInfo,
		ResourceType: "trailer",
	}).Return([]*customfield.CustomFieldDefinition{}, nil)

	customFields := map[string]any{
		"unknown_field_id": "value",
	}

	multiErr := errortypes.NewMultiError()
	validator.Validate(t.Context(), tenantInfo, "trailer", customFields, multiErr)

	require.True(t, multiErr.HasErrors())
	assert.Equal(t, errortypes.ErrInvalid, multiErr.Errors[0].Code)
	repo.AssertExpectations(t)
}

func TestValidate_RepositoryError(t *testing.T) {
	t.Parallel()

	repo := new(mockCustomFieldRepository)
	validator := newTestValuesValidator(repo)
	tenantInfo := newTenantInfo()

	repo.On("GetActiveByResourceType", mock.Anything, repositories.GetActiveByResourceTypeRequest{
		TenantInfo:   tenantInfo,
		ResourceType: "trailer",
	}).Return(nil, errors.New("database error"))

	customFields := map[string]any{}

	multiErr := errortypes.NewMultiError()
	validator.Validate(t.Context(), tenantInfo, "trailer", customFields, multiErr)

	require.True(t, multiErr.HasErrors())
	assert.Equal(t, errortypes.ErrSystemError, multiErr.Errors[0].Code)
	repo.AssertExpectations(t)
}

func TestValidateText_Success(t *testing.T) {
	t.Parallel()

	repo := new(mockCustomFieldRepository)
	validator := newTestValuesValidator(repo)
	tenantInfo := newTenantInfo()

	defID := pulid.MustNew("cfd_")
	definitions := []*customfield.CustomFieldDefinition{
		{
			ID:        defID,
			Name:      "text_field",
			Label:     "Text Field",
			FieldType: customfield.FieldTypeText,
			IsActive:  true,
		},
	}

	repo.On("GetActiveByResourceType", mock.Anything, mock.Anything).Return(definitions, nil)

	customFields := map[string]any{
		defID.String(): "valid text",
	}

	multiErr := errortypes.NewMultiError()
	validator.Validate(t.Context(), tenantInfo, "trailer", customFields, multiErr)

	assert.False(t, multiErr.HasErrors())
}

func TestValidateText_InvalidType(t *testing.T) {
	t.Parallel()

	repo := new(mockCustomFieldRepository)
	validator := newTestValuesValidator(repo)
	tenantInfo := newTenantInfo()

	defID := pulid.MustNew("cfd_")
	definitions := []*customfield.CustomFieldDefinition{
		{
			ID:        defID,
			Name:      "text_field",
			Label:     "Text Field",
			FieldType: customfield.FieldTypeText,
			IsActive:  true,
		},
	}

	repo.On("GetActiveByResourceType", mock.Anything, mock.Anything).Return(definitions, nil)

	customFields := map[string]any{
		defID.String(): 123,
	}

	multiErr := errortypes.NewMultiError()
	validator.Validate(t.Context(), tenantInfo, "trailer", customFields, multiErr)

	require.True(t, multiErr.HasErrors())
	assert.Equal(t, errortypes.ErrInvalid, multiErr.Errors[0].Code)
}

func TestValidateText_MinLength(t *testing.T) {
	t.Parallel()

	repo := new(mockCustomFieldRepository)
	validator := newTestValuesValidator(repo)
	tenantInfo := newTenantInfo()

	minLen := 5
	defID := pulid.MustNew("cfd_")
	definitions := []*customfield.CustomFieldDefinition{
		{
			ID:              defID,
			Name:            "text_field",
			Label:           "Text Field",
			FieldType:       customfield.FieldTypeText,
			IsActive:        true,
			ValidationRules: &customfield.ValidationRules{MinLength: &minLen},
		},
	}

	repo.On("GetActiveByResourceType", mock.Anything, mock.Anything).Return(definitions, nil)

	customFields := map[string]any{
		defID.String(): "ab",
	}

	multiErr := errortypes.NewMultiError()
	validator.Validate(t.Context(), tenantInfo, "trailer", customFields, multiErr)

	require.True(t, multiErr.HasErrors())
	assert.Contains(t, multiErr.Errors[0].Message, "at least 5 characters")
}

func TestValidateText_MaxLength(t *testing.T) {
	t.Parallel()

	repo := new(mockCustomFieldRepository)
	validator := newTestValuesValidator(repo)
	tenantInfo := newTenantInfo()

	maxLen := 5
	defID := pulid.MustNew("cfd_")
	definitions := []*customfield.CustomFieldDefinition{
		{
			ID:              defID,
			Name:            "text_field",
			Label:           "Text Field",
			FieldType:       customfield.FieldTypeText,
			IsActive:        true,
			ValidationRules: &customfield.ValidationRules{MaxLength: &maxLen},
		},
	}

	repo.On("GetActiveByResourceType", mock.Anything, mock.Anything).Return(definitions, nil)

	customFields := map[string]any{
		defID.String(): "this is too long",
	}

	multiErr := errortypes.NewMultiError()
	validator.Validate(t.Context(), tenantInfo, "trailer", customFields, multiErr)

	require.True(t, multiErr.HasErrors())
	assert.Contains(t, multiErr.Errors[0].Message, "at most 5 characters")
}

func TestValidateText_Pattern(t *testing.T) {
	t.Parallel()

	repo := new(mockCustomFieldRepository)
	validator := newTestValuesValidator(repo)
	tenantInfo := newTenantInfo()

	pattern := "^[A-Z]{3}$"
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
		defID.String(): "abc",
	}

	multiErr := errortypes.NewMultiError()
	validator.Validate(t.Context(), tenantInfo, "trailer", customFields, multiErr)

	require.True(t, multiErr.HasErrors())
	assert.Contains(t, multiErr.Errors[0].Message, "pattern")
}

func TestValidateNumber_Success_Int(t *testing.T) {
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
		defID.String(): 42,
	}

	multiErr := errortypes.NewMultiError()
	validator.Validate(t.Context(), tenantInfo, "trailer", customFields, multiErr)

	assert.False(t, multiErr.HasErrors())
}

func TestValidateNumber_Success_Float(t *testing.T) {
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
		defID.String(): 42.5,
	}

	multiErr := errortypes.NewMultiError()
	validator.Validate(t.Context(), tenantInfo, "trailer", customFields, multiErr)

	assert.False(t, multiErr.HasErrors())
}

func TestValidateNumber_InvalidType(t *testing.T) {
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
		defID.String(): "not a number",
	}

	multiErr := errortypes.NewMultiError()
	validator.Validate(t.Context(), tenantInfo, "trailer", customFields, multiErr)

	require.True(t, multiErr.HasErrors())
	assert.Equal(t, errortypes.ErrInvalid, multiErr.Errors[0].Code)
}

func TestValidateNumber_BelowMin(t *testing.T) {
	t.Parallel()

	repo := new(mockCustomFieldRepository)
	validator := newTestValuesValidator(repo)
	tenantInfo := newTenantInfo()

	minVal := 10
	defID := pulid.MustNew("cfd_")
	definitions := []*customfield.CustomFieldDefinition{
		{
			ID:              defID,
			Name:            "number_field",
			Label:           "Number Field",
			FieldType:       customfield.FieldTypeNumber,
			IsActive:        true,
			ValidationRules: &customfield.ValidationRules{Min: &minVal},
		},
	}

	repo.On("GetActiveByResourceType", mock.Anything, mock.Anything).Return(definitions, nil)

	customFields := map[string]any{
		defID.String(): float64(5),
	}

	multiErr := errortypes.NewMultiError()
	validator.Validate(t.Context(), tenantInfo, "trailer", customFields, multiErr)

	require.True(t, multiErr.HasErrors())
	assert.Contains(t, multiErr.Errors[0].Message, "at least 10")
}

func TestValidateNumber_AboveMax(t *testing.T) {
	t.Parallel()

	repo := new(mockCustomFieldRepository)
	validator := newTestValuesValidator(repo)
	tenantInfo := newTenantInfo()

	maxVal := 100
	defID := pulid.MustNew("cfd_")
	definitions := []*customfield.CustomFieldDefinition{
		{
			ID:              defID,
			Name:            "number_field",
			Label:           "Number Field",
			FieldType:       customfield.FieldTypeNumber,
			IsActive:        true,
			ValidationRules: &customfield.ValidationRules{Max: &maxVal},
		},
	}

	repo.On("GetActiveByResourceType", mock.Anything, mock.Anything).Return(definitions, nil)

	customFields := map[string]any{
		defID.String(): float64(150),
	}

	multiErr := errortypes.NewMultiError()
	validator.Validate(t.Context(), tenantInfo, "trailer", customFields, multiErr)

	require.True(t, multiErr.HasErrors())
	assert.Contains(t, multiErr.Errors[0].Message, "at most 100")
}

func TestValidateDate_Success_RFC3339(t *testing.T) {
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
		defID.String(): "2024-01-15T10:30:00Z",
	}

	multiErr := errortypes.NewMultiError()
	validator.Validate(t.Context(), tenantInfo, "trailer", customFields, multiErr)

	assert.False(t, multiErr.HasErrors())
}

func TestValidateDate_Success_DateOnly(t *testing.T) {
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
		defID.String(): "2024-01-15",
	}

	multiErr := errortypes.NewMultiError()
	validator.Validate(t.Context(), tenantInfo, "trailer", customFields, multiErr)

	assert.False(t, multiErr.HasErrors())
}

func TestValidateDate_Success_Timestamp(t *testing.T) {
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
		defID.String(): float64(1705312200),
	}

	multiErr := errortypes.NewMultiError()
	validator.Validate(t.Context(), tenantInfo, "trailer", customFields, multiErr)

	assert.False(t, multiErr.HasErrors())
}

func TestValidateDate_InvalidFormat(t *testing.T) {
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
		defID.String(): "01-15-2024",
	}

	multiErr := errortypes.NewMultiError()
	validator.Validate(t.Context(), tenantInfo, "trailer", customFields, multiErr)

	require.True(t, multiErr.HasErrors())
	assert.Contains(t, multiErr.Errors[0].Message, "Invalid date format")
}

func TestValidateBoolean_Success(t *testing.T) {
	t.Parallel()

	repo := new(mockCustomFieldRepository)
	validator := newTestValuesValidator(repo)
	tenantInfo := newTenantInfo()

	defID := pulid.MustNew("cfd_")
	definitions := []*customfield.CustomFieldDefinition{
		{
			ID:        defID,
			Name:      "bool_field",
			Label:     "Bool Field",
			FieldType: customfield.FieldTypeBoolean,
			IsActive:  true,
		},
	}

	repo.On("GetActiveByResourceType", mock.Anything, mock.Anything).Return(definitions, nil)

	customFields := map[string]any{
		defID.String(): true,
	}

	multiErr := errortypes.NewMultiError()
	validator.Validate(t.Context(), tenantInfo, "trailer", customFields, multiErr)

	assert.False(t, multiErr.HasErrors())
}

func TestValidateBoolean_InvalidType(t *testing.T) {
	t.Parallel()

	repo := new(mockCustomFieldRepository)
	validator := newTestValuesValidator(repo)
	tenantInfo := newTenantInfo()

	defID := pulid.MustNew("cfd_")
	definitions := []*customfield.CustomFieldDefinition{
		{
			ID:        defID,
			Name:      "bool_field",
			Label:     "Bool Field",
			FieldType: customfield.FieldTypeBoolean,
			IsActive:  true,
		},
	}

	repo.On("GetActiveByResourceType", mock.Anything, mock.Anything).Return(definitions, nil)

	customFields := map[string]any{
		defID.String(): "true",
	}

	multiErr := errortypes.NewMultiError()
	validator.Validate(t.Context(), tenantInfo, "trailer", customFields, multiErr)

	require.True(t, multiErr.HasErrors())
	assert.Contains(t, multiErr.Errors[0].Message, "boolean")
}

func TestValidateSelect_Success(t *testing.T) {
	t.Parallel()

	repo := new(mockCustomFieldRepository)
	validator := newTestValuesValidator(repo)
	tenantInfo := newTenantInfo()

	defID := pulid.MustNew("cfd_")
	definitions := []*customfield.CustomFieldDefinition{
		{
			ID:        defID,
			Name:      "select_field",
			Label:     "Select Field",
			FieldType: customfield.FieldTypeSelect,
			IsActive:  true,
			Options: []customfield.SelectOption{
				{Value: "option1", Label: "Option 1"},
				{Value: "option2", Label: "Option 2"},
			},
		},
	}

	repo.On("GetActiveByResourceType", mock.Anything, mock.Anything).Return(definitions, nil)

	customFields := map[string]any{
		defID.String(): "option1",
	}

	multiErr := errortypes.NewMultiError()
	validator.Validate(t.Context(), tenantInfo, "trailer", customFields, multiErr)

	assert.False(t, multiErr.HasErrors())
}

func TestValidateSelect_InvalidOption(t *testing.T) {
	t.Parallel()

	repo := new(mockCustomFieldRepository)
	validator := newTestValuesValidator(repo)
	tenantInfo := newTenantInfo()

	defID := pulid.MustNew("cfd_")
	definitions := []*customfield.CustomFieldDefinition{
		{
			ID:        defID,
			Name:      "select_field",
			Label:     "Select Field",
			FieldType: customfield.FieldTypeSelect,
			IsActive:  true,
			Options: []customfield.SelectOption{
				{Value: "option1", Label: "Option 1"},
				{Value: "option2", Label: "Option 2"},
			},
		},
	}

	repo.On("GetActiveByResourceType", mock.Anything, mock.Anything).Return(definitions, nil)

	customFields := map[string]any{
		defID.String(): "invalid_option",
	}

	multiErr := errortypes.NewMultiError()
	validator.Validate(t.Context(), tenantInfo, "trailer", customFields, multiErr)

	require.True(t, multiErr.HasErrors())
	assert.Contains(t, multiErr.Errors[0].Message, "not a valid option")
}

func TestValidateSelect_InvalidType(t *testing.T) {
	t.Parallel()

	repo := new(mockCustomFieldRepository)
	validator := newTestValuesValidator(repo)
	tenantInfo := newTenantInfo()

	defID := pulid.MustNew("cfd_")
	definitions := []*customfield.CustomFieldDefinition{
		{
			ID:        defID,
			Name:      "select_field",
			Label:     "Select Field",
			FieldType: customfield.FieldTypeSelect,
			IsActive:  true,
			Options: []customfield.SelectOption{
				{Value: "option1", Label: "Option 1"},
			},
		},
	}

	repo.On("GetActiveByResourceType", mock.Anything, mock.Anything).Return(definitions, nil)

	customFields := map[string]any{
		defID.String(): 123,
	}

	multiErr := errortypes.NewMultiError()
	validator.Validate(t.Context(), tenantInfo, "trailer", customFields, multiErr)

	require.True(t, multiErr.HasErrors())
	assert.Equal(t, errortypes.ErrInvalid, multiErr.Errors[0].Code)
}

func TestValidateMultiSelect_Success(t *testing.T) {
	t.Parallel()

	repo := new(mockCustomFieldRepository)
	validator := newTestValuesValidator(repo)
	tenantInfo := newTenantInfo()

	defID := pulid.MustNew("cfd_")
	definitions := []*customfield.CustomFieldDefinition{
		{
			ID:        defID,
			Name:      "multi_select_field",
			Label:     "Multi Select Field",
			FieldType: customfield.FieldTypeMultiSelect,
			IsActive:  true,
			Options: []customfield.SelectOption{
				{Value: "option1", Label: "Option 1"},
				{Value: "option2", Label: "Option 2"},
				{Value: "option3", Label: "Option 3"},
			},
		},
	}

	repo.On("GetActiveByResourceType", mock.Anything, mock.Anything).Return(definitions, nil)

	customFields := map[string]any{
		defID.String(): []any{"option1", "option3"},
	}

	multiErr := errortypes.NewMultiError()
	validator.Validate(t.Context(), tenantInfo, "trailer", customFields, multiErr)

	assert.False(t, multiErr.HasErrors())
}

func TestValidateMultiSelect_InvalidOption(t *testing.T) {
	t.Parallel()

	repo := new(mockCustomFieldRepository)
	validator := newTestValuesValidator(repo)
	tenantInfo := newTenantInfo()

	defID := pulid.MustNew("cfd_")
	definitions := []*customfield.CustomFieldDefinition{
		{
			ID:        defID,
			Name:      "multi_select_field",
			Label:     "Multi Select Field",
			FieldType: customfield.FieldTypeMultiSelect,
			IsActive:  true,
			Options: []customfield.SelectOption{
				{Value: "option1", Label: "Option 1"},
				{Value: "option2", Label: "Option 2"},
			},
		},
	}

	repo.On("GetActiveByResourceType", mock.Anything, mock.Anything).Return(definitions, nil)

	customFields := map[string]any{
		defID.String(): []any{"option1", "invalid_option"},
	}

	multiErr := errortypes.NewMultiError()
	validator.Validate(t.Context(), tenantInfo, "trailer", customFields, multiErr)

	require.True(t, multiErr.HasErrors())
	assert.Contains(t, multiErr.Errors[0].Message, "not a valid option")
}

func TestValidateMultiSelect_InvalidType(t *testing.T) {
	t.Parallel()

	repo := new(mockCustomFieldRepository)
	validator := newTestValuesValidator(repo)
	tenantInfo := newTenantInfo()

	defID := pulid.MustNew("cfd_")
	definitions := []*customfield.CustomFieldDefinition{
		{
			ID:        defID,
			Name:      "multi_select_field",
			Label:     "Multi Select Field",
			FieldType: customfield.FieldTypeMultiSelect,
			IsActive:  true,
			Options: []customfield.SelectOption{
				{Value: "option1", Label: "Option 1"},
			},
		},
	}

	repo.On("GetActiveByResourceType", mock.Anything, mock.Anything).Return(definitions, nil)

	customFields := map[string]any{
		defID.String(): "not an array",
	}

	multiErr := errortypes.NewMultiError()
	validator.Validate(t.Context(), tenantInfo, "trailer", customFields, multiErr)

	require.True(t, multiErr.HasErrors())
	assert.Contains(t, multiErr.Errors[0].Message, "array")
}

func TestValidateMultiSelect_EmptyArray(t *testing.T) {
	t.Parallel()

	repo := new(mockCustomFieldRepository)
	validator := newTestValuesValidator(repo)
	tenantInfo := newTenantInfo()

	defID := pulid.MustNew("cfd_")
	definitions := []*customfield.CustomFieldDefinition{
		{
			ID:        defID,
			Name:      "multi_select_field",
			Label:     "Multi Select Field",
			FieldType: customfield.FieldTypeMultiSelect,
			IsActive:  true,
			Options: []customfield.SelectOption{
				{Value: "option1", Label: "Option 1"},
			},
		},
	}

	repo.On("GetActiveByResourceType", mock.Anything, mock.Anything).Return(definitions, nil)

	customFields := map[string]any{
		defID.String(): []any{},
	}

	multiErr := errortypes.NewMultiError()
	validator.Validate(t.Context(), tenantInfo, "trailer", customFields, multiErr)

	assert.False(t, multiErr.HasErrors())
}

func TestValidate_NilValue_Required(t *testing.T) {
	t.Parallel()

	repo := new(mockCustomFieldRepository)
	validator := newTestValuesValidator(repo)
	tenantInfo := newTenantInfo()

	defID := pulid.MustNew("cfd_")
	definitions := []*customfield.CustomFieldDefinition{
		{
			ID:         defID,
			Name:       "required_field",
			Label:      "Required Field",
			FieldType:  customfield.FieldTypeText,
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

func TestValidate_NilValue_NotRequired(t *testing.T) {
	t.Parallel()

	repo := new(mockCustomFieldRepository)
	validator := newTestValuesValidator(repo)
	tenantInfo := newTenantInfo()

	defID := pulid.MustNew("cfd_")
	definitions := []*customfield.CustomFieldDefinition{
		{
			ID:         defID,
			Name:       "optional_field",
			Label:      "Optional Field",
			FieldType:  customfield.FieldTypeText,
			IsRequired: false,
			IsActive:   true,
		},
	}

	repo.On("GetActiveByResourceType", mock.Anything, mock.Anything).Return(definitions, nil)

	customFields := map[string]any{
		defID.String(): nil,
	}

	multiErr := errortypes.NewMultiError()
	validator.Validate(t.Context(), tenantInfo, "trailer", customFields, multiErr)

	assert.False(t, multiErr.HasErrors())
}
