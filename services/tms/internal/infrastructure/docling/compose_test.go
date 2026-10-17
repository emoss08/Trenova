package docling

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
