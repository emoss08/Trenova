package pagination

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKeyCursorRoundTripsWithinItsBinding(t *testing.T) {
	t.Parallel()

	binding := BindCursor(testTenant(), "agent_tool_policy", nil)
	encoded := EncodeKeyCursor(binding, "customer_send_email")
	key, err := DecodeKeyCursor(binding, encoded)
	require.NoError(t, err)
	assert.Equal(t, "customer_send_email", key)

	_, err = DecodeKeyCursor(BindCursor(testTenant(), "shipment", nil), encoded)
	require.ErrorIs(t, err, ErrCursorInvalid)

	_, err = DecodeKeyCursor(binding, "%%%")
	require.ErrorIs(t, err, ErrCursorInvalid)
}

func TestOffsetCursorRoundTripsWithinItsBinding(t *testing.T) {
	t.Parallel()

	binding := BindCursor(testTenant(), "/api/v1/items", nil)
	offset, err := DecodeOffsetCursor(binding, EncodeOffsetCursor(binding, 40))
	require.NoError(t, err)
	assert.Equal(t, 40, offset)

	offset, err = DecodeOffsetCursor(binding, "")
	require.NoError(t, err)
	assert.Zero(t, offset, "no cursor is the first page")

	_, err = DecodeOffsetCursor(binding, EncodeOffsetCursor(binding, MaxOffsetCursor+1))
	require.ErrorIs(t, err, ErrCursorInvalid)

	_, err = DecodeOffsetCursor(binding, EncodeKeyCursor(binding, "40"))
	require.ErrorIs(t, err, ErrCursorInvalid, "a key cursor is not an offset cursor")
}
