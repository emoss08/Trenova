package timeutils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseWindow(t *testing.T) {
	t.Parallel()

	chicago, err := time.LoadLocation("America/Chicago")
	require.NoError(t, err)
	now := time.Date(2026, time.October, 5, 12, 0, 0, 0, chicago)

	tests := []struct {
		name      string
		text      string
		wantStart time.Time
		wantEnd   time.Time
		ok        bool
	}{
		{
			name:      "named day with window",
			text:      "Oct 7, 08:00-14:00",
			wantStart: time.Date(2026, time.October, 7, 8, 0, 0, 0, chicago),
			wantEnd:   time.Date(2026, time.October, 7, 14, 0, 0, 0, chicago),
			ok:        true,
		},
		{
			name:      "iso instant",
			text:      "2026-10-08T06:30",
			wantStart: time.Date(2026, time.October, 8, 6, 30, 0, 0, chicago),
			ok:        true,
		},
		{
			name:      "numeric with meridiem",
			text:      "10/09/2026 2pm",
			wantStart: time.Date(2026, time.October, 9, 14, 0, 0, 0, chicago),
			ok:        true,
		},
		{
			name:      "date only",
			text:      "Thu Oct 8",
			wantStart: time.Date(2026, time.October, 8, 0, 0, 0, 0, chicago),
			ok:        true,
		},
		{
			name:      "next year when long past",
			text:      "Jan 3 09:00",
			wantStart: time.Date(2027, time.January, 3, 9, 0, 0, 0, chicago),
			ok:        true,
		},
		{name: "no date", text: "tomorrow morning", ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			window, ok := ParseWindow(tt.text, chicago, now)
			assert.Equal(t, tt.ok, ok)
			if !tt.ok {
				return
			}
			assert.Equal(t, tt.wantStart.Unix(), window.Start)
			if tt.wantEnd.IsZero() {
				assert.Zero(t, window.End)
			} else {
				assert.Equal(t, tt.wantEnd.Unix(), window.End)
			}
		})
	}
}
