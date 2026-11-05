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
		Networks []string `yaml:"networks"`
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
