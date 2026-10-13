package timeutils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestWindowLabelIn_CollapsesTheEndOnTheSameLocalDay(t *testing.T) {
	chicago := LoadLocation("America/Chicago")
	start := time.Date(2026, 7, 19, 22, 0, 0, 0, chicago).Unix()
	end := time.Date(2026, 7, 19, 23, 30, 0, 0, chicago).Unix()

	assert.Equal(t, "Sun 19 Jul 2026 22:00 CDT–23:30 CDT", WindowLabelIn(start, &end, chicago))
	assert.Equal(t, "Mon 20 Jul 2026 03:00 UTC–04:30 UTC", WindowLabelUTC(start, &end))
}

func TestWindowLabelIn_SpellsOutAnEndOnALaterDay(t *testing.T) {
	start := time.Date(2026, 7, 19, 9, 0, 0, 0, time.UTC).Unix()
	end := time.Date(2026, 7, 20, 9, 0, 0, 0, time.UTC).Unix()

	assert.Equal(t,
		"Sun 19 Jul 2026 09:00 UTC – Mon 20 Jul 2026 09:00 UTC",
		WindowLabelIn(start, &end, nil),
	)
	assert.Empty(t, WindowLabelIn(0, nil, nil))
	assert.Equal(t, "Sun 19 Jul 2026 09:00 UTC", WindowLabelIn(start, nil, nil))
}
