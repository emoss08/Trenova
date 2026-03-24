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

type mockScopedEntity struct {
	id            pulid.ID
	tableName     string
	businessUnit  pulid.ID
	name          string
	scacCode      string
	validationErr bool
}

func newMockScopedEntity() *mockScopedEntity {
	return &mockScopedEntity{
		id:           pulid.Nil,
		tableName:    "organizations",
		businessUnit: pulid.MustNew("bu_"),
		name:         "Acme",
		scacCode:     "ACME",
	}
}

func (m *mockScopedEntity) GetID() pulid.ID {
	return m.id
}

func (m *mockScopedEntity) GetTableName() string {
	return m.tableName
}

func (m *mockScopedEntity) Validate(multiErr *errortypes.MultiError) {
	if m.validationErr {
		multiErr.Add("name", errortypes.ErrRequired, "Name is required")
	}
}

func TestNewScopedValidatorBuilder(t *testing.T) {
	t.Parallel()

	validator := NewScopedValidatorBuilder[*mockScopedEntity]().Build()
	require.NotNil(t, validator)
	assert.Equal(t, "Entity", validator.modelName)
	assert.NotNil(t, validator.engineConfig)
}

func TestScopedValidator_ValidateCreate_IDMustBeNil(t *testing.T) {
	t.Parallel()

	entity := newMockScopedEntity()
	entity.id = pulid.MustNew("org_")

	validator := NewScopedValidatorBuilder[*mockScopedEntity]().Build()

	result := validator.ValidateCreate(t.Context(), entity)
	require.NotNil(t, result)

	hasIDError := false
	for _, err := range result.Errors {
		if err.Field == "id" && err.Code == errortypes.ErrInvalid {
			hasIDError = true
		}
	}
	assert.True(t, hasIDError)
}

func TestScopedValidator_ValidateCreate_UniqueConflict(t *testing.T) {
	t.Parallel()

	entity := newMockScopedEntity()
	checker := newMockUniquenessChecker()
	checker.On("CheckUniqueness", mock.Anything, mock.Anything).Return(true, nil)

	validator := NewScopedValidatorBuilder[*mockScopedEntity]().
		WithModelName("Organization").
		WithUniquenessChecker(checker).
		WithScopeField("businessUnitId", "business_unit_id", func(e *mockScopedEntity) any {
			return e.businessUnit
		}).
		WithUniqueField("name", "name", "Organization with this name already exists", func(e *mockScopedEntity) any {
			return e.name
		}).
		Build()

	result := validator.ValidateCreate(t.Context(), entity)
	require.NotNil(t, result)

	hasNameError := false
	for _, err := range result.Errors {
		if err.Field == "name" && err.Code == errortypes.ErrDuplicate {
			hasNameError = true
		}
	}
	assert.True(t, hasNameError)
	checker.AssertExpectations(t)
}

func TestScopedValidator_ValidateUpdate_SendsExcludeIDAndScope(t *testing.T) {
	t.Parallel()

	entity := newMockScopedEntity()
	entity.id = pulid.MustNew("org_")

	checker := newMockUniquenessChecker()
	checker.On("CheckUniqueness", mock.Anything, mock.MatchedBy(func(req *UniquenessRequest) bool {
		if req.ExcludeID != entity.id {
			return false
		}
		if len(req.ScopeFields) != 1 {
			return false
		}
		return req.ScopeFields[0].Column == "business_unit_id"
	})).Return(false, nil)

	validator := NewScopedValidatorBuilder[*mockScopedEntity]().
		WithUniquenessChecker(checker).
		WithScopeField("businessUnitId", "business_unit_id", func(e *mockScopedEntity) any {
			return e.businessUnit
		}).
		WithUniqueField("name", "name", "", func(e *mockScopedEntity) any {
			return e.name
		}).
		Build()

	result := validator.ValidateUpdate(t.Context(), entity)
	assert.Nil(t, result)
	checker.AssertExpectations(t)
}

func TestScopedValidator_ValidateCreate_UniquenessErrorAsSystemError(t *testing.T) {
	t.Parallel()

	entity := newMockScopedEntity()
	checker := newMockUniquenessChecker()
	checker.On("CheckUniqueness", mock.Anything, mock.Anything).
		Return(false, errors.New("db failure"))

	validator := NewScopedValidatorBuilder[*mockScopedEntity]().
		WithUniquenessChecker(checker).
		WithUniqueField("name", "name", "", func(e *mockScopedEntity) any {
			return e.name
		}).
		Build()

	result := validator.ValidateCreate(t.Context(), entity)
	require.NotNil(t, result)

	hasSystemError := false
	for _, err := range result.Errors {
		if err.Field == "system" && err.Code == errortypes.ErrSystemError {
			hasSystemError = true
		}
	}
	assert.True(t, hasSystemError)
	checker.AssertExpectations(t)
}

func TestScopedValidator_CustomRule_OnUpdateOnly(t *testing.T) {
	t.Parallel()

	entity := newMockScopedEntity()
	entity.id = pulid.MustNew("org_")

	executed := false
	rule := NewScopedRule[*mockScopedEntity]("custom_update_only").
		OnUpdate().
		WithValidation(func(
			_ context.Context,
			_ *mockScopedEntity,
			_ *ScopedValidationContext,
			_ *errortypes.MultiError,
		) error {
			executed = true
			return nil
		})

	validator := NewScopedValidatorBuilder[*mockScopedEntity]().
		WithCustomRule(rule).
		Build()

	_ = validator.ValidateCreate(t.Context(), entity)
	assert.False(t, executed)

	_ = validator.ValidateUpdate(t.Context(), entity)
	assert.True(t, executed)
}
