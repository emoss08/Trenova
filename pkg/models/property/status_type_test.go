package property_test

import (
	"testing"

	"github.com/emoss08/trenova/pkg/models/property"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatus_String(t *testing.T) {
	status := property.StatusActive
	assert.Equal(t, "Active", status.String())
}

func TestStatus_Values(t *testing.T) {
	values := property.Status("").Values()
	assert.Equal(t, []string{"Active", "Inactive"}, values)
}

func TestStatus_Value(t *testing.T) {
	status := property.StatusActive
	val, err := status.Value()
	require.NoError(t, err)
	assert.Equal(t, "Active", val)
}

func TestStatus_Scan(t *testing.T) {
	var status property.Status

	err := status.Scan("Active")
	require.NoError(t, err)
	assert.Equal(t, property.StatusActive, status)

	err = status.Scan(nil)
	require.Error(t, err)

	err = status.Scan(123)
	assert.Error(t, err)
}
