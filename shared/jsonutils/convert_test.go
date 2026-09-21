package jsonutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvert(t *testing.T) {
	t.Parallel()

	type stop struct {
		LocationID string `json:"locationId"`
		Type       string `json:"type"`
	}
	type payload struct {
		Stops  []stop `json:"stops"`
		Pieces *int64 `json:"pieces"`
	}

	var out payload
	require.NoError(t, Convert(map[string]any{
		"stops":  []any{map[string]any{"locationId": "loc_1", "type": "Pickup"}},
		"pieces": 4,
	}, &out))

	require.Len(t, out.Stops, 1)
	assert.Equal(t, "loc_1", out.Stops[0].LocationID)
	require.NotNil(t, out.Pieces)
	assert.EqualValues(t, 4, *out.Pieces)

	var wrong payload
	assert.Error(t, Convert(map[string]any{"stops": "not a list"}, &wrong),
		"a value that does not fit the destination is refused")
}
