package sessioncookie

import (
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/gin-gonic/gin"
)

func Set(c *gin.Context, cfg *config.SessionConfig, token string, expiresAt int64) {
	maxAge := max(0, int(expiresAt-timeutils.NowUnix()))

	c.SetSameSite(cfg.GetSameSite())
	c.SetCookie(cfg.Name, token, maxAge, cfg.Path, cfg.Domain, cfg.Secure, cfg.HTTPOnly)
}

func Clear(c *gin.Context, cfg *config.SessionConfig) {
	c.SetSameSite(cfg.GetSameSite())
	c.SetCookie(cfg.Name, "", -1, cfg.Path, cfg.Domain, cfg.Secure, cfg.HTTPOnly)
}
