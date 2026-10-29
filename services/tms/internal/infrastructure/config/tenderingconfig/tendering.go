package tenderingconfig

import (
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
)

type Config struct {
	PublicBaseURL string `mapstructure:"publicBaseUrl" validate:"omitempty,url"`
}

func (c *Config) GetPublicBaseURL() string {
	return strings.TrimSuffix(c.PublicBaseURL, "/")
}

func (c *Config) PublicLinkBaseURL(app *appconfig.Config) string {
	if base := c.GetPublicBaseURL(); base != "" {
		return base
	}
	return app.GetWebBaseURL()
}
