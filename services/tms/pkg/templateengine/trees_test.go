package templateengine

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractTemplatePosition(t *testing.T) {
	tests := []struct {
		name       string
		in         string
		wantLine   int
		wantColumn int
		wantMsg    string
	}{
		{
			"line and column",
			`template: invoice.pdf:PDF:12:4: unexpected "}" in operand`,
			12, 4, `unexpected "}" in operand`,
		},
		{
			"line only",
			`template: invoice.pdf:7: unexpected EOF`,
			7, 0, "unexpected EOF",
		},
		{
			"no position",
			"some other failure",
			0, 0, "some other failure",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			line, column, msg := extractTemplatePosition(tt.in)
			assert.Equal(t, tt.wantLine, line)
			assert.Equal(t, tt.wantColumn, column)
			assert.Equal(t, tt.wantMsg, msg)
		})
	}
}
