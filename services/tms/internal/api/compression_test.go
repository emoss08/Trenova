//revive:disable-next-line:var-naming
package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func compressionRouter(t *testing.T, paths ...string) *gin.Engine {
	t.Helper()

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(responseCompression())
	body := strings.Repeat("data: {\"delta\":\"token\"}\n\n", 200)
	for _, path := range paths {
		router.GET(path, func(c *gin.Context) {
			c.Header("Content-Type", "text/event-stream")
			c.String(http.StatusOK, body)
		})
	}

	return router
}

func TestResponseCompression_LeavesEventStreamsUncompressed(t *testing.T) {
	t.Parallel()

	router := compressionRouter(t,
		"/api/v1/assistant/turns/:turnID/stream/",
		"/api/v1/realtime/stream/",
	)

	for _, path := range []string{
		"/api/v1/assistant/turns/aturn_01JTURN0000000000000000/stream/",
		"/api/v1/realtime/stream/",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Accept-Encoding", "gzip")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code, path)
		assert.Empty(t, rec.Header().Get("Content-Encoding"), path)
	}
}

func TestResponseCompression_StillCompressesOrdinaryRoutes(t *testing.T) {
	t.Parallel()

	router := compressionRouter(t, "/api/v1/assistant/threads/")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/assistant/threads/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, "gzip", rec.Header().Get("Content-Encoding"))
}
