package validationframework

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConcreteRule(t *testing.T) {
	t.Parallel()

	rule := NewConcreteRule("test")

	assert.Equal(t, "test", rule.Name())
	assert.Equal(t, ValidationStageBasic, rule.Stage())
	assert.Equal(t, ValidationPriorityMedium, rule.Priority())
	assert.Equal(t, "", rule.Description())
}

func TestConcreteRule_WithDescription(t *testing.T) {
	t.Parallel()

	rule := NewConcreteRule("test").WithDescription("a description")

	assert.Equal(t, "a description", rule.Description())
}

func TestConcreteRule_WithCondition_True(t *testing.T) {
	t.Parallel()

	executed := false
	rule := NewConcreteRule("test").
		WithCondition(func() bool { return true }).
		WithValidation(func(_ context.Context, _ *errortypes.MultiError) error {
			executed = true
			return nil
		})

	multiErr := errortypes.NewMultiError()
	err := rule.Validate(t.Context(), multiErr)

	require.NoError(t, err)
	assert.True(t, executed)
}

func TestConcreteRule_WithCondition_False(t *testing.T) {
	t.Parallel()

	executed := false
	rule := NewConcreteRule("test").
		WithCondition(func() bool { return false }).
		WithValidation(func(_ context.Context, _ *errortypes.MultiError) error {
			executed = true
			return nil
		})

	multiErr := errortypes.NewMultiError()
	err := rule.Validate(t.Context(), multiErr)

	require.NoError(t, err)
	assert.False(t, executed)
}

func TestConcreteRule_Validate_NilFunction(t *testing.T) {
	t.Parallel()

	rule := NewConcreteRule("no_fn")
	multiErr := errortypes.NewMultiError()
	err := rule.Validate(t.Context(), multiErr)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "validation function not set")
	assert.Contains(t, err.Error(), "no_fn")
}

func TestConcreteRule_Validate_ReturnsError(t *testing.T) {
	t.Parallel()

	rule := NewConcreteRule("err_rule").
		WithValidation(func(_ context.Context, _ *errortypes.MultiError) error {
			return errors.New("validation failed")
		})

	multiErr := errortypes.NewMultiError()
	err := rule.Validate(t.Context(), multiErr)

	require.Error(t, err)
	assert.Equal(t, "validation failed", err.Error())
}

func TestNewFieldValidator(t *testing.T) {
	t.Parallel()

	type entity struct{ Name string }
	fv := NewFieldValidator("name", func(e entity) string { return e.Name })

	require.NotNil(t, fv)
	assert.Equal(t, "name", fv.fieldName)
}

func TestFieldValidator_Required_Empty(t *testing.T) {
	t.Parallel()

	type entity struct{ Name string }
	fv := NewFieldValidator("name", func(e entity) string { return e.Name }).
		Required("Name is required")

	result := fv.Validate(entity{Name: ""})

	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
}

func TestFieldValidator_Required_Filled(t *testing.T) {
	t.Parallel()

	type entity struct{ Name string }
	fv := NewFieldValidator("name", func(e entity) string { return e.Name }).
		Required("Name is required")

	result := fv.Validate(entity{Name: "hello"})

	assert.Nil(t, result)
}

func TestFieldValidator_Custom(t *testing.T) {
	t.Parallel()

	type entity struct{ Age int }
	fv := NewFieldValidator("age", func(e entity) int { return e.Age }).
		Custom(func(val int) *errortypes.Error {
			if val < 18 {
				return errortypes.NewValidationError("age", errortypes.ErrInvalid, "Must be 18+")
			}
			return nil
		})

	result := fv.Validate(entity{Age: 10})
	require.NotNil(t, result)

	result = fv.Validate(entity{Age: 25})
	assert.Nil(t, result)
}

func TestFieldValidator_MultipleValidators(t *testing.T) {
	t.Parallel()

	type entity struct{ Name string }
	fv := NewFieldValidator("name", func(e entity) string { return e.Name }).
		Required("Required").
		Custom(func(val string) *errortypes.Error {
			if len(val) > 5 {
				return errortypes.NewValidationError("name", errortypes.ErrInvalid, "Too long")
			}
			return nil
		})

	result := fv.Validate(entity{Name: ""})
	require.NotNil(t, result)

	result = fv.Validate(entity{Name: "toolongname"})
	require.NotNil(t, result)

	result = fv.Validate(entity{Name: "ok"})
	assert.Nil(t, result)
}

func TestNewFieldRule(t *testing.T) {
	t.Parallel()

	rule := NewFieldRule("email")

	assert.Equal(t, "field_email", rule.Name())
	assert.Equal(t, "email", rule.fieldName)
}

func TestNewBusinessRule(t *testing.T) {
	t.Parallel()

	rule := NewBusinessRule("check_balance").
		WithDependencies("account", "balance")

	assert.Equal(t, "check_balance", rule.Name())
	assert.Equal(t, ValidationStageBusinessRules, rule.Stage())
	assert.Equal(t, []string{"account", "balance"}, rule.GetDependencies())
}

func TestBusinessRule_WithValidation(t *testing.T) {
	t.Parallel()

	executed := false
	rule := NewBusinessRule("test").
		WithValidation(func(_ context.Context, _ *errortypes.MultiError) error {
			executed = true
			return nil
		})

	multiErr := errortypes.NewMultiError()
	err := rule.Validate(t.Context(), multiErr)

	require.NoError(t, err)
	assert.True(t, executed)
}

func TestNewComplianceRule(t *testing.T) {
	t.Parallel()

	rule := NewComplianceRule("FMCSA", "395.1")

	assert.Equal(t, "compliance_FMCSA_395.1", rule.Name())
	assert.Equal(t, ValidationStageCompliance, rule.Stage())
	assert.Equal(t, ValidationPriorityHigh, rule.Priority())
	assert.Equal(t, "FMCSA", rule.GetRegulation())
	assert.Equal(t, "395.1", rule.GetSection())
}

func TestComplianceRule_WithValidation(t *testing.T) {
	t.Parallel()

	rule := NewComplianceRule("DOT", "49.1").
		WithValidation(func(_ context.Context, multiErr *errortypes.MultiError) error {
			multiErr.Add("hours", errortypes.ErrInvalid, "Hours exceeded")
			return nil
		})

	multiErr := errortypes.NewMultiError()
	err := rule.Validate(t.Context(), multiErr)

	require.NoError(t, err)
	assert.True(t, multiErr.HasErrors())
}

func TestNewValidationRule(t *testing.T) {
	t.Parallel()

	rule := NewValidationRule(
		ValidationStageDataIntegrity,
		ValidationPriorityLow,
		func(_ context.Context, _ *errortypes.MultiError) error { return nil },
	)

	assert.Equal(t, ValidationStageDataIntegrity, rule.Stage())
	assert.Equal(t, ValidationPriorityLow, rule.Priority())
	assert.NoError(t, rule.Validate(t.Context(), errortypes.NewMultiError()))
}

func TestNewValidationContext(t *testing.T) {
	t.Parallel()

	ctx := NewValidationContext(&ValidationContext{IsCreate: true, IsUpdate: false})

	assert.True(t, ctx.IsCreate)
	assert.False(t, ctx.IsUpdate)
}

func TestValidatorFunc(t *testing.T) {
	t.Parallel()

	vf := ValidatorFunc[string](func(_ context.Context, entity string) *errortypes.MultiError {
		if entity == "" {
			me := errortypes.NewMultiError()
			me.Add("value", errortypes.ErrRequired, "required")
			return me
		}
		return nil
	})

	result := vf.Validate(t.Context(), "")
	require.NotNil(t, result)

	result = vf.Validate(t.Context(), "hello")
	assert.Nil(t, result)
}

func TestConcreteRule_WithStage(t *testing.T) {
	t.Parallel()

	rule := NewConcreteRule("test").WithStage(ValidationStageCompliance)

	assert.Equal(t, ValidationStageCompliance, rule.Stage())
}

func TestConcreteRule_WithPriority(t *testing.T) {
	t.Parallel()

	rule := NewConcreteRule("test").WithPriority(ValidationPriorityLow)

	assert.Equal(t, ValidationPriorityLow, rule.Priority())
}

func TestConcreteRule_FluentChaining(t *testing.T) {
	t.Parallel()

	executed := false
	rule := NewConcreteRule("chained").
		WithStage(ValidationStageDataIntegrity).
		WithPriority(ValidationPriorityHigh).
		WithDescription("A chained rule").
		WithCondition(func() bool { return true }).
		WithValidation(func(_ context.Context, _ *errortypes.MultiError) error {
			executed = true
			return nil
		})

	assert.Equal(t, "chained", rule.Name())
	assert.Equal(t, ValidationStageDataIntegrity, rule.Stage())
	assert.Equal(t, ValidationPriorityHigh, rule.Priority())
	assert.Equal(t, "A chained rule", rule.Description())

	multiErr := errortypes.NewMultiError()
	err := rule.Validate(t.Context(), multiErr)
	require.NoError(t, err)
	assert.True(t, executed)
}

func TestFieldRule_InheritsConcreteRule(t *testing.T) {
	t.Parallel()

	rule := NewFieldRule("email")

	assert.Equal(t, ValidationStageBasic, rule.Stage())
	assert.Equal(t, ValidationPriorityMedium, rule.Priority())
}

func TestBusinessRule_DefaultStage(t *testing.T) {
	t.Parallel()

	rule := NewBusinessRule("check")

	assert.Equal(t, ValidationStageBusinessRules, rule.Stage())
	assert.Equal(t, ValidationPriorityMedium, rule.Priority())
}

func TestBusinessRule_EmptyDependencies(t *testing.T) {
	t.Parallel()

	rule := NewBusinessRule("check")

	assert.Empty(t, rule.GetDependencies())
}

func TestComplianceRule_WithValidation_NoError(t *testing.T) {
	t.Parallel()

	rule := NewComplianceRule("DOT", "49.1").
		WithValidation(func(_ context.Context, _ *errortypes.MultiError) error {
			return nil
		})

	multiErr := errortypes.NewMultiError()
	err := rule.Validate(t.Context(), multiErr)

	require.NoError(t, err)
	assert.False(t, multiErr.HasErrors())
}

func TestValidationRuleFunc_Implements_Interface(t *testing.T) {
	t.Parallel()

	ruleFunc := ValidationRuleFunc{
		StageFunc:    func() ValidationStage { return ValidationStageCompliance },
		PriorityFunc: func() ValidationPriority { return ValidationPriorityLow },
		ValidateFunc: func(_ context.Context, multiErr *errortypes.MultiError) error {
			multiErr.Add("test", errortypes.ErrInvalid, "test error")
			return nil
		},
	}

	var rule ValidationRule = ruleFunc
	assert.Equal(t, ValidationStageCompliance, rule.Stage())
	assert.Equal(t, ValidationPriorityLow, rule.Priority())

	multiErr := errortypes.NewMultiError()
	err := rule.Validate(t.Context(), multiErr)
	require.NoError(t, err)
	assert.True(t, multiErr.HasErrors())
}

func TestValidationRuleFunc_ReturnsError(t *testing.T) {
	t.Parallel()

	ruleFunc := ValidationRuleFunc{
		StageFunc:    func() ValidationStage { return ValidationStageBasic },
		PriorityFunc: func() ValidationPriority { return ValidationPriorityHigh },
		ValidateFunc: func(_ context.Context, _ *errortypes.MultiError) error {
			return errors.New("system error")
		},
	}

	multiErr := errortypes.NewMultiError()
	err := ruleFunc.Validate(t.Context(), multiErr)
	require.Error(t, err)
	assert.Equal(t, "system error", err.Error())
}

func TestValidationContext_Fields(t *testing.T) {
	t.Parallel()

	t.Run("update mode", func(t *testing.T) {
		t.Parallel()
		ctx := NewValidationContext(&ValidationContext{IsCreate: false, IsUpdate: true})
		assert.False(t, ctx.IsCreate)
		assert.True(t, ctx.IsUpdate)
	})

	t.Run("create mode", func(t *testing.T) {
		t.Parallel()
		ctx := NewValidationContext(&ValidationContext{IsCreate: true, IsUpdate: false})
		assert.True(t, ctx.IsCreate)
		assert.False(t, ctx.IsUpdate)
	})
}

func TestFieldValidator_Custom_NoError(t *testing.T) {
	t.Parallel()

	type entity struct{ Val int }
	fv := NewFieldValidator("val", func(e entity) int { return e.Val }).
		Custom(func(val int) *errortypes.Error {
			return nil
		})

	result := fv.Validate(entity{Val: 42})
	assert.Nil(t, result)
}

func TestFieldValidator_Required_ZeroInt(t *testing.T) {
	t.Parallel()

	type entity struct{ Count int }
	fv := NewFieldValidator("count", func(e entity) int { return e.Count }).
		Required("Count is required")

	result := fv.Validate(entity{Count: 0})
	require.NotNil(t, result)
	assert.True(t, result.HasErrors())

	result = fv.Validate(entity{Count: 5})
	assert.Nil(t, result)
}
