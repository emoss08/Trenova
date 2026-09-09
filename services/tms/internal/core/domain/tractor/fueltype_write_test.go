package tractor_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/tractor"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

// The fuel type decides which lines of the quarterly IFTA return a tractor's
// miles land on, so it has to be written from the entity every time. A bun
// `default:` tag would quietly send DEFAULT for an unset value instead, which
// is why the column default lives only in the migration.
func TestTractorFuelTypeCarriesNoBunDefault(t *testing.T) {
	t.Parallel()

	field, ok := reflect.TypeFor[tractor.Tractor]().FieldByName("FuelType")
	require.True(t, ok, "Tractor must declare FuelType")

	require.NotContains(
		t,
		field.Tag.Get("bun"),
		"default:",
		"fuel_type must not carry a bun default; keep the default in the migration",
	)
}

func TestTractorNormalizesFuelTypeOnWrite(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		given domaintypes.IFTAFuelType
		want  domaintypes.IFTAFuelType
	}{
		{"keeps a chosen fuel type", domaintypes.IFTAFuelTypeGasoline, domaintypes.IFTAFuelTypeGasoline},
		{"falls back to diesel when unset", "", domaintypes.IFTAFuelTypeDiesel},
	}

	for _, tt := range tests {
		t.Run(tt.name+" on insert", func(t *testing.T) {
			t.Parallel()

			entity := &tractor.Tractor{FuelType: tt.given}
			require.NoError(
				t,
				entity.BeforeAppendModel(t.Context(), (*bun.InsertQuery)(nil)),
			)
			require.Equal(t, tt.want, entity.FuelType)
		})

		t.Run(tt.name+" on update", func(t *testing.T) {
			t.Parallel()

			entity := &tractor.Tractor{FuelType: tt.given}
			require.NoError(
				t,
				entity.BeforeAppendModel(t.Context(), (*bun.UpdateQuery)(nil)),
			)
			require.Equal(t, tt.want, entity.FuelType)
		})
	}
}

func TestTractorFuelTypeColumnStillMapsToTheEnum(t *testing.T) {
	t.Parallel()

	field, ok := reflect.TypeFor[tractor.Tractor]().FieldByName("FuelType")
	require.True(t, ok)

	tag := field.Tag.Get("bun")
	require.True(t, strings.HasPrefix(tag, "fuel_type,"), "unexpected column mapping: %s", tag)
	require.Contains(t, tag, "type:ifta_fuel_type_enum")
	require.Contains(t, tag, "notnull")
}
