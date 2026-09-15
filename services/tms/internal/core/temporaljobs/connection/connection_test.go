package connection_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/emoss08/trenova/internal/core/temporaljobs/connection"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const cloudProfileTOML = `
[profile.default]
address = "localhost:7233"
namespace = "default"

[profile.cloud]
address = "example-ns.abc12.tmprl.cloud:7233"
namespace = "example-ns.abc12"
api_key = "test-api-key"
`

func writeProfileFile(t *testing.T, contents string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "temporal.toml")
	require.NoError(t, os.WriteFile(path, []byte(contents), 0o600))

	return path
}

func TestBuildOptions_ConfigPlaintext(t *testing.T) {
	t.Parallel()

	opts, summary, err := connection.BuildOptions(&config.TemporalConfig{
		HostPort: "localhost:7233",
		Identity: "trenova-test",
	})
	require.NoError(t, err)

	assert.Equal(t, "localhost:7233", opts.HostPort)
	assert.Equal(t, "default", opts.Namespace)
	assert.Equal(t, "trenova-test", opts.Identity)
	assert.Nil(t, opts.Credentials)
	assert.Nil(t, opts.ConnectionOptions.TLS)
	assert.Equal(t, connection.Summary{
		Source:    "config",
		HostPort:  "localhost:7233",
		Namespace: "default",
		Auth:      "none",
		TLS:       false,
	}, summary)
}

func TestBuildOptions_ConfigAPIKeyEnablesTLS(t *testing.T) {
	t.Parallel()

	opts, summary, err := connection.BuildOptions(&config.TemporalConfig{
		HostPort:  "example-ns.abc12.tmprl.cloud:7233",
		Namespace: "example-ns.abc12",
		APIKey:    "  test-api-key  ",
	})
	require.NoError(t, err)

	assert.Equal(t, "example-ns.abc12", opts.Namespace)
	assert.NotNil(t, opts.Credentials)
	require.NotNil(t, opts.ConnectionOptions.TLS)
	assert.Equal(t, "api-key", summary.Auth)
	assert.True(t, summary.TLS)
	assert.Equal(t, "trenova-tms", opts.Identity)
}

func TestBuildOptions_ConfigTLSServerName(t *testing.T) {
	t.Parallel()

	opts, summary, err := connection.BuildOptions(&config.TemporalConfig{
		HostPort: "temporal.internal:7233",
		TLS: config.TemporalTLSConfig{
			ServerName: "temporal.example.com",
		},
	})
	require.NoError(t, err)

	require.NotNil(t, opts.ConnectionOptions.TLS)
	assert.Equal(t, "temporal.example.com", opts.ConnectionOptions.TLS.ServerName)
	assert.Nil(t, opts.Credentials)
	assert.Equal(t, "none", summary.Auth)
	assert.True(t, summary.TLS)
}

func TestBuildOptions_ConfigInvalidClientCertificate(t *testing.T) {
	t.Parallel()

	missing := filepath.Join(t.TempDir(), "missing.pem")
	_, _, err := connection.BuildOptions(&config.TemporalConfig{
		HostPort: "temporal.internal:7233",
		TLS: config.TemporalTLSConfig{
			ClientCertPath: missing,
			ClientKeyPath:  missing,
		},
	})
	require.Error(t, err)
}

func TestBuildOptions_ConfigMissingAddress(t *testing.T) {
	t.Parallel()

	_, _, err := connection.BuildOptions(&config.TemporalConfig{})
	require.ErrorIs(t, err, connection.ErrTemporalAddressRequired)
}

func TestBuildOptions_Profile(t *testing.T) {
	path := writeProfileFile(t, cloudProfileTOML)

	opts, summary, err := connection.BuildOptions(&config.TemporalConfig{
		HostPort:   "localhost:7233",
		Namespace:  "ignored",
		Profile:    " cloud ",
		ConfigFile: path,
		Identity:   "trenova-cloud",
	})
	require.NoError(t, err)

	assert.Equal(t, "example-ns.abc12.tmprl.cloud:7233", opts.HostPort)
	assert.Equal(t, "example-ns.abc12", opts.Namespace)
	assert.Equal(t, "trenova-cloud", opts.Identity)
	assert.NotNil(t, opts.Credentials)
	require.NotNil(t, opts.ConnectionOptions.TLS)
	assert.Equal(t, connection.Summary{
		Source:    "profile",
		Profile:   "cloud",
		HostPort:  "example-ns.abc12.tmprl.cloud:7233",
		Namespace: "example-ns.abc12",
		Auth:      "api-key",
		TLS:       true,
	}, summary)
}

func TestBuildOptions_ProfileEnvironmentOverride(t *testing.T) {
	path := writeProfileFile(t, cloudProfileTOML)
	t.Setenv("TEMPORAL_NAMESPACE", "override-ns.abc12")

	opts, _, err := connection.BuildOptions(&config.TemporalConfig{
		Profile:    "cloud",
		ConfigFile: path,
	})
	require.NoError(t, err)

	assert.Equal(t, "override-ns.abc12", opts.Namespace)
}

func TestBuildOptions_ProfileRejectsAPIKeyWithoutTLS(t *testing.T) {
	path := writeProfileFile(t, cloudProfileTOML)
	t.Setenv("TEMPORAL_TLS", "false")

	_, _, err := connection.BuildOptions(&config.TemporalConfig{
		Profile:    "cloud",
		ConfigFile: path,
	})
	require.ErrorIs(t, err, connection.ErrTemporalAPIKeyRequiresTLS)
}

func TestBuildOptions_ProfileNotFound(t *testing.T) {
	path := writeProfileFile(t, cloudProfileTOML)

	_, _, err := connection.BuildOptions(&config.TemporalConfig{
		Profile:    "staging",
		ConfigFile: path,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), `"staging"`)
}

func TestBuildOptions_ProfileWithoutAddress(t *testing.T) {
	path := writeProfileFile(t, "[profile.empty]\nnamespace = \"only-namespace\"\n")

	_, _, err := connection.BuildOptions(&config.TemporalConfig{
		HostPort:   "localhost:7233",
		Profile:    "empty",
		ConfigFile: path,
	})
	require.ErrorIs(t, err, connection.ErrTemporalAddressRequired)
}

func TestSummary_Fields(t *testing.T) {
	t.Parallel()

	withoutProfile := connection.Summary{Source: "config", HostPort: "localhost:7233"}.Fields()
	assert.Len(t, withoutProfile, 5)

	withProfile := connection.Summary{Source: "profile", Profile: "cloud"}.Fields()
	require.Len(t, withProfile, 6)
	assert.Equal(t, "profile", withProfile[5].Key)
	assert.Equal(t, "cloud", withProfile[5].String)
}
