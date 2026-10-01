package middleware

import (
	"github.com/emoss08/trenova/pkg/requestmeta"
	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
)

func NewRequestMetaMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		meta := requestmeta.New(requestid.Get(c), c.ClientIP(), c.Request.UserAgent())

		c.Set(requestmeta.GinContextKey, meta)
		c.Request = c.Request.WithContext(requestmeta.With(c.Request.Context(), meta))

		c.Next()
	}
}
