package domaintypes_test

import (
	"testing"

	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatus(t *testing.T) {
	t.Parallel()

	t.Run("Active constant", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "Active", domaintypes.StatusActive.String())
	})

	t.Run("Inactive constant", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "Inactive", domaintypes.StatusInactive.String())
	})
}

func TestStatusFromString(t *testing.T) {
	t.Parallel()

	t.Run("parses Active", func(t *testing.T) {
		t.Parallel()
		s, err := domaintypes.StatusFromString("Active")
		require.NoError(t, err)
		assert.Equal(t, domaintypes.StatusActive, s)
	})

	t.Run("parses Inactive", func(t *testing.T) {
		t.Parallel()
		s, err := domaintypes.StatusFromString("Inactive")
		require.NoError(t, err)
		assert.Equal(t, domaintypes.StatusInactive, s)
	})

	t.Run("returns error for invalid status", func(t *testing.T) {
		t.Parallel()
		_, err := domaintypes.StatusFromString("Unknown")
		assert.ErrorIs(t, err, domaintypes.ErrInvalidStatus)
	})

	t.Run("returns error for empty string", func(t *testing.T) {
		t.Parallel()
		_, err := domaintypes.StatusFromString("")
		assert.ErrorIs(t, err, domaintypes.ErrInvalidStatus)
	})

	t.Run("case sensitive", func(t *testing.T) {
		t.Parallel()
		_, err := domaintypes.StatusFromString("active")
		assert.ErrorIs(t, err, domaintypes.ErrInvalidStatus)
	})
}

func TestEquipmentStatusFromString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected domaintypes.EquipmentStatus
		wantErr  bool
	}{
		{"Available", "Available", domaintypes.EquipmentStatusAvailable, false},
		{"OutOfService", "OutOfService", domaintypes.EquipmentStatusOOS, false},
		{"AtMaintenance", "AtMaintenance", domaintypes.EquipmentStatusAtMaintenance, false},
		{"Sold", "Sold", domaintypes.EquipmentStatusSold, false},
		{"invalid", "BadValue", "", true},
		{"empty", "", "", true},
		{"lowercase", "available", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, err := domaintypes.EquipmentStatusFromString(tt.input)
			if tt.wantErr {
				assert.ErrorIs(t, err, domaintypes.ErrInvalidEquipmentStatus)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestValidateVin(t *testing.T) {
	t.Parallel()

	t.Run("valid VIN", func(t *testing.T) {
		t.Parallel()
		err := domaintypes.ValidateVin("1HGCM82633A004352")
		assert.NoError(t, err)
	})

	t.Run("empty VIN returns nil", func(t *testing.T) {
		t.Parallel()
		err := domaintypes.ValidateVin("")
		assert.NoError(t, err)
	})

	t.Run("invalid VIN too short", func(t *testing.T) {
		t.Parallel()
		err := domaintypes.ValidateVin("1HGCM826")
		assert.ErrorIs(t, err, domaintypes.ErrInvalidVin)
	})

	t.Run("invalid VIN with I character", func(t *testing.T) {
		t.Parallel()
		err := domaintypes.ValidateVin("1HGCM82633I004352")
		assert.ErrorIs(t, err, domaintypes.ErrInvalidVin)
	})

	t.Run("invalid VIN with O character", func(t *testing.T) {
		t.Parallel()
		err := domaintypes.ValidateVin("1HGCM82633O004352")
		assert.ErrorIs(t, err, domaintypes.ErrInvalidVin)
	})

	t.Run("invalid VIN with Q character", func(t *testing.T) {
		t.Parallel()
		err := domaintypes.ValidateVin("1HGCM82633Q004352")
		assert.ErrorIs(t, err, domaintypes.ErrInvalidVin)
	})

	t.Run("non-string value returns nil", func(t *testing.T) {
		t.Parallel()
		err := domaintypes.ValidateVin(12345)
		assert.NoError(t, err)
	})
}

func TestValidatePostalCode(t *testing.T) {
	t.Parallel()

	t.Run("valid 5-digit zip", func(t *testing.T) {
		t.Parallel()
		err := domaintypes.ValidatePostalCode("90210")
		assert.NoError(t, err)
	})

	t.Run("valid zip+4", func(t *testing.T) {
		t.Parallel()
		err := domaintypes.ValidatePostalCode("90210-1234")
		assert.NoError(t, err)
	})

	t.Run("empty returns nil", func(t *testing.T) {
		t.Parallel()
		err := domaintypes.ValidatePostalCode("")
		assert.NoError(t, err)
	})

	t.Run("invalid format", func(t *testing.T) {
		t.Parallel()
		err := domaintypes.ValidatePostalCode("ABCDE")
		assert.ErrorIs(t, err, domaintypes.ErrInvalidPostalCode)
	})

	t.Run("too short", func(t *testing.T) {
		t.Parallel()
		err := domaintypes.ValidatePostalCode("1234")
		assert.ErrorIs(t, err, domaintypes.ErrInvalidPostalCode)
	})

	t.Run("too long without dash", func(t *testing.T) {
		t.Parallel()
		err := domaintypes.ValidatePostalCode("123456789")
		assert.ErrorIs(t, err, domaintypes.ErrInvalidPostalCode)
	})

	t.Run("non-string value returns nil", func(t *testing.T) {
		t.Parallel()
		err := domaintypes.ValidatePostalCode(12345)
		assert.NoError(t, err)
	})
}

func TestSearchWeight_GetScore(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		weight   domaintypes.SearchWeight
		expected int
	}{
		{"weight A", domaintypes.SearchWeightA, 4},
		{"weight B", domaintypes.SearchWeightB, 3},
		{"weight C", domaintypes.SearchWeightC, 2},
		{"weight D", domaintypes.SearchWeightD, 1},
		{"weight blank", domaintypes.SearchWeightBlank, 0},
		{"unknown weight", domaintypes.SearchWeight("X"), 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.weight.GetScore())
		})
	}
}
