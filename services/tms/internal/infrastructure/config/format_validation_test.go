package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatValidationError_WithValidationErrors(t *testing.T) {
	t.Parallel()

	l := NewLoader()

	type testStruct struct {
		Name    string `validate:"required"`
		Port    int    `validate:"required,min=1,max=65535"`
		Mode    string `validate:"required,oneof=debug release"`
		Version string `validate:"required"`
	}

	err := l.validator.Struct(testStruct{})
	require.Error(t, err)

	result := l.formatValidationError(err)
	require.Error(t, result)
	assert.Contains(t, result.Error(), "validation errors:")
	assert.Contains(t, result.Error(), "is required")
}

func TestFormatValidationError_WithNonValidationError(t *testing.T) {
	t.Parallel()

	l := NewLoader()
	nonValidationErr := errors.New("some other error")

	result := l.formatValidationError(nonValidationErr)
	require.Error(t, result)
	assert.Equal(t, nonValidationErr, result)
}

func TestFormatValidationError_RequiredTag(t *testing.T) {
	t.Parallel()

	l := NewLoader()

	type testStruct struct {
		Name string `validate:"required"`
	}

	err := l.validator.Struct(testStruct{})
	require.Error(t, err)

	var validationErrs validator.ValidationErrors
	require.True(t, errors.As(err, &validationErrs))

	result := l.formatValidationError(err)
	assert.Contains(t, result.Error(), "is required")
}

func TestFormatValidationError_MinTag(t *testing.T) {
	t.Parallel()

	l := NewLoader()

	type testStruct struct {
		Port int `validate:"min=100"`
	}

	err := l.validator.Struct(testStruct{Port: 0})
	require.Error(t, err)

	result := l.formatValidationError(err)
	assert.Contains(t, result.Error(), "validation errors:")
}

func TestFormatValidationError_MaxTag(t *testing.T) {
	t.Parallel()

	l := NewLoader()

	type testStruct struct {
		Port int `validate:"max=100"`
	}

	err := l.validator.Struct(testStruct{Port: 200})
	require.Error(t, err)

	result := l.formatValidationError(err)
	assert.Contains(t, result.Error(), "validation errors:")
}

func TestFormatValidationError_OneofTag(t *testing.T) {
	t.Parallel()

	l := NewLoader()

	type testStruct struct {
		Mode string `validate:"oneof=debug release"`
	}

	err := l.validator.Struct(testStruct{Mode: "invalid"})
	require.Error(t, err)

	result := l.formatValidationError(err)
	assert.Contains(t, result.Error(), "validation errors:")
}

func TestFormatValidationError_NoTrailingSlashTag(t *testing.T) {
	t.Parallel()

	l := NewLoader()

	type testStruct struct {
		Path string `validate:"no_trailing_slash"`
	}

	err := l.validator.Struct(testStruct{Path: "/api/"})
	require.Error(t, err)

	result := l.formatValidationError(err)
	assert.Contains(t, result.Error(), "validation errors:")
}

func TestFormatValidationError_RequiredIfTag(t *testing.T) {
	t.Parallel()

	l := NewLoader()

	type testStruct struct {
		Enabled bool   `validate:""`
		Path    string `validate:"required_if=Enabled true"`
	}

	err := l.validator.Struct(testStruct{Enabled: true, Path: ""})
	require.Error(t, err)

	result := l.formatValidationError(err)
	assert.Contains(t, result.Error(), "validation errors:")
}

func TestFormatValidationError_DefaultTag(t *testing.T) {
	t.Parallel()

	l := NewLoader()

	type testStruct struct {
		Email string `validate:"email"`
	}

	err := l.validator.Struct(testStruct{Email: "not-an-email"})
	require.Error(t, err)

	result := l.formatValidationError(err)
	assert.Contains(t, result.Error(), "validation errors:")
}

func TestValidateConfig_ValidationStructError(t *testing.T) {
	t.Parallel()

	l := NewLoader()

	cfg := &Config{
		App: AppConfig{
			Name:    "",
			Version: "",
			Env:     "development",
		},
	}

	err := l.validateConfig(cfg)
	require.Error(t, err)
}

func TestValidateConfig_CorsEnabledWithOrigins(t *testing.T) {
	t.Parallel()

	l := NewLoader()
	cfg := newValidConfig()
	cfg.Server.CORS.Enabled = true
	cfg.Server.CORS.AllowedOrigins = []string{"http://localhost:3000"}
	cfg.Server.CORS.AllowedMethods = []string{"GET", "POST"}
	cfg.Server.CORS.AllowedHeaders = []string{"Content-Type"}

	err := l.validateConfig(cfg)
	require.NoError(t, err)
}

func TestValidateConfig_LoggingOutputFileWithFileConfig(t *testing.T) {
	t.Parallel()

	l := NewLoader()
	cfg := newValidConfig()
	cfg.Logging.Output = "file"
	cfg.Logging.File = &LogFileConfig{
		Path:       "/var/log/app.log",
		MaxSize:    100,
		MaxAge:     30,
		MaxBackups: 5,
	}

	err := l.validateConfig(cfg)
	require.NoError(t, err)
}

func TestLoad_ProductionWithValidConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configContent := validConfigYAML()
	err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(configContent), 0o600)
	require.NoError(t, err)

	t.Setenv("APP_ENV", "production")

	l := NewLoader(WithConfigPath(tmpDir), WithEnvironment("production"))

	cfg, err := l.Load()
	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, "production", cfg.App.Env)
	assert.False(t, cfg.App.Debug)
	assert.Equal(t, "release", cfg.Server.Mode)
	assert.True(t, cfg.Security.Session.Secure)
}

func TestLoad_StagingWithEnvSpecificConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configContent := validConfigYAML()
	err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(configContent), 0o600)
	require.NoError(t, err)

	stagingContent := `
server:
  port: 9090
`
	err = os.WriteFile(filepath.Join(tmpDir, "config.staging.yaml"), []byte(stagingContent), 0o600)
	require.NoError(t, err)

	t.Setenv("APP_ENV", "staging")

	l := NewLoader(WithConfigPath(tmpDir), WithEnvironment("staging"))

	cfg, err := l.Load()
	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, 9090, cfg.Server.Port)
	assert.Equal(t, "staging", cfg.App.Env)
}
