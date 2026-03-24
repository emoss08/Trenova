package validationframework

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultEngineConfig(t *testing.T) {
	t.Parallel()

	cfg := DefaultEngineConfig()

	assert.False(t, cfg.FailFast)
	assert.Equal(t, 10, cfg.MaxParallel)
	assert.False(t, cfg.EnableMetrics)
	assert.False(t, cfg.EnableTracing)
}

func TestNewEngine_NilConfig(t *testing.T) {
	t.Parallel()

	engine := NewEngine(nil)

	require.NotNil(t, engine)
	assert.Equal(t, 0, engine.RuleCount())
	assert.NotNil(t, engine.config)
	assert.Equal(t, 10, engine.config.MaxParallel)
}

func TestNewEngine_CustomConfig(t *testing.T) {
	t.Parallel()

	cfg := &EngineConfig{FailFast: true, MaxParallel: 5}
	engine := NewEngine(cfg)

	require.NotNil(t, engine)
	assert.True(t, engine.config.FailFast)
	assert.Equal(t, 5, engine.config.MaxParallel)
}

func TestEngine_AddRule(t *testing.T) {
	t.Parallel()

	engine := NewEngine(nil)
	rule := NewConcreteRule("test_rule").
		WithStage(ValidationStageBasic).
		WithPriority(ValidationPriorityHigh).
		WithValidation(func(_ context.Context, _ *errortypes.MultiError) error {
			return nil
		})

	engine.AddRule(rule)

	assert.Equal(t, 1, engine.RuleCount())
}

func TestEngine_AddRules(t *testing.T) {
	t.Parallel()

	engine := NewEngine(nil)
	rule1 := NewConcreteRule("rule1").
		WithStage(ValidationStageBasic).
		WithPriority(ValidationPriorityHigh).
		WithValidation(func(_ context.Context, _ *errortypes.MultiError) error { return nil })
	rule2 := NewConcreteRule("rule2").
		WithStage(ValidationStageBasic).
		WithPriority(ValidationPriorityMedium).
		WithValidation(func(_ context.Context, _ *errortypes.MultiError) error { return nil })

	engine.AddRules(rule1, rule2)

	assert.Equal(t, 2, engine.RuleCount())
}

func TestEngine_Validate_NoRules(t *testing.T) {
	t.Parallel()

	engine := NewEngine(nil)
	result := engine.Validate(t.Context())

	assert.Nil(t, result)
}

func TestEngine_Validate_PassingRule(t *testing.T) {
	t.Parallel()

	engine := NewEngine(nil)
	engine.AddRule(NewConcreteRule("pass").
		WithStage(ValidationStageBasic).
		WithPriority(ValidationPriorityHigh).
		WithValidation(func(_ context.Context, _ *errortypes.MultiError) error { return nil }))

	result := engine.Validate(t.Context())

	assert.Nil(t, result)
}

func TestEngine_Validate_FailingRule(t *testing.T) {
	t.Parallel()

	engine := NewEngine(nil)
	engine.AddRule(NewConcreteRule("fail").
		WithStage(ValidationStageBasic).
		WithPriority(ValidationPriorityHigh).
		WithValidation(func(_ context.Context, multiErr *errortypes.MultiError) error {
			multiErr.Add("field", errortypes.ErrRequired, "Field is required")
			return nil
		}))

	result := engine.Validate(t.Context())

	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
}

func TestEngine_Validate_RuleReturnsError(t *testing.T) {
	t.Parallel()

	engine := NewEngine(nil)
	engine.AddRule(NewConcreteRule("system_error").
		WithStage(ValidationStageBasic).
		WithPriority(ValidationPriorityHigh).
		WithValidation(func(_ context.Context, _ *errortypes.MultiError) error {
			return errors.New("system failure")
		}))

	result := engine.Validate(t.Context())

	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
	found := false
	for _, e := range result.Errors {
		if e.Field == "system" && e.Code == errortypes.ErrSystemError {
			found = true
			break
		}
	}
	assert.True(t, found)
}

func TestEngine_Validate_FailFast(t *testing.T) {
	t.Parallel()

	cfg := &EngineConfig{FailFast: true, MaxParallel: 1}
	engine := NewEngine(cfg)

	secondRuleRan := false
	engine.AddRule(NewConcreteRule("first_fail").
		WithStage(ValidationStageBasic).
		WithPriority(ValidationPriorityHigh).
		WithValidation(func(_ context.Context, multiErr *errortypes.MultiError) error {
			multiErr.Add("field1", errortypes.ErrRequired, "Required")
			return nil
		}))
	engine.AddRule(NewConcreteRule("second_rule").
		WithStage(ValidationStageDataIntegrity).
		WithPriority(ValidationPriorityHigh).
		WithValidation(func(_ context.Context, _ *errortypes.MultiError) error {
			secondRuleRan = true
			return nil
		}))

	result := engine.Validate(t.Context())

	require.NotNil(t, result)
	assert.False(t, secondRuleRan)
}

func TestEngine_Validate_MultipleStages(t *testing.T) {
	t.Parallel()

	engine := NewEngine(nil)
	order := make([]string, 0, 3)

	engine.AddRule(NewConcreteRule("compliance").
		WithStage(ValidationStageCompliance).
		WithPriority(ValidationPriorityHigh).
		WithValidation(func(_ context.Context, _ *errortypes.MultiError) error {
			order = append(order, "compliance")
			return nil
		}))
	engine.AddRule(NewConcreteRule("basic").
		WithStage(ValidationStageBasic).
		WithPriority(ValidationPriorityHigh).
		WithValidation(func(_ context.Context, _ *errortypes.MultiError) error {
			order = append(order, "basic")
			return nil
		}))
	engine.AddRule(NewConcreteRule("business").
		WithStage(ValidationStageBusinessRules).
		WithPriority(ValidationPriorityHigh).
		WithValidation(func(_ context.Context, _ *errortypes.MultiError) error {
			order = append(order, "business")
			return nil
		}))

	engine.Validate(t.Context())

	require.Len(t, order, 3)
	assert.Equal(t, "basic", order[0])
	assert.Equal(t, "business", order[1])
	assert.Equal(t, "compliance", order[2])
}

func TestEngine_Validate_ParallelExecution(t *testing.T) {
	t.Parallel()

	cfg := &EngineConfig{MaxParallel: 5}
	engine := NewEngine(cfg)

	engine.AddRule(NewConcreteRule("rule1").
		WithStage(ValidationStageBasic).
		WithPriority(ValidationPriorityHigh).
		WithValidation(func(_ context.Context, multiErr *errortypes.MultiError) error {
			multiErr.Add("f1", errortypes.ErrRequired, "err1")
			return nil
		}))
	engine.AddRule(NewConcreteRule("rule2").
		WithStage(ValidationStageBasic).
		WithPriority(ValidationPriorityHigh).
		WithValidation(func(_ context.Context, multiErr *errortypes.MultiError) error {
			multiErr.Add("f2", errortypes.ErrRequired, "err2")
			return nil
		}))

	result := engine.Validate(t.Context())

	require.NotNil(t, result)
	assert.Len(t, result.Errors, 2)
}

func TestEngine_Validate_ParallelWithSystemError(t *testing.T) {
	t.Parallel()

	cfg := &EngineConfig{MaxParallel: 5}
	engine := NewEngine(cfg)

	engine.AddRule(NewConcreteRule("rule1").
		WithStage(ValidationStageBasic).
		WithPriority(ValidationPriorityHigh).
		WithValidation(func(_ context.Context, _ *errortypes.MultiError) error {
			return errors.New("system error in parallel")
		}))
	engine.AddRule(NewConcreteRule("rule2").
		WithStage(ValidationStageBasic).
		WithPriority(ValidationPriorityHigh).
		WithValidation(func(_ context.Context, _ *errortypes.MultiError) error {
			return nil
		}))

	result := engine.Validate(t.Context())

	require.NotNil(t, result)
	found := false
	for _, e := range result.Errors {
		if e.Field == "system" {
			found = true
			break
		}
	}
	assert.True(t, found)
}

func TestEngine_ForField(t *testing.T) {
	t.Parallel()

	parent := errortypes.NewMultiError()
	engine := NewEngine(nil).ForField("address").WithParent(parent)

	engine.AddRule(NewConcreteRule("nested").
		WithStage(ValidationStageBasic).
		WithPriority(ValidationPriorityHigh).
		WithValidation(func(_ context.Context, multiErr *errortypes.MultiError) error {
			multiErr.Add("street", errortypes.ErrRequired, "Street is required")
			return nil
		}))

	result := engine.Validate(t.Context())

	assert.Nil(t, result)
	assert.True(t, parent.HasErrors())
}

func TestEngine_AtIndex(t *testing.T) {
	t.Parallel()

	parent := errortypes.NewMultiError()
	engine := NewEngine(nil).ForField("items").AtIndex(0).WithParent(parent)

	engine.AddRule(NewConcreteRule("indexed").
		WithStage(ValidationStageBasic).
		WithPriority(ValidationPriorityHigh).
		WithValidation(func(_ context.Context, multiErr *errortypes.MultiError) error {
			multiErr.Add("name", errortypes.ErrRequired, "Name is required")
			return nil
		}))

	result := engine.Validate(t.Context())

	assert.Nil(t, result)
	assert.True(t, parent.HasErrors())
}

func TestEngine_ValidateInto(t *testing.T) {
	t.Parallel()

	engine := NewEngine(nil)
	engine.AddRule(NewConcreteRule("rule").
		WithStage(ValidationStageBasic).
		WithPriority(ValidationPriorityHigh).
		WithValidation(func(_ context.Context, multiErr *errortypes.MultiError) error {
			multiErr.Add("x", errortypes.ErrRequired, "X required")
			return nil
		}))

	multiErr := errortypes.NewMultiError()
	engine.ValidateInto(t.Context(), multiErr)

	assert.True(t, multiErr.HasErrors())
}

func TestEngine_Clear(t *testing.T) {
	t.Parallel()

	engine := NewEngine(nil)
	engine.AddRule(NewConcreteRule("rule").
		WithStage(ValidationStageBasic).
		WithPriority(ValidationPriorityHigh).
		WithValidation(func(_ context.Context, _ *errortypes.MultiError) error { return nil }))

	assert.Equal(t, 1, engine.RuleCount())

	engine.Clear()

	assert.Equal(t, 0, engine.RuleCount())
}

func TestEngine_RulesByStageAndPriority(t *testing.T) {
	t.Parallel()

	engine := NewEngine(nil)
	engine.AddRule(NewConcreteRule("basic_high").
		WithStage(ValidationStageBasic).
		WithPriority(ValidationPriorityHigh).
		WithValidation(func(_ context.Context, _ *errortypes.MultiError) error { return nil }))
	engine.AddRule(NewConcreteRule("basic_medium").
		WithStage(ValidationStageBasic).
		WithPriority(ValidationPriorityMedium).
		WithValidation(func(_ context.Context, _ *errortypes.MultiError) error { return nil }))

	highRules := engine.RulesByStageAndPriority(ValidationStageBasic, ValidationPriorityHigh)
	assert.Len(t, highRules, 1)

	medRules := engine.RulesByStageAndPriority(ValidationStageBasic, ValidationPriorityMedium)
	assert.Len(t, medRules, 1)

	noRules := engine.RulesByStageAndPriority(ValidationStageCompliance, ValidationPriorityLow)
	assert.Nil(t, noRules)
}

func TestEngine_WithConfig(t *testing.T) {
	t.Parallel()

	engine := NewEngine(nil)
	newCfg := &EngineConfig{FailFast: true, MaxParallel: 3}
	engine.WithConfig(newCfg)

	assert.True(t, engine.config.FailFast)
	assert.Equal(t, 3, engine.config.MaxParallel)
}

func TestEngine_String(t *testing.T) {
	t.Parallel()

	engine := NewEngine(nil)
	engine.AddRule(NewConcreteRule("rule").
		WithStage(ValidationStageBasic).
		WithPriority(ValidationPriorityHigh).
		WithValidation(func(_ context.Context, _ *errortypes.MultiError) error { return nil }))

	s := engine.String()

	assert.Contains(t, s, "ValidationEngine")
	assert.Contains(t, s, "rules: 1")
}

func TestEngine_PriorityToErrorPriority(t *testing.T) {
	t.Parallel()

	engine := NewEngine(nil)

	assert.Equal(t, errortypes.PriorityHigh, engine.priorityToErrorPriority(ValidationPriorityHigh))
	assert.Equal(
		t,
		errortypes.PriorityMedium,
		engine.priorityToErrorPriority(ValidationPriorityMedium),
	)
	assert.Equal(t, errortypes.PriorityLow, engine.priorityToErrorPriority(ValidationPriorityLow))
	assert.Equal(
		t,
		errortypes.PriorityHigh,
		engine.priorityToErrorPriority(ValidationPriority(999)),
	)
}

func TestEngine_SequentialExecutionWithFailFast(t *testing.T) {
	t.Parallel()

	cfg := &EngineConfig{FailFast: true, MaxParallel: 1}
	engine := NewEngine(cfg)

	thirdRan := false
	engine.AddRule(NewConcreteRule("r1").
		WithStage(ValidationStageBasic).
		WithPriority(ValidationPriorityHigh).
		WithValidation(func(_ context.Context, multiErr *errortypes.MultiError) error {
			multiErr.Add("f1", errortypes.ErrRequired, "err")
			return nil
		}))
	engine.AddRule(NewConcreteRule("r2").
		WithStage(ValidationStageBasic).
		WithPriority(ValidationPriorityHigh).
		WithValidation(func(_ context.Context, _ *errortypes.MultiError) error {
			thirdRan = true
			return nil
		}))

	result := engine.Validate(t.Context())

	require.NotNil(t, result)
	assert.False(t, thirdRan)
}

func TestEngine_ValidateInto_WithIndexedContext(t *testing.T) {
	t.Parallel()

	engine := NewEngine(nil).ForField("items").AtIndex(2)

	engine.AddRule(NewConcreteRule("indexed_rule").
		WithStage(ValidationStageBasic).
		WithPriority(ValidationPriorityHigh).
		WithValidation(func(_ context.Context, multiErr *errortypes.MultiError) error {
			multiErr.Add("name", errortypes.ErrRequired, "Name is required")
			return nil
		}))

	multiErr := errortypes.NewMultiError()
	engine.ValidateInto(t.Context(), multiErr)

	assert.True(t, multiErr.HasErrors())
}

func TestEngine_ValidateInto_WithFieldContext(t *testing.T) {
	t.Parallel()

	engine := NewEngine(nil).ForField("address")

	engine.AddRule(NewConcreteRule("field_rule").
		WithStage(ValidationStageBasic).
		WithPriority(ValidationPriorityHigh).
		WithValidation(func(_ context.Context, multiErr *errortypes.MultiError) error {
			multiErr.Add("city", errortypes.ErrRequired, "City is required")
			return nil
		}))

	multiErr := errortypes.NewMultiError()
	engine.ValidateInto(t.Context(), multiErr)

	assert.True(t, multiErr.HasErrors())
}

func TestEngine_ValidateInto_WithoutContext(t *testing.T) {
	t.Parallel()

	engine := NewEngine(nil)

	engine.AddRule(NewConcreteRule("plain_rule").
		WithStage(ValidationStageBasic).
		WithPriority(ValidationPriorityHigh).
		WithValidation(func(_ context.Context, multiErr *errortypes.MultiError) error {
			multiErr.Add("field", errortypes.ErrRequired, "Field required")
			return nil
		}))

	multiErr := errortypes.NewMultiError()
	engine.ValidateInto(t.Context(), multiErr)

	assert.True(t, multiErr.HasErrors())
}

func TestEngine_Validate_IndexedWithoutParent(t *testing.T) {
	t.Parallel()

	engine := NewEngine(nil).ForField("items").AtIndex(0)

	engine.AddRule(NewConcreteRule("rule").
		WithStage(ValidationStageBasic).
		WithPriority(ValidationPriorityHigh).
		WithValidation(func(_ context.Context, multiErr *errortypes.MultiError) error {
			multiErr.Add("x", errortypes.ErrRequired, "X required")
			return nil
		}))

	result := engine.Validate(t.Context())

	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
}

func TestEngine_Validate_FieldWithoutParent(t *testing.T) {
	t.Parallel()

	engine := NewEngine(nil).ForField("nested")

	engine.AddRule(NewConcreteRule("rule").
		WithStage(ValidationStageBasic).
		WithPriority(ValidationPriorityHigh).
		WithValidation(func(_ context.Context, multiErr *errortypes.MultiError) error {
			multiErr.Add("y", errortypes.ErrRequired, "Y required")
			return nil
		}))

	result := engine.Validate(t.Context())

	require.NotNil(t, result)
	assert.True(t, result.HasErrors())
}

func TestEngine_Validate_PriorityOrdering(t *testing.T) {
	t.Parallel()

	cfg := &EngineConfig{MaxParallel: 1}
	engine := NewEngine(cfg)
	order := make([]string, 0, 3)

	engine.AddRule(NewConcreteRule("low").
		WithStage(ValidationStageBasic).
		WithPriority(ValidationPriorityLow).
		WithValidation(func(_ context.Context, _ *errortypes.MultiError) error {
			order = append(order, "low")
			return nil
		}))
	engine.AddRule(NewConcreteRule("high").
		WithStage(ValidationStageBasic).
		WithPriority(ValidationPriorityHigh).
		WithValidation(func(_ context.Context, _ *errortypes.MultiError) error {
			order = append(order, "high")
			return nil
		}))
	engine.AddRule(NewConcreteRule("medium").
		WithStage(ValidationStageBasic).
		WithPriority(ValidationPriorityMedium).
		WithValidation(func(_ context.Context, _ *errortypes.MultiError) error {
			order = append(order, "medium")
			return nil
		}))

	engine.Validate(t.Context())

	require.Len(t, order, 3)
	assert.Equal(t, "high", order[0])
	assert.Equal(t, "medium", order[1])
	assert.Equal(t, "low", order[2])
}

func TestEngine_Validate_FailFastAtPriorityLevel(t *testing.T) {
	t.Parallel()

	cfg := &EngineConfig{FailFast: true, MaxParallel: 1}
	engine := NewEngine(cfg)

	mediumRan := false
	engine.AddRule(NewConcreteRule("high_fail").
		WithStage(ValidationStageBasic).
		WithPriority(ValidationPriorityHigh).
		WithValidation(func(_ context.Context, multiErr *errortypes.MultiError) error {
			multiErr.Add("f", errortypes.ErrRequired, "err")
			return nil
		}))
	engine.AddRule(NewConcreteRule("medium_rule").
		WithStage(ValidationStageBasic).
		WithPriority(ValidationPriorityMedium).
		WithValidation(func(_ context.Context, _ *errortypes.MultiError) error {
			mediumRan = true
			return nil
		}))

	result := engine.Validate(t.Context())

	require.NotNil(t, result)
	assert.False(t, mediumRan)
}

func TestEngine_AddRule_ReturnsSelf(t *testing.T) {
	t.Parallel()

	engine := NewEngine(nil)
	rule := NewConcreteRule("test").
		WithStage(ValidationStageBasic).
		WithPriority(ValidationPriorityHigh).
		WithValidation(func(_ context.Context, _ *errortypes.MultiError) error { return nil })

	result := engine.AddRule(rule)

	assert.Same(t, engine, result)
}

func TestEngine_Clear_ReturnsSelf(t *testing.T) {
	t.Parallel()

	engine := NewEngine(nil)
	result := engine.Clear()

	assert.Same(t, engine, result)
}

func TestEngine_Validate_ParallelWithMaxZero(t *testing.T) {
	t.Parallel()

	cfg := &EngineConfig{MaxParallel: 0}
	engine := NewEngine(cfg)

	engine.AddRule(NewConcreteRule("r1").
		WithStage(ValidationStageBasic).
		WithPriority(ValidationPriorityHigh).
		WithValidation(func(_ context.Context, multiErr *errortypes.MultiError) error {
			multiErr.Add("a", errortypes.ErrRequired, "A required")
			return nil
		}))
	engine.AddRule(NewConcreteRule("r2").
		WithStage(ValidationStageBasic).
		WithPriority(ValidationPriorityHigh).
		WithValidation(func(_ context.Context, multiErr *errortypes.MultiError) error {
			multiErr.Add("b", errortypes.ErrRequired, "B required")
			return nil
		}))

	result := engine.Validate(t.Context())

	require.NotNil(t, result)
	assert.Len(t, result.Errors, 2)
}
