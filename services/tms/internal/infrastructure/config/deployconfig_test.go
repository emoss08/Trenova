package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
	"github.com/stretchr/testify/require"
)

const deployConfigsEnv = "TRENOVA_DEPLOY_CONFIGS"

func deploymentConfigs(t *testing.T) []string {
	t.Helper()

	paths := []string{
		filepath.Join("..", "..", "..", "..", "..", "deploy", "selfhost", "config", "config.yaml"),
	}
	for p := range strings.SplitSeq(os.Getenv(deployConfigsEnv), string(os.PathListSeparator)) {
		if p = strings.TrimSpace(p); p != "" {
			paths = append(paths, p)
		}
	}

	return paths
}

func TestDeploymentConfigsDecodeStrictly(t *testing.T) {
	t.Parallel()

	for _, path := range deploymentConfigs(t) {
		t.Run(filepath.Base(filepath.Dir(path))+"/"+filepath.Base(path), func(t *testing.T) {
			t.Parallel()

			body, err := os.ReadFile(path)
			require.NoError(t, err)

			dir := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(dir, "config.yaml"), body, 0o600))

			l := NewLoader(WithConfigPath(dir), WithEnvironment(appconfig.EnvProduction))
			l.configureViper()
			require.NoError(t, l.loadConfigFiles())
			require.NoError(t, l.rejectRetiredSections())

			_, err = l.decode()
			require.NoError(t, err, "%s does not decode against this build's configuration", path)
		})
	}
}
