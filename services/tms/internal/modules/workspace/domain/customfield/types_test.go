package customfield

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFieldType_IsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		ft       FieldType
		expected bool
	}{
		{name: "text is valid", ft: FieldTypeText, expected: true},
		{name: "number is valid", ft: FieldTypeNumber, expected: true},
		{name: "date is valid", ft: FieldTypeDate, expected: true},
		{name: "boolean is valid", ft: FieldTypeBoolean, expected: true},
		{name: "select is valid", ft: FieldTypeSelect, expected: true},
		{name: "multiSelect is valid", ft: FieldTypeMultiSelect, expected: true},
		{name: "empty is invalid", ft: FieldType(""), expected: false},
		{name: "unknown is invalid", ft: FieldType("unknown"), expected: false},
		{name: "integer is invalid", ft: FieldType("integer"), expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.ft.IsValid())
		})
	}
}

func TestFieldType_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		ft       FieldType
		expected string
	}{
		{name: "text", ft: FieldTypeText, expected: "text"},
		{name: "number", ft: FieldTypeNumber, expected: "number"},
		{name: "date", ft: FieldTypeDate, expected: "date"},
		{name: "boolean", ft: FieldTypeBoolean, expected: "boolean"},
		{name: "select", ft: FieldTypeSelect, expected: "select"},
		{name: "multiSelect", ft: FieldTypeMultiSelect, expected: "multiSelect"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.ft.String())
		})
	}
}

func TestFieldType_RequiresOptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		ft       FieldType
		expected bool
	}{
		{name: "text does not require options", ft: FieldTypeText, expected: false},
		{name: "number does not require options", ft: FieldTypeNumber, expected: false},
		{name: "date does not require options", ft: FieldTypeDate, expected: false},
		{name: "boolean does not require options", ft: FieldTypeBoolean, expected: false},
		{name: "select requires options", ft: FieldTypeSelect, expected: true},
		{name: "multiSelect requires options", ft: FieldTypeMultiSelect, expected: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.ft.RequiresOptions())
		})
	}
}
