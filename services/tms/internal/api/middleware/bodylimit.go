package middleware

import (
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/gin-gonic/gin"
)

func NewBodyLimit(limit int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.ContentLength > limit {
			c.Abort()
			helpers.JSON(c, http.StatusRequestEntityTooLarge, gin.H{
				"error": "request body too large",
			})
			return
		}

		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
		c.Next()
	}
}
