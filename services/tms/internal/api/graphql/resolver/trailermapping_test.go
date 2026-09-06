package resolver

import (
	"testing"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/mappers"
	"github.com/emoss08/trenova/internal/core/domain/trailer"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
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

	err := mappers.ApplyTrailerPatch(entity, gqlmodel.TrailerPatchInput{})
	require.NoError(t, err)
	assert.Equal(t, stateID, entity.RegistrationStateID)
	assert.Equal(t, fleetCodeID, entity.FleetCodeID)

	err = mappers.ApplyTrailerPatch(entity, gqlmodel.TrailerPatchInput{
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
	err = mappers.ApplyTrailerPatch(entity, gqlmodel.TrailerPatchInput{
		RegistrationStateID: graphql.OmittableOf(&nextStateIDValue),
		FleetCodeID:         graphql.OmittableOf(&nextFleetCodeIDValue),
	})
	require.NoError(t, err)
	assert.Equal(t, nextStateID, entity.RegistrationStateID)
	assert.Equal(t, nextFleetCodeID, entity.FleetCodeID)
}

func newTrailerPatchFixture() *trailer.Trailer {
	year := 2018
	maxLoadWeight := 45000
	lastInspection := int64(1_690_000_000)
	expiry := int64(1_700_000_000)
	return &trailer.Trailer{
		EquipmentTypeID:         pulid.MustNew("et_"),
		EquipmentManufacturerID: pulid.MustNew("em_"),
		Status:                  domaintypes.EquipmentStatusAvailable,
		Code:                    "TRL-100",
		Model:                   "Dry Van",
		Make:                    "Great Dane",
		Year:                    &year,
		LicensePlateNumber:      "TRL123",
		Vin:                     "1GRAA0625CB700001",
		ExternalID:              "ext-1",
		RegistrationNumber:      "REG-1",
		MaxLoadWeight:           &maxLoadWeight,
		LastInspectionDate:      &lastInspection,
		RegistrationExpiry:      &expiry,
		Version:                 3,
		CustomFields:            map[string]any{"cf_1": "a"},
	}
}

func TestApplyTrailerPatch_AbsentLeavesUnchanged(t *testing.T) {
	t.Parallel()

	entity := newTrailerPatchFixture()
	expected := *entity
	require.NoError(t, mappers.ApplyTrailerPatch(entity, gqlmodel.TrailerPatchInput{}))
	assert.Equal(t, expected, *entity)
}

func TestApplyTrailerPatch_NullClearsOptionalFields(t *testing.T) {
	t.Parallel()

	entity := newTrailerPatchFixture()
	require.NoError(t, mappers.ApplyTrailerPatch(entity, gqlmodel.TrailerPatchInput{
		Model:              graphql.OmittableOf[*string](nil),
		Make:               graphql.OmittableOf[*string](nil),
		Year:               graphql.OmittableOf[*int](nil),
		LicensePlateNumber: graphql.OmittableOf[*string](nil),
		Vin:                graphql.OmittableOf[*string](nil),
		ExternalID:         graphql.OmittableOf[*string](nil),
		RegistrationNumber: graphql.OmittableOf[*string](nil),
		MaxLoadWeight:      graphql.OmittableOf[*int](nil),
		LastInspectionDate: graphql.OmittableOf[*int](nil),
		RegistrationExpiry: graphql.OmittableOf[*int](nil),
		CustomFields:       graphql.OmittableOf[map[string]any](nil),
	}))
	assert.Empty(t, entity.Model)
	assert.Empty(t, entity.Make)
	assert.Nil(t, entity.Year)
	assert.Empty(t, entity.LicensePlateNumber)
	assert.Empty(t, entity.Vin)
	assert.Empty(t, entity.ExternalID)
	assert.Empty(t, entity.RegistrationNumber)
	assert.Nil(t, entity.MaxLoadWeight)
	assert.Nil(t, entity.LastInspectionDate)
	assert.Nil(t, entity.RegistrationExpiry)
	require.NotNil(t, entity.CustomFields)
	assert.Empty(t, entity.CustomFields)
	assert.Equal(t, "TRL-100", entity.Code)
	assert.Equal(t, domaintypes.EquipmentStatusAvailable, entity.Status)
}

func TestApplyTrailerPatch_NullRejectedOnRequiredFields(t *testing.T) {
	t.Parallel()

	cases := []struct {
		field string
		input gqlmodel.TrailerPatchInput
	}{
		{
			field: "code",
			input: gqlmodel.TrailerPatchInput{Code: graphql.OmittableOf[*string](nil)},
		},
		{
			field: "status",
			input: gqlmodel.TrailerPatchInput{
				Status: graphql.OmittableOf[*domaintypes.EquipmentStatus](nil),
			},
		},
		{
			field: "version",
			input: gqlmodel.TrailerPatchInput{Version: graphql.OmittableOf[*int](nil)},
		},
		{
			field: "equipmentTypeId",
			input: gqlmodel.TrailerPatchInput{
				EquipmentTypeID: graphql.OmittableOf[*string](nil),
			},
		},
		{
			field: "equipmentManufacturerId",
			input: gqlmodel.TrailerPatchInput{
				EquipmentManufacturerID: graphql.OmittableOf[*string](nil),
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.field, func(t *testing.T) {
			t.Parallel()

			entity := newTrailerPatchFixture()
			expected := *entity
			err := mappers.ApplyTrailerPatch(entity, tc.input)
			requireRequiredPatchError(t, err, tc.field)
			assert.Equal(t, expected, *entity)
		})
	}
}

func TestApplyTrailerPatch_ValuesAreSet(t *testing.T) {
	t.Parallel()

	equipmentTypeID := pulid.MustNew("et_")
	equipmentManufacturerID := pulid.MustNew("em_")
	equipmentTypeIDValue := equipmentTypeID.String()
	equipmentManufacturerIDValue := equipmentManufacturerID.String()
	status := domaintypes.EquipmentStatusOOS
	code := "TRL-200"
	model := "Reefer"
	makeName := "Utility"
	year := 2023
	plate := "TRL789"
	vin := "1UYVS2538PM123456"
	externalID := "ext-2"
	registrationNumber := "REG-2"
	maxLoadWeight := 44000
	lastInspection := 1_750_000_000
	registrationExpiry := 1_800_000_000
	version := 4
	entity := newTrailerPatchFixture()
	require.NoError(t, mappers.ApplyTrailerPatch(entity, gqlmodel.TrailerPatchInput{
		EquipmentTypeID:         graphql.OmittableOf(&equipmentTypeIDValue),
		EquipmentManufacturerID: graphql.OmittableOf(&equipmentManufacturerIDValue),
		Status:                  graphql.OmittableOf(&status),
		Code:                    graphql.OmittableOf(&code),
		Model:                   graphql.OmittableOf(&model),
		Make:                    graphql.OmittableOf(&makeName),
		Year:                    graphql.OmittableOf(&year),
		LicensePlateNumber:      graphql.OmittableOf(&plate),
		Vin:                     graphql.OmittableOf(&vin),
		ExternalID:              graphql.OmittableOf(&externalID),
		RegistrationNumber:      graphql.OmittableOf(&registrationNumber),
		MaxLoadWeight:           graphql.OmittableOf(&maxLoadWeight),
		LastInspectionDate:      graphql.OmittableOf(&lastInspection),
		RegistrationExpiry:      graphql.OmittableOf(&registrationExpiry),
		Version:                 graphql.OmittableOf(&version),
		CustomFields:            graphql.OmittableOf(map[string]any{"cf_2": "b"}),
	}))
	assert.Equal(t, equipmentTypeID, entity.EquipmentTypeID)
	assert.Equal(t, equipmentManufacturerID, entity.EquipmentManufacturerID)
	assert.Equal(t, status, entity.Status)
	assert.Equal(t, code, entity.Code)
	assert.Equal(t, model, entity.Model)
	assert.Equal(t, makeName, entity.Make)
	require.NotNil(t, entity.Year)
	assert.Equal(t, year, *entity.Year)
	assert.Equal(t, plate, entity.LicensePlateNumber)
	assert.Equal(t, vin, entity.Vin)
	assert.Equal(t, externalID, entity.ExternalID)
	assert.Equal(t, registrationNumber, entity.RegistrationNumber)
	require.NotNil(t, entity.MaxLoadWeight)
	assert.Equal(t, maxLoadWeight, *entity.MaxLoadWeight)
	require.NotNil(t, entity.LastInspectionDate)
	assert.Equal(t, int64(lastInspection), *entity.LastInspectionDate)
	require.NotNil(t, entity.RegistrationExpiry)
	assert.Equal(t, int64(registrationExpiry), *entity.RegistrationExpiry)
	assert.Equal(t, int64(version), entity.Version)
	assert.Equal(t, map[string]any{"cf_2": "b"}, entity.CustomFields)
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

func TestTrailerConnectionToModel_PropagatesTotalCount(t *testing.T) {
	t.Parallel()

	total := 41
	connection, err := trailerConnectionToModel(&pagination.CursorListResult[*trailer.Trailer]{
		Items: []*trailer.Trailer{
			{
				ID:        pulid.MustNew("trl_"),
				CreatedAt: 1710000000000,
				Code:      "TRL-100",
			},
		},
		TotalCount: &total,
	})

	require.NoError(t, err)
	require.NotNil(t, connection.TotalCount)
	assert.Equal(t, total, *connection.TotalCount)
	assert.Len(t, connection.Edges, 1)
}
