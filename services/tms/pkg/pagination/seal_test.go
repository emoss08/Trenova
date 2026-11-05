package pagination

import (
	"encoding/base64"
	"testing"

	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testTenant() TenantInfo {
	return TenantInfo{
		OrgID: "org_0000000000000000000000000",
		BuID:  "bu_00000000000000000000000000",
	}
}

func TestSealedCursor_RefusesAnyEditedByte(t *testing.T) {
	t.Parallel()

	binding := BindCursor(testTenant(), "workers", keysetShape)
	encoded := SealCursor(binding, []byte("1.1710000000.wrk_01"))
	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	require.NoError(t, err)

	for i := range raw {
		edited := append([]byte(nil), raw...)
		edited[i] ^= 0x01
		_, err = OpenCursor(binding, base64.RawURLEncoding.EncodeToString(edited))
		require.ErrorIs(t, err, ErrCursorInvalid, "byte %d", i)
	}

	payload, err := OpenCursor(binding, encoded)
	require.NoError(t, err)
	assert.Equal(t, "1.1710000000.wrk_01", string(payload))
}

func TestSealedCursor_KindsDoNotStandInForEachOther(t *testing.T) {
	t.Parallel()

	binding := BindCursor(testTenant(), "workers", nil)
	_, err := OpenCursor(binding, EncodeKeyCursor(binding, "payload"))
	require.ErrorIs(t, err, ErrCursorInvalid)
}

func TestSealedCursor_AddsLittleToWhatItCarries(t *testing.T) {
	t.Parallel()

	binding := BindCursor(testTenant(), "workers", keysetShape)
	encoded, err := EncodeKeysetCursor(binding, testKeyset)
	require.NoError(t, err)

	assert.LessOrEqual(
		t,
		len(encoded),
		len(testKeyset)+base64.RawURLEncoding.EncodedLen(cursorTagSize)+1,
	)
}

func TestBindCursor_SeparatesSortShapes(t *testing.T) {
	t.Parallel()

	byName := BindCursor(testTenant(), "workers", SortShape([]domaintypes.SortField{
		{Field: "name", Direction: dbtype.SortDirectionAsc},
	}))
	byNameDesc := BindCursor(testTenant(), "workers", SortShape([]domaintypes.SortField{
		{Field: "name", Direction: dbtype.SortDirectionDesc},
	}))
	assert.NotEqual(t, byName.Query, byNameDesc.Query)

	encoded := EncodeOffsetCursor(byName, 20)
	_, err := DecodeOffsetCursor(byNameDesc, encoded)
	require.ErrorIs(t, err, ErrCursorInvalid)
}

//nolint:paralleltest // it replaces the process cursor key, which every other test reads
func TestInstallCursorKey_ReplicasWithOneSecretShareCursors(t *testing.T) {
	binding := BindCursor(testTenant(), "workers", nil)

	InstallCursorKey("a-session-secret-that-is-at-least-32-characters")
	issued := EncodeOffsetCursor(binding, 20)

	InstallCursorKey("another-session-secret-of-at-least-32-characters")
	_, err := DecodeOffsetCursor(binding, issued)
	require.ErrorIs(t, err, ErrCursorInvalid, "a cursor from another key is refused")

	InstallCursorKey("a-session-secret-that-is-at-least-32-characters")
	offset, err := DecodeOffsetCursor(binding, issued)
	require.NoError(t, err, "a replica with the same secret reads the cursor")
	assert.Equal(t, 20, offset)
}
