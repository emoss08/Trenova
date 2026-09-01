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

	lookupSpecs := map[string]struct{}{
		"lookup": {}, "lookupOr": {}, "lookup2": {}, "lookup2Or": {},
	}

	executable := 0
	for _, spec := range engine.FunctionSpecs() {
		if _, isLookup := lookupSpecs[spec.Name]; !isLookup {
			executable++
		}
	}

	assert.Len(t, engine.BuiltinFunctions(), executable)
}
