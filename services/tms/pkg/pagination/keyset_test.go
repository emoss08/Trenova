package pagination

import (
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

	encoded, err := EncodeKeysetCursor(keysetShape, "abc")
	require.NoError(t, err)

	decoded, err := DecodeCursor(encoded)
	require.NoError(t, err)
	assert.Equal(t, "abc", decoded.Keyset)
	assert.True(t, decoded.MatchesSort(keysetShape))
	assert.False(t, decoded.MatchesSort(keysetShape[:1]))
}

func TestKeysetCursor_NeedsAKeyset(t *testing.T) {
	t.Parallel()

	_, err := EncodeKeysetCursor(keysetShape, "")
	require.ErrorIs(t, err, ErrKeysetRequired)
}

func TestNewCursorInfo_ReadsAKeysetCursor(t *testing.T) {
	t.Parallel()

	encoded, err := EncodeKeysetCursor(keysetShape, "abc")
	require.NoError(t, err)

	info, err := NewCursorInfo(25, encoded)
	require.NoError(t, err)

	assert.Equal(t, 25, info.Limit)
	assert.Equal(t, encoded, info.After)
	assert.Equal(t, "abc", info.Cursor.Keyset)
}
