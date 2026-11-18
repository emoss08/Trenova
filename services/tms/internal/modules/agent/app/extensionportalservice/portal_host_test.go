package extensionportalservice

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/serverconfig"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentextensionports"
)

func TestEveryAddressTrenovaIsServedOnIsTrenova(t *testing.T) {
	t.Parallel()

	server := &serverconfig.Config{}
	server.CORS.AllowedOrigins = []string{
		"https://abc.trenova.cloud",
		"https://TMS.acme.com",
		"*",
	}
	hosts := appHostsOf(&appconfig.Config{WebBaseURL: "https://tms.acme.com"}, server)
	assert.Equal(t, []string{"tms.acme.com", "abc.trenova.cloud"}, hosts,
		"the address links use, then every origin the API admits, each once")
	assert.Empty(t, appHostsOf(nil, nil))

	f := newFixture(t)
	f.svc.appHosts = hosts
	for _, domains := range [][]string{{"abc.trenova.cloud"}, {"trenova.cloud"}, {"tms.acme.com"}} {
		_, err := f.svc.CreateSite(t.Context(), f.siteRequest(
			func(req *agentextensionports.SavePortalSiteRequest) {
				req.Domains = domains
				req.StartURL = "https://" + domains[0] + "/login"
			},
		))
		assert.Contains(t, fieldsOf(t, err), "domains",
			"%v covers an address Trenova itself is served on", domains)
	}
}
