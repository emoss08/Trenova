package querytooltest

import (
	"testing"

	"github.com/bytedance/sonic"
	"github.com/stretchr/testify/require"
)

func EncodedDocument(t *testing.T, result any) map[string]any {
	t.Helper()

	encoded, err := sonic.Marshal(result)
	require.NoError(t, err)
	var document map[string]any
	require.NoError(t, sonic.Unmarshal(encoded, &document))

	return document
}

func ObjectAt(t *testing.T, document any, path ...any) map[string]any {
	t.Helper()

	node := document
	for _, step := range path {
		switch key := step.(type) {
		case string:
			object, ok := node.(map[string]any)
			require.True(t, ok, "expected an object before %q", key)
			node = object[key]
		case int:
			list, ok := node.([]any)
			require.True(t, ok, "expected a list before [%d]", key)
			require.Greater(t, len(list), key)
			node = list[key]
		}
	}
	object, ok := node.(map[string]any)
	require.True(t, ok, "expected an object at %v", path)

	return object
}
