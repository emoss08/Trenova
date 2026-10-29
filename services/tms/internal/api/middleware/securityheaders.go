package middleware

import (
	"net/http"

	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
	"github.com/gin-gonic/gin"
)

const (
	apiContentSecurityPolicy = "default-src 'none'; frame-ancestors 'none'; base-uri 'none'"
	hstsHeaderValue          = "max-age=31536000; includeSubDomains"
	permissionsPolicyValue   = "camera=(), microphone=(self), geolocation=()"
)

func NewSecurityHeadersMiddleware(cfg *appconfig.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		ApplySecurityHeaders(c.Writer.Header(), cfg)
		c.Next()
	}
}

func ApplySecurityHeaders(header http.Header, cfg *appconfig.Config) {
	header.Set("X-Content-Type-Options", "nosniff")
	header.Set("X-Frame-Options", "DENY")
	header.Set("Referrer-Policy", "no-referrer")
	header.Set("Permissions-Policy", permissionsPolicyValue)
	header.Set("Content-Security-Policy", apiContentSecurityPolicy)

	if cfg.IsProduction() || cfg.IsStaging() {
		header.Set("Strict-Transport-Security", hstsHeaderValue)
	}
}
