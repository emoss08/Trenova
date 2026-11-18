package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// SupportEntryPath is where the control plane's admin posts a Trenova staff member's support
// assertion. The post comes from the control plane's origin by design, so the browser
// provenance guard and the cross-origin check let it through; it carries no cookie they
// protect, and the assertion in its body is the whole of its authority (managed-mode.md,
// "Support access").
const SupportEntryPath = "/api/v1/support/enter"

func isSupportEntry(path string) bool {
	return path == SupportEntryPath
}

func ExceptSupportEntry(check gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == http.MethodPost && isSupportEntry(c.Request.URL.Path) {
			c.Next()
			return
		}
		check(c)
	}
}
