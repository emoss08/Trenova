package archivestream_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"testing"

	"github.com/emoss08/trenova/internal/cloud/archivestream"
	"github.com/stretchr/testify/require"
)

func TestAZeroLengthReadIsNotAChangedPart(t *testing.T) {
	t.Parallel()

	body := []byte("a stored part")
	digest := sha256.Sum256(body)
	sum := archivestream.Sum{Bytes: int64(len(body)), SHA256: hex.EncodeToString(digest[:])}
	r, err := archivestream.NewReader(bytes.NewReader(body), sum, &archivestream.Options{
		Part: "documents/0001",
	})
	require.NoError(t, err)
	defer r.Close()

	n, err := r.Read(nil)
	require.NoError(t, err, "io.Reader allows a zero-length read")
	require.Zero(t, n)
	n, err = r.Read([]byte{})
	require.NoError(t, err)
	require.Zero(t, n)

	read, err := io.ReadAll(r)
	require.NoError(t, err)
	require.Equal(t, body, read)
	require.NoError(t, archivestream.Verify(bytes.NewReader(body), sum))
}

func compressed(t *testing.T, plain int64) ([]byte, archivestream.Sum) {
	t.Helper()
	var stored bytes.Buffer
	w, err := archivestream.NewWriter(&stored, &archivestream.Options{Part: "tables/0001-x", Compress: true})
	require.NoError(t, err)
	_, err = io.CopyN(w, zeros{}, plain)
	require.NoError(t, err)
	require.NoError(t, w.Close())
	return stored.Bytes(), w.Sum()
}

type zeros struct{}

func (zeros) Read(p []byte) (int, error) {
	clear(p)
	return len(p), nil
}

func TestAPartReadsBackWithinItsExpansionBound(t *testing.T) {
	t.Parallel()

	stored, sum := compressed(t, 8<<20)
	r, err := archivestream.NewReader(bytes.NewReader(stored), sum, &archivestream.Options{
		Part: "tables/0001-x", Compress: true,
	})
	require.NoError(t, err)
	defer r.Close()
	n, err := io.Copy(io.Discard, r)
	require.NoError(t, err)
	require.Equal(t, int64(8<<20), n)
}

type counting struct {
	src  io.Reader
	read int64
}

func (c *counting) Read(p []byte) (int, error) {
	n, err := c.src.Read(p)
	c.read += int64(n)
	return n, err
}

func TestAStoredObjectLongerThanItsManifestSaysStopsAtTheDeclaredSize(t *testing.T) {
	t.Parallel()

	declared := archivestream.Sum{Bytes: 1 << 10, SHA256: "unused"}
	endless := &counting{src: zeros{}}
	r, err := archivestream.NewReader(endless, declared, &archivestream.Options{Part: "documents/0001"})
	require.NoError(t, err)
	defer r.Close()
	_, err = io.Copy(io.Discard, r)
	require.ErrorIs(t, err, archivestream.ErrStoredBytesChanged)
	require.LessOrEqual(t, endless.read, int64(1<<10)+1,
		"an uncompressed part is never read past the size its manifest declares")
}

func TestAPartThatDecompressesWithoutLimitIsRefused(t *testing.T) {
	t.Parallel()

	stored, sum := compressed(t, 200<<20)
	require.Less(t, int64(len(stored))*archivestream.MaxExpansion+(64<<20), int64(200<<20),
		"the sample expands past the bound")
	r, err := archivestream.NewReader(bytes.NewReader(stored), sum, &archivestream.Options{
		Part: "tables/0001-x", Compress: true,
	})
	require.NoError(t, err)
	defer r.Close()
	_, err = io.Copy(io.Discard, r)
	require.ErrorIs(t, err, archivestream.ErrExpandsTooFar)
}
