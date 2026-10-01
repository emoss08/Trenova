package equipmentcontinuityresolver

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/equipmentcontinuity"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEquipmentContinuityResolver_NullableIDs(t *testing.T) {
	t.Parallel()

	resolver := &EquipmentContinuityResolver{}
	shipmentID := pulid.MustNew("shp_")
	moveID := pulid.MustNew("sm_")
	locationID := pulid.MustNew("loc_")

	entity := &equipmentcontinuity.EquipmentContinuity{
		SourceShipmentID:     shipmentID,
		SourceShipmentMoveID: moveID,
		CurrentLocationID:    locationID,
	}

	resolvedShipmentID, err := resolver.SourceShipmentID(t.Context(), entity)
	require.NoError(t, err)
	assert.Equal(t, shipmentID.String(), *resolvedShipmentID)

	resolvedMoveID, err := resolver.SourceShipmentMoveID(t.Context(), entity)
	require.NoError(t, err)
	assert.Equal(t, moveID.String(), *resolvedMoveID)

	resolvedLocationID, err := resolver.CurrentLocationID(t.Context(), entity)
	require.NoError(t, err)
	assert.Equal(t, locationID.String(), *resolvedLocationID)

	resolvedShipmentID, err = resolver.SourceShipmentID(
		t.Context(),
		&equipmentcontinuity.EquipmentContinuity{},
	)
	require.NoError(t, err)
	assert.Nil(t, resolvedShipmentID)
}
