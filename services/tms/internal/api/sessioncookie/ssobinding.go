package sessioncookie

import (
	"crypto/rand"
	"net/http"

	"github.com/emoss08/trenova/internal/infrastructure/config/securityconfig"
	"github.com/gin-gonic/gin"
)

const (
	ssoBindingName       = "trenova-sso-binding"
	ssoBindingSecureName = "__Host-" + ssoBindingName
	ssoBindingMaxAge     = 600
)

func SetSSOBinding(c *gin.Context, cfg *securityconfig.SessionConfig) string {
	token := rand.Text()
	writeSSOBinding(c, cfg, token, ssoBindingMaxAge)

	return token
}

func SSOBinding(c *gin.Context, cfg *securityconfig.SessionConfig) string {
	token, err := c.Cookie(ssoBindingCookie(cfg))
	if err != nil {
		return ""
	}

	return token
}

func ClearSSOBinding(c *gin.Context, cfg *securityconfig.SessionConfig) {
	writeSSOBinding(c, cfg, "", -1)
}

func writeSSOBinding(
	c *gin.Context,
	cfg *securityconfig.SessionConfig,
	token string,
	maxAge int,
) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     ssoBindingCookie(cfg),
		Value:    token,
		Path:     "/",
		MaxAge:   maxAge,
		Secure:   cfg.Secure,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func ssoBindingCookie(cfg *securityconfig.SessionConfig) string {
	if cfg.Secure {
		return ssoBindingSecureName
	}

	return ssoBindingName
}
