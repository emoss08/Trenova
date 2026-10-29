package portalconfig

import (
	"strings"
)

type Config struct {
	BaseURL string `mapstructure:"baseUrl" validate:"omitempty,url"`
}

func (c *Config) GetBaseURL() string {
	return strings.TrimSuffix(c.BaseURL, "/")
}
