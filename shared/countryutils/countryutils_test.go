package countryutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestISO3ToISO2(t *testing.T) {
	t.Parallel()

	tests := []struct {
		iso3 string
		want string
		ok   bool
	}{
		{iso3: "USA", want: "US", ok: true},
		{iso3: "can", want: "CA", ok: true},
		{iso3: " MEX ", want: "MX", ok: true},
		{iso3: "GBR", want: "", ok: false},
		{iso3: "", want: "", ok: false},
	}
	for _, tt := range tests {
		got, ok := ISO3ToISO2(tt.iso3)
		assert.Equal(t, tt.want, got, tt.iso3)
		assert.Equal(t, tt.ok, ok, tt.iso3)
	}
	assert.Equal(t, "US", ISO3ToISO2OrDefault("GBR", "US"))
	assert.Equal(t, "CA", ISO3ToISO2OrDefault("CAN", "US"))
}
