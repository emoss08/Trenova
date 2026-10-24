package tractor_test

import (
	"reflect"
	"testing"

	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/stretchr/testify/require"
)


func TestTractorNormalizesFuelTypeOnWrite(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		given domaintypes.IFTAFuelType
		want  domaintypes.IFTAFuelType
	}{
		{
			"keeps a chosen fuel type",
			domaintypes.IFTAFuelTypeGasoline,
			domaintypes.IFTAFuelTypeGasoline,
		},
		{"falls back to diesel when unset", "", domaintypes.IFTAFuelTypeDiesel},
	}

	for _, tt := range tests {
		t.Run(tt.name+" on insert", func(t *testing.T) {
			t.Parallel()

			entity := &tractor.Tractor{FuelType: tt.given}
			require.NoError(
				t,
				entity.BeforeCreate(t.Context()),
			)
			require.Equal(t, tt.want, entity.FuelType)
		})

		t.Run(tt.name+" on update", func(t *testing.T) {
			t.Parallel()

			entity := &tractor.Tractor{FuelType: tt.given}
			require.NoError(
				t,
				entity.BeforeUpdate(t.Context()),
			)
			require.Equal(t, tt.want, entity.FuelType)
		})
	}
}

func TestTractorFuelTypeColumnStillMapsToTheEnum(t *testing.T) {
	t.Parallel()

	field, ok := reflect.TypeFor[tractor.Tractor]().FieldByName("FuelType")
	require.True(t, ok)

	tag := field.Tag.Get("keel")
	require.Equal(t, "fuel_type", tag, "unexpected column mapping: %s", tag)
}
