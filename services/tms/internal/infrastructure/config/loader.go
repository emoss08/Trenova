package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/joho/godotenv"
	"github.com/samber/lo"
	"github.com/spf13/viper"
)

// Loader handles configuration loading from multiple sources
// Precedence (highest to lowest):
// 1. Environment variables
// 2. Environment-specific config file (e.g., config.staging.yaml)
// 3. Base config file (config.yaml)
// 4. Default values
type Loader struct {
	configPath string
	envPrefix  string
	env        string
	viper      *viper.Viper
	validator  *validator.Validate
}

// LoaderOption configures the loader
type LoaderOption func(*Loader)

// WithConfigPath sets the configuration path
func WithConfigPath(path string) LoaderOption {
	return func(l *Loader) {
		l.configPath = path
	}
}

// WithEnvironment sets the environment
func WithEnvironment(env string) LoaderOption {
	return func(l *Loader) {
		l.env = env
	}
}

// WithEnvPrefix sets the environment variable prefix
func WithEnvPrefix(prefix string) LoaderOption {
	return func(l *Loader) {
		l.envPrefix = prefix
	}
}

// NewLoader creates a new configuration loader
func NewLoader(opts ...LoaderOption) *Loader {
	l := &Loader{
		configPath: "config",
		envPrefix:  "TRENOVA",
		env:        "",
		viper:      viper.New(),
	}

	for _, opt := range opts {
		opt(l)
	}

	l.validator = validator.New()
	l.registerValidators()

	return l
}

// Load loads configuration from all sources
func (l *Loader) Load() (*Config, error) {
	if err := l.determineEnvironment(); err != nil {
		return nil, fmt.Errorf("failed to determine environment: %w", err)
	}

	if l.env == EnvDevelopment {
		if err := l.loadEnvFile(); err != nil {
			return nil, fmt.Errorf("failed to load .env file: %w", err)
		}
	}

	l.configureViper()

	if err := l.loadConfigFiles(); err != nil {
		return nil, fmt.Errorf("failed to load config files: %w", err)
	}

	config := &Config{}
	if err := l.viper.UnmarshalExact(config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if err := l.loadSecrets(config); err != nil {
		return nil, fmt.Errorf("failed to load secrets: %w", err)
	}

	if err := l.validateConfig(config); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	l.applyEnvironmentOverrides(config)

	return config, nil
}

// determineEnvironment determines the current environment
func (l *Loader) determineEnvironment() error {
	if l.env == "" {
		l.env = os.Getenv("APP_ENV")
	}
	if l.env == "" {
		l.env = EnvDevelopment
	}

	if !slices.Contains(ValidEnvs, l.env) {
		return fmt.Errorf(
			"invalid environment: %s (must be one of: %s)",
			l.env,
			strings.Join(ValidEnvs, ", "),
		)
	}

	return nil
}

// loadEnvFile loads .env file if it exists
func (l *Loader) loadEnvFile() error {
	if err := godotenv.Load(); err != nil {
		// It's okay if .env doesn't exist in development
		if !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

// configureViper sets up viper configuration
func (l *Loader) configureViper() {
	l.viper.SetConfigType("yaml")
	l.viper.SetEnvPrefix(l.envPrefix)
	l.viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	l.viper.AutomaticEnv()
	l.viper.AllowEmptyEnv(false)

	l.setDefaults()
}

// setDefaults sets default configuration values
func (l *Loader) setDefaults() { //nolint:funlen // sets default configs
	// App defaults
	l.viper.SetDefault("app.env", l.env)
	l.viper.SetDefault("app.debug", false)
	l.viper.SetDefault("app.version", "0.0.0")

	// Server defaults
	l.viper.SetDefault("server.host", "0.0.0.0")
	l.viper.SetDefault("server.port", 8080)
	l.viper.SetDefault("server.mode", "release")
	l.viper.SetDefault("server.readTimeout", "30s")
	l.viper.SetDefault("server.writeTimeout", "30s")
	l.viper.SetDefault("server.idleTimeout", "120s")
	l.viper.SetDefault("server.shutdownTimeout", "10s")

	// Database defaults
	l.viper.SetDefault("database.passwordSource", "env")
	l.viper.SetDefault("database.sslMode", "prefer")
	l.viper.SetDefault("database.maxIdleConns", 10)
	l.viper.SetDefault("database.maxOpenConns", 100)
	l.viper.SetDefault("database.connMaxLifetime", "1h")
	l.viper.SetDefault("database.connMaxIdleTime", "10m")

	// Session defaults
	l.viper.SetDefault("security.session.name", "trv-session-id")
	l.viper.SetDefault("security.session.maxAge", "24h")
	l.viper.SetDefault("security.session.httpOnly", true)
	l.viper.SetDefault("security.session.secure", false)
	l.viper.SetDefault("security.session.sameSite", "lax")
	l.viper.SetDefault("security.session.path", "/")
	l.viper.SetDefault("security.session.refreshWindow", "1h")

	// CSRF defaults
	l.viper.SetDefault("security.csrf.tokenName", "csrf_token")
	l.viper.SetDefault("security.csrf.headerName", "X-CSRF-Token")

	// Rate limit defaults
	l.viper.SetDefault("security.rateLimit.enabled", true)
	l.viper.SetDefault("security.rateLimit.requestsPerMinute", 60)
	l.viper.SetDefault("security.rateLimit.burstSize", 10)
	l.viper.SetDefault("security.rateLimit.cleanupInterval", "1m")

	// Logging defaults
	l.viper.SetDefault("logging.level", "info")
	l.viper.SetDefault("logging.format", "json")
	l.viper.SetDefault("logging.output", "stdout")
	l.viper.SetDefault("logging.sampling", false)
	l.viper.SetDefault("logging.stacktrace", false)

	// Health check defaults
	l.viper.SetDefault("monitoring.health.path", "/health")
	l.viper.SetDefault("monitoring.health.readinessPath", "/ready")
	l.viper.SetDefault("monitoring.health.livenessPath", "/live")
	l.viper.SetDefault("monitoring.health.checkInterval", "30s")
	l.viper.SetDefault("monitoring.health.timeout", "5s")

	// Cache defaults
	l.viper.SetDefault("cache.provider", "memory")
	l.viper.SetDefault("cache.host", "localhost")
	l.viper.SetDefault("cache.port", 6379)
	l.viper.SetDefault("cache.db", 0)
	l.viper.SetDefault("cache.poolSize", 10)
	l.viper.SetDefault("cache.minIdleConns", 5)
	l.viper.SetDefault("cache.maxRetries", 3)
	l.viper.SetDefault("cache.defaultTTL", "1h")
	l.viper.SetDefault("cache.maxRetryBackoff", "1s")
	l.viper.SetDefault("cache.minRetryBackoff", "100ms")
	l.viper.SetDefault("cache.dialTimeout", "5s")
	l.viper.SetDefault("cache.readTimeout", "3s")
	l.viper.SetDefault("cache.writeTimeout", "3s")
	l.viper.SetDefault("cache.poolTimeout", "10s")
	l.viper.SetDefault("cache.connMaxIdleTime", "10m")
	l.viper.SetDefault("cache.connMaxLifetime", "1h")
}

// loadConfigFiles loads base and environment-specific config files
func (l *Loader) loadConfigFiles() error {
	baseConfig := filepath.Join(l.configPath, "config.yaml")
	l.viper.SetConfigFile(baseConfig)

	if err := l.viper.ReadInConfig(); err != nil {
		if l.env == EnvProduction {
			return fmt.Errorf("config file required in production: %w", err)
		}
		if !os.IsNotExist(err) {
			return fmt.Errorf("error reading config: %w", err)
		}
	}

	if l.env != EnvDevelopment {
		envConfig := filepath.Join(l.configPath, fmt.Sprintf("config.%s.yaml", l.env))
		if _, err := os.Stat(envConfig); err == nil {
			l.viper.SetConfigFile(envConfig)
			if err = l.viper.MergeInConfig(); err != nil {
				return fmt.Errorf("error merging %s config: %w", l.env, err)
			}
		}
	}

	return nil
}

// loadSecrets loads secrets from various sources
func (l *Loader) loadSecrets(config *Config) error {
	switch config.Database.PasswordSource {
	case "env":
		if config.Database.Password == "" {
			return ErrDatabasePasswordNotSet
		}
	case "file":
		if config.Database.PasswordFile == "" {
			return ErrDatabasePasswordFileNotSet
		}
		password, err := os.ReadFile(config.Database.PasswordFile)
		if err != nil {
			return fmt.Errorf("failed to read password file: %w", err)
		}
		config.Database.Password = strings.TrimSpace(string(password))
	case "secret":
		// NOTE: For secret manager integration, the password would be loaded
		// by the secret provider during dependency injection
		// Here we just validate that the secret key is specified
		if config.Database.PasswordSecret == "" {
			return ErrDatabasePasswordSecretNotSet
		}
		// NOTE: The actual secret will be loaded when creating the database connection
	default:
		return fmt.Errorf("unknown password source: %s", config.Database.PasswordSource)
	}

	if config.Security.Session.Secret == "" {
		return ErrSessionSecretIsRequired
	}

	if l.env == "production" {
		if lo.Contains(InsecureDefaultValues, strings.ToLower(config.Security.Session.Secret)) {
			return ErrSessionSecretIsInsecure
		}
	}

	return nil
}

// validateConfig validates the configuration
func (l *Loader) validateConfig(config *Config) error {
	if err := l.validator.Struct(config); err != nil {
		return l.formatValidationError(err)
	}

	if config.Database.MaxIdleConns > config.Database.MaxOpenConns {
		return ErrMaxIdleConnsExceedsMaxOpenConns
	}

	if config.Cache != nil && config.Cache.Provider == "redis" {
		if config.Cache.MinIdleConns > config.Cache.PoolSize {
			return ErrCacheMinIdleConnsExceedsPoolSize
		}
	}

	if config.Server.CORS.Enabled {
		if len(config.Server.CORS.AllowedOrigins) == 0 {
			return ErrCorsEnabledButNoAllowedOrigins
		}
	}

	if config.Logging.Output == "file" && config.Logging.File == nil {
		return ErrLoggingOutputIsFileButFileConfigIsMissing
	}

	return nil
}

// applyEnvironmentOverrides applies environment-specific overrides
func (l *Loader) applyEnvironmentOverrides(config *Config) {
	if l.env == "production" {
		config.App.Debug = false
		config.Server.Mode = "release"
		config.Security.Session.Secure = true
		config.Security.Session.HTTPOnly = true
		config.Logging.Stacktrace = false

		// Ensure SSL for database if not explicitly disabled
		if config.Database.SSLMode == "disable" {
			config.Database.SSLMode = "require"
		}
	}

	if l.env == "development" {
		config.App.Debug = true
		config.Server.Mode = "debug"
		config.Logging.Stacktrace = true
	}

	config.App.Env = l.env

	os.Setenv("GIN_MODE", config.Server.Mode)
}

// registerValidators registers custom validators
func (l *Loader) registerValidators() {
	_ = l.validator.RegisterValidation("semver", func(fl validator.FieldLevel) bool {
		version := fl.Field().String()
		parts := strings.Split(version, ".")
		if len(parts) != 3 {
			return false
		}
		// TODO(wolfred): Basic check - could be enhanced with proper semver parsing
		return true
	})

	// Register hostname_port validator
	_ = l.validator.RegisterValidation("hostname_port", func(fl validator.FieldLevel) bool {
		addr := fl.Field().String()
		parts := strings.Split(addr, ":")
		return len(parts) == 2
	})

	// Register no_trailing_slash validator
	_ = l.validator.RegisterValidation("no_trailing_slash", func(fl validator.FieldLevel) bool {
		addr := fl.Field().String()
		return !strings.HasSuffix(addr, "/")
	})
}

// formatValidationError formats validation errors for better readability
func (l *Loader) formatValidationError(err error) error {
	var validationErrs validator.ValidationErrors
	if errors.As(err, &validationErrs) {
		return err
	}

	var errs []string
	for _, e := range validationErrs {
		field := strings.ToLower(e.Namespace())
		field = strings.ReplaceAll(field, "config.", "")

		switch e.Tag() {
		case "required":
			errs = append(errs, fmt.Sprintf("%s is required", field))
		case "min":
			errs = append(errs, fmt.Sprintf("%s must be at least %s", field, e.Param()))
		case "max":
			errs = append(errs, fmt.Sprintf("%s must be at most %s", field, e.Param()))
		case "oneof":
			errs = append(errs, fmt.Sprintf("%s must be one of: %s", field, e.Param()))
		case "required_if":
			errs = append(errs, fmt.Sprintf("%s is required when %s", field, e.Param()))
		case "excluded_if":
			errs = append(errs, fmt.Sprintf("%s is excluded when %s", field, e.Param()))
		case "no_trailing_slash":
			errs = append(errs, fmt.Sprintf("%s must not have a trailing slash", field))
		default:
			errs = append(errs, fmt.Sprintf("%s failed %s validation", field, e.Tag()))
		}
	}

	return fmt.Errorf("validation errors:\n  - %s", strings.Join(errs, "\n  - "))
}

// GetEnvironment returns the current environment
func GetEnvironment() string {
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "development"
	}
	return env
}
