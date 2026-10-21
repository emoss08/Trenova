package fileutils_test

import (
	"bytes"
	"mime/multipart"
	"net/http/httptest"
	"testing"

	"github.com/emoss08/trenova/shared/fileutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func multipartHeader(t *testing.T, content []byte) *multipart.FileHeader {
	t.Helper()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "logo.png")
	require.NoError(t, err)
	_, err = part.Write(content)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	req := httptest.NewRequestWithContext(t.Context(), "POST", "/", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	require.NoError(t, req.ParseMultipartForm(1<<20))
	return req.MultipartForm.File["file"][0]
}

func TestReadMultipartReturnsTheWholeFile(t *testing.T) {
	t.Parallel()

	content := []byte("\x89PNG\r\n\x1a\nimage")
	got, err := fileutils.ReadMultipart(multipartHeader(t, content))

	require.NoError(t, err)
	assert.Equal(t, content, got)
}

func TestReadMultipartRefusesMoreThanTheDeclaredSize(t *testing.T) {
	t.Parallel()

	header := multipartHeader(t, []byte("0123456789"))
	header.Size = 4

	_, err := fileutils.ReadMultipart(header)

	require.ErrorIs(t, err, fileutils.ErrMultipartSizeMismatch)
}

func TestReadMultipartRefusesAMissingFile(t *testing.T) {
	t.Parallel()

	_, err := fileutils.ReadMultipart(nil)

	require.Error(t, err)
}
