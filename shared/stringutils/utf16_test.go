package stringutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestByteOffsetOfUTF16(t *testing.T) {
	t.Parallel()

	source := "-5 °F é 🚚 x"
	tests := []struct {
		name  string
		units int
		want  string
	}{
		{name: "the start", units: 0, want: ""},
		{name: "before a two-byte character", units: 3, want: "-5 "},
		{name: "after a two-byte character", units: 4, want: "-5 °"},
		{name: "after é", units: 7, want: "-5 °F é"},
		{name: "after an emoji's two code units", units: 10, want: "-5 °F é 🚚"},
		{name: "between an emoji's code units", units: 9, want: "-5 °F é "},
		{name: "the end", units: 12, want: source},
		{name: "past the end", units: 99, want: source},
		{name: "before the start", units: -3, want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, source[:ByteOffsetOfUTF16(source, tt.units)])
		})
	}
}
