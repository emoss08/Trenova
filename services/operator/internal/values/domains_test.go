package values_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/emoss08/trenova/operator/internal/instanceapi"
	"github.com/emoss08/trenova/operator/internal/values"
)

const customHost = "tms.acme-freight.com"

func TestACustomHostnameIsServedBesideTheFirstAddressAndLinksUseIt(t *testing.T) {
	t.Parallel()
	s := spec(plans[0], instanceapi.Active)
	s.CustomHostname = ptr(customHost)

	r, err := values.Render(&s, base(), operatorNS, values.Storage{})

	require.NoError(t, err)
	require.Equal(t, s.Hostname, r.Values["domain"], "the first address is always served")
	require.Equal(t, []any{customHost}, r.Values["aliases"])
	require.Equal(t, customHost, r.Values["primaryDomain"])
	require.Equal(t, values.StorageHost(s.Hostname), r.Values["storageDomain"],
		"files stay under the first address")

	out := renderStorageChart(t, r)
	ingress := out["trenova/templates/ingress.yaml"]
	require.Contains(t, ingress, `host: "`+s.Hostname+`"`)
	require.Contains(t, ingress, `host: "`+customHost+`"`)
	config := out["trenova/templates/configmap.yaml"]
	require.Contains(t, config, "webBaseUrl: https://"+customHost)
	require.Contains(t, config, "https://"+s.Hostname)
}

func TestAnOperatorThatServesACustomHostnameSaysItCan(t *testing.T) {
	t.Parallel()
	s := spec(plans[0], instanceapi.Active)
	s.CustomHostname = ptr(customHost)
	r, err := values.Render(&s, base(), operatorNS, values.Storage{})
	require.NoError(t, err)
	require.Contains(t, r.Values, "aliases", "this build serves a custom hostname")

	require.Contains(t, values.Capabilities(), instanceapi.CapabilityCustomHostname,
		"a capability the code carries out is declared, or the control plane never offers it")
	for _, c := range values.Capabilities() {
		require.True(t, c.Valid(), "%q is not in the contract", c)
	}
	first := values.Capabilities()
	first[0] = "changed"
	require.Equal(t, instanceapi.CapabilityCustomHostname, values.Capabilities()[0],
		"each caller gets its own list")
}

func TestWithoutACustomHostnameNeitherValueIsSet(t *testing.T) {
	t.Parallel()
	b := base()
	b["aliases"] = []any{"left.over.example.com"}
	b["primaryDomain"] = "left.over.example.com"
	for _, custom := range []*string{nil, ptr("")} {
		s := spec(plans[0], instanceapi.Active)
		s.CustomHostname = custom

		r, err := values.Render(&s, b, operatorNS, values.Storage{})

		require.NoError(t, err)
		require.NotContains(t, r.Values, "aliases")
		require.NotContains(t, r.Values, "primaryDomain")
		config := renderStorageChart(t, r)["trenova/templates/configmap.yaml"]
		require.Contains(t, config, "webBaseUrl: https://"+s.Hostname)
	}
}

func TestACustomHostnameIsRefusedLikeTheHostname(t *testing.T) {
	t.Parallel()
	first := spec(plans[0], instanceapi.Active).Hostname
	for name, custom := range map[string]string{
		"not a host":      "Not A Host",
		"upper case":      "TMS.Acme.com",
		"one label":       "acme",
		"a wildcard":      "*.acme.com",
		"an address":      "10.0.0.1",
		"a port":          "tms.acme.com:443",
		"too long label":  strings.Repeat("a", 64) + ".acme.com",
		"the first":       first,
		"the files host":  values.StorageHost(first),
		"a trailing dot":  "tms.acme.com.",
		"a leading label": "-tms.acme.com",
	} {
		s := spec(plans[0], instanceapi.Active)
		s.CustomHostname = ptr(custom)
		_, err := values.Render(&s, base(), operatorNS, values.Storage{})
		require.ErrorIs(t, err, values.ErrInvalidSpec, name)
	}
}
