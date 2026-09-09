package domaintypes_test

import (
	"testing"

	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIFTAFuelType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		fuelType      domaintypes.IFTAFuelType
		valid         bool
		countsForIFTA bool
		gaseous       bool
	}{
		{domaintypes.IFTAFuelTypeDiesel, true, true, false},
		{domaintypes.IFTAFuelTypeGasoline, true, true, false},
		{domaintypes.IFTAFuelTypeGasohol, true, true, false},
		{domaintypes.IFTAFuelTypePropane, true, true, true},
		{domaintypes.IFTAFuelTypeCNG, true, true, true},
		{domaintypes.IFTAFuelTypeLNG, true, true, true},
		{domaintypes.IFTAFuelTypeEthanol, true, true, false},
		{domaintypes.IFTAFuelTypeMethanol, true, true, false},
		{domaintypes.IFTAFuelTypeE85, true, true, false},
		{domaintypes.IFTAFuelTypeM85, true, true, false},
		{domaintypes.IFTAFuelTypeA55, true, true, false},
		{domaintypes.IFTAFuelTypeBiodiesel, true, true, false},
		{domaintypes.IFTAFuelTypeElectricity, true, true, false},
		{domaintypes.IFTAFuelTypeHydrogen, true, true, true},
		{domaintypes.IFTAFuelTypeDEF, true, false, false},
		{domaintypes.IFTAFuelTypeReefer, true, false, false},
		{domaintypes.IFTAFuelTypeOther, true, false, false},
		{domaintypes.IFTAFuelType(""), false, false, false},
		{domaintypes.IFTAFuelType("diesel"), false, false, false},
		{domaintypes.IFTAFuelType("Kerosene"), false, false, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.fuelType), func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.valid, tt.fuelType.IsValid())
			assert.Equal(t, tt.countsForIFTA, tt.fuelType.CountsForIFTA())
			assert.Equal(t, tt.gaseous, tt.fuelType.IsGaseous())
			assert.Equal(t, string(tt.fuelType), tt.fuelType.String())
			if tt.valid {
				assert.NotEmpty(t, tt.fuelType.Label())
			}
		})
	}
}

func TestIFTAFuelTypes_ListsMembersInOrder(t *testing.T) {
	t.Parallel()

	members := domaintypes.IFTAFuelTypes()
	require.Len(t, members, 14)

	assert.Equal(t, domaintypes.IFTAFuelTypeDiesel, members[0])
	assert.Equal(t, domaintypes.IFTAFuelTypeHydrogen, members[13])

	seen := make(map[domaintypes.IFTAFuelType]struct{}, len(members))
	for _, member := range members {
		assert.True(t, member.CountsForIFTA(), "%s should count for IFTA", member)
		_, dup := seen[member]
		assert.False(t, dup, "%s listed twice", member)
		seen[member] = struct{}{}
	}

	members[0] = domaintypes.IFTAFuelTypeOther
	assert.Equal(t, domaintypes.IFTAFuelTypeDiesel, domaintypes.IFTAFuelTypes()[0],
		"callers must receive a copy")
}

func TestIFTAFuelType_LabelFallsBackToValue(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "Kerosene", domaintypes.IFTAFuelType("Kerosene").Label())
	assert.Equal(t, "E-85", domaintypes.IFTAFuelTypeE85.Label())
}

func TestAllIFTAFuelTypes_ListsEveryFuelInReturnOrder(t *testing.T) {
	t.Parallel()

	fuelTypes := domaintypes.AllIFTAFuelTypes()
	require.Len(t, fuelTypes, 17)

	assert.Equal(t, domaintypes.IFTAFuelTypes(), fuelTypes[:14])
	assert.Equal(
		t,
		[]domaintypes.IFTAFuelType{
			domaintypes.IFTAFuelTypeDEF,
			domaintypes.IFTAFuelTypeReefer,
			domaintypes.IFTAFuelTypeOther,
		},
		fuelTypes[14:],
	)

	seen := make(map[domaintypes.IFTAFuelType]struct{}, len(fuelTypes))
	for _, fuelType := range fuelTypes {
		assert.True(t, fuelType.IsValid(), "%s should be valid", fuelType)
		_, dup := seen[fuelType]
		assert.False(t, dup, "%s listed twice", fuelType)
		seen[fuelType] = struct{}{}
	}

	fuelTypes[0] = domaintypes.IFTAFuelTypeOther
	assert.Equal(t, domaintypes.IFTAFuelTypeDiesel, domaintypes.AllIFTAFuelTypes()[0],
		"callers must receive a copy")
}
