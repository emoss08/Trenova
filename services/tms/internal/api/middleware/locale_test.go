package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestLocaleMiddleware_ResolvesFromAcceptLanguage(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name   string
		header string
		want   i18n.Locale
	}{
		{"no header falls back to english", "", i18n.EN},
		{"simple tag", "es", i18n.ES},
		{"quality ordering wins", "en;q=0.2,zh-TW;q=0.9", i18n.ZhTW},
		{"script subtag maps to a written form", "zh-Hans", i18n.ZhCN},
		{"unsupported language falls back", "de-DE", i18n.EN},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var seen i18n.Locale
			router := gin.New()
			router.Use(NewLocaleMiddleware().Resolve())
			router.GET("/", func(c *gin.Context) {
				seen = i18n.FromContext(c.Request.Context())
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.header != "" {
				req.Header.Set("Accept-Language", tt.header)
			}
			router.ServeHTTP(httptest.NewRecorder(), req)

			assert.Equal(t, tt.want, seen)
		})
	}
}

func TestLocaleMiddleware_StoredPreferenceBeatsHeader(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	var seen i18n.Locale
	router := gin.New()
	router.Use(func(c *gin.Context) {
		authctx.SetLocale(c, string(i18n.ZhTW))
		c.Next()
	})
	router.Use(NewLocaleMiddleware().Resolve())
	router.GET("/", func(c *gin.Context) {
		seen = i18n.FromContext(c.Request.Context())
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Language", "es")
	router.ServeHTTP(httptest.NewRecorder(), req)

	assert.Equal(t, i18n.ZhTW, seen,
		"a signed-in user's saved language must win over the browser's header")
}

func TestLocaleMiddleware_IgnoresUnsupportedStoredPreference(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	var seen i18n.Locale
	router := gin.New()
	router.Use(func(c *gin.Context) {
		authctx.SetLocale(c, "fr")
		c.Next()
	})
	router.Use(NewLocaleMiddleware().Resolve())
	router.GET("/", func(c *gin.Context) {
		seen = i18n.FromContext(c.Request.Context())
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Language", "es")
	router.ServeHTTP(httptest.NewRecorder(), req)

	assert.Equal(t, i18n.ES, seen,
		"a stored language we no longer ship must fall through to the header, not to English")
}
