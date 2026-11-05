package middleware

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestMultipartCleanupRemovesSpilledFilesAfterTheRequestWasReplaced(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dir := t.TempDir()
	t.Setenv("TMPDIR", dir)

	spilled := 0
	router := gin.New()
	router.MaxMultipartMemory = 1
	router.Use(
		NewMultipartCleanupMiddleware(zap.NewNop()),
		requestid.New(),
		NewRequestMetaMiddleware(),
	)
	router.POST("/", func(c *gin.Context) {
		_, err := c.FormFile("file")
		assert.NoError(t, err)
		entries, err := os.ReadDir(dir)
		assert.NoError(t, err)
		spilled = len(entries)
		c.Status(http.StatusBadRequest)
	})

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "upload.bin")
	require.NoError(t, err)
	_, err = part.Write(bytes.Repeat([]byte("x"), 4096))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	router.ServeHTTP(httptest.NewRecorder(), req)

	require.Positive(t, spilled, "the upload spilled to disk while the handler ran")
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	assert.Empty(t, entries)
}
