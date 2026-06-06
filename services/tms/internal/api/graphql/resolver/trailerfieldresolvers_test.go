package resolver

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/equipmentcontinuity"
	"github.com/emoss08/trenova/internal/core/domain/trailer"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTrailerResolver_EquipmentStatusAliasesStatus(t *testing.T) {
	t.Parallel()

	resolved, err := (&trailerResolver{}).EquipmentStatus(t.Context(), &trailer.Trailer{
		Status: domaintypes.EquipmentStatusAtMaintenance,
	})

	require.NoError(t, err)
	assert.Equal(t, domaintypes.EquipmentStatusAtMaintenance, resolved)
}

func TestTrailerResolver_NullableIDs(t *testing.T) {
	t.Parallel()

	resolver := &trailerResolver{}
	stateID := pulid.MustNew("us_")
	fleetCodeID := pulid.MustNew("fc_")
	locationID := pulid.MustNew("loc_")

	entity := &trailer.Trailer{
		RegistrationStateID: stateID,
		FleetCodeID:         fleetCodeID,
		LastKnownLocationID: locationID,
	}

	resolvedStateID, err := resolver.RegistrationStateID(t.Context(), entity)
	require.NoError(t, err)
	assert.Equal(t, stateID.String(), *resolvedStateID)

	resolvedFleetCodeID, err := resolver.FleetCodeID(t.Context(), entity)
	require.NoError(t, err)
	assert.Equal(t, fleetCodeID.String(), *resolvedFleetCodeID)

	resolvedLocationID, err := resolver.LastKnownLocationID(t.Context(), entity)
	require.NoError(t, err)
	assert.Equal(t, locationID.String(), *resolvedLocationID)

	resolvedStateID, err = resolver.RegistrationStateID(t.Context(), &trailer.Trailer{})
	require.NoError(t, err)
	assert.Nil(t, resolvedStateID)
}

func TestEquipmentContinuityResolver_NullableIDs(t *testing.T) {
	t.Parallel()

	resolver := &equipmentContinuityResolver{}
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
