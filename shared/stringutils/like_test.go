package stringutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEscapeLikePattern(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "plain text", input: "acme", want: "acme"},
		{name: "percent", input: "100%", want: `100\%`},
		{name: "underscore", input: "pro_number", want: `pro\_number`},
		{name: "backslash", input: `a\b`, want: `a\\b`},
		{name: "mixed", input: `%_\`, want: `\%\_\\`},
		{name: "empty", input: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, EscapeLikePattern(tt.input))
		})
	}
}
