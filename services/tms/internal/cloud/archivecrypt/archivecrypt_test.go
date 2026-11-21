package archivecrypt_test

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"io"
	"testing"

	"github.com/emoss08/trenova/internal/cloud/archivecrypt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	vectorPart     = "tables/0001-shipments.copy.zst"
	vectorPlain    = "trenova tenant archive test vector"
	vectorStored   = "a0a1a2a3a4a5a603132c70d77081e07700b64ea1fa9b3975684380b4875c62d6f8746c3a3a9ece96482733c53ba47cb6a5d94a335cc25bf814"
	vectorInstance = "5d1a3c1e-8c1f-4a39-9a43-5b0d7c4e2f10"
	vectorWrapped  = "AAECAwQFBgcICQoLOgRue3OJQqfKG4rJtODCtJPVPxp4WbtE9jnyjV2sd0HjOpUN/WNdYvuT0gTO1xdD"
)

func sequence(start byte) []byte {
	out := make([]byte, archivecrypt.KeySize)
	for i := range out {
		out[i] = start + byte(i)
	}
	return out
}

func TestThePartEncryptionMatchesTheContractsTestVector(t *testing.T) {
	t.Parallel()

	key, err := archivecrypt.DataKeyOf(sequence(0))
	require.NoError(t, err)
	var stored bytes.Buffer
	w, err := key.EncryptWithPrefix(
		&stored,
		vectorPart,
		[]byte{0xa0, 0xa1, 0xa2, 0xa3, 0xa4, 0xa5, 0xa6},
	)
	require.NoError(t, err)
	_, err = w.Write([]byte(vectorPlain))
	require.NoError(t, err)
	require.NoError(t, w.Close())
	assert.Equal(t, vectorStored, hex.EncodeToString(stored.Bytes()))

	r, err := key.Decrypt(bytes.NewReader(stored.Bytes()), vectorPart)
	require.NoError(t, err)
	plain, err := io.ReadAll(r)
	require.NoError(t, err)
	assert.Equal(t, vectorPlain, string(plain))
}

func TestTheWrappedDataKeyMatchesTheContractsTestVector(t *testing.T) {
	t.Parallel()

	key, err := archivecrypt.Unwrap(vectorWrapped, sequence(0x40), vectorInstance)
	require.NoError(t, err)
	var stored bytes.Buffer
	w, err := key.EncryptWithPrefix(
		&stored,
		vectorPart,
		[]byte{0xa0, 0xa1, 0xa2, 0xa3, 0xa4, 0xa5, 0xa6},
	)
	require.NoError(t, err)
	_, err = w.Write([]byte(vectorPlain))
	require.NoError(t, err)
	require.NoError(t, w.Close())
	assert.Equal(t, vectorStored, hex.EncodeToString(stored.Bytes()))

	_, err = archivecrypt.Unwrap(
		vectorWrapped,
		sequence(0x40),
		"6e2b4d2f-9d2a-4b4a-8b54-6c1e8d5f3a21",
	)
	require.ErrorIs(
		t,
		err,
		archivecrypt.ErrUnwrap,
		"a data key never unwraps for another tenant's archive",
	)
	_, err = archivecrypt.Unwrap(vectorWrapped, sequence(0x41), vectorInstance)
	require.ErrorIs(t, err, archivecrypt.ErrUnwrap)
}

func seal(t *testing.T, key *archivecrypt.DataKey, part string, plain []byte) []byte {
	t.Helper()
	var stored bytes.Buffer
	w, err := key.Encrypt(&stored, part)
	require.NoError(t, err)
	_, err = w.Write(plain)
	require.NoError(t, err)
	require.NoError(t, w.Close())
	return stored.Bytes()
}

func open(key *archivecrypt.DataKey, part string, stored []byte) ([]byte, error) {
	r, err := key.Decrypt(bytes.NewReader(stored), part)
	if err != nil {
		return nil, err
	}
	return io.ReadAll(r)
}

func TestPartsOfEverySizeRoundTripAndTheirSizeIsDerivable(t *testing.T) {
	t.Parallel()

	key, err := archivecrypt.NewDataKey()
	require.NoError(t, err)
	for _, size := range []int{
		0, 1, archivecrypt.SegmentSize - 1, archivecrypt.SegmentSize, archivecrypt.SegmentSize + 1,
		3 * archivecrypt.SegmentSize,
	} {
		plain := make([]byte, size)
		_, _ = rand.Read(plain)
		stored := seal(t, key, "documents/a", plain)
		got, oerr := open(key, "documents/a", stored)
		require.NoError(t, oerr, size)
		assert.True(t, bytes.Equal(plain, got), size)
		derived, ok := archivecrypt.PlainSize(int64(len(stored)))
		require.True(t, ok)
		assert.Equal(t, int64(size), derived, size)
	}
}

func TestAnyChangeToAPartIsDetected(t *testing.T) {
	t.Parallel()

	key, err := archivecrypt.NewDataKey()
	require.NoError(t, err)
	plain := make([]byte, 2*archivecrypt.SegmentSize+100)
	_, _ = rand.Read(plain)
	stored := seal(t, key, "tables/0002-users.copy.zst", plain)

	for _, at := range []int{0, 3, 7, 50, archivecrypt.SegmentSize + 40, len(stored) - 1} {
		flipped := bytes.Clone(stored)
		flipped[at] ^= 0x01
		_, oerr := open(key, "tables/0002-users.copy.zst", flipped)
		require.Error(t, oerr, "a flipped byte at %d", at)
	}
	segment := archivecrypt.NoncePrefixSize + archivecrypt.SegmentSize + archivecrypt.TagSize
	for _, cut := range []int{segment, 2 * segment, len(stored) - 1, 5} {
		_, oerr := open(key, "tables/0002-users.copy.zst", stored[:cut])
		require.ErrorIs(t, oerr, archivecrypt.ErrTampered, "cut at %d", cut)
	}
	_, err = open(key, "tables/0003-users.copy.zst", stored)
	require.ErrorIs(t, err, archivecrypt.ErrTampered, "a part moved to another key does not open")
	other, err := archivecrypt.NewDataKey()
	require.NoError(t, err)
	_, err = open(other, "tables/0002-users.copy.zst", stored)
	require.ErrorIs(t, err, archivecrypt.ErrTampered, "a part does not open with another data key")
}
