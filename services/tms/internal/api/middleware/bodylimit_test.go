package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestBodyLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var readErr error
	router := gin.New()
	router.Use(NewBodyLimit(8))
	router.POST("/", func(c *gin.Context) {
		_, readErr = io.ReadAll(c.Request.Body)
		c.Status(http.StatusNoContent)
	})

	declared := httptest.NewRecorder()
	router.ServeHTTP(declared, httptest.NewRequestWithContext(
		t.Context(), http.MethodPost, "/", strings.NewReader(strings.Repeat("x", 16)),
	))
	assert.Equal(t, http.StatusRequestEntityTooLarge, declared.Code)

	undeclared := httptest.NewRequestWithContext(
		t.Context(), http.MethodPost, "/", strings.NewReader(strings.Repeat("x", 16)),
	)
	undeclared.ContentLength = -1
	router.ServeHTTP(httptest.NewRecorder(), undeclared)
	assert.Error(t, readErr, "a body that hides its length is cut off while it is read")
}
