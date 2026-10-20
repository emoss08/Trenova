package pagination

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKeyCursorRoundTripsWithinItsScope(t *testing.T) {
	t.Parallel()

	encoded := EncodeKeyCursor("agent_tool_policy", "customer_send_email")
	key, err := DecodeKeyCursor("agent_tool_policy", encoded)
	require.NoError(t, err)
	assert.Equal(t, "customer_send_email", key)

	_, err = DecodeKeyCursor("shipment", encoded)
	require.ErrorIs(t, err, ErrKeyCursorInvalid)

	_, err = DecodeKeyCursor("agent_tool_policy", "%%%")
	require.ErrorIs(t, err, ErrKeyCursorInvalid)

	_, err = DecodeKeyCursor("agent_tool_policy", EncodeKeyCursor("agent_tool_policy", ""))
	require.ErrorIs(t, err, ErrKeyCursorInvalid)
}
