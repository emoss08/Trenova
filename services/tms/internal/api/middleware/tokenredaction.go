package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
)

const redactedTokenPlaceholder = "REDACTED"

func NewTokenRedactionMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		_ = c.Query("token")

		if c.Request.URL.RawQuery != "" {
			values := c.Request.URL.Query()
			if values.Has("token") {
				tokens := values["token"]
				for i := range tokens {
					tokens[i] = redactedTokenPlaceholder
				}
				c.Request.URL.RawQuery = values.Encode()
			}
		}

		if v := c.Params.ByName("token"); v != "" {
			c.Request.URL.Path = strings.ReplaceAll(
				c.Request.URL.Path,
				v,
				redactedTokenPlaceholder,
			)
			c.Request.URL.RawPath = ""
		}

		c.Next()
	}
}
