package stringutils_test

import (
	"testing"

	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/stretchr/testify/assert"
)

func TestFirstName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "full name", input: "Dana Whitfield", expected: "Dana"},
		{name: "single name", input: "Dana", expected: "Dana"},
		{name: "surrounding whitespace", input: "  Dana Whitfield  ", expected: "Dana"},
		{name: "tab separated", input: "Dana\tWhitfield", expected: "Dana"},
		{name: "blank", input: "   ", expected: ""},
		{name: "empty", input: "", expected: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, stringutils.FirstName(tt.input))
		})
	}
}
