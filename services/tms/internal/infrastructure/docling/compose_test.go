package docling

import (
	"os"
	"path/filepath"
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
