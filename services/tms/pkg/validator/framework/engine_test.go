package framework_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validator/framework"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidationEngine_BasicValidation(t *testing.T) {
	t.Run("Single Rule Success", func(t *testing.T) {
		engine := framework.NewValidationEngine(framework.DefaultEngineConfig())

		rule := framework.NewConcreteRule("test_rule").
			WithValidation(func(_ context.Context, me *errortypes.MultiError) error {
				return nil
			})

		engine.AddRule(rule)
		multiErr := engine.Validate(context.Background())

		assert.Nil(t, multiErr)
	})

	t.Run("Single Rule With Error", func(t *testing.T) {
		engine := framework.NewValidationEngine(framework.DefaultEngineConfig())

		rule := framework.NewConcreteRule("test_rule").
			WithValidation(func(_ context.Context, me *errortypes.MultiError) error {
				me.Add("email", errortypes.ErrRequired, "Email is required")
				return nil
			})

		engine.AddRule(rule)
		multiErr := engine.Validate(context.Background())

		require.NotNil(t, multiErr)
		assert.True(t, multiErr.HasErrors())
		assert.Len(t, multiErr.Errors, 1)
		assert.Equal(t, "email", multiErr.Errors[0].Field)
	})

	t.Run("Multiple Rules", func(t *testing.T) {
		engine := framework.NewValidationEngine(framework.DefaultEngineConfig())

		rule1 := framework.NewConcreteRule("rule1").
			WithValidation(func(_ context.Context, me *errortypes.MultiError) error {
				me.Add("email", errortypes.ErrRequired, "Email is required")
				return nil
			})

		rule2 := framework.NewConcreteRule("rule2").
			WithValidation(func(_ context.Context, me *errortypes.MultiError) error {
				me.Add("password", errortypes.ErrRequired, "Password is required")
				return nil
			})

		engine.AddRule(rule1)
		engine.AddRule(rule2)
		multiErr := engine.Validate(context.Background())

		require.NotNil(t, multiErr)
		assert.Len(t, multiErr.Errors, 2)
	})
}

func TestValidationEngine_ForField(t *testing.T) {
	t.Run("Field Prefix Applied", func(t *testing.T) {
		engine := framework.NewValidationEngine(framework.DefaultEngineConfig()).
			ForField("billingProfile")

		rule := framework.NewConcreteRule("test_rule").
			WithValidation(func(_ context.Context, me *errortypes.MultiError) error {
				me.Add("revenueAccountId", errortypes.ErrInvalid, "Invalid revenue account")
				return nil
			})

		engine.AddRule(rule)
		multiErr := engine.Validate(context.Background())

		require.NotNil(t, multiErr)
		assert.Len(t, multiErr.Errors, 1)
		assert.Equal(t, "billingProfile.revenueAccountId", multiErr.Errors[0].Field)
	})

	t.Run("Field Prefix With Multiple Errors", func(t *testing.T) {
		engine := framework.NewValidationEngine(framework.DefaultEngineConfig()).
			ForField("user")

		rule := framework.NewConcreteRule("test_rule").
			WithValidation(func(_ context.Context, me *errortypes.MultiError) error {
				me.Add("email", errortypes.ErrRequired, "Email is required")
				me.Add("password", errortypes.ErrRequired, "Password is required")
				return nil
			})

		engine.AddRule(rule)
		multiErr := engine.Validate(context.Background())

		require.NotNil(t, multiErr)
		assert.Len(t, multiErr.Errors, 2)
		assert.Equal(t, "user.email", multiErr.Errors[0].Field)
		assert.Equal(t, "user.password", multiErr.Errors[1].Field)
	})

	t.Run("Field Prefix With Empty Field Name", func(t *testing.T) {
		engine := framework.NewValidationEngine(framework.DefaultEngineConfig()).
			ForField("billingProfile")

		rule := framework.NewConcreteRule("test_rule").
			WithValidation(func(_ context.Context, me *errortypes.MultiError) error {
				me.Add("", errortypes.ErrInvalid, "General error")
				return nil
			})

		engine.AddRule(rule)
		multiErr := engine.Validate(context.Background())

		require.NotNil(t, multiErr)
		assert.Len(t, multiErr.Errors, 1)
		assert.Equal(t, "billingProfile", multiErr.Errors[0].Field)
	})
}

func TestValidationEngine_WithParent(t *testing.T) {
	t.Run("Parent MultiError Receives Errors", func(t *testing.T) {
		parent := errortypes.NewMultiError()
		engine := framework.NewValidationEngine(framework.DefaultEngineConfig()).
			ForField("profile").
			WithParent(parent)

		rule := framework.NewConcreteRule("test_rule").
			WithValidation(func(_ context.Context, me *errortypes.MultiError) error {
				me.Add("name", errortypes.ErrRequired, "Name is required")
				return nil
			})

		engine.AddRule(rule)
		result := engine.Validate(context.Background())

		assert.Nil(t, result)
		assert.True(t, parent.HasErrors())
		assert.Len(t, parent.Errors, 1)
		assert.Equal(t, "profile.name", parent.Errors[0].Field)
	})

	t.Run("ValidateInto With Field Prefix", func(t *testing.T) {
		parent := errortypes.NewMultiError()
		engine := framework.NewValidationEngine(framework.DefaultEngineConfig()).
			ForField("address")

		rule := framework.NewConcreteRule("test_rule").
			WithValidation(func(_ context.Context, me *errortypes.MultiError) error {
				me.Add("street", errortypes.ErrRequired, "Street is required")
				me.Add("city", errortypes.ErrRequired, "City is required")
				return nil
			})

		engine.AddRule(rule)
		engine.ValidateInto(context.Background(), parent)

		assert.True(t, parent.HasErrors())
		assert.Len(t, parent.Errors, 2)
		assert.Equal(t, "address.street", parent.Errors[0].Field)
		assert.Equal(t, "address.city", parent.Errors[1].Field)
	})
}

func TestValidationEngine_AtIndex(t *testing.T) {
	t.Run("Index Prefix Applied", func(t *testing.T) {
		engine := framework.NewValidationEngine(framework.DefaultEngineConfig()).
			ForField("items").
			AtIndex(0)

		rule := framework.NewConcreteRule("test_rule").
			WithValidation(func(_ context.Context, me *errortypes.MultiError) error {
				me.Add("name", errortypes.ErrRequired, "Name is required")
				return nil
			})

		engine.AddRule(rule)
		multiErr := engine.Validate(context.Background())

		require.NotNil(t, multiErr)
		assert.Len(t, multiErr.Errors, 1)
		assert.Equal(t, "items[0].name", multiErr.Errors[0].Field)
	})

	t.Run("Index With Parent", func(t *testing.T) {
		parent := errortypes.NewMultiError()
		engine := framework.NewValidationEngine(framework.DefaultEngineConfig()).
			ForField("contacts").
			AtIndex(2).
			WithParent(parent)

		rule := framework.NewConcreteRule("test_rule").
			WithValidation(func(_ context.Context, me *errortypes.MultiError) error {
				me.Add("email", errortypes.ErrInvalidFormat, "Invalid email format")
				return nil
			})

		engine.AddRule(rule)
		engine.Validate(context.Background())

		assert.True(t, parent.HasErrors())
		assert.Len(t, parent.Errors, 1)
		assert.Equal(t, "contacts[2].email", parent.Errors[0].Field)
	})
}

func TestValidationEngine_NestedPrefixes(t *testing.T) {
	t.Run("Multiple Nested Prefixes", func(t *testing.T) {
		parent := errortypes.NewMultiError()
		level1 := parent.WithPrefix("customer")
		level2 := level1.WithPrefix("billingProfile")

		engine := framework.NewValidationEngine(framework.DefaultEngineConfig())
		rule := framework.NewConcreteRule("test_rule").
			WithValidation(func(_ context.Context, me *errortypes.MultiError) error {
				me.Add("revenueAccountId", errortypes.ErrInvalid, "Invalid account")
				return nil
			})

		engine.AddRule(rule)
		engine.ValidateInto(context.Background(), level2)

		assert.True(t, parent.HasErrors())
		assert.Len(t, parent.Errors, 1)
		assert.Equal(t, "customer.billingProfile.revenueAccountId", parent.Errors[0].Field)
	})

	t.Run("Prefix With Index In Nested Structure", func(t *testing.T) {
		parent := errortypes.NewMultiError()
		level1 := parent.WithPrefix("order")
		level2 := level1.WithIndex("items", 0)
		level3 := level2.WithPrefix("product")

		engine := framework.NewValidationEngine(framework.DefaultEngineConfig())
		rule := framework.NewConcreteRule("test_rule").
			WithValidation(func(_ context.Context, me *errortypes.MultiError) error {
				me.Add("sku", errortypes.ErrRequired, "SKU is required")
				return nil
			})

		engine.AddRule(rule)
		engine.ValidateInto(context.Background(), level3)

		assert.True(t, parent.HasErrors())
		assert.Len(t, parent.Errors, 1)
		assert.Equal(t, "order.items[0].product.sku", parent.Errors[0].Field)
	})
}

func TestValidationEngine_Stages(t *testing.T) {
	t.Run("Rules Execute In Stage Order", func(t *testing.T) {
		engine := framework.NewValidationEngine(framework.DefaultEngineConfig())
		executionOrder := []string{}

		basicRule := framework.NewConcreteRule("basic").
			WithStage(framework.ValidationStageBasic).
			WithValidation(func(_ context.Context, me *errortypes.MultiError) error {
				executionOrder = append(executionOrder, "basic")
				return nil
			})

		businessRule := framework.NewConcreteRule("business").
			WithStage(framework.ValidationStageBusinessRules).
			WithValidation(func(_ context.Context, me *errortypes.MultiError) error {
				executionOrder = append(executionOrder, "business")
				return nil
			})

		dataIntegrityRule := framework.NewConcreteRule("data").
			WithStage(framework.ValidationStageDataIntegrity).
			WithValidation(func(_ context.Context, me *errortypes.MultiError) error {
				executionOrder = append(executionOrder, "data")
				return nil
			})

		engine.AddRule(businessRule)
		engine.AddRule(dataIntegrityRule)
		engine.AddRule(basicRule)

		engine.Validate(context.Background())

		assert.Equal(t, []string{"basic", "data", "business"}, executionOrder)
	})
}

func TestValidationEngine_Priority(t *testing.T) {
	t.Run("Rules Execute In Priority Order Within Stage", func(t *testing.T) {
		engine := framework.NewValidationEngine(framework.DefaultEngineConfig())
		executionOrder := []string{}

		highRule := framework.NewConcreteRule("high").
			WithStage(framework.ValidationStageBasic).
			WithPriority(framework.ValidationPriorityHigh).
			WithValidation(func(_ context.Context, me *errortypes.MultiError) error {
				executionOrder = append(executionOrder, "high")
				return nil
			})

		mediumRule := framework.NewConcreteRule("medium").
			WithStage(framework.ValidationStageBasic).
			WithPriority(framework.ValidationPriorityMedium).
			WithValidation(func(_ context.Context, me *errortypes.MultiError) error {
				executionOrder = append(executionOrder, "medium")
				return nil
			})

		lowRule := framework.NewConcreteRule("low").
			WithStage(framework.ValidationStageBasic).
			WithPriority(framework.ValidationPriorityLow).
			WithValidation(func(_ context.Context, me *errortypes.MultiError) error {
				executionOrder = append(executionOrder, "low")
				return nil
			})

		engine.AddRule(lowRule)
		engine.AddRule(mediumRule)
		engine.AddRule(highRule)

		engine.Validate(context.Background())

		assert.Equal(t, []string{"high", "medium", "low"}, executionOrder)
	})

	t.Run("Priority Sets Error Priority", func(t *testing.T) {
		engine := framework.NewValidationEngine(framework.DefaultEngineConfig())

		highRule := framework.NewConcreteRule("high").
			WithPriority(framework.ValidationPriorityHigh).
			WithValidation(func(_ context.Context, me *errortypes.MultiError) error {
				me.Add("field1", errortypes.ErrRequired, "High priority error")
				return nil
			})

		mediumRule := framework.NewConcreteRule("medium").
			WithPriority(framework.ValidationPriorityMedium).
			WithValidation(func(_ context.Context, me *errortypes.MultiError) error {
				me.Add("field2", errortypes.ErrRequired, "Medium priority error")
				return nil
			})

		engine.AddRule(highRule)
		engine.AddRule(mediumRule)
		multiErr := engine.Validate(context.Background())

		require.NotNil(t, multiErr)
		assert.Len(t, multiErr.Errors, 2)

		for _, err := range multiErr.Errors {
			switch err.Field {
			case "field1":
				assert.Equal(t, errortypes.PriorityHigh, err.Priority)
			case "field2":
				assert.Equal(t, errortypes.PriorityMedium, err.Priority)
			default:
				assert.Fail(t, "unexpected field: %s", err.Field)
			}
		}
	})
}

func TestValidationEngine_FailFast(t *testing.T) {
	t.Run("FailFast Stops After First Error", func(t *testing.T) {
		config := &framework.EngineConfig{
			FailFast:    true,
			MaxParallel: 1,
		}
		engine := framework.NewValidationEngine(config)
		executionCount := 0

		rule1 := framework.NewConcreteRule("rule1").
			WithValidation(func(_ context.Context, me *errortypes.MultiError) error {
				executionCount++
				me.Add("field1", errortypes.ErrRequired, "Error 1")
				return nil
			})

		rule2 := framework.NewConcreteRule("rule2").
			WithValidation(func(_ context.Context, me *errortypes.MultiError) error {
				executionCount++
				me.Add("field2", errortypes.ErrRequired, "Error 2")
				return nil
			})

		engine.AddRule(rule1)
		engine.AddRule(rule2)
		multiErr := engine.Validate(context.Background())

		assert.Equal(t, 1, executionCount, "Only first rule should execute")
		require.NotNil(t, multiErr)
		assert.Len(t, multiErr.Errors, 1)
	})

	t.Run("FailFast False Executes All Rules", func(t *testing.T) {
		config := &framework.EngineConfig{
			FailFast:    false,
			MaxParallel: 1,
		}
		engine := framework.NewValidationEngine(config)
		executionCount := 0

		rule1 := framework.NewConcreteRule("rule1").
			WithValidation(func(_ context.Context, me *errortypes.MultiError) error {
				executionCount++
				me.Add("field1", errortypes.ErrRequired, "Error 1")
				return nil
			})

		rule2 := framework.NewConcreteRule("rule2").
			WithValidation(func(_ context.Context, me *errortypes.MultiError) error {
				executionCount++
				me.Add("field2", errortypes.ErrRequired, "Error 2")
				return nil
			})

		engine.AddRule(rule1)
		engine.AddRule(rule2)
		multiErr := engine.Validate(context.Background())

		assert.Equal(t, 2, executionCount, "Both rules should execute")
		require.NotNil(t, multiErr)
		assert.Len(t, multiErr.Errors, 2)
	})
}

func TestValidationEngine_RuleConditions(t *testing.T) {
	t.Run("Rule With Condition True Executes", func(t *testing.T) {
		engine := framework.NewValidationEngine(framework.DefaultEngineConfig())
		executed := false

		rule := framework.NewConcreteRule("conditional").
			WithCondition(func() bool {
				return true
			}).
			WithValidation(func(_ context.Context, me *errortypes.MultiError) error {
				executed = true
				return nil
			})

		engine.AddRule(rule)
		engine.Validate(context.Background())

		assert.True(t, executed, "Rule should have executed")
	})

	t.Run("Rule With Condition False Skips", func(t *testing.T) {
		engine := framework.NewValidationEngine(framework.DefaultEngineConfig())
		executed := false

		rule := framework.NewConcreteRule("conditional").
			WithCondition(func() bool {
				return false
			}).
			WithValidation(func(_ context.Context, me *errortypes.MultiError) error {
				executed = true
				return nil
			})

		engine.AddRule(rule)
		engine.Validate(context.Background())

		assert.False(t, executed, "Rule should not have executed")
	})
}

func TestValidationEngine_AddRules(t *testing.T) {
	t.Run("AddRules Adds Multiple Rules", func(t *testing.T) {
		engine := framework.NewValidationEngine(framework.DefaultEngineConfig())

		rule1 := framework.NewConcreteRule("rule1").
			WithValidation(func(_ context.Context, me *errortypes.MultiError) error {
				me.Add("field1", errortypes.ErrRequired, "Error 1")
				return nil
			})

		rule2 := framework.NewConcreteRule("rule2").
			WithValidation(func(_ context.Context, me *errortypes.MultiError) error {
				me.Add("field2", errortypes.ErrRequired, "Error 2")
				return nil
			})

		engine.AddRules(rule1, rule2)
		assert.Equal(t, 2, engine.RuleCount())

		multiErr := engine.Validate(context.Background())
		require.NotNil(t, multiErr)
		assert.Len(t, multiErr.Errors, 2)
	})
}

func TestValidationEngine_Clear(t *testing.T) {
	t.Run("Clear Removes All Rules", func(t *testing.T) {
		engine := framework.NewValidationEngine(framework.DefaultEngineConfig())

		rule := framework.NewConcreteRule("test").
			WithValidation(func(_ context.Context, me *errortypes.MultiError) error {
				me.Add("field", errortypes.ErrRequired, "Error")
				return nil
			})

		engine.AddRule(rule)
		assert.Equal(t, 1, engine.RuleCount())

		engine.Clear()
		assert.Equal(t, 0, engine.RuleCount())

		multiErr := engine.Validate(context.Background())
		assert.Nil(t, multiErr, "No rules should execute after clear")
	})
}

func TestValidationEngine_RulesByStageAndPriority(t *testing.T) {
	t.Run("Retrieve Rules By Stage And Priority", func(t *testing.T) {
		engine := framework.NewValidationEngine(framework.DefaultEngineConfig())

		rule1 := framework.NewConcreteRule("rule1").
			WithStage(framework.ValidationStageBasic).
			WithPriority(framework.ValidationPriorityHigh).
			WithValidation(func(_ context.Context, me *errortypes.MultiError) error {
				return nil
			})

		rule2 := framework.NewConcreteRule("rule2").
			WithStage(framework.ValidationStageBasic).
			WithPriority(framework.ValidationPriorityMedium).
			WithValidation(func(_ context.Context, me *errortypes.MultiError) error {
				return nil
			})

		rule3 := framework.NewConcreteRule("rule3").
			WithStage(framework.ValidationStageDataIntegrity).
			WithPriority(framework.ValidationPriorityHigh).
			WithValidation(func(_ context.Context, me *errortypes.MultiError) error {
				return nil
			})

		engine.AddRules(rule1, rule2, rule3)

		basicHighRules := engine.RulesByStageAndPriority(
			framework.ValidationStageBasic,
			framework.ValidationPriorityHigh,
		)
		assert.Len(t, basicHighRules, 1)

		basicMediumRules := engine.RulesByStageAndPriority(
			framework.ValidationStageBasic,
			framework.ValidationPriorityMedium,
		)
		assert.Len(t, basicMediumRules, 1)

		dataHighRules := engine.RulesByStageAndPriority(
			framework.ValidationStageDataIntegrity,
			framework.ValidationPriorityHigh,
		)
		assert.Len(t, dataHighRules, 1)

		nonExistentRules := engine.RulesByStageAndPriority(
			framework.ValidationStageCompliance,
			framework.ValidationPriorityLow,
		)
		assert.Nil(t, nonExistentRules)
	})
}

func TestValidationEngine_ParallelExecution(t *testing.T) {
	t.Run("Parallel Execution Handles Multiple Rules", func(t *testing.T) {
		config := &framework.EngineConfig{
			FailFast:    false,
			MaxParallel: 5,
		}
		engine := framework.NewValidationEngine(config)

		for i := range 5 {
			index := i
			rule := framework.NewConcreteRule("rule" + string(rune(index))).
				WithValidation(func(_ context.Context, me *errortypes.MultiError) error {
					me.Add("field", errortypes.ErrRequired, "Error")
					return nil
				})
			engine.AddRule(rule)
		}

		multiErr := engine.Validate(context.Background())
		require.NotNil(t, multiErr)
		assert.Len(t, multiErr.Errors, 5)
	})
}

func TestValidationEngine_SystemErrors(t *testing.T) {
	t.Run("Rule Returns Error Added As System Error", func(t *testing.T) {
		engine := framework.NewValidationEngine(framework.DefaultEngineConfig())

		rule := framework.NewConcreteRule("failing_rule").
			WithValidation(func(_ context.Context, me *errortypes.MultiError) error {
				return assert.AnError
			})

		engine.AddRule(rule)
		multiErr := engine.Validate(context.Background())

		require.NotNil(t, multiErr)
		assert.True(t, multiErr.HasErrors())
		assert.Equal(t, "system", multiErr.Errors[0].Field)
		assert.Equal(t, errortypes.ErrSystemError, multiErr.Errors[0].Code)
	})
}
