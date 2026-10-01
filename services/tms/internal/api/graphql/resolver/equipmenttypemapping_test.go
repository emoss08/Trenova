package resolver

import (
	"testing"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/mappers"
	"github.com/emoss08/trenova/internal/core/domain/equipmenttype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newEquipmentTypeFixture() *equipmenttype.EquipmentType {
	interiorLength := 53.0
	return &equipmenttype.EquipmentType{
		Status:         domaintypes.StatusActive,
		Code:           "DRYVAN",
		Description:    "53ft dry van",
		Class:          equipmenttype.ClassTrailer,
		Color:          "#ff0000",
		InteriorLength: &interiorLength,
		Version:        4,
	}
}

func TestApplyEquipmentTypePatch_AbsentLeavesUnchanged(t *testing.T) {
	t.Parallel()

	entity := newEquipmentTypeFixture()
	require.NoError(t, mappers.ApplyEquipmentTypePatch(entity, gqlmodel.EquipmentTypePatchInput{}))
	assert.Equal(t, newEquipmentTypeFixture(), entity)
}

func TestApplyEquipmentTypePatch_NullClearsOptionalFields(t *testing.T) {
	t.Parallel()

	entity := newEquipmentTypeFixture()
	require.NoError(t, mappers.ApplyEquipmentTypePatch(entity, gqlmodel.EquipmentTypePatchInput{
		Description:    graphql.OmittableOf[*string](nil),
		Color:          graphql.OmittableOf[*string](nil),
		InteriorLength: graphql.OmittableOf[*float64](nil),
	}))
	assert.Empty(t, entity.Description)
	assert.Empty(t, entity.Color)
	assert.Nil(t, entity.InteriorLength)
	assert.Equal(t, "DRYVAN", entity.Code)
	assert.Equal(t, equipmenttype.ClassTrailer, entity.Class)
}

func TestApplyEquipmentTypePatch_NullRejectedOnRequiredFields(t *testing.T) {
	t.Parallel()

	cases := []struct {
		field string
		input gqlmodel.EquipmentTypePatchInput
	}{
		{
			field: "code",
			input: gqlmodel.EquipmentTypePatchInput{Code: graphql.OmittableOf[*string](nil)},
		},
		{
			field: "class",
			input: gqlmodel.EquipmentTypePatchInput{
				Class: graphql.OmittableOf[*equipmenttype.Class](nil),
			},
		},
		{
			field: "status",
			input: gqlmodel.EquipmentTypePatchInput{
				Status: graphql.OmittableOf[*domaintypes.Status](nil),
			},
		},
		{
			field: "version",
			input: gqlmodel.EquipmentTypePatchInput{Version: graphql.OmittableOf[*int](nil)},
		},
	}
	for _, tc := range cases {
		t.Run(tc.field, func(t *testing.T) {
			t.Parallel()

			entity := newEquipmentTypeFixture()
			err := mappers.ApplyEquipmentTypePatch(entity, tc.input)
			requireRequiredPatchError(t, err, tc.field)
			assert.Equal(t, newEquipmentTypeFixture(), entity)
		})
	}
}

func TestApplyEquipmentTypePatch_ValuesAreSet(t *testing.T) {
	t.Parallel()

	code := "REEFER"
	description := "Refrigerated"
	class := equipmenttype.ClassContainer
	color := "#00ff00"
	interiorLength := 48.5
	status := domaintypes.StatusInactive
	version := 9
	entity := newEquipmentTypeFixture()
	require.NoError(t, mappers.ApplyEquipmentTypePatch(entity, gqlmodel.EquipmentTypePatchInput{
		Code:           graphql.OmittableOf(&code),
		Description:    graphql.OmittableOf(&description),
		Class:          graphql.OmittableOf(&class),
		Color:          graphql.OmittableOf(&color),
		InteriorLength: graphql.OmittableOf(&interiorLength),
		Status:         graphql.OmittableOf(&status),
		Version:        graphql.OmittableOf(&version),
	}))
	assert.Equal(t, code, entity.Code)
	assert.Equal(t, description, entity.Description)
	assert.Equal(t, class, entity.Class)
	assert.Equal(t, color, entity.Color)
	require.NotNil(t, entity.InteriorLength)
	assert.InDelta(t, interiorLength, *entity.InteriorLength, 0)
	assert.Equal(t, status, entity.Status)
	assert.Equal(t, int64(version), entity.Version)
}
