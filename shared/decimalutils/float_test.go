package decimalutils_test

import (
	"math"
	"testing"

	"github.com/emoss08/trenova/shared/decimalutils"
	"github.com/stretchr/testify/assert"
)

func TestFromFloat_DropsBinaryNoise(t *testing.T) {
	t.Parallel()

	cases := map[float64]string{
		0.1 + 0.2:          "0.3",
		0.145 * 100:        "14.5",
		1.15 * 100:         "115",
		1.005:              "1.005",
		0.6000000000000001: "0.6",
		-2.675:             "-2.675",
		123456789.125:      "123456789.125",
		1e-7:               "0.0000001",
		0:                  "0",
	}

	for value, want := range cases {
		got, ok := decimalutils.FromFloat(value)
		assert.True(t, ok, "%v", value)
		assert.Equal(t, want, got.String(), "%v", value)
	}
}

func TestFromFloat_RefusesNonFiniteValues(t *testing.T) {
	t.Parallel()

	for _, value := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		_, ok := decimalutils.FromFloat(value)
		assert.False(t, ok, "%v", value)
	}
}
