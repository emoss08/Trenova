package helpers_test

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type countingReader struct {
	r    io.Reader
	read int64
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.read += int64(n)
	return n, err
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) {
	return 0, errors.New("boom")
}

func newBodyContext(body io.Reader, contentLength int64) *gin.Context {
	gin.SetMode(gin.TestMode)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/", body)
	c.Request.ContentLength = contentLength

	return c
}

func TestReadBoundedRequestBody(t *testing.T) {
	t.Parallel()

	t.Run("reads a body under the limit", func(t *testing.T) {
		t.Parallel()
		payload := []byte(`{"ok":true}`)
		c := newBodyContext(bytes.NewReader(payload), int64(len(payload)))

		body, err := helpers.ReadBoundedRequestBody(c, 1024)

		require.NoError(t, err)
		assert.Equal(t, payload, body)
	})

	t.Run("reads a body exactly at the limit", func(t *testing.T) {
		t.Parallel()
		payload := bytes.Repeat([]byte("a"), 64)
		c := newBodyContext(bytes.NewReader(payload), int64(len(payload)))

		body, err := helpers.ReadBoundedRequestBody(c, 64)

		require.NoError(t, err)
		assert.Equal(t, payload, body)
	})

	t.Run("rejects a declared content length over the limit without reading", func(t *testing.T) {
		t.Parallel()
		payload := bytes.Repeat([]byte("a"), 128)
		source := &countingReader{r: bytes.NewReader(payload)}
		c := newBodyContext(source, int64(len(payload)))

		body, err := helpers.ReadBoundedRequestBody(c, 64)

		require.Error(t, err)
		assert.Nil(t, body)
		assert.True(t, helpers.IsRequestTooLargeError(err))
		assert.Zero(t, source.read)
	})

	t.Run("rejects an unknown content length over the limit", func(t *testing.T) {
		t.Parallel()
		source := &countingReader{r: bytes.NewReader(bytes.Repeat([]byte("a"), 1<<20))}
		c := newBodyContext(source, -1)

		body, err := helpers.ReadBoundedRequestBody(c, 64)

		require.Error(t, err)
		assert.Nil(t, body)
		assert.True(t, helpers.IsRequestTooLargeError(err))
		assert.LessOrEqual(t, source.read, int64(65))
	})

	t.Run("does not preallocate the limit for an unknown content length", func(t *testing.T) {
		t.Parallel()
		payload := []byte("small")
		c := newBodyContext(bytes.NewReader(payload), -1)

		body, err := helpers.ReadBoundedRequestBody(c, 1<<20)

		require.NoError(t, err)
		assert.Equal(t, payload, body)
		assert.Less(t, cap(body), 1<<20)
	})

	t.Run("propagates a transport read failure unchanged", func(t *testing.T) {
		t.Parallel()
		c := newBodyContext(failingReader{}, -1)

		body, err := helpers.ReadBoundedRequestBody(c, 64)

		require.Error(t, err)
		assert.Nil(t, body)
		assert.False(t, helpers.IsRequestTooLargeError(err))
	})
}

func TestRequestTooLargeError(t *testing.T) {
	t.Parallel()

	t.Run("names the limit it enforced", func(t *testing.T) {
		t.Parallel()
		err := helpers.NewRequestTooLargeError(1024)

		assert.Equal(t, "request body exceeds the 1024 byte limit", err.Error())
	})

	t.Run("is recognized through a wrapping error", func(t *testing.T) {
		t.Parallel()
		wrapped := errors.Join(errors.New("context"), helpers.NewRequestTooLargeError(1024))

		assert.True(t, helpers.IsRequestTooLargeError(wrapped))
	})

	t.Run("recognizes the stdlib max bytes error", func(t *testing.T) {
		t.Parallel()
		assert.True(t, helpers.IsRequestTooLargeError(&http.MaxBytesError{Limit: 1024}))
	})

	t.Run("does not match unrelated errors", func(t *testing.T) {
		t.Parallel()
		assert.False(t, helpers.IsRequestTooLargeError(errors.New("boom")))
		assert.False(t, helpers.IsRequestTooLargeError(nil))
	})
}

func TestClassifyRequestTooLarge(t *testing.T) {
	t.Parallel()

	classifier := helpers.NewDefaultClassifier()

	t.Run("maps the bounded reader error to 413", func(t *testing.T) {
		t.Parallel()
		problemType := classifier.Classify(helpers.NewRequestTooLargeError(1024))

		assert.Equal(t, helpers.ProblemTypeRequestTooLarge, problemType)
		assert.Equal(
			t,
			http.StatusRequestEntityTooLarge,
			helpers.ProblemTypeRequestTooLarge.Info().StatusCode,
		)
	})

	t.Run("maps the stdlib max bytes error to 413", func(t *testing.T) {
		t.Parallel()
		problemType := classifier.Classify(&http.MaxBytesError{Limit: 1024})

		assert.Equal(t, helpers.ProblemTypeRequestTooLarge, problemType)
	})
}
