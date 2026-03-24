package validationframework

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestNewTenantedValidatorBuilder(t *testing.T) {
	t.Parallel()

	builder := NewTenantedValidatorBuilder[*mockTenantedEntity]()

	require.NotNil(t, builder)
	require.NotNil(t, builder.validator)
	assert.Equal(t, "Entity", builder.validator.modelName)
	assert.NotNil(t, builder.validator.uniqueFields)
	assert.NotNil(t, builder.validator.customRules)
	assert.NotNil(t, builder.validator.engineConfig)
}

func TestTenantedValidatorBuilder_WithModelName(t *testing.T) {
	t.Parallel()

	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithModelName("TestModel").
		Build()

	assert.Equal(t, "TestModel", validator.modelName)
}

func TestTenantedValidatorBuilder_WithUniquenessChecker(t *testing.T) {
	t.Parallel()

	checker := newMockUniquenessChecker()
	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithUniquenessChecker(checker).
		Build()

	assert.Equal(t, checker, validator.uniquenessChecker)
}

func TestTenantedValidatorBuilder_WithUniqueField(t *testing.T) {
	t.Parallel()

	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithUniqueField("name", "name", "Name must be unique", func(e *mockTenantedEntity) any {
			return e.Name
		}).
		Build()

	require.Len(t, validator.uniqueFields, 1)
	assert.Equal(t, "name", validator.uniqueFields[0].FieldName)
	assert.Equal(t, "name", validator.uniqueFields[0].Column)
	assert.Equal(t, "Name must be unique", validator.uniqueFields[0].Message)
	assert.False(t, validator.uniqueFields[0].CaseSensitive)
}

func TestTenantedValidatorBuilder_WithCaseSensitiveUniqueField(t *testing.T) {
	t.Parallel()

	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithCaseSensitiveUniqueField(
			"code",
			"code",
			"Code must be unique",
			func(e *mockTenantedEntity) any { return e.Name },
		).
		Build()

	require.Len(t, validator.uniqueFields, 1)
	assert.True(t, validator.uniqueFields[0].CaseSensitive)
}

func TestTenantedValidatorBuilder_WithCustomRule(t *testing.T) {
	t.Parallel()

	rule := NewTenantedRule[*mockTenantedEntity]("custom_rule")
	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithCustomRule(rule).
		Build()

	require.Len(t, validator.customRules, 1)
	assert.Equal(t, "custom_rule", validator.customRules[0].Name())
}

func TestTenantedValidatorBuilder_WithEngineConfig(t *testing.T) {
	t.Parallel()

	config := &EngineConfig{
		FailFast:    true,
		MaxParallel: 5,
	}
	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithEngineConfig(config).
		Build()

	assert.Equal(t, config, validator.engineConfig)
}

func TestTenantedValidatorBuilder_FluentChaining(t *testing.T) {
	t.Parallel()

	checker := newMockUniquenessChecker()
	rule := NewTenantedRule[*mockTenantedEntity]("custom_rule")

	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithModelName("TestModel").
		WithUniquenessChecker(checker).
		WithUniqueField("name", "name", "Name must be unique", func(e *mockTenantedEntity) any {
			return e.Name
		}).
		WithCustomRule(rule).
		Build()

	assert.Equal(t, "TestModel", validator.modelName)
	assert.Equal(t, checker, validator.uniquenessChecker)
	require.Len(t, validator.uniqueFields, 1)
	require.Len(t, validator.customRules, 1)
}

func TestTenantedValidator_ValidateCreate_Success(t *testing.T) {
	t.Parallel()

	entity := newMockTenantedEntity()
	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().Build()

	result := validator.ValidateCreate(t.Context(), entity)

	assert.Nil(t, result)
}

func TestTenantedValidator_ValidateCreate_DomainValidationError(t *testing.T) {
	t.Parallel()

	entity := newMockTenantedEntity()
	entity.validationErr = true

	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().Build()

	result := validator.ValidateCreate(t.Context(), entity)

	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
}

func TestTenantedValidator_ValidateCreate_IDMustBeNil(t *testing.T) {
	t.Parallel()

	entity := newMockTenantedEntity()
	entity.id = pulid.MustNew("ent_")

	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().Build()

	result := validator.ValidateCreate(t.Context(), entity)

	require.NotNil(t, result)
	assert.True(t, result.HasErrors())

	hasIDError := false
	for _, err := range result.Errors {
		if err.Field == "id" {
			hasIDError = true
			break
		}
	}
	assert.True(t, hasIDError)
}

func TestTenantedValidator_ValidateUpdate_Success(t *testing.T) {
	t.Parallel()

	entity := newMockTenantedEntity()
	entity.id = pulid.MustNew("ent_")

	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().Build()

	result := validator.ValidateUpdate(t.Context(), entity)

	assert.Nil(t, result)
}

func TestTenantedValidator_ValidateUpdate_NoIDValidation(t *testing.T) {
	t.Parallel()

	entity := newMockTenantedEntity()
	entity.id = pulid.MustNew("ent_")

	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().Build()

	result := validator.ValidateUpdate(t.Context(), entity)

	assert.Nil(t, result)
}

func TestTenantedValidator_ValidateCreate_UniquenessCheck_NoConflict(t *testing.T) {
	t.Parallel()

	entity := newMockTenantedEntity()
	checker := newMockUniquenessChecker()
	checker.On("CheckUniqueness", mock.Anything, mock.Anything).Return(false, nil)

	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithUniquenessChecker(checker).
		WithUniqueField("name", "name", "Name must be unique", func(e *mockTenantedEntity) any {
			return e.Name
		}).
		Build()

	result := validator.ValidateCreate(t.Context(), entity)

	assert.Nil(t, result)
	checker.AssertExpectations(t)
}

func TestTenantedValidator_ValidateCreate_UniquenessCheck_Conflict(t *testing.T) {
	t.Parallel()

	entity := newMockTenantedEntity()
	checker := newMockUniquenessChecker()
	checker.On("CheckUniqueness", mock.Anything, mock.Anything).Return(true, nil)

	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithUniquenessChecker(checker).
		WithUniqueField("name", "name", "Name must be unique", func(e *mockTenantedEntity) any {
			return e.Name
		}).
		Build()

	result := validator.ValidateCreate(t.Context(), entity)

	require.NotNil(t, result)
	assert.True(t, result.HasErrors())

	hasNameError := false
	for _, err := range result.Errors {
		if err.Field == "name" && err.Code == errortypes.ErrDuplicate {
			hasNameError = true
			assert.Equal(t, "Name must be unique", err.Message)
			break
		}
	}
	assert.True(t, hasNameError)
	checker.AssertExpectations(t)
}

func TestTenantedValidator_ValidateUpdate_UniquenessCheck_ExcludesCurrentEntity(t *testing.T) {
	t.Parallel()

	entity := newMockTenantedEntity()
	entity.id = pulid.MustNew("ent_")
	checker := newMockUniquenessChecker()

	checker.On("CheckUniqueness", mock.Anything, mock.MatchedBy(func(req *UniquenessRequest) bool {
		return req.ExcludeID == entity.id
	})).Return(false, nil)

	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithUniquenessChecker(checker).
		WithUniqueField("name", "name", "Name must be unique", func(e *mockTenantedEntity) any {
			return e.Name
		}).
		Build()

	result := validator.ValidateUpdate(t.Context(), entity)

	assert.Nil(t, result)
	checker.AssertExpectations(t)
}

func TestTenantedValidator_ValidateCreate_UniquenessCheck_Error(t *testing.T) {
	t.Parallel()

	entity := newMockTenantedEntity()
	checker := newMockUniquenessChecker()
	checker.On("CheckUniqueness", mock.Anything, mock.Anything).
		Return(false, errors.New("database error"))

	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithUniquenessChecker(checker).
		WithUniqueField("name", "name", "Name must be unique", func(e *mockTenantedEntity) any {
			return e.Name
		}).
		Build()

	result := validator.ValidateCreate(t.Context(), entity)

	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
	checker.AssertExpectations(t)
}

func TestTenantedValidator_ValidateCreate_SkipsEmptyUniqueFields(t *testing.T) {
	t.Parallel()

	entity := newMockTenantedEntity()
	entity.Name = ""
	checker := newMockUniquenessChecker()

	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithUniquenessChecker(checker).
		WithUniqueField("name", "name", "Name must be unique", func(e *mockTenantedEntity) any {
			return e.Name
		}).
		Build()

	result := validator.ValidateCreate(t.Context(), entity)

	assert.Nil(t, result)
	checker.AssertNotCalled(t, "CheckUniqueness", mock.Anything, mock.Anything)
}

func TestTenantedValidator_ValidateCreate_DefaultErrorMessage(t *testing.T) {
	t.Parallel()

	entity := newMockTenantedEntity()
	checker := newMockUniquenessChecker()
	checker.On("CheckUniqueness", mock.Anything, mock.Anything).Return(true, nil)

	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithModelName("TestEntity").
		WithUniquenessChecker(checker).
		WithUniqueField("name", "name", "", func(e *mockTenantedEntity) any {
			return e.Name
		}).
		Build()

	result := validator.ValidateCreate(t.Context(), entity)

	require.NotNil(t, result)
	assert.True(t, result.HasErrors())

	for _, err := range result.Errors {
		if err.Field == "name" {
			assert.Equal(t, "TestEntity with this name already exists", err.Message)
			break
		}
	}
	checker.AssertExpectations(t)
}

func TestTenantedValidator_CustomRules_OnCreate(t *testing.T) {
	t.Parallel()

	executed := false
	rule := NewTenantedRule[*mockTenantedEntity]("custom_create_rule").
		OnCreate().
		WithValidation(func(
			_ context.Context,
			_ *mockTenantedEntity,
			_ *TenantedValidationContext,
			_ *errortypes.MultiError,
		) error {
			executed = true
			return nil
		})

	entity := newMockTenantedEntity()
	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithCustomRule(rule).
		Build()

	_ = validator.ValidateCreate(t.Context(), entity)

	assert.True(t, executed)
}

func TestTenantedValidator_CustomRules_OnUpdate(t *testing.T) {
	t.Parallel()

	executed := false
	rule := NewTenantedRule[*mockTenantedEntity]("custom_update_rule").
		OnUpdate().
		WithValidation(func(
			_ context.Context,
			_ *mockTenantedEntity,
			_ *TenantedValidationContext,
			_ *errortypes.MultiError,
		) error {
			executed = true
			return nil
		})

	entity := newMockTenantedEntity()
	entity.id = pulid.MustNew("ent_")
	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithCustomRule(rule).
		Build()

	_ = validator.ValidateUpdate(t.Context(), entity)

	assert.True(t, executed)
}

func TestTenantedValidator_CustomRules_NotExecutedOnWrongMode(t *testing.T) {
	t.Parallel()

	executed := false
	rule := NewTenantedRule[*mockTenantedEntity]("custom_update_rule").
		OnUpdate().
		WithValidation(func(
			_ context.Context,
			_ *mockTenantedEntity,
			_ *TenantedValidationContext,
			_ *errortypes.MultiError,
		) error {
			executed = true
			return nil
		})

	entity := newMockTenantedEntity()
	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithCustomRule(rule).
		Build()

	_ = validator.ValidateCreate(t.Context(), entity)

	assert.False(t, executed)
}

func TestTenantedValidator_CustomRules_AddsErrors(t *testing.T) {
	t.Parallel()

	rule := NewTenantedRule[*mockTenantedEntity]("custom_rule").
		OnCreate().
		WithValidation(func(
			_ context.Context,
			_ *mockTenantedEntity,
			_ *TenantedValidationContext,
			multiErr *errortypes.MultiError,
		) error {
			multiErr.Add("custom", errortypes.ErrBusinessLogic, "Custom validation failed")
			return nil
		})

	entity := newMockTenantedEntity()
	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithCustomRule(rule).
		Build()

	result := validator.ValidateCreate(t.Context(), entity)

	require.NotNil(t, result)
	assert.True(t, result.HasErrors())

	hasCustomError := false
	for _, err := range result.Errors {
		if err.Field == "custom" {
			hasCustomError = true
			break
		}
	}
	assert.True(t, hasCustomError)
}

func TestTenantedValidator_MultipleUniqueFields(t *testing.T) {
	t.Parallel()

	entity := newMockTenantedEntity()
	checker := newMockUniquenessChecker()
	checker.On("CheckUniqueness", mock.Anything, mock.Anything).Return(false, nil)

	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithUniquenessChecker(checker).
		WithUniqueField("name", "name", "Name must be unique", func(e *mockTenantedEntity) any {
			return e.Name
		}).
		WithCaseSensitiveUniqueField(
			"code",
			"code",
			"Code must be unique",
			func(e *mockTenantedEntity) any { return e.Name },
		).
		Build()

	result := validator.ValidateCreate(t.Context(), entity)

	assert.Nil(t, result)
	checker.AssertNumberOfCalls(t, "CheckUniqueness", 2)
}

func TestTenantedValidatorBuilder_WithReferenceChecker(t *testing.T) {
	t.Parallel()

	checker := newMockReferenceChecker()
	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithReferenceChecker(checker).
		Build()

	assert.Equal(t, checker, validator.referenceChecker)
}

func TestTenantedValidatorBuilder_WithReferenceCheck(t *testing.T) {
	t.Parallel()

	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithReferenceCheck("parentId", "parents", "Parent does not exist", func(e *mockTenantedEntity) pulid.ID {
			return e.ParentID
		}).
		Build()

	require.Len(t, validator.referenceFields, 1)
	assert.Equal(t, "parentId", validator.referenceFields[0].FieldName)
	assert.Equal(t, "parents", validator.referenceFields[0].TableName)
	assert.Equal(t, "Parent does not exist", validator.referenceFields[0].Message)
	assert.False(t, validator.referenceFields[0].Optional)
}

func TestTenantedValidatorBuilder_WithOptionalReferenceCheck(t *testing.T) {
	t.Parallel()

	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithOptionalReferenceCheck("parentId", "parents", "Parent does not exist", func(e *mockTenantedEntity) pulid.ID {
			return e.ParentID
		}).
		Build()

	require.Len(t, validator.referenceFields, 1)
	assert.True(t, validator.referenceFields[0].Optional)
}

func TestTenantedValidator_ValidateCreate_ReferenceCheck_Exists(t *testing.T) {
	t.Parallel()

	entity := newMockTenantedEntity()
	entity.ParentID = pulid.MustNew("par_")

	checker := newMockReferenceChecker()
	checker.On("CheckReference", mock.Anything, mock.Anything).Return(true, nil)

	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithReferenceChecker(checker).
		WithReferenceCheck("parentId", "parents", "Parent does not exist", func(e *mockTenantedEntity) pulid.ID {
			return e.ParentID
		}).
		Build()

	result := validator.ValidateCreate(t.Context(), entity)

	assert.Nil(t, result)
	checker.AssertExpectations(t)
}

func TestTenantedValidator_ValidateCreate_ReferenceCheck_NotFound(t *testing.T) {
	t.Parallel()

	entity := newMockTenantedEntity()
	entity.ParentID = pulid.MustNew("par_")

	checker := newMockReferenceChecker()
	checker.On("CheckReference", mock.Anything, mock.Anything).Return(false, nil)

	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithReferenceChecker(checker).
		WithReferenceCheck("parentId", "parents", "Parent does not exist", func(e *mockTenantedEntity) pulid.ID {
			return e.ParentID
		}).
		Build()

	result := validator.ValidateCreate(t.Context(), entity)

	require.NotNil(t, result)
	assert.True(t, result.HasErrors())

	hasRefError := false
	for _, err := range result.Errors {
		if err.Field == "parentId" && err.Code == errortypes.ErrInvalidReference {
			hasRefError = true
			assert.Equal(t, "Parent does not exist", err.Message)
			break
		}
	}
	assert.True(t, hasRefError)
	checker.AssertExpectations(t)
}

func TestTenantedValidator_ValidateCreate_ReferenceCheck_RequiredFieldNil(t *testing.T) {
	t.Parallel()

	entity := newMockTenantedEntity()

	checker := newMockReferenceChecker()

	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithReferenceChecker(checker).
		WithReferenceCheck("parentId", "parents", "Parent is required", func(e *mockTenantedEntity) pulid.ID {
			return e.ParentID
		}).
		Build()

	result := validator.ValidateCreate(t.Context(), entity)

	require.NotNil(t, result)
	assert.True(t, result.HasErrors())

	hasReqError := false
	for _, err := range result.Errors {
		if err.Field == "parentId" && err.Code == errortypes.ErrRequired {
			hasReqError = true
			break
		}
	}
	assert.True(t, hasReqError)
	checker.AssertNotCalled(t, "CheckReference", mock.Anything, mock.Anything)
}

func TestTenantedValidator_ValidateCreate_OptionalReferenceCheck_SkipsNil(t *testing.T) {
	t.Parallel()

	entity := newMockTenantedEntity()

	checker := newMockReferenceChecker()

	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithReferenceChecker(checker).
		WithOptionalReferenceCheck("parentId", "parents", "Parent does not exist", func(e *mockTenantedEntity) pulid.ID {
			return e.ParentID
		}).
		Build()

	result := validator.ValidateCreate(t.Context(), entity)

	assert.Nil(t, result)
	checker.AssertNotCalled(t, "CheckReference", mock.Anything, mock.Anything)
}

func TestTenantedValidator_ValidateCreate_OptionalReferenceCheck_ValidatesWhenSet(t *testing.T) {
	t.Parallel()

	entity := newMockTenantedEntity()
	entity.ParentID = pulid.MustNew("par_")

	checker := newMockReferenceChecker()
	checker.On("CheckReference", mock.Anything, mock.Anything).Return(true, nil)

	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithReferenceChecker(checker).
		WithOptionalReferenceCheck("parentId", "parents", "Parent does not exist", func(e *mockTenantedEntity) pulid.ID {
			return e.ParentID
		}).
		Build()

	result := validator.ValidateCreate(t.Context(), entity)

	assert.Nil(t, result)
	checker.AssertExpectations(t)
}

func TestTenantedValidator_ValidateCreate_ReferenceCheck_Error(t *testing.T) {
	t.Parallel()

	entity := newMockTenantedEntity()
	entity.ParentID = pulid.MustNew("par_")

	checker := newMockReferenceChecker()
	checker.On("CheckReference", mock.Anything, mock.Anything).
		Return(false, errors.New("database error"))

	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithReferenceChecker(checker).
		WithReferenceCheck("parentId", "parents", "Parent does not exist", func(e *mockTenantedEntity) pulid.ID {
			return e.ParentID
		}).
		Build()

	result := validator.ValidateCreate(t.Context(), entity)

	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
	checker.AssertExpectations(t)
}

func TestTenantedValidator_ValidateCreate_ReferenceCheck_DefaultMessage(t *testing.T) {
	t.Parallel()

	entity := newMockTenantedEntity()
	entity.ParentID = pulid.MustNew("par_")

	checker := newMockReferenceChecker()
	checker.On("CheckReference", mock.Anything, mock.Anything).Return(false, nil)

	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithReferenceChecker(checker).
		WithReferenceCheck("parentId", "parents", "", func(e *mockTenantedEntity) pulid.ID {
			return e.ParentID
		}).
		Build()

	result := validator.ValidateCreate(t.Context(), entity)

	require.NotNil(t, result)
	assert.True(t, result.HasErrors())

	for _, err := range result.Errors {
		if err.Field == "parentId" {
			assert.Equal(
				t,
				"Referenced parentId does not exist or belongs to a different organization",
				err.Message,
			)
			break
		}
	}
	checker.AssertExpectations(t)
}

func TestTenantedValidator_ValidateUpdate_ReferenceCheck_PassesTenantContext(t *testing.T) {
	t.Parallel()

	entity := newMockTenantedEntity()
	entity.id = pulid.MustNew("ent_")
	entity.ParentID = pulid.MustNew("par_")

	checker := newMockReferenceChecker()
	checker.On("CheckReference", mock.Anything, mock.MatchedBy(func(req *ReferenceRequest) bool {
		return req.OrganizationID == entity.organizationID &&
			req.BusinessUnitID == entity.businessUnitID &&
			req.TableName == "parents" &&
			req.ID == entity.ParentID
	})).Return(true, nil)

	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithReferenceChecker(checker).
		WithReferenceCheck("parentId", "parents", "", func(e *mockTenantedEntity) pulid.ID {
			return e.ParentID
		}).
		Build()

	result := validator.ValidateUpdate(t.Context(), entity)

	assert.Nil(t, result)
	checker.AssertExpectations(t)
}

func TestTenantedValidator_MultipleReferenceFields(t *testing.T) {
	t.Parallel()

	entity := newMockTenantedEntity()
	entity.ParentID = pulid.MustNew("par_")

	checker := newMockReferenceChecker()
	checker.On("CheckReference", mock.Anything, mock.Anything).Return(true, nil)

	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithReferenceChecker(checker).
		WithReferenceCheck("parentId", "parents", "", func(e *mockTenantedEntity) pulid.ID {
			return e.ParentID
		}).
		WithOptionalReferenceCheck("secondaryId", "secondaries", "", func(e *mockTenantedEntity) pulid.ID {
			return pulid.Nil
		}).
		Build()

	result := validator.ValidateCreate(t.Context(), entity)

	assert.Nil(t, result)
	checker.AssertNumberOfCalls(t, "CheckReference", 1)
}

func TestTenantedValidatorBuilder_WithCompositeUniqueFields(t *testing.T) {
	t.Parallel()

	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithCompositeUniqueFields(
			"name_code",
			"Name and Code combination must be unique",
			CompositeField[*mockTenantedEntity]{
				FieldName: "name",
				Column:    "name",
				GetValue:  func(e *mockTenantedEntity) any { return e.Name },
			},
			CompositeField[*mockTenantedEntity]{
				FieldName:     "code",
				Column:        "code",
				CaseSensitive: true,
				GetValue:      func(e *mockTenantedEntity) any { return e.Code },
			},
		).
		Build()

	require.Len(t, validator.compositeUniqueFields, 1)
	assert.Equal(t, "name_code", validator.compositeUniqueFields[0].Name)
	require.Len(t, validator.compositeUniqueFields[0].Fields, 2)
}

func TestTenantedValidator_CompositeUniqueness_NoConflict(t *testing.T) {
	t.Parallel()

	entity := newMockTenantedEntity()
	entity.Name = "Test"
	entity.Code = "CODE1"

	checker := newMockUniquenessChecker()
	checker.On("CheckUniqueness", mock.Anything, mock.MatchedBy(func(req *UniquenessRequest) bool {
		return len(req.Fields) == 2
	})).Return(false, nil)

	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithUniquenessChecker(checker).
		WithCompositeUniqueFields(
			"name_code",
			"",
			CompositeField[*mockTenantedEntity]{
				FieldName: "name",
				Column:    "name",
				GetValue:  func(e *mockTenantedEntity) any { return e.Name },
			},
			CompositeField[*mockTenantedEntity]{
				FieldName: "code",
				Column:    "code",
				GetValue:  func(e *mockTenantedEntity) any { return e.Code },
			},
		).
		Build()

	result := validator.ValidateCreate(t.Context(), entity)

	assert.Nil(t, result)
	checker.AssertExpectations(t)
}

func TestTenantedValidator_CompositeUniqueness_Conflict(t *testing.T) {
	t.Parallel()

	entity := newMockTenantedEntity()
	entity.Name = "Test"
	entity.Code = "CODE1"

	checker := newMockUniquenessChecker()
	checker.On("CheckUniqueness", mock.Anything, mock.Anything).Return(true, nil)

	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithModelName("TestEntity").
		WithUniquenessChecker(checker).
		WithCompositeUniqueFields(
			"name_code",
			"Name and Code combination already exists",
			CompositeField[*mockTenantedEntity]{
				FieldName: "name",
				Column:    "name",
				GetValue:  func(e *mockTenantedEntity) any { return e.Name },
			},
			CompositeField[*mockTenantedEntity]{
				FieldName: "code",
				Column:    "code",
				GetValue:  func(e *mockTenantedEntity) any { return e.Code },
			},
		).
		Build()

	result := validator.ValidateCreate(t.Context(), entity)

	require.NotNil(t, result)
	assert.True(t, result.HasErrors())

	hasCompositeError := false
	for _, err := range result.Errors {
		if err.Field == "name_code" && err.Code == errortypes.ErrDuplicate {
			hasCompositeError = true
			assert.Equal(t, "Name and Code combination already exists", err.Message)
			break
		}
	}
	assert.True(t, hasCompositeError)
}

func TestTenantedValidator_CompositeUniqueness_SkipsWhenFieldEmpty(t *testing.T) {
	t.Parallel()

	entity := newMockTenantedEntity()
	entity.Name = "Test"
	entity.Code = ""

	checker := newMockUniquenessChecker()

	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithUniquenessChecker(checker).
		WithCompositeUniqueFields(
			"name_code",
			"",
			CompositeField[*mockTenantedEntity]{
				FieldName: "name",
				Column:    "name",
				GetValue:  func(e *mockTenantedEntity) any { return e.Name },
			},
			CompositeField[*mockTenantedEntity]{
				FieldName: "code",
				Column:    "code",
				GetValue:  func(e *mockTenantedEntity) any { return e.Code },
			},
		).
		Build()

	result := validator.ValidateCreate(t.Context(), entity)

	assert.Nil(t, result)
	checker.AssertNotCalled(t, "CheckUniqueness", mock.Anything, mock.Anything)
}

func TestTenantedValidator_CompositeUniqueness_DefaultMessage(t *testing.T) {
	t.Parallel()

	entity := newMockTenantedEntity()
	entity.Name = "Test"
	entity.Code = "CODE1"

	checker := newMockUniquenessChecker()
	checker.On("CheckUniqueness", mock.Anything, mock.Anything).Return(true, nil)

	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithModelName("Equipment").
		WithUniquenessChecker(checker).
		WithCompositeUniqueFields(
			"name_code",
			"",
			CompositeField[*mockTenantedEntity]{
				FieldName: "name",
				Column:    "name",
				GetValue:  func(e *mockTenantedEntity) any { return e.Name },
			},
			CompositeField[*mockTenantedEntity]{
				FieldName: "code",
				Column:    "code",
				GetValue:  func(e *mockTenantedEntity) any { return e.Code },
			},
		).
		Build()

	result := validator.ValidateCreate(t.Context(), entity)

	require.NotNil(t, result)
	for _, err := range result.Errors {
		if err.Field == "name_code" {
			assert.Equal(t, "Equipment with this combination already exists", err.Message)
			break
		}
	}
}

func TestTenantedValidatorBuilder_WithImmutableField(t *testing.T) {
	t.Parallel()

	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithImmutableField("type", "Type cannot be changed", func(e *mockTenantedEntity) any {
			return e.Type
		}).
		Build()

	require.Len(t, validator.immutableFields, 1)
	assert.Equal(t, "type", validator.immutableFields[0].FieldName)
	assert.Equal(t, "Type cannot be changed", validator.immutableFields[0].Message)
}

func TestTenantedValidator_ImmutableField_NoChange(t *testing.T) {
	t.Parallel()

	original := newMockTenantedEntity()
	original.id = pulid.MustNew("ent_")
	original.Type = "TypeA"

	entity := newMockTenantedEntity()
	entity.id = original.id
	entity.organizationID = original.organizationID
	entity.businessUnitID = original.businessUnitID
	entity.Type = "TypeA"

	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithImmutableField("type", "", func(e *mockTenantedEntity) any {
			return e.Type
		}).
		Build()

	result := validator.ValidateUpdateWithOriginal(t.Context(), entity, original)

	assert.Nil(t, result)
}

func TestTenantedValidator_ImmutableField_Changed(t *testing.T) {
	t.Parallel()

	original := newMockTenantedEntity()
	original.id = pulid.MustNew("ent_")
	original.Type = "TypeA"

	entity := newMockTenantedEntity()
	entity.id = original.id
	entity.organizationID = original.organizationID
	entity.businessUnitID = original.businessUnitID
	entity.Type = "TypeB"

	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithImmutableField("type", "Type cannot be changed after creation", func(e *mockTenantedEntity) any {
			return e.Type
		}).
		Build()

	result := validator.ValidateUpdateWithOriginal(t.Context(), entity, original)

	require.NotNil(t, result)
	assert.True(t, result.HasErrors())

	hasImmutableError := false
	for _, err := range result.Errors {
		if err.Field == "type" && err.Code == errortypes.ErrInvalidOperation {
			hasImmutableError = true
			assert.Equal(t, "Type cannot be changed after creation", err.Message)
			break
		}
	}
	assert.True(t, hasImmutableError)
}

func TestTenantedValidator_ImmutableField_DefaultMessage(t *testing.T) {
	t.Parallel()

	original := newMockTenantedEntity()
	original.id = pulid.MustNew("ent_")
	original.Type = "TypeA"

	entity := newMockTenantedEntity()
	entity.id = original.id
	entity.organizationID = original.organizationID
	entity.businessUnitID = original.businessUnitID
	entity.Type = "TypeB"

	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithImmutableField("type", "", func(e *mockTenantedEntity) any {
			return e.Type
		}).
		Build()

	result := validator.ValidateUpdateWithOriginal(t.Context(), entity, original)

	require.NotNil(t, result)
	for _, err := range result.Errors {
		if err.Field == "type" {
			assert.Equal(t, "type cannot be changed after creation", err.Message)
			break
		}
	}
}

func TestTenantedValidator_ImmutableField_NotCheckedWithoutOriginal(t *testing.T) {
	t.Parallel()

	entity := newMockTenantedEntity()
	entity.id = pulid.MustNew("ent_")
	entity.Type = "TypeB"

	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithImmutableField("type", "", func(e *mockTenantedEntity) any {
			return e.Type
		}).
		Build()

	result := validator.ValidateUpdate(t.Context(), entity)

	assert.Nil(t, result)
}

func TestTenantedValidator_ImmutableField_NotCheckedOnCreate(t *testing.T) {
	t.Parallel()

	entity := newMockTenantedEntity()
	entity.Type = "TypeA"

	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithImmutableField("type", "", func(e *mockTenantedEntity) any {
			return e.Type
		}).
		Build()

	result := validator.ValidateCreate(t.Context(), entity)

	assert.Nil(t, result)
}

func TestTenantedValidator_MultipleImmutableFields(t *testing.T) {
	t.Parallel()

	original := newMockTenantedEntity()
	original.id = pulid.MustNew("ent_")
	original.Type = "TypeA"
	original.Code = "CODE1"

	entity := newMockTenantedEntity()
	entity.id = original.id
	entity.organizationID = original.organizationID
	entity.businessUnitID = original.businessUnitID
	entity.Type = "TypeB"
	entity.Code = "CODE2"

	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithImmutableField("type", "", func(e *mockTenantedEntity) any {
			return e.Type
		}).
		WithImmutableField("code", "", func(e *mockTenantedEntity) any {
			return e.Code
		}).
		Build()

	result := validator.ValidateUpdateWithOriginal(t.Context(), entity, original)

	require.NotNil(t, result)
	assert.Len(t, result.Errors, 2)
}

func TestValuesEqual(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		a        any
		b        any
		expected bool
	}{
		{"both nil", nil, nil, true},
		{"a nil b not", nil, "test", false},
		{"a not nil b nil", "test", nil, false},
		{"equal strings", "test", "test", true},
		{"different strings", "test1", "test2", false},
		{"equal ints", 42, 42, true},
		{"different ints", 42, 43, false},
		{"equal bools", true, true, true},
		{"different bools", true, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := valuesEqual(tt.a, tt.b)
			assert.Equal(t, tt.expected, result)
		})
	}
}

//go:fix inline
func ptr[T any](v T) *T {
	return new(v)
}

func TestTenantedValidatorBuilder_WithDateAfter(t *testing.T) {
	t.Parallel()

	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithDateAfter("endDate", "End date must be after start date",
			func(e *mockTenantedEntity) *int64 { return e.StartDate },
			func(e *mockTenantedEntity) *int64 { return e.EndDate },
		).
		Build()

	require.Len(t, validator.dateComparisons, 1)
	assert.Equal(t, "endDate", validator.dateComparisons[0].FieldName)
	assert.False(t, validator.dateComparisons[0].AllowEqual)
}

func TestTenantedValidatorBuilder_WithDateAfterOrEqual(t *testing.T) {
	t.Parallel()

	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithDateAfterOrEqual("endDate", "",
			func(e *mockTenantedEntity) *int64 { return e.StartDate },
			func(e *mockTenantedEntity) *int64 { return e.EndDate },
		).
		Build()

	require.Len(t, validator.dateComparisons, 1)
	assert.True(t, validator.dateComparisons[0].AllowEqual)
}

func TestTenantedValidator_DateAfter_Valid(t *testing.T) {
	t.Parallel()

	entity := newMockTenantedEntity()
	entity.StartDate = new(int64(1000))
	entity.EndDate = new(int64(2000))

	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithDateAfter("endDate", "",
			func(e *mockTenantedEntity) *int64 { return e.StartDate },
			func(e *mockTenantedEntity) *int64 { return e.EndDate },
		).
		Build()

	result := validator.ValidateCreate(t.Context(), entity)

	assert.Nil(t, result)
}

func TestTenantedValidator_DateAfter_Invalid(t *testing.T) {
	t.Parallel()

	entity := newMockTenantedEntity()
	entity.StartDate = new(int64(2000))
	entity.EndDate = new(int64(1000))

	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithDateAfter("endDate", "End date must be after start date",
			func(e *mockTenantedEntity) *int64 { return e.StartDate },
			func(e *mockTenantedEntity) *int64 { return e.EndDate },
		).
		Build()

	result := validator.ValidateCreate(t.Context(), entity)

	require.NotNil(t, result)
	assert.True(t, result.HasErrors())

	hasDateError := false
	for _, err := range result.Errors {
		if err.Field == "endDate" && err.Code == errortypes.ErrInvalid {
			hasDateError = true
			assert.Equal(t, "End date must be after start date", err.Message)
			break
		}
	}
	assert.True(t, hasDateError)
}

func TestTenantedValidator_DateAfter_EqualDatesInvalid(t *testing.T) {
	t.Parallel()

	entity := newMockTenantedEntity()
	entity.StartDate = new(int64(1000))
	entity.EndDate = new(int64(1000))

	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithDateAfter("endDate", "",
			func(e *mockTenantedEntity) *int64 { return e.StartDate },
			func(e *mockTenantedEntity) *int64 { return e.EndDate },
		).
		Build()

	result := validator.ValidateCreate(t.Context(), entity)

	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
}

func TestTenantedValidator_DateAfterOrEqual_EqualDatesValid(t *testing.T) {
	t.Parallel()

	entity := newMockTenantedEntity()
	entity.StartDate = new(int64(1000))
	entity.EndDate = new(int64(1000))

	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithDateAfterOrEqual("endDate", "",
			func(e *mockTenantedEntity) *int64 { return e.StartDate },
			func(e *mockTenantedEntity) *int64 { return e.EndDate },
		).
		Build()

	result := validator.ValidateCreate(t.Context(), entity)

	assert.Nil(t, result)
}

func TestTenantedValidator_DateAfter_SkipsNilDates(t *testing.T) {
	t.Parallel()

	entity := newMockTenantedEntity()
	entity.StartDate = new(int64(1000))
	entity.EndDate = nil

	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithDateAfter("endDate", "",
			func(e *mockTenantedEntity) *int64 { return e.StartDate },
			func(e *mockTenantedEntity) *int64 { return e.EndDate },
		).
		Build()

	result := validator.ValidateCreate(t.Context(), entity)

	assert.Nil(t, result)
}

func TestTenantedValidator_DateAfter_DefaultMessage(t *testing.T) {
	t.Parallel()

	entity := newMockTenantedEntity()
	entity.StartDate = new(int64(2000))
	entity.EndDate = new(int64(1000))

	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithDateAfter("endDate", "",
			func(e *mockTenantedEntity) *int64 { return e.StartDate },
			func(e *mockTenantedEntity) *int64 { return e.EndDate },
		).
		Build()

	result := validator.ValidateCreate(t.Context(), entity)

	require.NotNil(t, result)
	for _, err := range result.Errors {
		if err.Field == "endDate" {
			assert.Equal(t, "endDate must be after the start date", err.Message)
			break
		}
	}
}

func TestTenantedValidatorBuilder_WithNumericRange(t *testing.T) {
	t.Parallel()

	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithNumericRange("weight", "Weight out of range",
			func(e *mockTenantedEntity) *float64 { return e.Weight },
			new(0.0),
			new(80000.0),
		).
		Build()

	require.Len(t, validator.numericRanges, 1)
	assert.Equal(t, "weight", validator.numericRanges[0].FieldName)
	assert.Equal(t, 0.0, *validator.numericRanges[0].Min)
	assert.Equal(t, 80000.0, *validator.numericRanges[0].Max)
}

func TestTenantedValidator_NumericRange_Valid(t *testing.T) {
	t.Parallel()

	entity := newMockTenantedEntity()
	entity.Weight = new(50000.0)

	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithNumericRange("weight", "",
			func(e *mockTenantedEntity) *float64 { return e.Weight },
			new(0.0),
			new(80000.0),
		).
		Build()

	result := validator.ValidateCreate(t.Context(), entity)

	assert.Nil(t, result)
}

func TestTenantedValidator_NumericRange_BelowMin(t *testing.T) {
	t.Parallel()

	entity := newMockTenantedEntity()
	entity.Weight = new(-100.0)

	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithNumericRange("weight", "Weight must be positive",
			func(e *mockTenantedEntity) *float64 { return e.Weight },
			new(0.0),
			new(80000.0),
		).
		Build()

	result := validator.ValidateCreate(t.Context(), entity)

	require.NotNil(t, result)
	assert.True(t, result.HasErrors())

	hasRangeError := false
	for _, err := range result.Errors {
		if err.Field == "weight" && err.Code == errortypes.ErrInvalid {
			hasRangeError = true
			assert.Equal(t, "Weight must be positive", err.Message)
			break
		}
	}
	assert.True(t, hasRangeError)
}

func TestTenantedValidator_NumericRange_AboveMax(t *testing.T) {
	t.Parallel()

	entity := newMockTenantedEntity()
	entity.Weight = new(100000.0)

	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithNumericRange("weight", "",
			func(e *mockTenantedEntity) *float64 { return e.Weight },
			new(0.0),
			new(80000.0),
		).
		Build()

	result := validator.ValidateCreate(t.Context(), entity)

	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
}

func TestTenantedValidator_NumericRange_SkipsNil(t *testing.T) {
	t.Parallel()

	entity := newMockTenantedEntity()
	entity.Weight = nil

	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithNumericRange("weight", "",
			func(e *mockTenantedEntity) *float64 { return e.Weight },
			new(0.0),
			new(80000.0),
		).
		Build()

	result := validator.ValidateCreate(t.Context(), entity)

	assert.Nil(t, result)
}

func TestTenantedValidator_NumericRange_MinOnly(t *testing.T) {
	t.Parallel()

	entity := newMockTenantedEntity()
	entity.Weight = new(1000000.0)

	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithNumericRange("weight", "",
			func(e *mockTenantedEntity) *float64 { return e.Weight },
			new(0.0),
			nil,
		).
		Build()

	result := validator.ValidateCreate(t.Context(), entity)

	assert.Nil(t, result)
}

func TestTenantedValidator_NumericRange_MaxOnly(t *testing.T) {
	t.Parallel()

	entity := newMockTenantedEntity()
	entity.Weight = new(-1000.0)

	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithNumericRange("weight", "",
			func(e *mockTenantedEntity) *float64 { return e.Weight },
			nil,
			new(80000.0),
		).
		Build()

	result := validator.ValidateCreate(t.Context(), entity)

	assert.Nil(t, result)
}

func TestTenantedValidator_NumericRange_DefaultMessages(t *testing.T) {
	t.Parallel()

	t.Run("below min", func(t *testing.T) {
		t.Parallel()
		entity := newMockTenantedEntity()
		entity.Weight = new(-100.0)

		validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
			WithNumericRange("weight", "",
				func(e *mockTenantedEntity) *float64 { return e.Weight },
				new(0.0),
				new(80000.0),
			).
			Build()

		result := validator.ValidateCreate(t.Context(), entity)

		require.NotNil(t, result)
		for _, err := range result.Errors {
			if err.Field == "weight" {
				assert.Equal(t, "weight must be at least 0", err.Message)
				break
			}
		}
	})

	t.Run("above max", func(t *testing.T) {
		t.Parallel()
		entity := newMockTenantedEntity()
		entity.Weight = new(100000.0)

		validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
			WithNumericRange("weight", "",
				func(e *mockTenantedEntity) *float64 { return e.Weight },
				new(0.0),
				new(80000.0),
			).
			Build()

		result := validator.ValidateCreate(t.Context(), entity)

		require.NotNil(t, result)
		for _, err := range result.Errors {
			if err.Field == "weight" {
				assert.Equal(t, "weight must be at most 80000", err.Message)
				break
			}
		}
	})
}

func TestTenantedValidator_NumericRange_AtBoundary(t *testing.T) {
	t.Parallel()

	t.Run("at min", func(t *testing.T) {
		t.Parallel()
		entity := newMockTenantedEntity()
		entity.Weight = new(0.0)

		validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
			WithNumericRange("weight", "",
				func(e *mockTenantedEntity) *float64 { return e.Weight },
				new(0.0),
				new(80000.0),
			).
			Build()

		result := validator.ValidateCreate(t.Context(), entity)
		assert.Nil(t, result)
	})

	t.Run("at max", func(t *testing.T) {
		t.Parallel()
		entity := newMockTenantedEntity()
		entity.Weight = new(80000.0)

		validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
			WithNumericRange("weight", "",
				func(e *mockTenantedEntity) *float64 { return e.Weight },
				new(0.0),
				new(80000.0),
			).
			Build()

		result := validator.ValidateCreate(t.Context(), entity)
		assert.Nil(t, result)
	})
}
