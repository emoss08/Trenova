package stringutils_test

import (
	"testing"

	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/stretchr/testify/assert"
)

func TestMarkdownTableCell(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "plain", in: "Reads only", want: "Reads only"},
		{name: "pipe is escaped", in: "a | b", want: `a \| b`},
		{name: "line breaks collapse", in: "first\nsecond\r\n  third", want: "first second third"},
		{name: "runs of spaces collapse", in: "  spaced   out  ", want: "spaced out"},
		{name: "empty", in: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, stringutils.MarkdownTableCell(tt.in))
		})
	}
}
