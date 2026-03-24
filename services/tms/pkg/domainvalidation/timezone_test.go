package domainvalidation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateTimezone(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		value   any
		wantErr bool
	}{
		{"valid timezone", "America/New_York", false},
		{"valid UTC", "UTC", false},
		{"valid Europe", "Europe/London", false},
		{"empty string", "", false},
		{"auto value", "auto", false},
		{"invalid timezone", "Invalid/Timezone", true},
		{"not a string", 123, true},
		{"nil value", nil, true},
		{"bool value", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateTimezone(tt.value)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateTimezone_NonStringError(t *testing.T) {
	t.Parallel()

	err := ValidateTimezone(42)
	assert.Equal(t, ErrTimezoneMustBeString, err)
}
