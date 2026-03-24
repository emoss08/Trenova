package stringutils_test

import (
	"testing"

	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/stretchr/testify/assert"
)

func TestParseBool(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected bool
		ok       bool
	}{
		{"true lowercase", "true", true, true},
		{"false lowercase", "false", false, true},
		{"TRUE uppercase", "TRUE", true, true},
		{"FALSE uppercase", "FALSE", false, true},
		{"True mixed case", "True", true, true},
		{"False mixed case", "False", false, true},
		{"1", "1", true, true},
		{"0", "0", false, true},
		{"yes", "yes", true, true},
		{"no", "no", false, true},
		{"YES uppercase", "YES", true, true},
		{"NO uppercase", "NO", false, true},
		{"on", "on", true, true},
		{"off", "off", false, true},
		{"ON uppercase", "ON", true, true},
		{"OFF uppercase", "OFF", false, true},
		{"invalid string", "invalid", false, false},
		{"empty string", "", false, false},
		{"random text", "maybe", false, false},
		{"t shorthand", "t", true, true},
		{"f shorthand", "f", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result, ok := stringutils.ParseBool(tt.input)
			assert.Equal(t, tt.expected, result)
			assert.Equal(t, tt.ok, ok)
		})
	}
}
