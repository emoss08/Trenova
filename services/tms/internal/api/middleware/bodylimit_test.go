package middleware

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func publicBodyRouter(read *int64) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(NewPublicBodyLimit())
	router.POST("/public", func(c *gin.Context) {
		size, err := io.Copy(io.Discard, c.Request.Body)
		if err != nil {
			c.Status(http.StatusRequestEntityTooLarge)
			return
		}
		*read = size
		c.Status(http.StatusNoContent)
	})

	return router
}

func TestPublicBodyLimitRefusesADeclaredOversizedBodyBeforeReadingIt(t *testing.T) {
	t.Parallel()

	var read int64
	router := publicBodyRouter(&read)
	request := httptest.NewRequest(http.MethodPost, "/public", bytes.NewReader([]byte("x")))
	request.ContentLength = maxPublicBodyBytes + 1

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusRequestEntityTooLarge, recorder.Code)
	assert.Zero(t, read)
}

func TestPublicBodyLimitStopsAnUndeclaredBodyAtTheLimit(t *testing.T) {
	t.Parallel()

	var read int64
	router := publicBodyRouter(&read)
	request := httptest.NewRequest(
		http.MethodPost,
		"/public",
		io.LimitReader(zeroReader{}, maxPublicBodyBytes+1),
	)
	request.ContentLength = -1

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusRequestEntityTooLarge, recorder.Code)
	assert.Zero(t, read)
}

func TestPublicBodyLimitAcceptsTheLargestDriverUpload(t *testing.T) {
	t.Parallel()

	var read int64
	router := publicBodyRouter(&read)
	request := httptest.NewRequest(
		http.MethodPost,
		"/public",
		io.LimitReader(zeroReader{}, maxPublicBodyBytes),
	)
	request.ContentLength = maxPublicBodyBytes

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusNoContent, recorder.Code)
	assert.Equal(t, int64(maxPublicBodyBytes), read)
}

type zeroReader struct{}

func (zeroReader) Read(p []byte) (int, error) {
	clear(p)
	return len(p), nil
}
