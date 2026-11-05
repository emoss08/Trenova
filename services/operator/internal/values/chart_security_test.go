package values_test

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/engine"

	"github.com/emoss08/trenova/operator/internal/instanceapi"
	"github.com/emoss08/trenova/operator/internal/values"
)

var certificateOnly = regexp.MustCompile(
	`secretName: trenova-postgres-ca\s+items:\s+- key: ca\.crt\s+path: ca\.crt\n`,
)

func renderChart(t *testing.T, b map[string]any) (*chart.Chart, map[string]any, map[string]string) {
	t.Helper()
	ch, err := loader.Load(chartDir)
	require.NoError(t, err)
	r, err := values.Render(ptr(spec(plans[0], instanceapi.Active, "docling")), b, operatorNS)
	require.NoError(t, err)
	vals, err := chartutil.ToRenderValues(ch, r.Values, chartutil.ReleaseOptions{
		Name: "trenova", Namespace: "trenova", Revision: 1, IsInstall: true,
	}, chartutil.DefaultCapabilities)
	require.NoError(t, err)
	out, err := engine.Render(ch, vals)
	require.NoError(t, err)
	return ch, r.Values, out
}

func template(t *testing.T, out map[string]string, suffix string) string {
	t.Helper()
	for name, body := range out {
		if strings.HasSuffix(name, suffix) {
			return body
		}
	}
	require.Failf(t, "template not rendered", "%s", suffix)
	return ""
}

func TestChartHasNoModeThatSignsInAsTheOwner(t *testing.T) {
	t.Parallel()
	b := base()
	b["postgres"] = map[string]any{"rlsMode": "off"}
	ch, err := loader.Load(chartDir)
	require.NoError(t, err)
	r, err := values.Render(ptr(spec(plans[0], instanceapi.Active, "docling")), b, operatorNS)
	require.NoError(t, err)
	require.ErrorContains(t, chartutil.ValidateAgainstSchema(ch, r.Values), "rlsMode")
}

func TestApplicationSignsInAsItsOwnRoleInEveryMode(t *testing.T) {
	t.Parallel()
	for _, mode := range []string{"enforce", "observe"} {
		b := base()
		b["postgres"] = map[string]any{"rlsMode": mode}
		ch, vals, out := renderChart(t, b)
		require.NoError(t, chartutil.ValidateAgainstSchema(ch, vals), mode)

		config := template(t, out, "templates/configmap.yaml")
		require.Contains(t, config, "user: trenova_app", mode)
		require.Contains(t, config, "user: trenova_system", mode)
		require.Contains(t, template(t, out, "templates/release-job.yaml"), "name: rls-roles", mode)
	}
}

func TestClamAVRunsUnprivileged(t *testing.T) {
	t.Parallel()
	_, _, out := renderChart(t, base())
	services := template(t, out, "templates/services.yaml")
	clamav := services[strings.Index(services, "name: clamav\n          image:")-400:]
	clamav = clamav[:strings.Index(clamav, "volumeClaimTemplates")]

	require.Contains(t, clamav, "runAsNonRoot: true")
	require.Contains(t, clamav, "runAsUser: 100")
	require.Contains(t, clamav, "command: [/init-unprivileged]")
	require.Contains(t, clamav, "allowPrivilegeEscalation: false")
	require.Contains(t, clamav, "drop: [ALL]")
}

func TestEveryPostgresClientVerifiesTheServer(t *testing.T) {
	t.Parallel()
	_, _, out := renderChart(t, base())

	for name, body := range out {
		require.NotContains(t, body, "sslmode=require", name)
		require.NotRegexp(t, `name: PGSSLMODE\s+value: require`, body, name)
	}

	require.Contains(t, template(t, out, "templates/configmap.yaml"), "sslMode: verify-full")
	require.Contains(t, template(t, out, "templates/configmap.yaml"), "sslRootCert: /certs/postgres/ca.crt")

	services := template(t, out, "templates/services.yaml")
	require.Contains(t, services, "sslmode=verify-full&sslrootcert=/certs/postgres/ca.crt")
	require.Regexp(t, `name: SQL_HOST_VERIFICATION\s+value: "true"`, services)
	require.Regexp(t, `name: POSTGRES_TLS_DISABLE_HOST_VERIFICATION\s+value: "false"`, services)
	require.Regexp(t, `name: SQL_HOST_NAME\s+value: trenova-postgres-rw`, services)
}

func TestEveryPodThatMountsThePostgresCAHasItsVolume(t *testing.T) {
	t.Parallel()
	_, _, out := renderChart(t, base())

	for _, suffix := range []string{
		"templates/tms.yaml", "templates/services.yaml", "templates/release-job.yaml",
	} {
		for _, doc := range strings.Split(template(t, out, suffix), "\n---") {
			if !strings.Contains(doc, "mountPath: /certs/postgres") {
				continue
			}
			require.Contains(t, doc, "secretName: trenova-postgres-ca", suffix)
		}
	}
}

func TestPodsMountThePostgresCertificateNotTheCAKey(t *testing.T) {
	t.Parallel()
	_, _, out := renderChart(t, base())

	for name, body := range out {
		require.NotContains(t, body, "key: ca.key", name)
		require.Len(
			t,
			certificateOnly.FindAllString(body, -1),
			strings.Count(body, "secretName: trenova-postgres-ca"),
			name,
		)
	}
}
