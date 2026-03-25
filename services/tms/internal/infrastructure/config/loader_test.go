package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLoader_Defaults(t *testing.T) {
	t.Parallel()

	l := NewLoader()

	assert.Equal(t, "config", l.configPath)
	assert.Equal(t, "TRENOVA", l.envPrefix)
	assert.Equal(t, "", l.env)
	assert.NotNil(t, l.viper)
	assert.NotNil(t, l.validator)
}

func TestWithConfigPath(t *testing.T) {
	t.Parallel()

	l := NewLoader(WithConfigPath("/custom/path"))

	assert.Equal(t, "/custom/path", l.configPath)
}

func TestWithEnvironment(t *testing.T) {
	t.Parallel()

	l := NewLoader(WithEnvironment("production"))

	assert.Equal(t, "production", l.env)
}

func TestWithEnvPrefix(t *testing.T) {
	t.Parallel()

	l := NewLoader(WithEnvPrefix("MYAPP"))

	assert.Equal(t, "MYAPP", l.envPrefix)
}

func TestNewLoader_MultipleOptions(t *testing.T) {
	t.Parallel()

	l := NewLoader(
		WithConfigPath("/etc/myapp"),
		WithEnvironment("staging"),
		WithEnvPrefix("CUSTOM"),
	)

	assert.Equal(t, "/etc/myapp", l.configPath)
	assert.Equal(t, "staging", l.env)
	assert.Equal(t, "CUSTOM", l.envPrefix)
}

func TestDetermineEnvironment(t *testing.T) {
	t.Run("env already set", func(t *testing.T) {
		t.Parallel()

		l := NewLoader(WithEnvironment("staging"))
		err := l.determineEnvironment()

		require.NoError(t, err)
		assert.Equal(t, "staging", l.env)
	})

	t.Run("env empty uses APP_ENV", func(t *testing.T) {
		t.Setenv("APP_ENV", "production")

		l := NewLoader()
		err := l.determineEnvironment()

		require.NoError(t, err)
		assert.Equal(t, "production", l.env)
	})

	t.Run("env empty and APP_ENV empty defaults to development", func(t *testing.T) {
		t.Setenv("APP_ENV", "")

		l := NewLoader()
		err := l.determineEnvironment()

		require.NoError(t, err)
		assert.Equal(t, "development", l.env)
	})

	t.Run("invalid environment returns error", func(t *testing.T) {
		t.Parallel()

		l := NewLoader(WithEnvironment("invalid"))
		err := l.determineEnvironment()

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid environment: invalid")
	})
}

func newValidConfig() *Config {
	return &Config{
		App: AppConfig{
			Name:    "trenova",
			Env:     "development",
			Version: "1.0.0",
		},
		Server: ServerConfig{
			Host: "0.0.0.0",
			Port: 8080,
			Mode: "release",
		},
		Database: DatabaseConfig{
			Host:         "localhost",
			Port:         5432,
			Name:         "trenova",
			User:         "postgres",
			Password:     "password",
			SSLMode:      "disable",
			MaxIdleConns: 10,
			MaxOpenConns: 100,
		},
		Security: SecurityConfig{
			Session: SessionConfig{
				Secret:   "a-very-long-secret-that-is-at-least-32-chars",
				Name:     "trv-session-id",
				MaxAge:   24 * time.Hour,
				SameSite: "lax",
				Path:     "/",
			},
			CSRF: CSRFConfig{
				TokenName:  "csrf_token",
				HeaderName: "X-CSRF-Token",
			},
			RateLimit: RateLimitConfig{
				RequestsPerMinute: 60,
				BurstSize:         10,
			},
			Encryption: EncryptionConfig{
				Key: "a-very-long-encryption-key-that-is-at-least-32-chars",
			},
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
			Output: "stdout",
		},
		Monitoring: MonitoringConfig{
			Metrics: MetricsConfig{
				Enabled: true,
				Port:    9090,
				Path:    "/metrics",
			},
			Tracing: TracingConfig{
				Enabled:     true,
				Provider:    "otlp",
				Endpoint:    "localhost:4317",
				ServiceName: "trenova-tms",
			},
			Health: HealthConfig{
				Path: "/health",
			},
		},
		Cache: CacheConfig{
			Host: "localhost",
			Port: 6379,
		},
		Temporal: TemporalConfig{
			HostPort: "localhost:7233",
			Security: TemporalSecurityConfig{
				EnableEncryption: true,
				EncryptionKeyID:  "test-key-id",
			},
		},
		Audit: AuditConfig{
			BatchSize:          500,
			MaxEntriesPerFlush: 5000,
			DLQMaxRetries:      5,
		},
		Storage: StorageConfig{
			Endpoint:  "http://localhost:9000",
			AccessKey: "minioadmin",
			SecretKey: "minioadmin",
			Bucket:    "trenova",
		},
		Ably: AblyConfig{
			APIKey: "ably.test:key",
		},
		System: SystemConfig{
			SystemUserPassword: "test-system-password",
		},
	}
}

func TestValidateConfig(t *testing.T) {
	t.Parallel()

	t.Run("valid config", func(t *testing.T) {
		t.Parallel()

		l := NewLoader()
		cfg := newValidConfig()

		err := l.validateConfig(cfg)

		require.NoError(t, err)
	})

	t.Run("MaxIdleConns exceeds MaxOpenConns", func(t *testing.T) {
		t.Parallel()

		l := NewLoader()
		cfg := newValidConfig()
		cfg.Database.MaxIdleConns = 200
		cfg.Database.MaxOpenConns = 100

		err := l.validateConfig(cfg)

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrMaxIdleConnsExceedsMaxOpenConns)
	})

	t.Run("CORS enabled but no allowed origins", func(t *testing.T) {
		t.Parallel()

		l := NewLoader()
		cfg := newValidConfig()
		cfg.Server.CORS.Enabled = true
		cfg.Server.CORS.AllowedOrigins = nil
		cfg.Server.CORS.AllowedMethods = []string{"GET"}
		cfg.Server.CORS.AllowedHeaders = []string{"Content-Type"}

		err := l.validateConfig(cfg)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "allowedorigins")
	})

	t.Run("logging output file but no file config", func(t *testing.T) {
		t.Parallel()

		l := NewLoader()
		cfg := newValidConfig()
		cfg.Logging.Output = "file"
		cfg.Logging.File = nil

		err := l.validateConfig(cfg)

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrLoggingOutputIsFileButFileConfigIsMissing)
	})
}

func TestApplyEnvironmentOverrides(t *testing.T) {
	t.Parallel()

	t.Run("production overrides", func(t *testing.T) {
		t.Parallel()

		l := NewLoader(WithEnvironment("production"))
		_ = l.determineEnvironment()
		cfg := newValidConfig()
		cfg.App.Debug = true
		cfg.Server.Mode = "debug"
		cfg.Security.Session.Secure = false
		cfg.Security.Session.HTTPOnly = false
		cfg.Logging.Stacktrace = true
		cfg.Database.SSLMode = "disable"

		l.applyEnvironmentOverrides(cfg)

		assert.False(t, cfg.App.Debug)
		assert.Equal(t, "release", cfg.Server.Mode)
		assert.True(t, cfg.Security.Session.Secure)
		assert.True(t, cfg.Security.Session.HTTPOnly)
		assert.False(t, cfg.Logging.Stacktrace)
		assert.Equal(t, "disable", cfg.Database.SSLMode)
		assert.Equal(t, "production", cfg.App.Env)
	})

	t.Run("production does not override non-disable ssl", func(t *testing.T) {
		t.Parallel()

		l := NewLoader(WithEnvironment("production"))
		_ = l.determineEnvironment()
		cfg := newValidConfig()
		cfg.Database.SSLMode = "verify-full"

		l.applyEnvironmentOverrides(cfg)

		assert.Equal(t, "verify-full", cfg.Database.SSLMode)
	})

	t.Run("development overrides", func(t *testing.T) {
		t.Parallel()

		l := NewLoader(WithEnvironment("development"))
		_ = l.determineEnvironment()
		cfg := newValidConfig()
		cfg.App.Debug = false
		cfg.Server.Mode = "release"
		cfg.Logging.Stacktrace = false

		l.applyEnvironmentOverrides(cfg)

		assert.True(t, cfg.App.Debug)
		assert.Equal(t, "debug", cfg.Server.Mode)
		assert.True(t, cfg.Logging.Stacktrace)
		assert.Equal(t, "development", cfg.App.Env)
	})

	t.Run("sets App.Env to loader env", func(t *testing.T) {
		t.Parallel()

		l := NewLoader(WithEnvironment("staging"))
		_ = l.determineEnvironment()
		cfg := newValidConfig()

		l.applyEnvironmentOverrides(cfg)

		assert.Equal(t, "staging", cfg.App.Env)
	})
}

func TestGetEnvironment(t *testing.T) {
	t.Run("APP_ENV set", func(t *testing.T) {
		t.Setenv("APP_ENV", "staging")

		result := GetEnvironment()

		assert.Equal(t, "staging", result)
	})

	t.Run("APP_ENV empty defaults to development", func(t *testing.T) {
		t.Setenv("APP_ENV", "")

		result := GetEnvironment()

		assert.Equal(t, "development", result)
	})
}

func TestRegisterValidators(t *testing.T) {
	t.Parallel()

	t.Run("semver valid", func(t *testing.T) {
		t.Parallel()

		l := NewLoader()
		type s struct {
			V string `validate:"semver"`
		}
		err := l.validator.Struct(s{V: "1.2.3"})

		assert.NoError(t, err)
	})

	t.Run("semver invalid", func(t *testing.T) {
		t.Parallel()

		l := NewLoader()
		type s struct {
			V string `validate:"semver"`
		}
		err := l.validator.Struct(s{V: "abc"})

		assert.Error(t, err)
	})

	t.Run("hostname_port valid", func(t *testing.T) {
		t.Parallel()

		l := NewLoader()
		type s struct {
			V string `validate:"hostname_port"`
		}
		err := l.validator.Struct(s{V: "host:8080"})

		assert.NoError(t, err)
	})

	t.Run("hostname_port invalid", func(t *testing.T) {
		t.Parallel()

		l := NewLoader()
		type s struct {
			V string `validate:"hostname_port"`
		}
		err := l.validator.Struct(s{V: "noport"})

		assert.Error(t, err)
	})

	t.Run("no_trailing_slash valid", func(t *testing.T) {
		t.Parallel()

		l := NewLoader()
		type s struct {
			V string `validate:"no_trailing_slash"`
		}
		err := l.validator.Struct(s{V: "/api"})

		assert.NoError(t, err)
	})

	t.Run("no_trailing_slash invalid", func(t *testing.T) {
		t.Parallel()

		l := NewLoader()
		type s struct {
			V string `validate:"no_trailing_slash"`
		}
		err := l.validator.Struct(s{V: "/api/"})

		assert.Error(t, err)
	})
}

func TestConfigureViper(t *testing.T) {
	t.Parallel()

	t.Run("sets config type and defaults", func(t *testing.T) {
		t.Parallel()

		l := NewLoader(WithEnvironment("development"), WithEnvPrefix("TEST"))
		_ = l.determineEnvironment()
		l.configureViper()

		assert.Equal(t, 8080, l.viper.GetInt("server.port"))
		assert.Equal(t, "0.0.0.0", l.viper.GetString("server.host"))
		assert.Equal(t, "release", l.viper.GetString("server.mode"))
		assert.Equal(t, 6379, l.viper.GetInt("cache.port"))
		assert.Equal(t, "localhost", l.viper.GetString("cache.host"))
		assert.Equal(t, "info", l.viper.GetString("logging.level"))
		assert.Equal(t, false, l.viper.GetBool("app.debug"))
	})

	t.Run("env prefix is applied", func(t *testing.T) {
		t.Parallel()

		l := NewLoader(WithEnvironment("development"), WithEnvPrefix("MYPREFIX"))
		_ = l.determineEnvironment()
		l.configureViper()

		assert.Equal(t, 8080, l.viper.GetInt("server.port"))
	})
}

func TestSetDefaults(t *testing.T) {
	t.Parallel()

	t.Run("app defaults", func(t *testing.T) {
		t.Parallel()

		l := NewLoader(WithEnvironment("staging"))
		_ = l.determineEnvironment()
		l.configureViper()

		assert.Equal(t, "staging", l.viper.GetString("app.env"))
		assert.Equal(t, false, l.viper.GetBool("app.debug"))
		assert.Equal(t, "0.0.0", l.viper.GetString("app.version"))
	})

	t.Run("server defaults", func(t *testing.T) {
		t.Parallel()

		l := NewLoader(WithEnvironment("development"))
		_ = l.determineEnvironment()
		l.configureViper()

		assert.Equal(t, "0.0.0.0", l.viper.GetString("server.host"))
		assert.Equal(t, 8080, l.viper.GetInt("server.port"))
		assert.Equal(t, "release", l.viper.GetString("server.mode"))
		assert.Equal(t, "30s", l.viper.GetString("server.readTimeout"))
		assert.Equal(t, "30s", l.viper.GetString("server.writeTimeout"))
		assert.Equal(t, "120s", l.viper.GetString("server.idleTimeout"))
		assert.Equal(t, "10s", l.viper.GetString("server.shutdownTimeout"))
	})

	t.Run("database defaults", func(t *testing.T) {
		t.Parallel()

		l := NewLoader(WithEnvironment("development"))
		_ = l.determineEnvironment()
		l.configureViper()

		assert.Equal(t, "prefer", l.viper.GetString("database.sslMode"))
		assert.Equal(t, 10, l.viper.GetInt("database.maxIdleConns"))
		assert.Equal(t, 100, l.viper.GetInt("database.maxOpenConns"))
		assert.Equal(t, "1h", l.viper.GetString("database.connMaxLifetime"))
		assert.Equal(t, "10m", l.viper.GetString("database.connMaxIdleTime"))
	})

	t.Run("session defaults", func(t *testing.T) {
		t.Parallel()

		l := NewLoader(WithEnvironment("development"))
		_ = l.determineEnvironment()
		l.configureViper()

		assert.Equal(t, "trv-session-id", l.viper.GetString("security.session.name"))
		assert.Equal(t, "24h", l.viper.GetString("security.session.maxAge"))
		assert.Equal(t, true, l.viper.GetBool("security.session.httpOnly"))
		assert.Equal(t, false, l.viper.GetBool("security.session.secure"))
		assert.Equal(t, "lax", l.viper.GetString("security.session.sameSite"))
		assert.Equal(t, "/", l.viper.GetString("security.session.path"))
		assert.Equal(t, "1h", l.viper.GetString("security.session.refreshWindow"))
	})

	t.Run("csrf defaults", func(t *testing.T) {
		t.Parallel()

		l := NewLoader(WithEnvironment("development"))
		_ = l.determineEnvironment()
		l.configureViper()

		assert.Equal(t, "csrf_token", l.viper.GetString("security.csrf.tokenName"))
		assert.Equal(t, "X-CSRF-Token", l.viper.GetString("security.csrf.headerName"))
	})

	t.Run("rate limit defaults", func(t *testing.T) {
		t.Parallel()

		l := NewLoader(WithEnvironment("development"))
		_ = l.determineEnvironment()
		l.configureViper()

		assert.Equal(t, true, l.viper.GetBool("security.rateLimit.enabled"))
		assert.Equal(t, 60, l.viper.GetInt("security.rateLimit.requestsPerMinute"))
		assert.Equal(t, 10, l.viper.GetInt("security.rateLimit.burstSize"))
		assert.Equal(t, "1m", l.viper.GetString("security.rateLimit.cleanupInterval"))
	})

	t.Run("logging defaults", func(t *testing.T) {
		t.Parallel()

		l := NewLoader(WithEnvironment("development"))
		_ = l.determineEnvironment()
		l.configureViper()

		assert.Equal(t, "info", l.viper.GetString("logging.level"))
		assert.Equal(t, "json", l.viper.GetString("logging.format"))
		assert.Equal(t, "stdout", l.viper.GetString("logging.output"))
		assert.Equal(t, false, l.viper.GetBool("logging.sampling"))
		assert.Equal(t, false, l.viper.GetBool("logging.stacktrace"))
	})

	t.Run("health defaults", func(t *testing.T) {
		t.Parallel()

		l := NewLoader(WithEnvironment("development"))
		_ = l.determineEnvironment()
		l.configureViper()

		assert.Equal(t, "/health", l.viper.GetString("monitoring.health.path"))
		assert.Equal(t, "/ready", l.viper.GetString("monitoring.health.readinessPath"))
		assert.Equal(t, "/live", l.viper.GetString("monitoring.health.livenessPath"))
		assert.Equal(t, "30s", l.viper.GetString("monitoring.health.checkInterval"))
		assert.Equal(t, "5s", l.viper.GetString("monitoring.health.timeout"))
	})

	t.Run("cache defaults", func(t *testing.T) {
		t.Parallel()

		l := NewLoader(WithEnvironment("development"))
		_ = l.determineEnvironment()
		l.configureViper()

		assert.Equal(t, "localhost", l.viper.GetString("cache.host"))
		assert.Equal(t, 6379, l.viper.GetInt("cache.port"))
		assert.Equal(t, 0, l.viper.GetInt("cache.db"))
		assert.Equal(t, 10, l.viper.GetInt("cache.poolSize"))
		assert.Equal(t, 5, l.viper.GetInt("cache.minIdleConns"))
		assert.Equal(t, 3, l.viper.GetInt("cache.maxRetries"))
		assert.Equal(t, "1s", l.viper.GetString("cache.maxRetryBackoff"))
		assert.Equal(t, "100ms", l.viper.GetString("cache.minRetryBackoff"))
		assert.Equal(t, "5s", l.viper.GetString("cache.dialTimeout"))
		assert.Equal(t, "3s", l.viper.GetString("cache.readTimeout"))
		assert.Equal(t, "3s", l.viper.GetString("cache.writeTimeout"))
		assert.Equal(t, "10s", l.viper.GetString("cache.poolTimeout"))
		assert.Equal(t, "10m", l.viper.GetString("cache.connMaxIdleTime"))
		assert.Equal(t, "1h", l.viper.GetString("cache.connMaxLifetime"))
	})
}

func TestLoadConfigFiles(t *testing.T) {
	t.Run("development env with no config file succeeds", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		l := NewLoader(WithConfigPath(tmpDir), WithEnvironment("development"))
		_ = l.determineEnvironment()
		l.configureViper()

		err := l.loadConfigFiles()

		require.NoError(t, err)
	})

	t.Run("production env with no config file errors", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		l := NewLoader(WithConfigPath(tmpDir), WithEnvironment("production"))
		_ = l.determineEnvironment()
		l.configureViper()

		err := l.loadConfigFiles()

		require.Error(t, err)
		assert.Contains(t, err.Error(), "config file required in production")
	})

	t.Run("valid config file exists", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		configContent := validConfigYAML()
		err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(configContent), 0o600)
		require.NoError(t, err)

		l := NewLoader(WithConfigPath(tmpDir), WithEnvironment("development"))
		_ = l.determineEnvironment()
		l.configureViper()

		err = l.loadConfigFiles()

		require.NoError(t, err)
		assert.Equal(t, "trenova", l.viper.GetString("app.name"))
		assert.Equal(t, 5432, l.viper.GetInt("database.port"))
	})

	t.Run("non-development env with environment-specific config merges", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		baseContent := validConfigYAML()
		err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(baseContent), 0o600)
		require.NoError(t, err)

		stagingContent := `
server:
  port: 9090
`
		err = os.WriteFile(
			filepath.Join(tmpDir, "config.staging.yaml"),
			[]byte(stagingContent),
			0o600,
		)
		require.NoError(t, err)

		l := NewLoader(WithConfigPath(tmpDir), WithEnvironment("staging"))
		_ = l.determineEnvironment()
		l.configureViper()

		err = l.loadConfigFiles()

		require.NoError(t, err)
		assert.Equal(t, 9090, l.viper.GetInt("server.port"))
		assert.Equal(t, "trenova", l.viper.GetString("app.name"))
	})

	t.Run("config file with permission error", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")
		err := os.WriteFile(configPath, []byte("invalid"), 0o000)
		require.NoError(t, err)

		l := NewLoader(WithConfigPath(tmpDir), WithEnvironment("staging"))
		_ = l.determineEnvironment()
		l.configureViper()

		err = l.loadConfigFiles()

		if err != nil {
			assert.Contains(t, err.Error(), "error reading config")
		}
	})
}

func TestLoadRepositoryEnvironmentConfigs(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		env        string
		envContent string
	}{
		{
			name: "test",
			env:  "test",
			envContent: `
app:
  env: test
  debug: true
server:
  port: 8081
  mode: test
monitoring:
  metrics:
    port: 9090
    path: /metrics
  health:
    path: /health
    readinessPath: /ready
    livenessPath: /live
  tracing:
    provider: stdout
`,
		},
		{
			name: "production",
			env:  "production",
			envContent: `
app:
  env: production
  debug: false
server:
  mode: release
monitoring:
  metrics:
    enabled: true
    port: 9090
    path: /metrics
  health:
    path: /health
    readinessPath: /ready
    livenessPath: /live
  tracing:
    enabled: true
    provider: otlp
    endpoint: localhost:4317
    serviceName: trenova
    samplingRate: 1.0
`,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			configPath := t.TempDir()
			require.NoError(
				t,
				os.WriteFile(filepath.Join(configPath, "config.yaml"), []byte(validConfigYAML()), 0o600),
			)
			require.NoError(
				t,
				os.WriteFile(
					filepath.Join(configPath, "config."+tc.env+".yaml"),
					[]byte(tc.envContent),
					0o600,
				),
			)

			l := NewLoader(WithConfigPath(configPath), WithEnvironment(tc.env))

			cfg, loadErr := l.Load()

			require.NoError(t, loadErr)
			require.NotNil(t, cfg)
			assert.NotZero(t, cfg.Monitoring.Metrics.Port)
			assert.NotEmpty(t, cfg.Monitoring.Metrics.Path)
			assert.NotEmpty(t, cfg.Monitoring.Health.Path)
			assert.NotEmpty(t, cfg.Monitoring.Health.ReadinessPath)
			assert.NotEmpty(t, cfg.Monitoring.Health.LivenessPath)
			assert.NotEmpty(t, cfg.Monitoring.Tracing.Provider)
		})
	}
}

func TestLoadEnvFile(t *testing.T) {
	t.Run("no env file exists succeeds", func(t *testing.T) {
		t.Parallel()

		l := NewLoader(WithConfigPath(t.TempDir()))

		err := l.loadEnvFile()

		require.NoError(t, err)
	})
}

func TestLoad(t *testing.T) {
	t.Run("loads valid config from file", func(t *testing.T) {
		tmpDir := t.TempDir()
		configContent := validConfigYAML()
		err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(configContent), 0o600)
		require.NoError(t, err)

		t.Setenv("APP_ENV", "development")

		l := NewLoader(WithConfigPath(tmpDir), WithEnvironment("development"))

		cfg, err := l.Load()

		require.NoError(t, err)
		require.NotNil(t, cfg)
		assert.Equal(t, "trenova", cfg.App.Name)
		assert.Equal(t, "1.0.0", cfg.App.Version)
		assert.Equal(t, "development", cfg.App.Env)
		assert.True(t, cfg.App.Debug)
		assert.Equal(t, "debug", cfg.Server.Mode)
		assert.Equal(t, 8080, cfg.Server.Port)
		assert.Equal(t, "localhost", cfg.Database.Host)
		assert.Equal(t, 5432, cfg.Database.Port)
		assert.Equal(t, "testdb", cfg.Database.Name)
		assert.Equal(t, "postgres", cfg.Database.User)
		assert.Equal(t, "testpass", cfg.Database.Password)
		assert.Equal(t, "localhost", cfg.Cache.Host)
		assert.Equal(t, 6379, cfg.Cache.Port)
		assert.Equal(t, "localhost:7233", cfg.Temporal.HostPort)
	})

	t.Run("env vars override config file values", func(t *testing.T) {
		tmpDir := t.TempDir()
		configContent := validConfigYAML()
		err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(configContent), 0o600)
		require.NoError(t, err)

		t.Setenv("APP_ENV", "development")
		t.Setenv("TRENOVA_SERVER_PORT", "9999")
		t.Setenv("TRENOVA_DATABASE_HOST", "remotehost")

		l := NewLoader(WithConfigPath(tmpDir), WithEnvironment("development"))

		cfg, err := l.Load()

		require.NoError(t, err)
		require.NotNil(t, cfg)
		assert.Equal(t, 9999, cfg.Server.Port)
		assert.Equal(t, "remotehost", cfg.Database.Host)
	})

	t.Run("error on invalid config values", func(t *testing.T) {
		tmpDir := t.TempDir()
		invalidConfig := `
app:
  name: ""
  version: "1.0.0"
server:
  host: "0.0.0.0"
  port: 8080
database:
  host: localhost
  port: 5432
  name: testdb
  user: postgres
  password: testpass
security:
  session:
    secret: "short"
    name: "trv-session-id"
    maxAge: "24h"
    sameSite: "lax"
    path: "/"
  csrf:
    tokenName: "csrf_token"
    headerName: "X-CSRF-Token"
  rateLimit:
    requestsPerMinute: 60
    burstSize: 10
  encryption:
    key: "short"
logging:
  level: info
  format: json
  output: stdout
monitoring:
  metrics:
    enabled: false
    port: 9090
    path: "/metrics"
  tracing:
    enabled: false
    provider: "otlp"
    endpoint: "localhost:4317"
    serviceName: "trenova-tms"
cache:
  host: localhost
  port: 6379
temporal:
  hostPort: "localhost:7233"
  security:
    enableEncryption: false
    encryptionKeyID: "test-key-id"
audit:
  batchSize: 500
  maxEntriesPerFlush: 5000
  dlqMaxRetries: 5
storage:
  endpoint: "http://localhost:9000"
  accessKey: "test"
  secretKey: "test"
  bucket: "test"
`
		err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(invalidConfig), 0o600)
		require.NoError(t, err)

		t.Setenv("APP_ENV", "development")

		l := NewLoader(WithConfigPath(tmpDir), WithEnvironment("development"))

		cfg, err := l.Load()

		require.Error(t, err)
		assert.Nil(t, cfg)
		assert.Contains(t, err.Error(), "validation failed")
	})

	t.Run("error on invalid environment", func(t *testing.T) {
		t.Parallel()

		l := NewLoader(WithEnvironment("bogus"))

		cfg, err := l.Load()

		require.Error(t, err)
		assert.Nil(t, cfg)
		assert.Contains(t, err.Error(), "failed to determine environment")
	})
}

func validConfigYAML() string {
	return `
app:
  name: trenova
  version: "1.0.0"
server:
  host: "0.0.0.0"
  port: 8080
database:
  host: localhost
  port: 5432
  name: testdb
  user: postgres
  password: testpass
  sslMode: disable
security:
  session:
    secret: "a-very-long-secret-that-is-at-least-32-characters-long"
    name: "trv-session-id"
    maxAge: "24h"
    sameSite: "lax"
    path: "/"
  csrf:
    tokenName: "csrf_token"
    headerName: "X-CSRF-Token"
  rateLimit:
    requestsPerMinute: 60
    burstSize: 10
  encryption:
    key: "a-very-long-encryption-key-that-is-at-least-32-chars"
logging:
  level: info
  format: json
  output: stdout
monitoring:
  metrics:
    enabled: false
    port: 9090
    path: "/metrics"
  tracing:
    enabled: false
    provider: "otlp"
    endpoint: "localhost:4317"
    serviceName: "trenova-tms"
cache:
  host: localhost
  port: 6379
temporal:
  hostPort: "localhost:7233"
  security:
    enableEncryption: false
    encryptionKeyID: "test-key-id"
audit:
  batchSize: 500
  maxEntriesPerFlush: 5000
  dlqMaxRetries: 5
storage:
  endpoint: "http://localhost:9000"
  accessKey: "test"
  secretKey: "test"
  bucket: "test"
ably:
  apiKey: "ably.test:key"
system:
  systemUserPassword: "test-system-password"
`
}
