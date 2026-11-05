package editransport

import (
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/emoss08/trenova/shared/sftp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeInboundReader struct {
	files map[string]string
	reads []string
}

func (f *fakeInboundReader) ReadFile(remotePath string) (string, error) {
	f.reads = append(f.reads, remotePath)
	contents, ok := f.files[remotePath]
	if !ok {
		return "", errors.New("file is too large")
	}
	return contents, nil
}

func TestReadInboundBatch_OneBadFileLeavesTheRestReadable(t *testing.T) {
	t.Parallel()

	reader := &fakeInboundReader{files: map[string]string{"/in/a": "ISA", "/in/c": "ISA"}}

	files := readInboundBatch(reader, []sftp.RemoteFile{
		{Path: "/in/a", Name: "a"},
		{Path: "/in/b", Name: "b"},
		{Path: "/in/c", Name: "c"},
	})

	require.Len(t, files, 3)
	require.NoError(t, files[0].ReadError)
	require.Error(t, files[1].ReadError)
	assert.Contains(t, files[1].ReadError.Error(), "/in/b")
	assert.Empty(t, files[1].Contents)
	require.NoError(t, files[2].ReadError)
	assert.Equal(t, "ISA", files[2].Contents)
}

func TestReadInboundBatch_StopsAtTheFileLimit(t *testing.T) {
	t.Parallel()

	reader := &fakeInboundReader{files: map[string]string{}}
	entries := make([]sftp.RemoteFile, 0, maxInboundFilesPerPoll+5)
	for index := range maxInboundFilesPerPoll + 5 {
		path := "/in/" + strconv.Itoa(index)
		reader.files[path] = "ISA"
		entries = append(entries, sftp.RemoteFile{Path: path})
	}

	files := readInboundBatch(reader, entries)

	assert.Len(t, files, maxInboundFilesPerPoll)
	assert.Len(t, reader.reads, maxInboundFilesPerPoll)
}

func TestReadInboundBatch_StopsAtTheByteBudgetButAlwaysTakesOneFile(t *testing.T) {
	t.Parallel()

	first := strings.Repeat("x", 16)
	reader := &fakeInboundReader{files: map[string]string{"/in/a": first, "/in/b": "ISA"}}

	files := readInboundBatch(reader, []sftp.RemoteFile{
		{Path: "/in/a", Size: maxInboundBytesPerPoll + 1},
		{Path: "/in/b", Size: maxInboundBytesPerPoll},
	})

	require.Len(t, files, 1)
	assert.Equal(t, first, files[0].Contents)
}

func TestReadInboundBatch_StopsAfterTooManyFailures(t *testing.T) {
	t.Parallel()

	reader := &fakeInboundReader{files: map[string]string{}}
	entries := make([]sftp.RemoteFile, maxInboundFilesPerPoll+5)

	files := readInboundBatch(reader, entries)

	assert.Len(t, files, maxInboundFilesPerPoll)
}
