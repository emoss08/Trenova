package extensionportalservice

import (
	"net/url"
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
)

// appHostOf is the host Trenova's own web app is served from, which no portal
// site may cover.
func appHostOf(cfg *appconfig.Config) string {
	if cfg == nil {
		return ""
	}
	parsed, err := url.Parse(cfg.GetWebBaseURL())
	if err != nil {
		return ""
	}

	return strings.ToLower(parsed.Hostname())
}
