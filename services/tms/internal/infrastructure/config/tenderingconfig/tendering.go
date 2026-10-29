package tenderingconfig

import (
	"strings"
)

type Config struct {
	PublicBaseURL string `mapstructure:"publicBaseUrl" validate:"omitempty,url"`
}

func (c *Config) GetPublicBaseURL() string {
	return strings.TrimSuffix(c.PublicBaseURL, "/")
}
