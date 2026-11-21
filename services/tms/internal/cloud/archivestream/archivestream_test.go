package archivestream_test

import (
	"bytes"
	"io"
	"testing"

	"github.com/emoss08/trenova/internal/cloud/archivestream"
	"github.com/stretchr/testify/require"
)

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
