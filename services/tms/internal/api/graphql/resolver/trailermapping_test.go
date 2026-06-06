package resolver

import (
	"testing"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/trailer"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyTrailerPatch_NullableIDs(t *testing.T) {
	t.Parallel()

	stateID := pulid.MustNew("us_")
	fleetCodeID := pulid.MustNew("fc_")
	entity := &trailer.Trailer{
		RegistrationStateID: stateID,
		FleetCodeID:         fleetCodeID,
	}

	err := applyTrailerPatch(entity, gqlmodel.TrailerPatchInput{})
	require.NoError(t, err)
	assert.Equal(t, stateID, entity.RegistrationStateID)
	assert.Equal(t, fleetCodeID, entity.FleetCodeID)

	err = applyTrailerPatch(entity, gqlmodel.TrailerPatchInput{
		RegistrationStateID: graphql.OmittableOf[*string](nil),
		FleetCodeID:         graphql.OmittableOf[*string](nil),
	})
	require.NoError(t, err)
	assert.True(t, entity.RegistrationStateID.IsNil())
	assert.True(t, entity.FleetCodeID.IsNil())

	nextStateID := pulid.MustNew("us_")
	nextFleetCodeID := pulid.MustNew("fc_")
	nextStateIDValue := nextStateID.String()
	nextFleetCodeIDValue := nextFleetCodeID.String()
	err = applyTrailerPatch(entity, gqlmodel.TrailerPatchInput{
		RegistrationStateID: graphql.OmittableOf(&nextStateIDValue),
		FleetCodeID:         graphql.OmittableOf(&nextFleetCodeIDValue),
	})
	require.NoError(t, err)
	assert.Equal(t, nextStateID, entity.RegistrationStateID)
	assert.Equal(t, nextFleetCodeID, entity.FleetCodeID)
}

func TestTrailerRelationIncludesForFields_UsesSelectedFields(t *testing.T) {
	t.Parallel()

	selected := map[string]bool{
		"businessUnit":          true,
		"equipmentType":         true,
		"fleetCode":             true,
		"fleetCode.manager":     true,
		"lastKnownLocation":     true,
		"customFields":          true,
		"registrationState":     true,
		"equipmentManufacturer": false,
	}
	includeEquipment := true
	includeFleet := true
	includes := trailerRelationIncludesForFields(
		func(path string) bool { return selected[path] },
		"",
		&includeEquipment,
		&includeFleet,
	)
	assert.True(t, includes.IncludeBusinessUnit)
	assert.False(t, includes.IncludeOrganization)
	assert.True(t, includes.IncludeRegistrationState)
	assert.True(t, includes.IncludeEquipmentType)
	assert.False(t, includes.IncludeEquipmentManufacturer)
	assert.True(t, includes.IncludeFleetCode)
	assert.True(t, includes.IncludeFleetManager)
	assert.True(t, includes.IncludeLastKnownLocation)
	assert.True(t, includes.IncludeCustomFields)
	assert.False(t, includes.IncludeTenantDetails)
	assert.False(t, includes.IncludeRegistrationDetails)
	assert.False(t, includes.IncludeEquipmentDetails)
	assert.False(t, includes.IncludeFleetDetails)
	assert.Equal(
		t,
		[]string{
			"id",
			"created_at",
			"business_unit_id",
			"equipment_type_id",
			"fleet_code_id",
			"registration_state_id",
		},
		includes.TrailerColumns,
	)
	assert.Equal(t, []string{"id", "created_at"}, includes.EquipmentTypeColumns)
	assert.Equal(t, []string{"id", "created_at", "manager_id"}, includes.FleetCodeColumns)
	assert.Nil(t, includes.EquipmentManufacturerColumns)
}

func TestTrailerRelationIncludesForFields_ProjectsTrailerListColumns(t *testing.T) {
	t.Parallel()

	selected := map[string]bool{
		"id":                                true,
		"code":                              true,
		"equipmentStatus":                   true,
		"model":                             true,
		"make":                              true,
		"licensePlateNumber":                true,
		"vin":                               true,
		"registrationNumber":                true,
		"maxLoadWeight":                     true,
		"lastInspectionDate":                true,
		"registrationExpiry":                true,
		"lastKnownLocationId":               true,
		"lastKnownLocation":                 true,
		"equipmentManufacturer":             true,
		"equipmentManufacturer.status":      true,
		"equipmentManufacturer.name":        true,
		"equipmentManufacturer.description": true,
		"equipmentManufacturer.version":     true,
		"equipmentManufacturer.createdAt":   true,
		"equipmentManufacturer.updatedAt":   true,
		"equipmentType":                     true,
		"equipmentType.id":                  true,
		"equipmentType.code":                true,
		"equipmentType.class":               true,
		"equipmentType.description":         true,
		"equipmentType.color":               true,
		"equipmentType.interiorLength":      true,
		"equipmentType.createdAt":           true,
		"equipmentType.updatedAt":           true,
		"fleetCode":                         true,
		"fleetCode.id":                      true,
		"fleetCode.code":                    true,
		"fleetCode.description":             true,
		"fleetCode.revenueGoal":             true,
		"fleetCode.deadheadGoal":            true,
		"fleetCode.mileageGoal":             true,
		"fleetCode.status":                  true,
		"fleetCode.color":                   true,
		"fleetCode.version":                 true,
		"fleetCode.createdAt":               true,
		"fleetCode.updatedAt":               true,
	}

	includes := trailerRelationIncludesForFields(
		func(path string) bool { return selected[path] },
		"",
		nil,
		nil,
	)

	assert.True(t, includes.IncludeEquipmentManufacturer)
	assert.True(t, includes.IncludeEquipmentType)
	assert.True(t, includes.IncludeFleetCode)
	assert.True(t, includes.IncludeLastKnownLocation)
	assert.False(t, includes.IncludeFleetManager)
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
			"vin",
			"registration_number",
			"max_load_weight",
			"last_inspection_date",
			"registration_expiry",
			"equipment_type_id",
			"equipment_manufacturer_id",
			"fleet_code_id",
		},
		includes.TrailerColumns,
	)
	assert.Equal(
		t,
		[]string{
			"id",
			"created_at",
			"status",
			"name",
			"description",
			"version",
			"updated_at",
		},
		includes.EquipmentManufacturerColumns,
	)
	assert.Equal(
		t,
		[]string{
			"id",
			"created_at",
			"code",
			"description",
			"class",
			"color",
			"interior_length",
			"updated_at",
		},
		includes.EquipmentTypeColumns,
	)
	assert.Equal(
		t,
		[]string{
			"id",
			"created_at",
			"status",
			"code",
			"description",
			"revenue_goal",
			"deadhead_goal",
			"mileage_goal",
			"color",
			"version",
			"updated_at",
		},
		includes.FleetCodeColumns,
	)
}

func TestTrailerRelationIncludesForFields_ListFlagsGateEquipmentAndFleetFields(t *testing.T) {
	t.Parallel()

	selected := map[string]bool{
		"equipmentManufacturer":        true,
		"equipmentManufacturer.status": true,
		"equipmentType":                true,
		"equipmentType.code":           true,
		"fleetCode":                    true,
		"fleetCode.manager":            true,
		"fleetCode.code":               true,
	}
	includeEquipment := false
	includeFleet := false

	includes := trailerRelationIncludesForFields(
		func(path string) bool { return selected[path] },
		"",
		&includeEquipment,
		&includeFleet,
	)

	assert.False(t, includes.IncludeEquipmentDetails)
	assert.False(t, includes.IncludeEquipmentType)
	assert.False(t, includes.IncludeEquipmentManufacturer)
	assert.False(t, includes.IncludeFleetDetails)
	assert.False(t, includes.IncludeFleetCode)
	assert.False(t, includes.IncludeFleetManager)
	assert.Nil(t, includes.EquipmentTypeColumns)
	assert.Nil(t, includes.EquipmentManufacturerColumns)
	assert.Nil(t, includes.FleetCodeColumns)
	assert.Equal(t, []string{"id", "created_at"}, includes.TrailerColumns)
}

func TestTrailerRelationIncludesForFields_ListFlagsDoNotForceUnselectedGroups(t *testing.T) {
	t.Parallel()

	includeEquipment := true
	includeFleet := true
	includes := trailerRelationIncludesForFields(
		func(string) bool { return false },
		"",
		&includeEquipment,
		&includeFleet,
	)

	assert.False(t, includes.IncludeEquipmentDetails)
	assert.False(t, includes.IncludeEquipmentType)
	assert.False(t, includes.IncludeEquipmentManufacturer)
	assert.False(t, includes.IncludeFleetDetails)
	assert.False(t, includes.IncludeFleetCode)
	assert.False(t, includes.IncludeFleetManager)
	assert.Nil(t, includes.EquipmentTypeColumns)
	assert.Nil(t, includes.EquipmentManufacturerColumns)
	assert.Nil(t, includes.FleetCodeColumns)
}

func TestTrailerFieldPath(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "customFields", trailerFieldPath("", "customFields"))
	assert.Equal(t, "edges.node.customFields", trailerFieldPath("edges.node", "customFields"))
}
