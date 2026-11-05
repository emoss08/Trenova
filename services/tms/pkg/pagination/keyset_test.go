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

var testKeyset = base64.RawURLEncoding.EncodeToString([]byte("\x01keyset-values"))

func TestKeysetCursor_RoundTripsUnderTheQueryItWasIssuedFor(t *testing.T) {
	t.Parallel()

	binding := BindCursor(testTenant(), "workers", keysetShape)
	encoded, err := EncodeKeysetCursor(binding, testKeyset)
	require.NoError(t, err)

	keyset, err := DecodeKeysetCursor(binding, encoded)
	require.NoError(t, err)
	assert.Equal(t, testKeyset, keyset)
}

func TestKeysetCursor_IsRefusedUnderAnotherSortTableOrTenant(t *testing.T) {
	t.Parallel()

	encoded, err := EncodeKeysetCursor(BindCursor(testTenant(), "workers", keysetShape), testKeyset)
	require.NoError(t, err)

	other := testTenant()
	other.OrgID = "org_other_000000000000000000"
	for name, binding := range map[string]CursorBinding{
		"sort":   BindCursor(testTenant(), "workers", keysetShape[:1]),
		"table":  BindCursor(testTenant(), "tractors", keysetShape),
		"tenant": BindCursor(other, "workers", keysetShape),
	} {
		_, err = DecodeKeysetCursor(binding, encoded)
		require.ErrorIs(t, err, ErrCursorInvalid, name)
	}
}

func TestKeysetCursor_NeedsAKeyset(t *testing.T) {
	t.Parallel()

	_, err := EncodeKeysetCursor(BindCursor(testTenant(), "workers", keysetShape), "")
	require.ErrorIs(t, err, ErrKeysetRequired)
}

func TestNewCursorInfo_CarriesTheCursorForTheListToVerify(t *testing.T) {
	t.Parallel()

	encoded, err := EncodeKeysetCursor(BindCursor(testTenant(), "workers", keysetShape), testKeyset)
	require.NoError(t, err)

	info, err := NewCursorInfo(25, encoded)
	require.NoError(t, err)

	assert.Equal(t, 25, info.Limit)
	assert.Equal(t, encoded, info.After)
}
