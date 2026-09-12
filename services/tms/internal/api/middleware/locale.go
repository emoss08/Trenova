package middleware

import (
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/gin-gonic/gin"
)

type LocaleMiddleware struct{}

func NewLocaleMiddleware() *LocaleMiddleware {
	return &LocaleMiddleware{}
}

func (m *LocaleMiddleware) Resolve() gin.HandlerFunc {
	return func(c *gin.Context) {
		locale := resolveLocale(c)

		authctx.SetLocale(c, locale.String())
		c.Request = c.Request.WithContext(i18n.WithLocale(c.Request.Context(), locale))

		c.Next()
	}
}

func resolveLocale(c *gin.Context) i18n.Locale {
	if stored, ok := authctx.GetLocale(c); ok {
		if locale, known := i18n.Parse(stored); known {
			return locale
		}
	}

	return i18n.ParseAcceptLanguage(c.GetHeader("Accept-Language"))
}
