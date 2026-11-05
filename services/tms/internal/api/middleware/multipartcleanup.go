package middleware

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func NewMultipartCleanupMiddleware(l *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			form := c.Request.MultipartForm
			if form == nil {
				return
			}
			if err := form.RemoveAll(); err != nil {
				l.Warn("failed to remove multipart temp files", zap.Error(err))
			}
		}()

		c.Next()
	}
}
