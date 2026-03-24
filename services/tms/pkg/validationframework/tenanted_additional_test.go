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
	"github.com/uptrace/bun"
)

func TestTenantedValidator_CustomReferenceCheck_Exists(t *testing.T) {
	t.Parallel()

	entity := newMockTenantedEntity()
	entity.ParentID = pulid.MustNew("par_")

	checker := newMockReferenceChecker()

	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithReferenceChecker(checker).
		WithCustomReferenceCheck("parentId", "Custom parent not found",
			func(e *mockTenantedEntity) pulid.ID { return e.ParentID },
			func(_ context.Context, _, _, _ pulid.ID) (bool, error) {
				return true, nil
			},
		).
		Build()

	result := validator.ValidateCreate(t.Context(), entity)

	assert.Nil(t, result)
	checker.AssertNotCalled(t, "CheckReference", mock.Anything, mock.Anything)
}

func TestTenantedValidator_CustomReferenceCheck_NotFound(t *testing.T) {
	t.Parallel()

	entity := newMockTenantedEntity()
	entity.ParentID = pulid.MustNew("par_")

	checker := newMockReferenceChecker()

	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithReferenceChecker(checker).
		WithCustomReferenceCheck("parentId", "Custom parent not found",
			func(e *mockTenantedEntity) pulid.ID { return e.ParentID },
			func(_ context.Context, _, _, _ pulid.ID) (bool, error) {
				return false, nil
			},
		).
		Build()

	result := validator.ValidateCreate(t.Context(), entity)

	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
	found := false
	for _, err := range result.Errors {
		if err.Field == "parentId" && err.Code == errortypes.ErrInvalidReference {
			found = true
			assert.Equal(t, "Custom parent not found", err.Message)
			break
		}
	}
	assert.True(t, found)
}

func TestTenantedValidator_CustomReferenceCheck_Error(t *testing.T) {
	t.Parallel()

	entity := newMockTenantedEntity()
	entity.ParentID = pulid.MustNew("par_")

	checker := newMockReferenceChecker()

	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithReferenceChecker(checker).
		WithCustomReferenceCheck("parentId", "",
			func(e *mockTenantedEntity) pulid.ID { return e.ParentID },
			func(_ context.Context, _, _, _ pulid.ID) (bool, error) {
				return false, errors.New("custom check failed")
			},
		).
		Build()

	result := validator.ValidateCreate(t.Context(), entity)

	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
}

func TestTenantedValidator_OptionalCustomReferenceCheck_SkipsNil(t *testing.T) {
	t.Parallel()

	entity := newMockTenantedEntity()

	checker := newMockReferenceChecker()

	customCalled := false
	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithReferenceChecker(checker).
		WithOptionalCustomReferenceCheck("parentId", "",
			func(e *mockTenantedEntity) pulid.ID { return e.ParentID },
			func(_ context.Context, _, _, _ pulid.ID) (bool, error) {
				customCalled = true
				return true, nil
			},
		).
		Build()

	result := validator.ValidateCreate(t.Context(), entity)

	assert.Nil(t, result)
	assert.False(t, customCalled)
}

func TestTenantedValidator_OptionalCustomReferenceCheck_ValidatesWhenSet(t *testing.T) {
	t.Parallel()

	entity := newMockTenantedEntity()
	entity.ParentID = pulid.MustNew("par_")

	checker := newMockReferenceChecker()

	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithReferenceChecker(checker).
		WithOptionalCustomReferenceCheck("parentId", "",
			func(e *mockTenantedEntity) pulid.ID { return e.ParentID },
			func(_ context.Context, _, _, _ pulid.ID) (bool, error) {
				return true, nil
			},
		).
		Build()

	result := validator.ValidateCreate(t.Context(), entity)

	assert.Nil(t, result)
}

func TestTenantedValidator_CompositeUniqueness_Error(t *testing.T) {
	t.Parallel()

	entity := newMockTenantedEntity()
	entity.Name = "Test"
	entity.Code = "CODE1"

	checker := newMockUniquenessChecker()
	checker.On("CheckUniqueness", mock.Anything, mock.Anything).
		Return(false, errors.New("db error"))

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

	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
}

func TestTenantedValidator_DateAfterOrEqual_Invalid(t *testing.T) {
	t.Parallel()

	entity := newMockTenantedEntity()
	entity.StartDate = new(int64(3000))
	entity.EndDate = new(int64(1000))

	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithDateAfterOrEqual("endDate", "End must be on or after start",
			func(e *mockTenantedEntity) *int64 { return e.StartDate },
			func(e *mockTenantedEntity) *int64 { return e.EndDate },
		).
		Build()

	result := validator.ValidateCreate(t.Context(), entity)

	require.NotNil(t, result)
	found := false
	for _, err := range result.Errors {
		if err.Field == "endDate" {
			found = true
			assert.Equal(t, "End must be on or after start", err.Message)
			break
		}
	}
	assert.True(t, found)
}

func TestTenantedValidator_DateAfterOrEqual_DefaultMessage(t *testing.T) {
	t.Parallel()

	entity := newMockTenantedEntity()
	entity.StartDate = new(int64(3000))
	entity.EndDate = new(int64(1000))

	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithDateAfterOrEqual("endDate", "",
			func(e *mockTenantedEntity) *int64 { return e.StartDate },
			func(e *mockTenantedEntity) *int64 { return e.EndDate },
		).
		Build()

	result := validator.ValidateCreate(t.Context(), entity)

	require.NotNil(t, result)
	for _, err := range result.Errors {
		if err.Field == "endDate" {
			assert.Equal(t, "endDate must be on or after the start date", err.Message)
			break
		}
	}
}

func TestTenantedValidator_RequiredReferenceCheck_DefaultMessage(t *testing.T) {
	t.Parallel()

	entity := newMockTenantedEntity()

	checker := newMockReferenceChecker()

	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithReferenceChecker(checker).
		WithReferenceCheck("parentId", "parents", "", func(e *mockTenantedEntity) pulid.ID {
			return e.ParentID
		}).
		Build()

	result := validator.ValidateCreate(t.Context(), entity)

	require.NotNil(t, result)
	for _, err := range result.Errors {
		if err.Field == "parentId" && err.Code == errortypes.ErrRequired {
			assert.Equal(t, "parentId is required", err.Message)
			break
		}
	}
}

func TestTenantedValidator_UniquenessCheck_NilValue(t *testing.T) {
	t.Parallel()

	entity := newMockTenantedEntity()
	checker := newMockUniquenessChecker()

	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithUniquenessChecker(checker).
		WithUniqueField("parentId", "parent_id", "", func(e *mockTenantedEntity) any {
			return nil
		}).
		Build()

	result := validator.ValidateCreate(t.Context(), entity)

	assert.Nil(t, result)
	checker.AssertNotCalled(t, "CheckUniqueness", mock.Anything, mock.Anything)
}

func TestTenantedValidator_CompositeUniqueness_NilFieldValue(t *testing.T) {
	t.Parallel()

	entity := newMockTenantedEntity()
	checker := newMockUniquenessChecker()

	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithUniquenessChecker(checker).
		WithCompositeUniqueFields(
			"name_parent",
			"",
			CompositeField[*mockTenantedEntity]{
				FieldName: "name",
				Column:    "name",
				GetValue:  func(e *mockTenantedEntity) any { return e.Name },
			},
			CompositeField[*mockTenantedEntity]{
				FieldName: "parent",
				Column:    "parent_id",
				GetValue:  func(_ *mockTenantedEntity) any { return nil },
			},
		).
		Build()

	result := validator.ValidateCreate(t.Context(), entity)

	assert.Nil(t, result)
	checker.AssertNotCalled(t, "CheckUniqueness", mock.Anything, mock.Anything)
}

func TestTenantedValidator_CustomRule_ReturnsError(t *testing.T) {
	t.Parallel()

	rule := NewTenantedRule[*mockTenantedEntity]("error_rule").
		OnCreate().
		WithValidation(func(
			_ context.Context,
			_ *mockTenantedEntity,
			_ *TenantedValidationContext,
			_ *errortypes.MultiError,
		) error {
			return errors.New("custom rule system error")
		})

	entity := newMockTenantedEntity()
	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithCustomRule(rule).
		Build()

	result := validator.ValidateCreate(t.Context(), entity)

	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
}

func TestTenantedValidator_CustomRule_OnBoth(t *testing.T) {
	t.Parallel()

	callCount := 0
	rule := NewTenantedRule[*mockTenantedEntity]("both_rule").
		OnBoth().
		WithValidation(func(
			_ context.Context,
			_ *mockTenantedEntity,
			_ *TenantedValidationContext,
			_ *errortypes.MultiError,
		) error {
			callCount++
			return nil
		})

	entity := newMockTenantedEntity()
	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithCustomRule(rule).
		Build()

	validator.ValidateCreate(t.Context(), entity)
	entity.id = pulid.MustNew("ent_")
	validator.ValidateUpdate(t.Context(), entity)

	assert.Equal(t, 2, callCount)
}

func TestTenantedValidator_ImmutableField_NilValues(t *testing.T) {
	t.Parallel()

	original := newMockTenantedEntity()
	original.id = pulid.MustNew("ent_")
	original.Weight = nil

	entity := newMockTenantedEntity()
	entity.id = original.id
	entity.organizationID = original.organizationID
	entity.businessUnitID = original.businessUnitID
	entity.Weight = nil

	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithImmutableField("weight", "", func(e *mockTenantedEntity) any {
			return e.Weight
		}).
		Build()

	result := validator.ValidateUpdateWithOriginal(t.Context(), entity, original)

	assert.Nil(t, result)
}

func TestTenantedValidator_ImmutableField_NilToNonNil(t *testing.T) {
	t.Parallel()

	original := newMockTenantedEntity()
	original.id = pulid.MustNew("ent_")
	original.Weight = nil

	entity := newMockTenantedEntity()
	entity.id = original.id
	entity.organizationID = original.organizationID
	entity.businessUnitID = original.businessUnitID
	entity.Weight = new(100.0)

	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithImmutableField("weight", "", func(e *mockTenantedEntity) any {
			return e.Weight
		}).
		Build()

	result := validator.ValidateUpdateWithOriginal(t.Context(), entity, original)

	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
}

func TestTenantedValidator_CompositeUniqueness_UpdateExcludesID(t *testing.T) {
	t.Parallel()

	entity := newMockTenantedEntity()
	entity.id = pulid.MustNew("ent_")
	entity.Name = "Test"
	entity.Code = "CODE1"

	checker := newMockUniquenessChecker()
	checker.On("CheckUniqueness", mock.Anything, mock.MatchedBy(func(req *UniquenessRequest) bool {
		return req.ExcludeID == entity.id
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

	result := validator.ValidateUpdate(t.Context(), entity)

	assert.Nil(t, result)
	checker.AssertExpectations(t)
}

func TestTenantedValidator_NumericRange_CustomMessage_AboveMax(t *testing.T) {
	t.Parallel()

	entity := newMockTenantedEntity()
	entity.Weight = new(99999.0)

	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithNumericRange("weight", "Weight exceeds maximum",
			func(e *mockTenantedEntity) *float64 { return e.Weight },
			new(0.0),
			new(80000.0),
		).
		Build()

	result := validator.ValidateCreate(t.Context(), entity)

	require.NotNil(t, result)
	for _, err := range result.Errors {
		if err.Field == "weight" {
			assert.Equal(t, "Weight exceeds maximum", err.Message)
			break
		}
	}
}

func TestBunUniquenessChecker_NilDB(t *testing.T) {
	t.Parallel()

	checker := NewBunUniquenessChecker(nil)

	req := &UniquenessRequest{
		TableName: "test_table",
		Fields:    []FieldCheck{{Column: "name", Value: "test"}},
	}

	_, err := checker.CheckUniqueness(t.Context(), req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "database connection is not initialized")
}

func TestBunUniquenessCheckerLazy(t *testing.T) {
	t.Parallel()

	getter := DBGetter(func() bun.IDB { return nil })
	checker := NewBunUniquenessCheckerLazy(getter)

	require.NotNil(t, checker)
	assert.NotNil(t, checker.dbGetter)
}

func TestBunReferenceChecker_Validations(t *testing.T) {
	t.Parallel()

	checker := NewBunReferenceChecker(nil)

	t.Run("empty table name", func(t *testing.T) {
		t.Parallel()
		req := &ReferenceRequest{
			TableName: "",
			ID:        pulid.MustNew("ref_"),
		}
		_, err := checker.CheckReference(t.Context(), req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "table name is required")
	})

	t.Run("nil reference ID", func(t *testing.T) {
		t.Parallel()
		req := &ReferenceRequest{
			TableName: "test_table",
			ID:        pulid.Nil,
		}
		_, err := checker.CheckReference(t.Context(), req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "reference ID is required")
	})

	t.Run("nil database", func(t *testing.T) {
		t.Parallel()
		req := &ReferenceRequest{
			TableName: "test_table",
			ID:        pulid.MustNew("ref_"),
		}
		_, err := checker.CheckReference(t.Context(), req)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "database connection is not initialized")
	})
}

func TestBunReferenceCheckerLazy(t *testing.T) {
	t.Parallel()

	getter := DBGetter(func() bun.IDB { return nil })
	checker := NewBunReferenceCheckerLazy(getter)

	require.NotNil(t, checker)
	assert.NotNil(t, checker.dbGetter)
}

func TestTenantedValidator_DateAfter_ValidEndAfterStart(t *testing.T) {
	t.Parallel()

	entity := newMockTenantedEntity()
	entity.StartDate = new(int64(1000))
	entity.EndDate = new(int64(3000))

	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithDateAfter("endDate", "",
			func(e *mockTenantedEntity) *int64 { return e.StartDate },
			func(e *mockTenantedEntity) *int64 { return e.EndDate },
		).
		Build()

	result := validator.ValidateCreate(t.Context(), entity)

	assert.Nil(t, result)
}

func TestTenantedValidator_DateAfter_EqualNotAllowed(t *testing.T) {
	t.Parallel()

	entity := newMockTenantedEntity()
	entity.StartDate = new(int64(1000))
	entity.EndDate = new(int64(1000))

	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithDateAfter("endDate", "End must be after start",
			func(e *mockTenantedEntity) *int64 { return e.StartDate },
			func(e *mockTenantedEntity) *int64 { return e.EndDate },
		).
		Build()

	result := validator.ValidateCreate(t.Context(), entity)

	require.NotNil(t, result)
	found := false
	for _, err := range result.Errors {
		if err.Field == "endDate" {
			found = true
			assert.Equal(t, "End must be after start", err.Message)
			break
		}
	}
	assert.True(t, found)
}

func TestTenantedValidator_DateAfter_UsesDefaultMessage(t *testing.T) {
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

func TestTenantedValidator_DateAfterOrEqual_EqualIsAllowed(t *testing.T) {
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

func TestTenantedValidator_DateComparison_NilStartDate(t *testing.T) {
	t.Parallel()

	entity := newMockTenantedEntity()
	entity.StartDate = nil
	entity.EndDate = new(int64(1000))

	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithDateAfter("endDate", "",
			func(e *mockTenantedEntity) *int64 { return e.StartDate },
			func(e *mockTenantedEntity) *int64 { return e.EndDate },
		).
		Build()

	result := validator.ValidateCreate(t.Context(), entity)

	assert.Nil(t, result)
}

func TestTenantedValidator_DateComparison_NilEndDate(t *testing.T) {
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

func TestTenantedValidator_NumericRange_BelowMinimum(t *testing.T) {
	t.Parallel()

	entity := newMockTenantedEntity()
	entity.Weight = new(-5.0)

	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithNumericRange("weight", "",
			func(e *mockTenantedEntity) *float64 { return e.Weight },
			new(0.0),
			new(100.0),
		).
		Build()

	result := validator.ValidateCreate(t.Context(), entity)

	require.NotNil(t, result)
	for _, err := range result.Errors {
		if err.Field == "weight" {
			assert.Contains(t, err.Message, "at least")
			break
		}
	}
}

func TestTenantedValidator_NumericRange_AboveMaxDefaultMessage(t *testing.T) {
	t.Parallel()

	entity := newMockTenantedEntity()
	entity.Weight = new(200.0)

	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithNumericRange("weight", "",
			func(e *mockTenantedEntity) *float64 { return e.Weight },
			new(0.0),
			new(100.0),
		).
		Build()

	result := validator.ValidateCreate(t.Context(), entity)

	require.NotNil(t, result)
	for _, err := range result.Errors {
		if err.Field == "weight" {
			assert.Contains(t, err.Message, "at most")
			break
		}
	}
}

func TestTenantedValidator_NumericRange_NilValueSkipped(t *testing.T) {
	t.Parallel()

	entity := newMockTenantedEntity()
	entity.Weight = nil

	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithNumericRange("weight", "",
			func(e *mockTenantedEntity) *float64 { return e.Weight },
			new(0.0),
			new(100.0),
		).
		Build()

	result := validator.ValidateCreate(t.Context(), entity)

	assert.Nil(t, result)
}

func TestTenantedValidator_NumericRange_InRange(t *testing.T) {
	t.Parallel()

	entity := newMockTenantedEntity()
	entity.Weight = new(50.0)

	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithNumericRange("weight", "",
			func(e *mockTenantedEntity) *float64 { return e.Weight },
			new(0.0),
			new(100.0),
		).
		Build()

	result := validator.ValidateCreate(t.Context(), entity)

	assert.Nil(t, result)
}

func TestTenantedValidator_NumericRange_MinOnlyNoBound(t *testing.T) {
	t.Parallel()

	entity := newMockTenantedEntity()
	entity.Weight = new(1000.0)

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

func TestTenantedValidator_NumericRange_MaxOnlyNoBound(t *testing.T) {
	t.Parallel()

	entity := newMockTenantedEntity()
	entity.Weight = new(-100.0)

	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithNumericRange("weight", "",
			func(e *mockTenantedEntity) *float64 { return e.Weight },
			nil,
			new(100.0),
		).
		Build()

	result := validator.ValidateCreate(t.Context(), entity)

	assert.Nil(t, result)
}

func TestTenantedValidator_UniquenessCheck_UsesDefaultMessage(t *testing.T) {
	t.Parallel()

	entity := newMockTenantedEntity()
	entity.Name = "TestName"

	checker := newMockUniquenessChecker()
	checker.On("CheckUniqueness", mock.Anything, mock.Anything).Return(true, nil)

	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithModelName("Widget").
		WithUniquenessChecker(checker).
		WithUniqueField("name", "name", "", func(e *mockTenantedEntity) any {
			return e.Name
		}).
		Build()

	result := validator.ValidateCreate(t.Context(), entity)

	require.NotNil(t, result)
	for _, err := range result.Errors {
		if err.Field == "name" {
			assert.Equal(t, "Widget with this name already exists", err.Message)
			break
		}
	}
}

func TestTenantedValidator_CompositeUniqueness_UsesDefaultMessage(t *testing.T) {
	t.Parallel()

	entity := newMockTenantedEntity()
	entity.Name = "Test"
	entity.Code = "CODE1"

	checker := newMockUniquenessChecker()
	checker.On("CheckUniqueness", mock.Anything, mock.Anything).Return(true, nil)

	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithModelName("Widget").
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
			assert.Equal(t, "Widget with this combination already exists", err.Message)
			break
		}
	}
}

func TestTenantedValidator_ReferenceCheck_NotFoundDefaultMessage(t *testing.T) {
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
	for _, err := range result.Errors {
		if err.Field == "parentId" && err.Code == errortypes.ErrInvalidReference {
			assert.Contains(t, err.Message, "does not exist")
			break
		}
	}
}

func TestTenantedValidator_CustomRule_SkippedOnWrongMode(t *testing.T) {
	t.Parallel()

	rule := NewTenantedRule[*mockTenantedEntity]("update_only_rule").
		OnUpdate().
		WithValidation(func(
			_ context.Context,
			_ *mockTenantedEntity,
			_ *TenantedValidationContext,
			multiErr *errortypes.MultiError,
		) error {
			multiErr.Add("x", errortypes.ErrInvalid, "should not appear")
			return nil
		})

	entity := newMockTenantedEntity()
	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithCustomRule(rule).
		Build()

	result := validator.ValidateCreate(t.Context(), entity)

	assert.Nil(t, result)
}

func TestTenantedValidator_UniquenessCheck_SkipsEmptyString(t *testing.T) {
	t.Parallel()

	entity := newMockTenantedEntity()
	entity.Name = ""

	checker := newMockUniquenessChecker()

	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithUniquenessChecker(checker).
		WithUniqueField("name", "name", "", func(e *mockTenantedEntity) any {
			return e.Name
		}).
		Build()

	result := validator.ValidateCreate(t.Context(), entity)

	assert.Nil(t, result)
	checker.AssertNotCalled(t, "CheckUniqueness", mock.Anything, mock.Anything)
}

func TestTenantedValidator_ImmutableField_UsesDefaultMessage(t *testing.T) {
	t.Parallel()

	original := newMockTenantedEntity()
	original.id = pulid.MustNew("ent_")
	original.Name = "Original"

	entity := newMockTenantedEntity()
	entity.id = original.id
	entity.organizationID = original.organizationID
	entity.businessUnitID = original.businessUnitID
	entity.Name = "Changed"

	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithImmutableField("name", "", func(e *mockTenantedEntity) any {
			return e.Name
		}).
		Build()

	result := validator.ValidateUpdateWithOriginal(t.Context(), entity, original)

	require.NotNil(t, result)
	for _, err := range result.Errors {
		if err.Field == "name" {
			assert.Contains(t, err.Message, "cannot be changed after creation")
			break
		}
	}
}

func TestValuesEqual_Comparisons(t *testing.T) {
	t.Parallel()

	t.Run("both nil", func(t *testing.T) {
		t.Parallel()
		assert.True(t, valuesEqual(nil, nil))
	})

	t.Run("first nil second not", func(t *testing.T) {
		t.Parallel()
		assert.False(t, valuesEqual(nil, "value"))
	})

	t.Run("first not nil second nil", func(t *testing.T) {
		t.Parallel()
		assert.False(t, valuesEqual("value", nil))
	})

	t.Run("equal strings", func(t *testing.T) {
		t.Parallel()
		assert.True(t, valuesEqual("abc", "abc"))
	})

	t.Run("different strings", func(t *testing.T) {
		t.Parallel()
		assert.False(t, valuesEqual("abc", "xyz"))
	})

	t.Run("equal integers", func(t *testing.T) {
		t.Parallel()
		assert.True(t, valuesEqual(42, 42))
	})

	t.Run("different integers", func(t *testing.T) {
		t.Parallel()
		assert.False(t, valuesEqual(42, 99))
	})
}

func TestTenantedValidator_WithReferenceCheckerSet(t *testing.T) {
	t.Parallel()

	checker := newMockReferenceChecker()
	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithReferenceChecker(checker).
		Build()

	assert.Equal(t, checker, validator.referenceChecker)
}

func TestTenantedValidator_WithCustomEngineConfig(t *testing.T) {
	t.Parallel()

	cfg := &EngineConfig{FailFast: true, MaxParallel: 3}
	validator := NewTenantedValidatorBuilder[*mockTenantedEntity]().
		WithEngineConfig(cfg).
		Build()

	assert.Equal(t, cfg, validator.engineConfig)
}

func TestBunUniquenessChecker_EmptyTableNameValidation(t *testing.T) {
	t.Parallel()

	checker := NewBunUniquenessChecker(nil)

	req := &UniquenessRequest{
		TableName: "",
		Fields:    []FieldCheck{{Column: "name", Value: "test"}},
	}

	_, err := checker.CheckUniqueness(t.Context(), req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "table name is required")
}

func TestBunUniquenessChecker_EmptyFieldsValidation(t *testing.T) {
	t.Parallel()

	checker := NewBunUniquenessChecker(nil)

	req := &UniquenessRequest{
		TableName: "test_table",
		Fields:    []FieldCheck{},
	}

	_, err := checker.CheckUniqueness(t.Context(), req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least one field is required")
}

func TestBunUniquenessCheckerLazy_NilDBFromGetter(t *testing.T) {
	t.Parallel()

	checker := NewBunUniquenessCheckerLazy(func() bun.IDB { return nil })

	req := &UniquenessRequest{
		TableName: "test_table",
		Fields:    []FieldCheck{{Column: "name", Value: "test"}},
	}

	_, err := checker.CheckUniqueness(t.Context(), req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "database connection is not initialized")
}

func TestBunReferenceCheckerLazy_NilDBFromGetter(t *testing.T) {
	t.Parallel()

	checker := NewBunReferenceCheckerLazy(func() bun.IDB { return nil })

	req := &ReferenceRequest{
		TableName: "test_table",
		ID:        pulid.MustNew("ref_"),
	}

	_, err := checker.CheckReference(t.Context(), req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "database connection is not initialized")
}
