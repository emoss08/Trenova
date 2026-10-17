package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func NewVaryOriginMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		AddVary(c.Writer.Header(), "Origin")
		c.Next()
	}
}

func AddVary(header http.Header, field string) {
	for _, value := range header.Values("Vary") {
		for token := range strings.SplitSeq(value, ",") {
			token = strings.TrimSpace(token)
			if token == "*" || strings.EqualFold(token, field) {
				return
			}
		}
	}
	header.Add("Vary", field)
}
