package reflectutils_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/emoss08/trenova/shared/reflectutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type widget struct{ name string }

var errBroken = errors.New("broken")

func TestConstructReturnsTheConstructorsError(t *testing.T) {
	t.Parallel()

	built, err := reflectutils.Construct(func() (*widget, error) { return nil, errBroken }, nil)

	require.ErrorIs(t, err, errBroken)
	assert.Nil(t, built)
}

func TestConstructSuppliesWhatItHasAndZeroesTheRest(t *testing.T) {
	t.Parallel()

	name := "gear"
	built, err := reflectutils.Construct(func(n string, other *widget) (*widget, error) {
		if other != nil {
			return nil, errBroken
		}
		return &widget{name: n}, nil
	}, map[reflect.Type]reflect.Value{reflect.TypeFor[string](): reflect.ValueOf(name)})

	require.NoError(t, err)
	assert.Equal(t, &widget{name: name}, built)
}
