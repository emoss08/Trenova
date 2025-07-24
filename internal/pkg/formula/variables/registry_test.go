/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package variables_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/types/formula"
	"github.com/emoss08/trenova/internal/pkg/formula/variables"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistry_Register(t *testing.T) {
	registry := variables.NewRegistry()

	// * Create a test variable
	var1 := variables.NewVariable(
		"test_var",
		"Test variable",
		formula.ValueTypeNumber,
		"test",
		func(ctx variables.VariableContext) (any, error) {
			return 42.0, nil
		},
	)

	// * Register should succeed
	err := registry.Register(var1)
	require.NoError(t, err)

	// * Duplicate registration should fail
	err = registry.Register(var1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")

	// * Empty name should fail
	var2 := variables.NewVariable(
		"",
		"Invalid variable",
		formula.ValueTypeString,
		"test",
		nil,
	)
	err = registry.Register(var2)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "name cannot be empty")
}

func TestRegistry_Get(t *testing.T) {
	registry := variables.NewRegistry()

	// * Register a variable
	var1 := variables.NewVariable(
		"test_var",
		"Test variable",
		formula.ValueTypeNumber,
		"test",
		func(ctx variables.VariableContext) (any, error) {
			return 42.0, nil
		},
	)
	registry.MustRegister(var1)

	// * Get should return the variable
	retrieved, err := registry.Get("test_var")
	require.NoError(t, err)
	assert.Equal(t, "test_var", retrieved.Name())
	assert.Equal(t, "Test variable", retrieved.Description())
	assert.Equal(t, formula.ValueTypeNumber, retrieved.Type())

	// * Get non-existent should fail
	_, err = registry.Get("non_existent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRegistry_GetByCategory(t *testing.T) {
	registry := variables.NewRegistry()

	// * Register variables in different categories
	var1 := variables.NewVariable("var1", "Var 1", formula.ValueTypeNumber, "cat1", nil)
	var2 := variables.NewVariable("var2", "Var 2", formula.ValueTypeString, "cat1", nil)
	var3 := variables.NewVariable("var3", "Var 3", formula.ValueTypeBoolean, "cat2", nil)

	registry.MustRegister(var1)
	registry.MustRegister(var2)
	registry.MustRegister(var3)

	// * Get by category
	cat1Vars := registry.GetByCategory("cat1")
	assert.Len(t, cat1Vars, 2)

	cat2Vars := registry.GetByCategory("cat2")
	assert.Len(t, cat2Vars, 1)
	assert.Equal(t, "var3", cat2Vars[0].Name())

	// * Non-existent category returns empty
	emptyVars := registry.GetByCategory("non_existent")
	assert.Empty(t, emptyVars)
}

func TestRegistry_List(t *testing.T) {
	registry := variables.NewRegistry()

	// * Initially empty
	assert.Empty(t, registry.List())
	assert.Empty(t, registry.ListNames())

	// * Add variables
	var1 := variables.NewVariable("var1", "Var 1", formula.ValueTypeNumber, "cat1", nil)
	var2 := variables.NewVariable("var2", "Var 2", formula.ValueTypeString, "cat2", nil)

	registry.MustRegister(var1)
	registry.MustRegister(var2)

	// * List all
	allVars := registry.List()
	assert.Len(t, allVars, 2)

	// * List names
	names := registry.ListNames()
	assert.Len(t, names, 2)
	assert.Contains(t, names, "var1")
	assert.Contains(t, names, "var2")
}

func TestRegistry_Categories(t *testing.T) {
	registry := variables.NewRegistry()

	// * Register variables in different categories
	registry.MustRegister(variables.NewVariable("v1", "V1", formula.ValueTypeNumber, "cat1", nil))
	registry.MustRegister(variables.NewVariable("v2", "V2", formula.ValueTypeString, "cat2", nil))
	registry.MustRegister(variables.NewVariable("v3", "V3", formula.ValueTypeBoolean, "cat1", nil))

	categories := registry.Categories()
	assert.Len(t, categories, 2)
	assert.Contains(t, categories, "cat1")
	assert.Contains(t, categories, "cat2")
}

func TestRegistry_Clear(t *testing.T) {
	registry := variables.NewRegistry()

	// * Add variables
	registry.MustRegister(variables.NewVariable("v1", "V1", formula.ValueTypeNumber, "cat1", nil))
	registry.MustRegister(variables.NewVariable("v2", "V2", formula.ValueTypeString, "cat2", nil))

	assert.Len(t, registry.List(), 2)

	// * Clear
	registry.Clear()

	assert.Empty(t, registry.List())
	assert.Empty(t, registry.Categories())
}

func TestVariableDefinition_Resolve(t *testing.T) {
	// * Create a variable with resolver
	var1 := variables.NewVariable(
		"computed",
		"Computed value",
		formula.ValueTypeNumber,
		"test",
		func(ctx variables.VariableContext) (any, error) {
			// * Simulate computation
			val1, _ := ctx.GetField("Field1")
			val2, _ := ctx.GetField("Field2")

			num1, _ := val1.(float64)
			num2, _ := val2.(float64)

			return num1 + num2, nil
		},
	)

	// * Create mock context
	mockCtx := &mockVariableContext{
		fields: map[string]any{
			"Field1": 10.0,
			"Field2": 32.0,
		},
	}

	// * Resolve should compute
	result, err := var1.Resolve(mockCtx)
	require.NoError(t, err)
	assert.Equal(t, 42.0, result)
}

func TestVariableDefinition_Validate(t *testing.T) {
	// * Default validator for number
	var1 := variables.NewVariable(
		"num_var",
		"Number variable",
		formula.ValueTypeNumber,
		"test",
		nil,
	)

	// * Valid numbers
	assert.NoError(t, var1.Validate(42.0))
	assert.NoError(t, var1.Validate(42))
	assert.NoError(t, var1.Validate(int64(42)))
	assert.NoError(t, var1.Validate(nil))

	// * Invalid type
	assert.Error(t, var1.Validate("not a number"))

	// * Custom validator
	var2 := variables.NewVariableWithValidator(
		"positive_num",
		"Positive number only",
		formula.ValueTypeNumber,
		"test",
		nil,
		func(value any) error {
			if value == nil {
				return nil
			}

			num, ok := value.(float64)
			if !ok {
				return assert.AnError
			}

			if num < 0 {
				return assert.AnError
			}

			return nil
		},
	)

	assert.NoError(t, var2.Validate(42.0))
	assert.Error(t, var2.Validate(-42.0))
}

// * Mock context for testing
type mockVariableContext struct {
	entity   any
	fields   map[string]any
	computed map[string]any
	metadata map[string]any
}

func (m *mockVariableContext) GetEntity() any {
	return m.entity
}

func (m *mockVariableContext) GetField(path string) (any, error) {
	val, ok := m.fields[path]
	if !ok {
		return nil, assert.AnError
	}
	return val, nil
}

func (m *mockVariableContext) GetComputed(function string) (any, error) {
	val, ok := m.computed[function]
	if !ok {
		return nil, assert.AnError
	}
	return val, nil
}

func (m *mockVariableContext) GetMetadata() map[string]any {
	if m.metadata == nil {
		return make(map[string]any)
	}
	return m.metadata
}
