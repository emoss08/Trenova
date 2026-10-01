package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/emoss08/trenova/pkg/requestmeta"
	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequestMetaMiddlewareBindsTheRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var fromRequest, fromGin requestmeta.Meta
	var okRequest, okGin bool

	router := gin.New()
	router.Use(requestid.New(), NewRequestMetaMiddleware())
	router.GET("/", func(c *gin.Context) {
		fromRequest, okRequest = requestmeta.From(c.Request.Context())
		fromGin, okGin = requestmeta.From(c)
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	req.Header.Set("User-Agent", "audit-test")
	req.Header.Set("X-Request-ID", "req-123")
	req.RemoteAddr = "203.0.113.7:4321"
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.True(t, okRequest)
	require.True(t, okGin)
	assert.Equal(t, fromRequest, fromGin)
	assert.Equal(t, "req-123", fromRequest.RequestID)
	assert.Equal(t, "203.0.113.7", fromRequest.ClientIP)
	assert.Equal(t, "audit-test", fromRequest.UserAgent)
}
