package formula

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShipmentSchema_MatchesCurrentShipmentModel(t *testing.T) {
	t.Parallel()

	registry, err := newSchemaRegistry()
	require.NoError(t, err)

	definition, ok := registry.Get("shipment")
	require.True(t, ok)

	_, hasRatingMethod := definition.Properties["ratingMethod"]
	assert.False(t, hasRatingMethod)

	assert.NotContains(t, definition.Required, "ratingMethod")
	assert.Contains(t, definition.DataSource.Preloads, "Customer")
	assert.Contains(t, definition.DataSource.Preloads, "TractorType")
	assert.Contains(t, definition.DataSource.Preloads, "TrailerType")
	assert.Contains(t, definition.DataSource.Preloads, "Moves.Stops")
	assert.Contains(t, definition.DataSource.Preloads, "AdditionalCharges.AccessorialCharge")
	assert.Contains(t, definition.DataSource.Preloads, "Commodities.Commodity")
	assert.Contains(t, definition.DataSource.Preloads, "Commodities.Commodity.HazardousMaterial")

	_, hasRatingMethodField := definition.FieldSources["ratingMethod"]
	assert.False(t, hasRatingMethodField)

	_, hasHazmat := definition.Properties["hasHazmat"]
	assert.True(t, hasHazmat)

	_, hasHazmatField := definition.FieldSources["hasHazmat"]
	assert.True(t, hasHazmatField)

	_, hasOtherChargeAmount := definition.Properties["otherChargeAmount"]
	assert.True(t, hasOtherChargeAmount)

	_, hasTotalDistance := definition.Properties["totalDistance"]
	assert.True(t, hasTotalDistance)
}

func TestShipmentSchema_ExposesStopAndCommodityCollections(t *testing.T) {
	t.Parallel()

	registry, err := newSchemaRegistry()
	require.NoError(t, err)

	definition, ok := registry.Get("shipment")
	require.True(t, ok)

	for _, name := range []string{"stops", "commodities"} {
		prop, has := definition.Properties[name]
		require.True(t, has, "%s is a schema property", name)
		assert.Equal(t, "array", prop.Type, "%s is an array", name)
		assert.True(t, prop.Source.Computed)
		require.NotNil(t, prop.Items, "%s declares its element shape", name)
		assert.NotEmpty(t, prop.Items.Properties)
	}
	assert.NotEmpty(t, definition.Properties["stops"].Items.Properties["state"].Description)
	assert.NotEmpty(t, definition.Properties["commodities"].Items.Properties["freightClass"].Description)
}

func TestShipmentSchema_ExposesServiceAndShipmentTypesAndDimensionRollups(t *testing.T) {
	t.Parallel()

	registry, err := newSchemaRegistry()
	require.NoError(t, err)

	definition, ok := registry.Get("shipment")
	require.True(t, ok)

	assert.Contains(t, definition.DataSource.Preloads, "ServiceType")
	assert.Contains(t, definition.DataSource.Preloads, "ShipmentType")

	for _, name := range []string{"serviceType", "shipmentType"} {
		prop, has := definition.Properties[name]
		require.True(t, has, "%s is a schema property", name)
		assert.NotEmpty(t, prop.Source.Relation, "%s is loaded through its relation", name)
		assert.True(t, prop.Source.Nullable, "%s may be unset on a shipment", name)
		code, hasCode := prop.Properties["code"]
		require.True(t, hasCode, "%s exposes its code", name)
		assert.NotEmpty(t, code.Source.Path)
	}

	for _, name := range []string{"totalCubicFeet", "density", "primaryFreightClass", "highestFreightClass"} {
		prop, has := definition.Properties[name]
		require.True(t, has, "%s is a schema property", name)
		assert.True(t, prop.Source.Computed, "%s is computed", name)
		assert.NotEmpty(t, prop.Description)
	}
}
