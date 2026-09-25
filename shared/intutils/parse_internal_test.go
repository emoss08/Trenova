package intutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFirstInteger(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  int64
		ok    bool
	}{
		{input: "42,000 lbs", want: 42000, ok: true},
		{input: "Pieces: 26 pallets, 40 cases", want: 26, ok: true},
		{input: "1250.75", want: 1250, ok: true},
		{input: "none", ok: false},
		{input: "", ok: false},
		{input: "99999999999999999999999", ok: false},
	}

	for _, tt := range tests {
		got, ok := FirstInteger(tt.input)
		assert.Equal(t, tt.ok, ok, tt.input)
		assert.Equal(t, tt.want, got, tt.input)
	}
}
