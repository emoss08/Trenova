package jsonflex

import (
	"testing"

	"github.com/bytedance/sonic"
	"github.com/stretchr/testify/require"
)

func TestStringListAcceptsAStringOrAList(t *testing.T) {
	t.Parallel()

	var payload struct {
		To StringList `json:"to"`
	}
	require.NoError(t, sonic.Unmarshal([]byte(`{"to":["a@example.com"," b@example.com ",null,""]}`), &payload))
	require.Equal(t, StringList{"a@example.com", "b@example.com"}, payload.To)
	require.Equal(t, "a@example.com", payload.To.First())

	require.NoError(t, sonic.Unmarshal([]byte(`{"to":"c@example.com"}`), &payload))
	require.Equal(t, StringList{"c@example.com"}, payload.To)

	require.NoError(t, sonic.Unmarshal([]byte(`{"to":null}`), &payload))
	require.Empty(t, payload.To)
	require.Empty(t, payload.To.First())

	require.Error(t, sonic.Unmarshal([]byte(`{"to":[{"x":1}]}`), &payload))
}
