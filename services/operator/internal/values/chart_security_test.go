package values_test

import (
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/engine"
	"sigs.k8s.io/yaml"

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
	r, err := values.Render(ptr(spec(plans[0], instanceapi.Active, "docling")), b)
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
	r, err := values.Render(ptr(spec(plans[0], instanceapi.Active)), b)
	require.NoError(t, err)
	_, err = chartutil.ToRenderValues(ch, r.Values, chartutil.ReleaseOptions{
		Name: "trenova", Namespace: "trenova", Revision: 1, IsInstall: true,
	}, chartutil.DefaultCapabilities)
	require.ErrorContains(t, err, "rlsMode")
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

type renderedSecret struct {
	Metadata struct {
		Name string `json:"name"`
	} `json:"metadata"`
	Data map[string]string `json:"data"`
}

func renderedSecrets(t *testing.T, out map[string]string) map[string]renderedSecret {
	t.Helper()
	secrets := map[string]renderedSecret{}
	for _, doc := range strings.Split(template(t, out, "templates/secrets.yaml"), "\n---") {
		var s renderedSecret
		require.NoError(t, yaml.Unmarshal([]byte(doc), &s))
		if s.Metadata.Name != "" {
			secrets[s.Metadata.Name] = s
		}
	}
	return secrets
}

func certificate(t *testing.T, s renderedSecret, key string) *x509.Certificate {
	t.Helper()
	raw, err := base64.StdEncoding.DecodeString(s.Data[key])
	require.NoError(t, err)
	block, _ := pem.Decode(raw)
	require.NotNil(t, block, key)
	cert, err := x509.ParseCertificate(block.Bytes)
	require.NoError(t, err)
	return cert
}

func TestTemporalCertificatesComeFromOneCAAndServeItsName(t *testing.T) {
	t.Parallel()
	_, _, out := renderChart(t, base())
	secrets := renderedSecrets(t, out)

	server, ok := secrets["trenova-temporal-server-tls"]
	require.True(t, ok)
	client, ok := secrets["trenova-temporal-client-tls"]
	require.True(t, ok)
	require.Equal(t, server.Data["ca.crt"], client.Data["ca.crt"])

	pool := x509.NewCertPool()
	pool.AddCert(certificate(t, server, "ca.crt"))

	serverCert := certificate(t, server, "tls.crt")
	require.Contains(t, serverCert.DNSNames, "trenova-temporal")
	_, err := serverCert.Verify(x509.VerifyOptions{
		DNSName:   "trenova-temporal",
		Roots:     pool,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	})
	require.NoError(t, err)
	_, err = serverCert.Verify(x509.VerifyOptions{
		Roots:     pool,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	})
	require.NoError(t, err, "Temporal's services present the server certificate to each other")

	_, err = certificate(t, client, "tls.crt").Verify(x509.VerifyOptions{
		Roots:     pool,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	})
	require.NoError(t, err)

	require.NotEqual(t, secrets["trenova-imaging-server-tls"].Data["ca.crt"], server.Data["ca.crt"])
}

func TestTemporalRequiresAClientCertificateAndTheTMSPresentsOne(t *testing.T) {
	t.Parallel()
	_, _, out := renderChart(t, base())

	services := template(t, out, "templates/services.yaml")
	temporal := services[:strings.Index(services, "name: trenova-gotenberg")]
	require.Regexp(t, `name: TEMPORAL_TLS_REQUIRE_CLIENT_AUTH\s+value: "true"`, temporal)
	require.Regexp(t, `name: TEMPORAL_TLS_CLIENT1_CA_CERT\s+value: /certs/temporal/ca.crt`, temporal)
	require.Regexp(t, `name: TEMPORAL_TLS_FRONTEND_CERT\s+value: /certs/temporal/tls.crt`, temporal)
	require.Regexp(t, `name: TEMPORAL_TLS_SERVER_CERT\s+value: /certs/temporal/tls.crt`, temporal)
	require.Contains(t, temporal, "secretName: trenova-temporal-server-tls")

	config := template(t, out, "templates/configmap.yaml")
	require.Contains(t, config, "clientCertPath: /certs/temporal/tls.crt")
	require.Contains(t, config, "clientKeyPath: /certs/temporal/tls.key")
	require.Contains(t, config, "serverCACertPath: /certs/temporal/ca.crt")
	require.Contains(t, config, "serverName: trenova-temporal")

	for _, suffix := range []string{"templates/tms.yaml", "templates/release-job.yaml"} {
		body := template(t, out, suffix)
		require.Contains(t, body, "secretName: trenova-temporal-client-tls", suffix)
		require.NotContains(t, body, "trenova-temporal-server-tls", suffix)
	}
}

func TestGotenbergAndTheTMSShareOneGeneratedCredential(t *testing.T) {
	t.Parallel()
	_, _, out := renderChart(t, base())

	require.Contains(t, template(t, out, "templates/secrets.yaml"), "gotenbergPassword: ")
	require.Contains(t, template(t, out, "templates/configmap.yaml"), "gotenbergUsername: trenova")

	services := template(t, out, "templates/services.yaml")
	require.Contains(t, services, "--api-enable-basic-auth=true")
	require.Regexp(
		t,
		`name: GOTENBERG_API_BASIC_AUTH_PASSWORD\s+valueFrom:\s+secretKeyRef:\s+name: trenova-secrets\s+key: gotenbergPassword`,
		services,
	)

	for _, suffix := range []string{"templates/tms.yaml", "templates/release-job.yaml"} {
		require.Regexp(
			t,
			`name: TRENOVA_RENDERER_GOTENBERGPASSWORD\s+valueFrom:\s+secretKeyRef:\s+name: trenova-secrets\s+key: gotenbergPassword`,
			template(t, out, suffix),
			suffix,
		)
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
