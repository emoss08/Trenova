package extensionwebwatchservice

import (
	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
)

// webBaseOf is the public address Trenova is served from, where a vendor
// reports back.
func webBaseOf(cfg *appconfig.Config) string {
	if cfg == nil {
		return ""
	}

	return cfg.GetWebBaseURL()
}
