package pagination

import (
	"encoding/base64"
	"testing"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecodeCursor_MalformedBase64(t *testing.T) {
	t.Parallel()

	_, err := DecodeCursor("not base64")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode cursor")
}

func TestDecodeCursor_MalformedJSON(t *testing.T) {
	t.Parallel()

	encoded := base64.RawURLEncoding.EncodeToString([]byte("not-json"))

	_, err := DecodeCursor(encoded)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal cursor")
}

func TestDecodeCursor_RefusesACursorWithoutAKeyset(t *testing.T) {
	t.Parallel()

	encoded := base64.RawURLEncoding.EncodeToString(
		[]byte(`{"createdAt":1710000000,"id":"trac_01"}`),
	)

	_, err := DecodeCursor(encoded)
	require.ErrorIs(t, err, ErrKeysetRequired)
}

func TestCursorListResult_NextCursorIsTheLastEdgesCursor(t *testing.T) {
	t.Parallel()

	result := &CursorListResult[string]{
		Items:       []string{"a", "b"},
		HasNextPage: true,
		EdgeCursors: []string{"first", "second"},
	}
	next, err := result.NextCursor()
	require.NoError(t, err)
	assert.Equal(t, "second", next)

	result.HasNextPage = false
	next, err = result.NextCursor()
	require.NoError(t, err)
	assert.Empty(t, next, "the last page has no next cursor")

	result.HasNextPage = true
	result.EdgeCursors = nil
	_, err = result.NextCursor()
	require.ErrorIs(t, err, ErrNoEdgeCursor)
}

func TestPositionCursor_RoundTrip(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("wti_")
	encoded, err := EncodePositionCursor(PositionCursor{At: 1710000000, ID: id})
	require.NoError(t, err)

	decoded, err := DecodePositionCursor(encoded)
	require.NoError(t, err)
	assert.Equal(t, PositionCursor{At: 1710000000, ID: id}, decoded)

	_, err = EncodePositionCursor(PositionCursor{At: 1})
	require.ErrorIs(t, err, ErrPositionIDRequired)

	_, err = DecodePositionCursor(base64.RawURLEncoding.EncodeToString([]byte(`{"at":1}`)))
	require.ErrorIs(t, err, ErrPositionIDRequired)
}
