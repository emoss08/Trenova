package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func newOriginCheckedRouter() *gin.Engine {
	router := gin.New()
	router.Use(ExceptSupportEntry(cors.New(cors.Config{
		AllowOrigins: []string{"https://acme.trenova.app"},
		AllowMethods: []string{http.MethodGet, http.MethodPost},
	})))
	reached := func(c *gin.Context) { c.String(http.StatusOK, "reached") }
	router.POST(SupportEntryPath, reached)
	router.GET(SupportEntryPath, reached)
	router.POST("/api/v1/other", reached)
	return router
}

func postFrom(router *gin.Engine, method, path, origin string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, strings.NewReader("assertion=abc"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	router.ServeHTTP(w, req)
	return w
}

func TestSupportEntryPostReachesItsHandlerFromAnotherOrigin(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)
	router := newOriginCheckedRouter()

	for _, origin := range []string{"https://admin.trenova.app", "null", ""} {
		w := postFrom(router, http.MethodPost, SupportEntryPath, origin)
		assert.Equal(t, http.StatusOK, w.Code, "origin %q", origin)
		assert.Equal(t, "reached", w.Body.String(), "origin %q", origin)
	}
}

func TestOnlyTheSupportEntryPostSkipsTheOriginCheck(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)
	router := newOriginCheckedRouter()

	other := postFrom(router, http.MethodPost, "/api/v1/other", "https://admin.trenova.app")
	assert.Equal(t, http.StatusForbidden, other.Code, "another path is still refused")

	get := postFrom(router, http.MethodGet, SupportEntryPath, "https://admin.trenova.app")
	assert.Equal(t, http.StatusForbidden, get.Code, "only the post is let through")

	allowed := postFrom(router, http.MethodPost, "/api/v1/other", "https://acme.trenova.app")
	assert.Equal(t, http.StatusOK, allowed.Code, "the instance's own origin still passes")
}
