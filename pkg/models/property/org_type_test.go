package property_test

import (
	"testing"

	"github.com/emoss08/trenova/pkg/models/property"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrgType_String(t *testing.T) {
	orgType := property.OrganizationTypeAsset
	assert.Equal(t, "Asset", orgType.String())
}

func TestOrgType_Values(t *testing.T) {
	values := property.OrganizationType("").Values()
	assert.Equal(t, []string{"Asset", "Brokerage", "Both"}, values)
}

func TestOrgType_Value(t *testing.T) {
	orgType := property.OrganizationTypeAsset
	val, err := orgType.Value()
	require.NoError(t, err)
	assert.Equal(t, "Asset", val)
}

func TestOrgType_Scan(t *testing.T) {
	var orgType property.OrganizationType

	err := orgType.Scan("Asset")
	require.NoError(t, err)
	assert.Equal(t, property.OrganizationTypeAsset, orgType)

	err = orgType.Scan(nil)
	require.Error(t, err)

	err = orgType.Scan(123)
	assert.Error(t, err)
}
