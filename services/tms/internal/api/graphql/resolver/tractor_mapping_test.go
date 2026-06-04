package resolver

import (
	"testing"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/tractor"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyTractorPatch_NullableIDs(t *testing.T) {
	t.Parallel()

	stateID := pulid.MustNew("us_")
	fleetCodeID := pulid.MustNew("fc_")
	secondaryWorkerID := pulid.MustNew("wrk_")
	entity := &tractor.Tractor{
		StateID:           stateID,
		FleetCodeID:       fleetCodeID,
		SecondaryWorkerID: secondaryWorkerID,
	}

	err := applyTractorPatch(entity, gqlmodel.TractorPatchInput{})
	require.NoError(t, err)
	assert.Equal(t, stateID, entity.StateID)
	assert.Equal(t, fleetCodeID, entity.FleetCodeID)
	assert.Equal(t, secondaryWorkerID, entity.SecondaryWorkerID)

	err = applyTractorPatch(entity, gqlmodel.TractorPatchInput{
		StateID:           graphql.OmittableOf[*string](nil),
		FleetCodeID:       graphql.OmittableOf[*string](nil),
		SecondaryWorkerID: graphql.OmittableOf[*string](nil),
	})
	require.NoError(t, err)
	assert.True(t, entity.StateID.IsNil())
	assert.True(t, entity.FleetCodeID.IsNil())
	assert.True(t, entity.SecondaryWorkerID.IsNil())

	nextStateID := pulid.MustNew("us_")
	nextFleetCodeID := pulid.MustNew("fc_")
	nextSecondaryWorkerID := pulid.MustNew("wrk_")
	nextStateIDValue := nextStateID.String()
	nextFleetCodeIDValue := nextFleetCodeID.String()
	nextSecondaryWorkerIDValue := nextSecondaryWorkerID.String()
	err = applyTractorPatch(entity, gqlmodel.TractorPatchInput{
		StateID:           graphql.OmittableOf(&nextStateIDValue),
		FleetCodeID:       graphql.OmittableOf(&nextFleetCodeIDValue),
		SecondaryWorkerID: graphql.OmittableOf(&nextSecondaryWorkerIDValue),
	})
	require.NoError(t, err)
	assert.Equal(t, nextStateID, entity.StateID)
	assert.Equal(t, nextFleetCodeID, entity.FleetCodeID)
	assert.Equal(t, nextSecondaryWorkerID, entity.SecondaryWorkerID)
}

func TestTractorRelationIncludesForFields_UsesSelectedFields(t *testing.T) {
	t.Parallel()

	selected := map[string]bool{
		"businessUnit":            true,
		"equipmentType":           true,
		"fleetCode":               true,
		"lastKnownLocation":       true,
		"customFields":            true,
		"state":                   true,
		"primaryWorker":           true,
		"primaryWorker.firstName": true,
		"primaryWorker.manager":   true,
		"secondaryWorker":         true,
		"secondaryWorker.state":   true,
		"equipmentManufacturer":   false,
	}
	includeEquipment := true
	includeFleet := true
	includeWorker := true
	includes := tractorRelationIncludesForFields(
		func(path string) bool { return selected[path] },
		"",
		&includeEquipment,
		&includeFleet,
		&includeWorker,
	)

	assert.True(t, includes.IncludeBusinessUnit)
	assert.False(t, includes.IncludeOrganization)
	assert.True(t, includes.IncludeState)
	assert.True(t, includes.IncludeEquipmentType)
	assert.False(t, includes.IncludeEquipmentManufacturer)
	assert.True(t, includes.IncludeFleetCode)
	assert.True(t, includes.IncludePrimaryWorker)
	assert.True(t, includes.IncludePrimaryWorkerManager)
	assert.True(t, includes.IncludeSecondaryWorker)
	assert.True(t, includes.IncludeSecondaryWorkerState)
	assert.True(t, includes.IncludeLastKnownLocation)
	assert.True(t, includes.IncludeCustomFields)
	assert.Equal(t, []string{"id", "created_at"}, includes.TractorColumns)
	assert.Equal(t, []string{"id"}, includes.EquipmentTypeColumns)
	assert.Equal(t, []string{"id"}, includes.FleetCodeColumns)
	assert.Equal(t, []string{"id", "first_name", "manager_id"}, includes.PrimaryWorkerColumns)
	assert.Equal(t, []string{"id"}, includes.SecondaryWorkerColumns)
}

func TestTractorRelationIncludesForFields_ListFlagsGateDetails(t *testing.T) {
	t.Parallel()

	selected := map[string]bool{
		"equipmentManufacturer":        true,
		"equipmentManufacturer.status": true,
		"equipmentType":                true,
		"equipmentType.code":           true,
		"fleetCode":                    true,
		"fleetCode.code":               true,
		"primaryWorker":                true,
		"primaryWorker.firstName":      true,
	}
	includeEquipment := false
	includeFleet := false
	includeWorker := false

	includes := tractorRelationIncludesForFields(
		func(path string) bool { return selected[path] },
		"",
		&includeEquipment,
		&includeFleet,
		&includeWorker,
	)

	assert.False(t, includes.IncludeEquipmentType)
	assert.False(t, includes.IncludeEquipmentManufacturer)
	assert.False(t, includes.IncludeFleetCode)
	assert.False(t, includes.IncludePrimaryWorker)
	assert.Nil(t, includes.EquipmentTypeColumns)
	assert.Nil(t, includes.EquipmentManufacturerColumns)
	assert.Nil(t, includes.FleetCodeColumns)
	assert.Nil(t, includes.PrimaryWorkerColumns)
	assert.Equal(t, []string{"id", "created_at"}, includes.TractorColumns)
}

func TestTractorRelationIncludesForFields_ProjectsTractorListColumns(t *testing.T) {
	t.Parallel()

	selected := map[string]bool{
		"id":                         true,
		"code":                       true,
		"equipmentStatus":            true,
		"model":                      true,
		"make":                       true,
		"licensePlateNumber":         true,
		"registrationNumber":         true,
		"registrationExpiry":         true,
		"vin":                        true,
		"lastKnownLocationId":        true,
		"equipmentManufacturer":      true,
		"equipmentManufacturer.id":   true,
		"equipmentManufacturer.name": true,
		"equipmentType":              true,
		"equipmentType.id":           true,
		"equipmentType.code":         true,
		"fleetCode":                  true,
		"fleetCode.id":               true,
		"fleetCode.code":             true,
		"primaryWorker":              true,
		"primaryWorker.id":           true,
		"primaryWorker.firstName":    true,
		"primaryWorker.lastName":     true,
	}

	includes := tractorRelationIncludesForFields(
		func(path string) bool { return selected[path] },
		"",
		nil,
		nil,
		nil,
	)

	assert.True(t, includes.IncludeEquipmentManufacturer)
	assert.True(t, includes.IncludeEquipmentType)
	assert.True(t, includes.IncludeFleetCode)
	assert.True(t, includes.IncludePrimaryWorker)
	assert.True(t, includes.IncludeLastKnownLocation)
	assert.Equal(
		t,
		[]string{
			"id",
			"created_at",
			"status",
			"code",
			"model",
			"make",
			"license_plate_number",
			"registration_number",
			"registration_expiry",
			"vin",
		},
		includes.TractorColumns,
	)
	assert.Equal(t, []string{"id", "name"}, includes.EquipmentManufacturerColumns)
	assert.Equal(t, []string{"id", "code"}, includes.EquipmentTypeColumns)
	assert.Equal(t, []string{"id", "code"}, includes.FleetCodeColumns)
	assert.Equal(t, []string{"id", "first_name", "last_name"}, includes.PrimaryWorkerColumns)
}

func TestTractorLastKnownLocationReference(t *testing.T) {
	t.Parallel()

	locationID := pulid.MustNew("loc_")
	ref := tractorLastKnownLocationReference(&tractor.Tractor{
		LastKnownLocationID:   locationID,
		LastKnownLocationName: "North Yard",
	})

	require.NotNil(t, ref)
	assert.Equal(t, locationID.String(), ref.ID)
	assert.Equal(t, "North Yard", ref.Name)
	assert.Nil(t, tractorLastKnownLocationReference(&tractor.Tractor{}))
}
