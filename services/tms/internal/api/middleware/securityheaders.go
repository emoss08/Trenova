package middleware

import (
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/gin-gonic/gin"
)

const (
	apiContentSecurityPolicy = "default-src 'none'; frame-ancestors 'none'; base-uri 'none'"
	hstsHeaderValue         = "max-age=31536000; includeSubDomains"
	permissionsPolicyValue  = "camera=(), microphone=(), geolocation=()"
)

func NewSecurityHeadersMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "no-referrer")
		c.Header("Permissions-Policy", permissionsPolicyValue)
		c.Header("Content-Security-Policy", apiContentSecurityPolicy)

		if cfg.App.IsProduction() || cfg.App.IsStaging() {
			c.Header("Strict-Transport-Security", hstsHeaderValue)
		}

		c.Next()
	}
}
