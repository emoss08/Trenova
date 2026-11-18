package extensionportalservice

import (
	"net/url"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/serverconfig"
	"github.com/emoss08/trenova/internal/modules/agent/domain/extensionportal"
)

// appHostsOf is every host Trenova's own web app is served from (the address
// links use and every origin the API admits), none of which a portal site may
// cover.
func appHostsOf(app *appconfig.Config, server *serverconfig.Config) []string {
	var origins []string
	if app != nil {
		origins = append(origins, app.GetWebBaseURL())
	}
	if server != nil {
		origins = append(origins, server.CORS.AllowedOrigins...)
	}
	hosts := make([]string, 0, len(origins))
	for _, origin := range origins {
		parsed, err := url.Parse(strings.TrimSpace(origin))
		if err != nil {
			continue
		}
		host := strings.ToLower(parsed.Hostname())
		if host == "" || slices.Contains(hosts, host) {
			continue
		}
		hosts = append(hosts, host)
	}

	return hosts
}

func (s *Service) coversTrenova(domains []string) bool {
	for _, host := range s.appHosts {
		if extensionportal.HostAllowed(host, domains) {
			return true
		}
	}

	return false
}
