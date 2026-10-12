package shipment

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPickupWindowsAtPartitionsTheDay(t *testing.T) {
	t.Parallel()

	loc, err := time.LoadLocation("America/Chicago")
	require.NoError(t, err)

	morning := time.Date(2026, 10, 6, 9, 30, 0, 0, loc)
	windows := PickupWindowsAt(morning, loc)
	require.Len(t, windows, 4)
	assert.Equal(t, PickupWindowUnderTwoHours, windows[0].Window)
	assert.Equal(t, 0, windows[0].StartMinutes)
	assert.Equal(t, 120, *windows[0].EndMinutes)
	assert.Equal(t, 120, windows[1].StartMinutes)
	assert.Equal(t, 360, *windows[1].EndMinutes)
	assert.Equal(t, 360, windows[2].StartMinutes)
	assert.Equal(t, 870, *windows[2].EndMinutes)
	assert.Equal(t, 870, windows[3].StartMinutes)
	assert.Nil(t, windows[3].EndMinutes)

	evening := time.Date(2026, 10, 6, 22, 30, 0, 0, loc)
	windows = PickupWindowsAt(evening, loc)
	assert.Equal(t, 90, *windows[0].EndMinutes)
	assert.True(t, windows[1].Empty())
	assert.True(t, windows[2].Empty())
	assert.Equal(t, 90, windows[3].StartMinutes)

	for _, window := range windows {
		assert.True(t, window.Window.IsValid())
		assert.Equal(t, QuickFilterPickupWindow, window.Spec().Filter)
	}
}
