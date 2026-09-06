package resolver

import (
	"testing"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/mappers"
	"github.com/emoss08/trenova/internal/core/domain/tractor"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
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

	err := mappers.ApplyTractorPatch(entity, gqlmodel.TractorPatchInput{})
	require.NoError(t, err)
	assert.Equal(t, stateID, entity.StateID)
	assert.Equal(t, fleetCodeID, entity.FleetCodeID)
	assert.Equal(t, secondaryWorkerID, entity.SecondaryWorkerID)

	err = mappers.ApplyTractorPatch(entity, gqlmodel.TractorPatchInput{
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
	err = mappers.ApplyTractorPatch(entity, gqlmodel.TractorPatchInput{
		StateID:           graphql.OmittableOf(&nextStateIDValue),
		FleetCodeID:       graphql.OmittableOf(&nextFleetCodeIDValue),
		SecondaryWorkerID: graphql.OmittableOf(&nextSecondaryWorkerIDValue),
	})
	require.NoError(t, err)
	assert.Equal(t, nextStateID, entity.StateID)
	assert.Equal(t, nextFleetCodeID, entity.FleetCodeID)
	assert.Equal(t, nextSecondaryWorkerID, entity.SecondaryWorkerID)
}

func newTractorPatchFixture() *tractor.Tractor {
	year := 2019
	expiry := int64(1_700_000_000)
	return &tractor.Tractor{
		PrimaryWorkerID:         pulid.MustNew("wrk_"),
		EquipmentTypeID:         pulid.MustNew("et_"),
		EquipmentManufacturerID: pulid.MustNew("em_"),
		Status:                  domaintypes.EquipmentStatusAvailable,
		Code:                    "TRC-100",
		Model:                   "Cascadia",
		Make:                    "Freightliner",
		Year:                    &year,
		LicensePlateNumber:      "ABC123",
		RegistrationNumber:      "REG-1",
		RegistrationExpiry:      &expiry,
		Vin:                     "1FUJGLDR2CLBP8834",
		ExternalID:              "ext-1",
		Version:                 3,
		CustomFields:            map[string]any{"cf_1": "a"},
	}
}

func TestApplyTractorPatch_AbsentLeavesUnchanged(t *testing.T) {
	t.Parallel()

	entity := newTractorPatchFixture()
	expected := *entity
	require.NoError(t, mappers.ApplyTractorPatch(entity, gqlmodel.TractorPatchInput{}))
	assert.Equal(t, expected, *entity)
}

func TestApplyTractorPatch_NullClearsOptionalFields(t *testing.T) {
	t.Parallel()

	entity := newTractorPatchFixture()
	require.NoError(t, mappers.ApplyTractorPatch(entity, gqlmodel.TractorPatchInput{
		Model:              graphql.OmittableOf[*string](nil),
		Make:               graphql.OmittableOf[*string](nil),
		Year:               graphql.OmittableOf[*int](nil),
		LicensePlateNumber: graphql.OmittableOf[*string](nil),
		RegistrationNumber: graphql.OmittableOf[*string](nil),
		RegistrationExpiry: graphql.OmittableOf[*int](nil),
		Vin:                graphql.OmittableOf[*string](nil),
		ExternalID:         graphql.OmittableOf[*string](nil),
		CustomFields:       graphql.OmittableOf[map[string]any](nil),
	}))
	assert.Empty(t, entity.Model)
	assert.Empty(t, entity.Make)
	assert.Nil(t, entity.Year)
	assert.Empty(t, entity.LicensePlateNumber)
	assert.Empty(t, entity.RegistrationNumber)
	assert.Nil(t, entity.RegistrationExpiry)
	assert.Empty(t, entity.Vin)
	assert.Empty(t, entity.ExternalID)
	require.NotNil(t, entity.CustomFields)
	assert.Empty(t, entity.CustomFields)
	assert.Equal(t, "TRC-100", entity.Code)
	assert.Equal(t, domaintypes.EquipmentStatusAvailable, entity.Status)
}

func TestApplyTractorPatch_NullRejectedOnRequiredFields(t *testing.T) {
	t.Parallel()

	cases := []struct {
		field string
		input gqlmodel.TractorPatchInput
	}{
		{
			field: "code",
			input: gqlmodel.TractorPatchInput{Code: graphql.OmittableOf[*string](nil)},
		},
		{
			field: "status",
			input: gqlmodel.TractorPatchInput{
				Status: graphql.OmittableOf[*domaintypes.EquipmentStatus](nil),
			},
		},
		{
			field: "version",
			input: gqlmodel.TractorPatchInput{Version: graphql.OmittableOf[*int](nil)},
		},
		{
			field: "primaryWorkerId",
			input: gqlmodel.TractorPatchInput{
				PrimaryWorkerID: graphql.OmittableOf[*string](nil),
			},
		},
		{
			field: "equipmentTypeId",
			input: gqlmodel.TractorPatchInput{
				EquipmentTypeID: graphql.OmittableOf[*string](nil),
			},
		},
		{
			field: "equipmentManufacturerId",
			input: gqlmodel.TractorPatchInput{
				EquipmentManufacturerID: graphql.OmittableOf[*string](nil),
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.field, func(t *testing.T) {
			t.Parallel()

			entity := newTractorPatchFixture()
			expected := *entity
			err := mappers.ApplyTractorPatch(entity, tc.input)
			requireRequiredPatchError(t, err, tc.field)
			assert.Equal(t, expected, *entity)
		})
	}
}

func TestApplyTractorPatch_ValuesAreSet(t *testing.T) {
	t.Parallel()

	primaryWorkerID := pulid.MustNew("wrk_")
	equipmentTypeID := pulid.MustNew("et_")
	equipmentManufacturerID := pulid.MustNew("em_")
	primaryWorkerIDValue := primaryWorkerID.String()
	equipmentTypeIDValue := equipmentTypeID.String()
	equipmentManufacturerIDValue := equipmentManufacturerID.String()
	status := domaintypes.EquipmentStatusAtMaintenance
	code := "TRC-200"
	model := "T680"
	makeName := "Kenworth"
	year := 2024
	plate := "XYZ789"
	registrationNumber := "REG-2"
	registrationExpiry := 1_800_000_000
	vin := "1XKYDP9X1RJ123456"
	externalID := "ext-2"
	version := 4
	entity := newTractorPatchFixture()
	require.NoError(t, mappers.ApplyTractorPatch(entity, gqlmodel.TractorPatchInput{
		PrimaryWorkerID:         graphql.OmittableOf(&primaryWorkerIDValue),
		EquipmentTypeID:         graphql.OmittableOf(&equipmentTypeIDValue),
		EquipmentManufacturerID: graphql.OmittableOf(&equipmentManufacturerIDValue),
		Status:                  graphql.OmittableOf(&status),
		Code:                    graphql.OmittableOf(&code),
		Model:                   graphql.OmittableOf(&model),
		Make:                    graphql.OmittableOf(&makeName),
		Year:                    graphql.OmittableOf(&year),
		LicensePlateNumber:      graphql.OmittableOf(&plate),
		RegistrationNumber:      graphql.OmittableOf(&registrationNumber),
		RegistrationExpiry:      graphql.OmittableOf(&registrationExpiry),
		Vin:                     graphql.OmittableOf(&vin),
		ExternalID:              graphql.OmittableOf(&externalID),
		Version:                 graphql.OmittableOf(&version),
		CustomFields:            graphql.OmittableOf(map[string]any{"cf_2": "b"}),
	}))
	assert.Equal(t, primaryWorkerID, entity.PrimaryWorkerID)
	assert.Equal(t, equipmentTypeID, entity.EquipmentTypeID)
	assert.Equal(t, equipmentManufacturerID, entity.EquipmentManufacturerID)
	assert.Equal(t, status, entity.Status)
	assert.Equal(t, code, entity.Code)
	assert.Equal(t, model, entity.Model)
	assert.Equal(t, makeName, entity.Make)
	require.NotNil(t, entity.Year)
	assert.Equal(t, year, *entity.Year)
	assert.Equal(t, plate, entity.LicensePlateNumber)
	assert.Equal(t, registrationNumber, entity.RegistrationNumber)
	require.NotNil(t, entity.RegistrationExpiry)
	assert.Equal(t, int64(registrationExpiry), *entity.RegistrationExpiry)
	assert.Equal(t, vin, entity.Vin)
	assert.Equal(t, externalID, entity.ExternalID)
	assert.Equal(t, int64(version), entity.Version)
	assert.Equal(t, map[string]any{"cf_2": "b"}, entity.CustomFields)
}

func TestApplyTractorPatch_InvalidRequiredIDIsRejected(t *testing.T) {
	t.Parallel()

	invalid := "not-an-id"
	entity := newTractorPatchFixture()
	err := mappers.ApplyTractorPatch(entity, gqlmodel.TractorPatchInput{
		PrimaryWorkerID: graphql.OmittableOf(&invalid),
	})
	require.Error(t, err)
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
	assert.Equal(
		t,
		[]string{
			"id",
			"created_at",
			"business_unit_id",
			"equipment_type_id",
			"fleet_code_id",
			"state_id",
			"primary_worker_id",
			"secondary_worker_id",
		},
		includes.TractorColumns,
	)
	assert.Equal(t, []string{"id", "created_at"}, includes.EquipmentTypeColumns)
	assert.Equal(t, []string{"id", "created_at"}, includes.FleetCodeColumns)
	assert.Equal(
		t,
		[]string{"id", "created_at", "first_name", "manager_id"},
		includes.PrimaryWorkerColumns,
	)
	assert.Equal(t, []string{"id", "created_at", "state_id"}, includes.SecondaryWorkerColumns)
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
			"equipment_type_id",
			"equipment_manufacturer_id",
			"fleet_code_id",
			"primary_worker_id",
		},
		includes.TractorColumns,
	)
	assert.Equal(t, []string{"id", "created_at", "name"}, includes.EquipmentManufacturerColumns)
	assert.Equal(t, []string{"id", "created_at", "code"}, includes.EquipmentTypeColumns)
	assert.Equal(t, []string{"id", "created_at", "code"}, includes.FleetCodeColumns)
	assert.Equal(
		t,
		[]string{"id", "created_at", "first_name", "last_name"},
		includes.PrimaryWorkerColumns,
	)
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

func TestTractorConnectionToModel_PropagatesTotalCount(t *testing.T) {
	t.Parallel()

	total := 37
	connection, err := tractorConnectionToModel(&pagination.CursorListResult[*tractor.Tractor]{
		Items: []*tractor.Tractor{
			{
				ID:        pulid.MustNew("trac_"),
				CreatedAt: 1710000000000,
				Code:      "TRC-100",
			},
		},
		TotalCount: &total,
	})

	require.NoError(t, err)
	require.NotNil(t, connection.TotalCount)
	assert.Equal(t, total, *connection.TotalCount)
	assert.Len(t, connection.Edges, 1)
}
