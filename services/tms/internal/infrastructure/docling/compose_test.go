package docling

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

const PinnedImage = "ghcr.io/docling-project/docling-serve-cpu:v1.36.0" +
	"@sha256:225c8586e20d5d0fc6811a9e0e044fa602bcc4393f00389009bad42d6787b58f"

func TestComposeFilesPinTheTestedImage(t *testing.T) {
	t.Parallel()

	root := filepath.Join("..", "..", "..", "..", "..")
	for _, file := range []string{
		filepath.Join(root, "dev", "compose.yml"),
		filepath.Join(root, "deploy", "selfhost", "compose.yml"),
	} {
		contents, err := os.ReadFile(file)
		require.NoError(t, err, file)
		assert.Contains(
			t,
			string(contents),
			"image: "+PinnedImage,
			"%s must run the docling-serve image the integration test pins",
			file,
		)
		assert.NotContains(t, string(contents), "docling-serve-cpu:latest", file)
		assert.Equal(
			t,
			1,
			strings.Count(string(contents), "docling-serve-cpu:"),
			"%s pins docling-serve-cpu exactly once",
			file,
		)
	}
}

var requiredAPIKeyInterpolation = regexp.MustCompile(
	`\$\{TRENOVA_DOCUMENTCONVERSION_APIKEY:?\?`,
)

func TestSelfHostComposeDoesNotRequireTheConverterKey(t *testing.T) {
	t.Parallel()

	file := filepath.Join("..", "..", "..", "..", "..", "deploy", "selfhost", "compose.yml")
	raw, err := os.ReadFile(file)
	require.NoError(t, err)
	contents := string(raw)

	assert.Empty(
		t,
		requiredAPIKeyInterpolation.FindAllString(contents, -1),
		"compose interpolates the whole file, so a required TRENOVA_DOCUMENTCONVERSION_APIKEY "+
			"breaks every compose command on an .env written before the key existed",
	)
	assert.Contains(t, contents, "DOCLING_SERVE_API_KEY: ${TRENOVA_DOCUMENTCONVERSION_APIKEY:-}")
	assert.Contains(
		t,
		contents,
		`if [ -z "$${DOCLING_SERVE_API_KEY}" ]; then`,
		"the sidecar must refuse to start without a key",
	)
}

func TestRequiredAPIKeyInterpolationPattern(t *testing.T) {
	t.Parallel()

	for _, refused := range []string{
		"${TRENOVA_DOCUMENTCONVERSION_APIKEY:?set it}",
		"${TRENOVA_DOCUMENTCONVERSION_APIKEY?set it}",
	} {
		assert.True(t, requiredAPIKeyInterpolation.MatchString(refused), refused)
	}
	assert.False(
		t,
		requiredAPIKeyInterpolation.MatchString("${TRENOVA_DOCUMENTCONVERSION_APIKEY:-}"),
	)
}

type selfHostCompose struct {
	Services map[string]struct {
		Networks   []string       `yaml:"networks"`
		User       string         `yaml:"user"`
		Entrypoint []string       `yaml:"entrypoint"`
		Volumes    []string       `yaml:"volumes"`
		Env        map[string]any `yaml:"environment"`
		DependsOn  map[string]any `yaml:"depends_on"`
	} `yaml:"services"`
	Networks map[string]struct {
		Internal bool `yaml:"internal"`
	} `yaml:"networks"`
}

func loadSelfHostCompose(t *testing.T) selfHostCompose {
	t.Helper()

	file := filepath.Join("..", "..", "..", "..", "..", "deploy", "selfhost", "compose.yml")
	raw, err := os.ReadFile(file)
	require.NoError(t, err)

	var compose selfHostCompose
	require.NoError(t, yaml.Unmarshal(raw, &compose))

	return compose
}

func TestSelfHostParsersShareANetworkOnlyWithTheTMS(t *testing.T) {
	t.Parallel()

	compose := loadSelfHostCompose(t)
	parsers := map[string]string{
		"gotenberg":   "render",
		"docling":     "documents",
		"docling-gpu": "documents",
		"docling-vlm": "documents",
		"imaging":     "imaging",
		"clamav":      "scan",
	}
	allowed := map[string][]string{
		"render":    {"gotenberg", "tms-api", "tms-worker"},
		"documents": {"docling", "docling-gpu", "docling-vlm", "tms-worker"},
		"imaging":   {"imaging", "tms-api", "tms-worker"},
		"scan":      {"clamav", "tms-api", "tms-worker"},
	}

	for parser, network := range parsers {
		assert.Contains(t, compose.Services[parser].Networks, network, parser)
		assert.NotContains(t, compose.Services[parser].Networks, "data", parser)
		assert.NotContains(t, compose.Services[parser].Networks, "edge", parser)
	}

	for network, members := range allowed {
		assert.True(t, compose.Networks[network].Internal, "%s must have no route out", network)

		for name, service := range compose.Services {
			if slices.Contains(service.Networks, network) {
				assert.Contains(t, members, name, "%s must not join %s", name, network)
			}
		}
	}
}

func TestSelfHostUnauthenticatedServicesStayOnTheDataNetwork(t *testing.T) {
	t.Parallel()

	compose := loadSelfHostCompose(t)

	assert.True(t, compose.Networks["data"].Internal)
	assert.Equal(t, []string{"data"}, compose.Services["temporal"].Networks)
	assert.Equal(t, []string{"data"}, compose.Services["gtc"].Networks)
}

func TestSelfHostClamAVRunsUnprivileged(t *testing.T) {
	t.Parallel()

	clamav := loadSelfHostCompose(t).Services["clamav"]

	assert.Equal(t, "clamav", clamav.User)
	assert.Equal(t, []string{"/init-unprivileged"}, clamav.Entrypoint)
}

func TestSelfHostTMSProcessesReachEveryServiceTheyCall(t *testing.T) {
	t.Parallel()

	compose := loadSelfHostCompose(t)

	assert.Subset(
		t,
		compose.Services["tms-api"].Networks,
		[]string{"data", "edge", "egress", "render", "imaging", "scan"},
	)
	assert.Subset(
		t,
		compose.Services["tms-worker"].Networks,
		[]string{"data", "egress", "render", "documents", "imaging", "scan"},
	)
}

func TestSelfHostClientsVerifyPostgres(t *testing.T) {
	t.Parallel()

	root := filepath.Join("..", "..", "..", "..", "..", "deploy", "selfhost")
	raw, err := os.ReadFile(filepath.Join(root, "compose.yml"))
	require.NoError(t, err)
	contents := string(raw)

	assert.NotContains(t, contents, "sslmode=require")
	assert.NotContains(t, contents, "PGSSLMODE: require")
	assert.Contains(t, contents, "sslmode=verify-full&sslrootcert=/certs/postgres/ca.pem")
	assert.Contains(t, contents, "PGSSLMODE: verify-full")
	assert.Contains(t, contents, `SQL_HOST_VERIFICATION: "true"`)
	assert.Contains(t, contents, `POSTGRES_TLS_DISABLE_HOST_VERIFICATION: "false"`)

	config, err := os.ReadFile(filepath.Join(root, "config", "config.yaml"))
	require.NoError(t, err)
	assert.Contains(t, string(config), "sslMode: verify-full")
	assert.Contains(t, string(config), "sslRootCert: /certs/postgres/ca.pem")
}

func TestPostgresKeyIsIssuedPerInstallAndStaysWithTheServer(t *testing.T) {
	t.Parallel()

	compose := loadSelfHostCompose(t)
	for name, service := range compose.Services {
		for _, volume := range service.Volumes {
			if strings.HasPrefix(volume, "postgres-server-tls:") {
				assert.Contains(t, []string{"postgres", "postgres-certs"}, name)
			}
			if strings.HasPrefix(volume, "postgres-ca:") {
				assert.Equal(t, "postgres-certs", name)
			}
		}
	}

	dockerfile, err := os.ReadFile(
		filepath.Join("..", "..", "..", "..", "..", "deploy", "Dockerfile.postgres"),
	)
	require.NoError(t, err)
	assert.NotContains(t, string(dockerfile), "server.key")
	assert.NotContains(t, string(dockerfile), "openssl req")
}

func TestSelfHostTemporalAndGTCSignInAsTheirOwnRoles(t *testing.T) {
	t.Parallel()

	compose := loadSelfHostCompose(t)

	temporal := compose.Services["temporal"]
	assert.Contains(t, temporal.Env["POSTGRES_USER"], "TRENOVA_DATABASE_TEMPORAL_USER")
	assert.Contains(t, temporal.Env["POSTGRES_PWD"], "TRENOVA_DATABASE_TEMPORAL_PASSWORD")
	assert.Equal(t, "true", temporal.Env["SKIP_DB_CREATE"])
	assert.Contains(t, temporal.DependsOn, "db-prepare")

	gtc := compose.Services["gtc"].Env["DATABASE_URL"]
	assert.Contains(t, gtc, "TRENOVA_DATABASE_GTC_USER")
	assert.Contains(t, gtc, "TRENOVA_DATABASE_GTC_PASSWORD")
	assert.NotContains(t, gtc, "${TRENOVA_DATABASE_PASSWORD")

	bootstrap := compose.Services["db-bootstrap"]
	assert.Contains(t, bootstrap.Env["GTC_ROLE"], "TRENOVA_DATABASE_GTC_USER")
	assert.Contains(t, bootstrap.DependsOn, "db-prepare")
}

func TestSelfHostTemporalTakesOnlyItsOwnCertificates(t *testing.T) {
	t.Parallel()

	compose := loadSelfHostCompose(t)

	temporal := compose.Services["temporal"]
	assert.Equal(t, "true", temporal.Env["TEMPORAL_TLS_REQUIRE_CLIENT_AUTH"])
	assert.Equal(t, "/certs/temporal/ca.pem", temporal.Env["TEMPORAL_TLS_CLIENT1_CA_CERT"])
	assert.Equal(t, "/certs/temporal/server.pem", temporal.Env["TEMPORAL_TLS_FRONTEND_CERT"])
	assert.Equal(t, "/certs/temporal/server.pem", temporal.Env["TEMPORAL_TLS_SERVER_CERT"])
	assert.Contains(t, temporal.DependsOn, "temporal-certs")

	for name, service := range compose.Services {
		for _, volume := range service.Volumes {
			switch {
			case strings.HasPrefix(volume, "temporal-ca:"):
				assert.Equal(t, "temporal-certs", name)
			case strings.HasPrefix(volume, "temporal-server-tls:"):
				assert.Contains(t, []string{"temporal", "temporal-certs"}, name)
			}
		}
	}

	root := filepath.Join("..", "..", "..", "..", "..", "deploy", "selfhost")
	raw, err := os.ReadFile(filepath.Join(root, "compose.yml"))
	require.NoError(t, err)
	assert.Contains(t, string(raw), "- temporal-client-tls:/certs/temporal:ro")
	assert.Contains(t, string(raw), "- --server-client-auth")

	config, err := os.ReadFile(filepath.Join(root, "config", "config.yaml"))
	require.NoError(t, err)
	assert.Contains(t, string(config), "clientCertPath: /certs/temporal/client.pem")
	assert.Contains(t, string(config), "clientKeyPath: /certs/temporal/client-key.pem")
	assert.Contains(t, string(config), "serverCACertPath: /certs/temporal/ca.pem")
}

func TestSelfHostOwnerPasswordStaysWithTheOneShots(t *testing.T) {
	t.Parallel()

	compose := loadSelfHostCompose(t)
	mayHoldIt := []string{
		"postgres", "db-prepare", "db-bootstrap", "trenova-db", "rls-roles", "tms-bootstrap",
	}

	for name, service := range compose.Services {
		if slices.Contains(mayHoldIt, name) {
			continue
		}
		for key, value := range service.Env {
			text, isText := value.(string)
			if !isText {
				continue
			}
			assert.NotContains(t, text, "${TRENOVA_DATABASE_PASSWORD", "%s %s", name, key)
		}
	}
}

func TestSelfHostInitEnvGeneratesTheTemporalAndGTCPasswords(t *testing.T) {
	t.Parallel()

	root := filepath.Join("..", "..", "..", "..", "..", "deploy", "selfhost")
	script, err := os.ReadFile(filepath.Join(root, "scripts", "init-env.sh"))
	require.NoError(t, err)
	example, err := os.ReadFile(filepath.Join(root, ".env.example"))
	require.NoError(t, err)

	for _, key := range []string{
		"TRENOVA_DATABASE_TEMPORAL_PASSWORD",
		"TRENOVA_DATABASE_GTC_PASSWORD",
	} {
		assert.Contains(t, string(script), "add_missing "+key, "--upgrade adds it")
		assert.Contains(t, string(script), "  "+key+" \\\n", "a new install generates it")
		assert.Contains(t, string(example), key+"=change-me", ".env.example lists it")
	}
}

func TestSelfHostAPIAndWorkerSignInAsTheirOwnRole(t *testing.T) {
	t.Parallel()

	compose := loadSelfHostCompose(t)

	for _, name := range []string{"tms-api", "tms-worker"} {
		env := compose.Services[name].Env
		assert.Contains(t, env["TRENOVA_DATABASE_USER"], "TRENOVA_DATABASE_APP_USER", name)
		assert.Contains(t, env["TRENOVA_DATABASE_PASSWORD"], "TRENOVA_DATABASE_APP_PASSWORD", name)
		assert.Contains(t, env, "TRENOVA_DATABASE_SYSTEM_PASSWORD", name)
		assert.Contains(t, env, "TRENOVA_DATABASE_RLS_SCOPEKEY", name)
		assert.NotContains(
			t,
			env,
			"TRENOVA_DATABASE_MIGRATOR_PASSWORD",
			"%s must not hold the superuser's password",
			name,
		)
	}
}

func TestSelfHostOneShotsHoldTheOwnerPassword(t *testing.T) {
	t.Parallel()

	compose := loadSelfHostCompose(t)

	for _, name := range []string{"trenova-db", "rls-roles", "tms-bootstrap"} {
		assert.Contains(t, compose.Services[name].Env, "TRENOVA_DATABASE_MIGRATOR_PASSWORD", name)
		assert.Contains(t, compose.Services[name].Env, "TRENOVA_DATABASE_RLS_SCOPEKEY", name)
	}
}

func TestSelfHostRolesExistBeforeAnythingSignsInAsThem(t *testing.T) {
	t.Parallel()

	compose := loadSelfHostCompose(t)

	assert.Contains(t, compose.Services["rls-roles"].DependsOn, "trenova-db")
	for _, name := range []string{"tms-api", "tms-worker", "tms-bootstrap"} {
		assert.Contains(t, compose.Services[name].DependsOn, "rls-roles", name)
	}
}

func TestSelfHostConfigEnforcesRowLevelSecurity(t *testing.T) {
	t.Parallel()

	root := filepath.Join("..", "..", "..", "..", "..", "deploy", "selfhost")
	raw, err := os.ReadFile(filepath.Join(root, "config", "config.yaml"))
	require.NoError(t, err)

	var config struct {
		Database struct {
			User   string `yaml:"user"`
			System struct {
				User string `yaml:"user"`
			} `yaml:"system"`
			Migrator struct {
				User     string `yaml:"user"`
				Password string `yaml:"password"`
			} `yaml:"migrator"`
			RLS struct {
				Mode string `yaml:"mode"`
			} `yaml:"rls"`
		} `yaml:"database"`
	}
	require.NoError(t, yaml.Unmarshal(raw, &config))

	assert.Equal(t, "enforce", config.Database.RLS.Mode)
	assert.Equal(t, "trenova_app", config.Database.User)
	assert.Equal(t, "trenova_system", config.Database.System.User)
	assert.Equal(t, "trenova", config.Database.Migrator.User)
	assert.Empty(t, config.Database.Migrator.Password)
}

func TestSelfHostInstallAndUpgradeWriteEverySecretComposeRequires(t *testing.T) {
	t.Parallel()

	root := filepath.Join("..", "..", "..", "..", "..", "deploy", "selfhost")
	example, err := os.ReadFile(filepath.Join(root, ".env.example"))
	require.NoError(t, err)
	script, err := os.ReadFile(filepath.Join(root, "scripts", "init-env.sh"))
	require.NoError(t, err)

	for _, name := range []string{
		"TRENOVA_DATABASE_APP_PASSWORD",
		"TRENOVA_DATABASE_SYSTEM_PASSWORD",
		"TRENOVA_DATABASE_RLS_SCOPEKEYID",
		"TRENOVA_DATABASE_RLS_SCOPEKEY",
	} {
		assert.Contains(t, string(example), name+"=change-me", name)
		assert.Contains(t, string(script), "add_missing "+name, name)
	}
}
