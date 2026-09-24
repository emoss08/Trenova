package pagination

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKeyCursorRoundTripsWithinItsScope(t *testing.T) {
	t.Parallel()

	encoded := EncodeKeyCursor("agent_tool_policy", "email_customer")
	key, err := DecodeKeyCursor("agent_tool_policy", encoded)
	require.NoError(t, err)
	assert.Equal(t, "email_customer", key)

	_, err = DecodeKeyCursor("shipment", encoded)
	require.ErrorIs(t, err, ErrKeyCursorInvalid)

	_, err = DecodeKeyCursor("agent_tool_policy", "%%%")
	require.ErrorIs(t, err, ErrKeyCursorInvalid)

	_, err = DecodeKeyCursor("agent_tool_policy", EncodeKeyCursor("agent_tool_policy", ""))
	require.ErrorIs(t, err, ErrKeyCursorInvalid)
}
