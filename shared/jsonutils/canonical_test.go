package jsonutils

import (
	"testing"

	"github.com/bytedance/sonic"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCanonicalMarshal_SortsKeysAndWritesExactNumbers(t *testing.T) {
	t.Parallel()

	encoded, err := CanonicalMarshal(map[string]any{
		"zeta":  1.10,
		"alpha": map[string]any{"b": int64(2), "a": []any{3, "x", nil, true}},
		"mid":   decimal.RequireFromString("40000.50"),
	})
	require.NoError(t, err)

	assert.JSONEq(t, `{"alpha":{"a":[3,"x",null,true],"b":2},"mid":"40000.5","zeta":1.1}`,
		string(encoded))
	assert.Equal(t,
		`{"alpha":{"a":[3,"x",null,true],"b":2},"mid":"40000.5","zeta":1.1}`,
		string(encoded),
	)
}

func TestCanonicalMarshal_IsTheSameWhateverProducedTheValue(t *testing.T) {
	t.Parallel()

	type row struct {
		Name   string  `json:"name"`
		Amount float64 `json:"amount"`
		Count  int     `json:"count"`
	}
	built := map[string]any{"rows": []row{{Name: "a", Amount: 12.5, Count: 3}}, "ok": true}

	raw, err := sonic.Marshal(built)
	require.NoError(t, err)
	var decoded any
	require.NoError(t, sonic.Unmarshal(raw, &decoded))

	fromGo, err := CanonicalMarshal(built)
	require.NoError(t, err)
	fromJSON, err := CanonicalMarshal(decoded)
	require.NoError(t, err)

	assert.Equal(t, string(fromGo), string(fromJSON))
}

func TestCanonicalDigest_IsStableAndSensitiveToContent(t *testing.T) {
	t.Parallel()

	first, err := CanonicalDigest(map[string]any{"b": 1, "a": "x"})
	require.NoError(t, err)
	again, err := CanonicalDigest(map[string]any{"a": "x", "b": 1.0})
	require.NoError(t, err)
	changed, err := CanonicalDigest(map[string]any{"a": "x", "b": 2})
	require.NoError(t, err)

	assert.Len(t, first, 64)
	assert.Equal(t, first, again)
	assert.NotEqual(t, first, changed)
}

func TestCanonicalValue_NilMapIsNull(t *testing.T) {
	t.Parallel()

	var empty map[string]any
	value, err := CanonicalValue(empty)
	require.NoError(t, err)
	assert.Nil(t, value)

	encoded, err := CanonicalMarshal(nil)
	require.NoError(t, err)
	assert.Equal(t, "null", string(encoded))
}
