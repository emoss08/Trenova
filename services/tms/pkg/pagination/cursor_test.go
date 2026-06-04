package pagination

import (
	"encoding/base64"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCursorCodec_RoundTrip(t *testing.T) {
	t.Parallel()

	expected := Cursor{
		CreatedAt: 1710000000,
		ID:        pulid.MustNew("tr_"),
	}

	encoded, err := EncodeCursor(expected)
	require.NoError(t, err)

	actual, err := DecodeCursor(encoded)
	require.NoError(t, err)
	assert.Equal(t, expected, actual)
}

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

func TestDecodeCursor_MissingID(t *testing.T) {
	t.Parallel()

	bytes, err := sonic.Marshal(map[string]any{"createdAt": int64(1710000000)})
	require.NoError(t, err)
	encoded := base64.RawURLEncoding.EncodeToString(bytes)

	_, err = DecodeCursor(encoded)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cursor id is required")
}

func TestDecodeCursor_InvalidID(t *testing.T) {
	t.Parallel()

	bytes, err := sonic.Marshal(map[string]any{
		"createdAt": int64(1710000000),
		"id":        "bad",
	})
	require.NoError(t, err)
	encoded := base64.RawURLEncoding.EncodeToString(bytes)

	_, err = DecodeCursor(encoded)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cursor id is invalid")
}

func TestEncodeCursor_OutputShape(t *testing.T) {
	t.Parallel()

	cursor := Cursor{
		CreatedAt: 1710000000,
		ID:        pulid.ID("tr_01ARZ3NDEKTSV4RRFFQ69G5FAV"),
	}

	encoded, err := EncodeCursor(cursor)
	require.NoError(t, err)

	bytes, err := base64.RawURLEncoding.DecodeString(encoded)
	require.NoError(t, err)
	assert.JSONEq(
		t,
		`{"createdAt":1710000000,"id":"tr_01ARZ3NDEKTSV4RRFFQ69G5FAV"}`,
		string(bytes),
	)
}
