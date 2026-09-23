package timeutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseInstant(t *testing.T) {
	t.Parallel()

	const noonSept1 = int64(1788264000)
	const midnightSept1 = int64(1788220800)

	tests := []struct {
		name  string
		input any
		want  int64
		ok    bool
	}{
		{name: "seconds", input: noonSept1, want: noonSept1, ok: true},
		{name: "milliseconds", input: noonSept1 * 1000, want: noonSept1, ok: true},
		{name: "float seconds", input: float64(noonSept1), want: noonSept1, ok: true},
		{name: "rfc3339", input: "2026-09-01T12:00:00Z", want: noonSept1, ok: true},
		{name: "seconds as text", input: "1788264000", want: noonSept1, ok: true},
		{name: "local date time", input: "2026-09-01T12:00", want: noonSept1, ok: true},
		{name: "calendar date", input: "2026-09-01", want: midnightSept1, ok: true},
		{name: "us date", input: "09/01/2026", want: midnightSept1, ok: true},
		{name: "words", input: "next tuesday", ok: false},
		{name: "empty", input: "", ok: false},
		{name: "nil", input: nil, ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, ok := ParseInstant(tt.input)
			assert.Equal(t, tt.ok, ok)
			if tt.ok {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
