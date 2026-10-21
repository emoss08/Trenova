package stringutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStripNUL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty", in: "", want: ""},
		{name: "no NUL is returned unchanged", in: "BOL 4471 — Dallas", want: "BOL 4471 — Dallas"},
		{name: "embedded NUL", in: "Load\x00 4471", want: "Load 4471"},
		{name: "only NULs", in: "\x00\x00", want: ""},
		{name: "leading and trailing", in: "\x00rate\x00", want: "rate"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, StripNUL(tt.in))
		})
	}
}
