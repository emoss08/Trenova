package formula_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDescribeSchema(t *testing.T) {
	t.Parallel()

	svc := setupService(t)

	description, err := svc.DescribeSchema("shipment")
	require.NoError(t, err)
	assert.Equal(t, "shipment", description.SchemaID)

	variablesByName := make(map[string]bool, len(description.Variables))
	computedByName := make(map[string]bool, len(description.Variables))
	for _, variable := range description.Variables {
		variablesByName[variable.Name] = true
		computedByName[variable.Name] = variable.Computed
		assert.NotEmpty(t, variable.Type, "type for %s", variable.Name)
		assert.NotEmpty(t, variable.Category, "category for %s", variable.Name)
	}

	assert.True(t, variablesByName["weight"], "field variables are described")
	assert.False(t, computedByName["weight"], "weight is a stored field")
	assert.True(t, variablesByName["totalDistance"], "computed variables are described")
	assert.True(t, computedByName["totalDistance"], "totalDistance is computed")

	functionNames := make(map[string]bool, len(description.Functions))
	for _, fn := range description.Functions {
		functionNames[fn.Name] = true
		assert.NotEmpty(t, fn.Signature, "signature for %s", fn.Name)
		assert.NotEmpty(t, fn.Description, "description for %s", fn.Name)
	}

	assert.True(t, functionNames["round"])
	assert.True(t, functionNames["lookup"])
}

func TestDescribeSchema_UnknownSchema(t *testing.T) {
	t.Parallel()

	svc := setupService(t)

	_, err := svc.DescribeSchema("nope")
	require.Error(t, err)
}
