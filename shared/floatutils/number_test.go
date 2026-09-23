package floatutils

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseNumber(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input any
		want  float64
		ok    bool
	}{
		{name: "float", input: 12.5, want: 12.5, ok: true},
		{name: "int", input: 7, want: 7, ok: true},
		{name: "int64", input: int64(9), want: 9, ok: true},
		{name: "plain string", input: "1350", want: 1350, ok: true},
		{name: "money string", input: " $1,350.25 ", want: 1350.25, ok: true},
		{name: "percent string", input: "82.4%", want: 82.4, ok: true},
		{name: "empty string", input: "", ok: false},
		{name: "words", input: "twelve", ok: false},
		{name: "nan", input: math.NaN(), ok: false},
		{name: "nil", input: nil, ok: false},
		{name: "bool", input: true, ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, ok := ParseNumber(tt.input)
			assert.Equal(t, tt.ok, ok)
			if tt.ok {
				assert.InDelta(t, tt.want, got, 1e-9)
			}
		})
	}
}
