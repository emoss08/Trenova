package temporaltype

import (
	"testing"

	"github.com/bytedance/sonic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStreamItem_AtIsOptional(t *testing.T) {
	t.Parallel()

	unstamped, err := sonic.MarshalString(StreamItem{Event: "tool_started", Data: map[string]any{}})
	require.NoError(t, err)
	assert.JSONEq(t, `{"event":"tool_started","data":{}}`, unstamped)

	stamped, err := sonic.MarshalString(StreamItem{Event: "tool_started", At: 1_790_000_000})
	require.NoError(t, err)
	assert.JSONEq(t, `{"event":"tool_started","data":null,"at":1790000000}`, stamped)

	var recorded StreamItem
	require.NoError(t, sonic.UnmarshalString(`{"event":"delta","data":{"text":"hi"}}`, &recorded))
	assert.Zero(t, recorded.At)
}
