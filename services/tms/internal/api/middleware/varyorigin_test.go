package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func newVaryOriginTestRouter() *gin.Engine {
	router := gin.New()
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173"},
		AllowMethods:     []string{http.MethodGet},
		AllowCredentials: true,
	}))
	router.Use(NewVaryOriginMiddleware())
	router.GET("/page", func(c *gin.Context) {
		c.Header("Cache-Control", "private, max-age=86400")
		c.Status(http.StatusOK)
	})

	return router
}

func TestVaryOriginMiddleware_VariesAResponseFetchedWithoutAnOrigin(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	newVaryOriginTestRouter().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/page", nil))

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(
		t,
		[]string{"Origin"},
		w.Header().Values("Vary"),
		"a cached copy fetched without an origin must not answer a request that has one",
	)
}

func TestVaryOriginMiddleware_NamesOriginOnceForACrossOriginRequest(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/page", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	newVaryOriginTestRouter().ServeHTTP(w, req)

	assert.Equal(t, "http://localhost:5173", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, []string{"Origin"}, w.Header().Values("Vary"))
}

func TestAddVary_KeepsWhatTheResponseAlreadyVariesOn(t *testing.T) {
	t.Parallel()

	header := http.Header{}
	header.Set("Vary", "Accept-Encoding")
	AddVary(header, "Origin")
	AddVary(header, "origin")
	assert.Equal(t, []string{"Accept-Encoding", "Origin"}, header.Values("Vary"))

	star := http.Header{}
	star.Set("Vary", "*")
	AddVary(star, "Origin")
	assert.Equal(t, []string{"*"}, star.Values("Vary"))
}
