package engine_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/services/formula/engine"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFunctionSpecsCarryCompleteMetadata(t *testing.T) {
	t.Parallel()

	specs := engine.FunctionSpecs()
	require.NotEmpty(t, specs)

	seen := make(map[string]struct{}, len(specs))
	for _, spec := range specs {
		assert.NotEmpty(t, spec.Name, "spec name")
		assert.NotEmpty(t, spec.Signature, "signature for %s", spec.Name)
		assert.NotEmpty(t, spec.Description, "description for %s", spec.Name)
		assert.NotEmpty(t, spec.Example, "example for %s", spec.Name)
		assert.NotEmpty(t, spec.Category, "category for %s", spec.Name)

		_, duplicate := seen[spec.Name]
		assert.False(t, duplicate, "duplicate spec for %s", spec.Name)
		seen[spec.Name] = struct{}{}
	}
}

func TestFunctionSpecsIncludeLookupDocumentation(t *testing.T) {
	t.Parallel()

	specs := engine.FunctionSpecs()
	names := make(map[string]string, len(specs))
	for _, spec := range specs {
		names[spec.Name] = spec.Category
	}

	assert.Equal(t, engine.FunctionCategoryRateTable, names["lookup"])
	assert.Equal(t, engine.FunctionCategoryRateTable, names["lookupOr"])
	assert.Equal(t, engine.FunctionCategoryRateTable, names["lookup2"])
	assert.Equal(t, engine.FunctionCategoryRateTable, names["lookup2Or"])
}

func TestBuiltinFunctionsMatchExecutableSpecs(t *testing.T) {
	t.Parallel()

	executable := 0
	for _, spec := range engine.FunctionSpecs() {
		if spec.Executable() {
			executable++
		}
	}

	assert.Len(t, engine.BuiltinFunctions(), executable)
}

func TestFunctionSpecsPublishStringBuiltins(t *testing.T) {
	t.Parallel()

	byName := make(map[string]engine.FunctionSpec)
	for _, spec := range engine.FunctionSpecs() {
		byName[spec.Name] = spec
	}

	for _, operator := range []string{"startsWith", "endsWith", "contains", "matches"} {
		spec, ok := byName[operator]
		require.True(t, ok, "%s must be published", operator)
		assert.True(t, spec.Operator, "%s is an infix operator", operator)
		assert.False(t, spec.Executable(), "%s ships with expr, not Trenova", operator)
		assert.Equal(t, engine.FunctionCategoryString, spec.Category)
	}

	for _, fn := range []string{"upper", "lower", "trim", "len", "indexOf", "replace"} {
		spec, ok := byName[fn]
		require.True(t, ok, "%s must be published", fn)
		assert.False(t, spec.Operator, "%s is called like a function", fn)
		assert.False(t, spec.Executable())
		assert.Equal(t, engine.FunctionCategoryString, spec.Category)
	}

	slice, ok := byName["[start:end]"]
	require.True(t, ok, "slicing must be published")
	assert.True(t, slice.Operator)

	assert.Equal(t, "min(...values)", byName["min"].Signature)
	assert.Equal(t, "max(...values)", byName["max"].Signature)
}
