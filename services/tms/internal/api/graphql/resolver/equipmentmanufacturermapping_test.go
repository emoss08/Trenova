package resolver

import (
	"testing"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/mappers"
	"github.com/emoss08/trenova/internal/core/domain/equipmentmanufacturer"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newEquipmentManufacturerFixture() *equipmentmanufacturer.EquipmentManufacturer {
	return &equipmentmanufacturer.EquipmentManufacturer{
		Status:      domaintypes.StatusActive,
		Name:        "Freightliner",
		Description: "Heavy trucks",
		Version:     2,
	}
}

func TestApplyEquipmentManufacturerPatch_AbsentLeavesUnchanged(t *testing.T) {
	t.Parallel()

	entity := newEquipmentManufacturerFixture()
	require.NoError(
		t,
		mappers.ApplyEquipmentManufacturerPatch(
			entity,
			gqlmodel.EquipmentManufacturerPatchInput{},
		),
	)
	assert.Equal(t, newEquipmentManufacturerFixture(), entity)
}

func TestApplyEquipmentManufacturerPatch_NullClearsDescription(t *testing.T) {
	t.Parallel()

	entity := newEquipmentManufacturerFixture()
	require.NoError(
		t,
		mappers.ApplyEquipmentManufacturerPatch(
			entity,
			gqlmodel.EquipmentManufacturerPatchInput{
				Description: graphql.OmittableOf[*string](nil),
			},
		),
	)
	assert.Empty(t, entity.Description)
	assert.Equal(t, "Freightliner", entity.Name)
}

func TestApplyEquipmentManufacturerPatch_NullRejectedOnRequiredFields(t *testing.T) {
	t.Parallel()

	cases := []struct {
		field string
		input gqlmodel.EquipmentManufacturerPatchInput
	}{
		{
			field: "name",
			input: gqlmodel.EquipmentManufacturerPatchInput{
				Name: graphql.OmittableOf[*string](nil),
			},
		},
		{
			field: "status",
			input: gqlmodel.EquipmentManufacturerPatchInput{
				Status: graphql.OmittableOf[*domaintypes.Status](nil),
			},
		},
		{
			field: "version",
			input: gqlmodel.EquipmentManufacturerPatchInput{
				Version: graphql.OmittableOf[*int](nil),
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.field, func(t *testing.T) {
			t.Parallel()

			entity := newEquipmentManufacturerFixture()
			err := mappers.ApplyEquipmentManufacturerPatch(entity, tc.input)
			requireRequiredPatchError(t, err, tc.field)
			assert.Equal(t, newEquipmentManufacturerFixture(), entity)
		})
	}
}

func TestApplyEquipmentManufacturerPatch_ValuesAreSet(t *testing.T) {
	t.Parallel()

	name := "Kenworth"
	description := "Class 8"
	status := domaintypes.StatusInactive
	version := 7
	entity := newEquipmentManufacturerFixture()
	require.NoError(
		t,
		mappers.ApplyEquipmentManufacturerPatch(
			entity,
			gqlmodel.EquipmentManufacturerPatchInput{
				Name:        graphql.OmittableOf(&name),
				Description: graphql.OmittableOf(&description),
				Status:      graphql.OmittableOf(&status),
				Version:     graphql.OmittableOf(&version),
			},
		),
	)
	assert.Equal(t, name, entity.Name)
	assert.Equal(t, description, entity.Description)
	assert.Equal(t, status, entity.Status)
	assert.Equal(t, int64(version), entity.Version)
}
