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
