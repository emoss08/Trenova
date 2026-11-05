package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
)

const redactedTokenPlaceholder = "REDACTED"

var redactedPathParams = [...]string{"token", "webhookToken", "mailboxToken", "as2InboundToken"}

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

		for _, name := range redactedPathParams {
			v := c.Params.ByName(name)
			if v == "" {
				continue
			}
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
