package stringutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeIdentifier(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "ABC123", NormalizeIdentifier("Abc-123"))
	assert.Equal(t, "INV2024001", NormalizeIdentifier("INV 2024/001"))
	assert.Equal(t, "", NormalizeIdentifier(" -_/ "))
}

func TestTruncateRunes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		value     string
		maxLength int
		want      string
	}{
		{name: "shorter than max", value: "ABC", maxLength: 5, want: "ABC"},
		{name: "equal to max", value: "ABC", maxLength: 3, want: "ABC"},
		{name: "longer than max", value: "ABCDEFG", maxLength: 3, want: "ABC"},
		{name: "unicode safe", value: "ÅBCDEF", maxLength: 3, want: "ÅBC"},
		{name: "zero max", value: "ABC", maxLength: 0, want: ""},
		{name: "negative max", value: "ABC", maxLength: -1, want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, TruncateRunes(tt.value, tt.maxLength))
		})
	}
}
