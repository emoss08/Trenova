package pagination

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var keysetShape = []CursorSortField{
	{Field: "createdAt", Direction: "desc"},
	{Field: "id", Direction: "desc"},
}

func TestKeysetCursor_RoundTrip(t *testing.T) {
	t.Parallel()

	encoded, err := EncodeKeysetCursor(keysetShape, "c2Rh", 3)
	require.NoError(t, err)

	decoded, err := DecodeCursor(encoded)
	require.NoError(t, err)

	assert.Equal(t, Cursor{Sort: keysetShape, Keyset: "c2Rh", Skip: 3}, decoded)
	assert.True(t, decoded.IsKeyset())
	assert.True(t, decoded.MatchesSort(keysetShape))
	assert.False(t, decoded.MatchesSort(keysetShape[:1]))
}

func TestKeysetCursor_NeedsAKeysetOrASkip(t *testing.T) {
	t.Parallel()

	_, err := EncodeKeysetCursor(keysetShape, "", 0)
	require.Error(t, err)

	_, err = EncodeKeysetCursor(keysetShape, "abc", -1)
	require.ErrorIs(t, err, ErrCursorSkipInvalid)
}

func TestKeysetCursor_DecodeRefusesASkipPastTheLimit(t *testing.T) {
	t.Parallel()

	encoded, err := EncodeKeysetCursor(keysetShape, "abc", MaxCursorSkip+1)
	require.NoError(t, err)

	_, err = DecodeCursor(encoded)
	require.ErrorIs(t, err, ErrCursorSkipInvalid)
}

func TestKeysetCursor_DecodeRefusesValues(t *testing.T) {
	t.Parallel()

	raw := `{"sort":[{"field":"id","direction":"desc"}],"values":["x"],"keyset":"abc"}`
	_, err := DecodeCursor(base64.RawURLEncoding.EncodeToString([]byte(raw)))
	require.Error(t, err)
}

func TestNewCursorInfo_ReadsAKeysetCursor(t *testing.T) {
	t.Parallel()

	encoded, err := EncodeKeysetCursor(keysetShape, "abc", 0)
	require.NoError(t, err)

	info, err := NewCursorInfo(25, encoded)
	require.NoError(t, err)

	assert.Equal(t, 25, info.Limit)
	assert.Equal(t, encoded, info.After)
	assert.Equal(t, "abc", info.Cursor.Keyset)
}

func TestCursorListResult_PrefersEdgeCursors(t *testing.T) {
	t.Parallel()

	result := &CursorListResult[*struct{ ID string }]{
		Items:       []*struct{ ID string }{{ID: "a"}, {ID: "b"}},
		HasNextPage: true,
		EdgeCursors: []string{"first", "second"},
	}

	cursor, ok := result.EdgeCursorAt(1)
	assert.True(t, ok)
	assert.Equal(t, "second", cursor)

	_, ok = result.EdgeCursorAt(2)
	assert.False(t, ok)

	next, err := result.NextCursor()
	require.NoError(t, err)
	assert.Equal(t, "second", next)
}
