package pagination

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCursorListResult_NextCursorIsTheLastEdgesCursor(t *testing.T) {
	t.Parallel()

	result := &CursorListResult[string]{
		Items:       []string{"a", "b"},
		HasNextPage: true,
		EdgeCursors: []string{"first", "second"},
	}
	next, err := NextCursor(result)
	require.NoError(t, err)
	assert.Equal(t, "second", next)

	result.HasNextPage = false
	next, err = NextCursor(result)
	require.NoError(t, err)
	assert.Empty(t, next, "the last page has no next cursor")

	result.HasNextPage = true
	result.EdgeCursors = nil
	_, err = NextCursor(result)
	require.ErrorIs(t, err, ErrNoEdgeCursor)
}

func TestPositionCursor_RoundTripsUnderItsBinding(t *testing.T) {
	t.Parallel()

	binding := BindCursor(testTenant(), "watchtower", nil)
	id := pulid.MustNew("wti_")
	encoded, err := EncodePositionCursor(binding, PositionCursor{At: 1710000000, ID: id})
	require.NoError(t, err)

	decoded, err := DecodePositionCursor(binding, encoded)
	require.NoError(t, err)
	assert.Equal(t, PositionCursor{At: 1710000000, ID: id}, decoded)

	_, err = DecodePositionCursor(BindCursor(testTenant(), "decisions", nil), encoded)
	require.ErrorIs(t, err, ErrCursorInvalid, "a cursor reads only under the list it came from")

	_, err = EncodePositionCursor(binding, PositionCursor{At: 1})
	require.ErrorIs(t, err, ErrPositionIDRequired)
}

func TestNewCursorInfo_RefusesWhatCannotBeACursor(t *testing.T) {
	t.Parallel()

	for _, after := range []string{
		"not base64!",
		base64.RawURLEncoding.EncodeToString([]byte("short")),
		strings.Repeat("A", MaxCursorLength+1),
	} {
		_, err := NewCursorInfo(10, after)
		require.ErrorIs(t, err, ErrCursorInvalid, after)
	}

	info, err := NewCursorInfo(10, "")
	require.NoError(t, err)
	assert.Empty(t, info.After)
}
