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

	types := make(map[string]string, len(description.Variables))
	for _, variable := range description.Variables {
		types[variable.Name] = variable.Type
	}
	assert.Equal(t, "array", types["stops"])
	assert.Equal(t, "string", types["stops[].type"], "a list's item fields are published")
	assert.Equal(t, "integer", types["stops[].sequence"])
	assert.Equal(t, "string", types["commodities[].freightClass"])
	assert.True(t, computedByName["stops[].city"], "an item field of a computed list is computed")
}

func TestDescribeSchema_UnknownSchema(t *testing.T) {
	t.Parallel()

	svc := setupService(t)

	_, err := svc.DescribeSchema("nope")
	require.Error(t, err)
}

func TestDescribeSchema_OptionalRelationFieldsAreNullable(t *testing.T) {
	t.Parallel()

	svc := setupService(t)

	description, err := svc.DescribeSchema("shipment")
	require.NoError(t, err)

	nullable := make(map[string]bool, len(description.Variables))
	for _, variable := range description.Variables {
		nullable[variable.Name] = variable.Nullable
	}

	assert.True(t, nullable["tractorType.code"])
	assert.True(t, nullable["trailerType.description"])
	assert.True(t, nullable["shipmentType.code"])
	assert.False(t, nullable["serviceType.code"], "every shipment has a service type")
	assert.False(t, nullable["customer.code"], "every shipment has a customer")
}
