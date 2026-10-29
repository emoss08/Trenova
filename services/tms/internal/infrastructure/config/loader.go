package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/emoss08/trenova/internal/infrastructure/config/aiauditconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/aiconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/cacheconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/databaseconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/loggingconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/monitoringconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/platformconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/reportingconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/searchconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/securityconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/serverconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/storageconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/temporalconfig"
	"github.com/emoss08/trenova/shared/urlutils"
	"github.com/go-playground/validator/v10"
	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type Loader struct {
	configPath string
	envPrefix  string
	env        string
	viper      *viper.Viper
	validator  *validator.Validate
	sections   []Section
}

type LoaderOption func(*Loader)

func WithConfigPath(path string) LoaderOption {
	return func(l *Loader) {
		l.configPath = path
	}
}

func WithEnvironment(env string) LoaderOption {
	return func(l *Loader) {
		l.env = env
	}
}

func WithEnvPrefix(prefix string) LoaderOption {
	return func(l *Loader) {
		l.envPrefix = prefix
	}
}

func WithSections(sections ...Section) LoaderOption {
	return func(l *Loader) {
		l.sections = slices.Clone(sections)
	}
}

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

	l.validator = newValidator()

	return l
}

// Load loads configuration from all sources
func (l *Loader) Load() (*Config, error) {
	if err := validateSections(l.sections); err != nil {
		return nil, err
	}

	if err := l.determineEnvironment(); err != nil {
		return nil, fmt.Errorf("failed to determine environment: %w", err)
	}

	if l.env == appconfig.EnvDevelopment {
		if err := l.loadEnvFile(); err != nil {
			return nil, fmt.Errorf("failed to load .env file: %w", err)
		}
	}

	l.configureViper()

	if err := l.loadConfigFiles(); err != nil {
		return nil, fmt.Errorf("failed to load config files: %w", err)
	}

	if err := l.rejectRetiredSections(); err != nil {
		return nil, err
	}

	if err := l.rejectUnownedEditionSections(); err != nil {
		return nil, err
	}

	config, err := l.decode()
	if err != nil {
		return nil, err
	}

	if err := l.applyEnvironmentOverrides(config); err != nil {
		return nil, fmt.Errorf("apply environment overrides: %w", err)
	}

	if err := l.applyAIAuditChainKeysEnv(config); err != nil {
		return nil, err
	}

	if err := l.validateConfig(config); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	return config, nil
}

// retiredSections names a section that has moved, and where it went. An
// exact unmarshal already refuses an unknown key, but its message says only
// that the key is unknown — which reads as a typo when the truth is that
// the settings were moved and are now being ignored.
var retiredSections = map[string]string{
	"documentIntelligence": "its settings now live under ai: " +
		"enableAI is ai.documentExtraction, aiTimeout is ai.timeout, " +
		"aiCompletionTimeout is ai.completionTimeout, " +
		"aiStreamIdleTimeout is ai.streamIdleTimeout, " +
		"aiMaxRetries is ai.maxRetries, aiMaxInputChars is ai.maxInputChars, " +
		"aiExtractionMaxTokens is ai.extractionMaxTokens, and the OCR keys " +
		"are gone because Docling reads scans",
}

func (l *Loader) rejectUnownedEditionSections() error {
	for _, path := range editionSectionPaths {
		if !l.viper.IsSet(path) || l.ownedBySection(path) {
			continue
		}

		return fmt.Errorf("%w: remove %q", ErrSectionRequiresEdition, path)
	}

	return nil
}

func (l *Loader) ownedBySection(path string) bool {
	for _, section := range l.sections {
		for _, owned := range section.Paths() {
			if pathsOverlap(path, owned) {
				return true
			}
		}
	}

	return false
}

func (l *Loader) decode() (*Config, error) {
	settings := l.viper.AllSettings()

	extensions := make(map[string]any, len(l.sections))
	for _, section := range l.sections {
		owned := make(map[string]any)
		for _, path := range section.Paths() {
			if value, ok := extractPath(settings, path); ok {
				insertPath(owned, path, value)
			}
		}

		sectionViper := viper.New()
		if err := sectionViper.MergeConfigMap(owned); err != nil {
			return nil, fmt.Errorf("failed to stage %s configuration: %w", section.Name(), err)
		}

		value, err := section.Decode(sectionViper)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal %s configuration: %w", section.Name(), err)
		}
		extensions[section.Name()] = value
	}

	base := viper.New()
	if err := base.MergeConfigMap(settings); err != nil {
		return nil, fmt.Errorf("failed to stage config: %w", err)
	}

	config := &Config{extensions: extensions}
	if err := base.UnmarshalExact(config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return config, nil
}

func (l *Loader) rejectRetiredSections() error {
	for section, guidance := range retiredSections {
		if l.viper.IsSet(section) {
			return fmt.Errorf(
				"configuration section %q has been removed: %s",
				section,
				guidance,
			)
		}
	}

	return nil
}

// determineEnvironment determines the current environment
func (l *Loader) determineEnvironment() error {
	if l.env == "" {
		l.env = os.Getenv("APP_ENV")
	}
	if l.env == "" {
		l.env = appconfig.EnvDevelopment
	}

	if !slices.Contains(appconfig.ValidEnvs, l.env) {
		return fmt.Errorf(
			"invalid environment: %s (must be one of: %s)",
			l.env,
			strings.Join(appconfig.ValidEnvs, ", "),
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

func (l *Loader) configureViper() {
	l.viper.SetConfigType("yaml")
	l.viper.SetEnvPrefix(l.envPrefix)
	l.viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	l.viper.AutomaticEnv()
	l.viper.AllowEmptyEnv(false)

	l.setDefaults()
	l.bindPlatformEnvAliases()
	l.bindAccountingEnv()
	l.bindAIAuditEnv()
	l.bindUnlistedEnv()

	for _, section := range l.sections {
		section.SetDefaults(l.viper, l.envPrefix)
	}
}

const (
	aiAuditChainKeysEnvSuffix        = "_AI_AUDIT_CHAIN_KEYS"
	aiAuditChainActiveKeyIDEnvSuffix = "_AI_AUDIT_CHAIN_ACTIVE_KEY_ID"
)

var envOnlyKeys = []string{
	"security.encryption.retiredKeys",
	"contentScan.failOpen",
	"contentScan.clamav.address",
	"contentScan.clamav.timeout",
	"contentScan.clamav.maxStreamBytes",
}

func (l *Loader) bindUnlistedEnv() {
	for _, key := range envOnlyKeys {
		_ = l.viper.BindEnv(key)
	}
}

func (l *Loader) bindAIAuditEnv() {
	_ = l.viper.BindEnv("aiAudit.chain.activeKeyId", l.envPrefix+aiAuditChainActiveKeyIDEnvSuffix)
}

func (l *Loader) applyAIAuditChainKeysEnv(config *Config) error {
	raw := strings.TrimSpace(os.Getenv(l.envPrefix + aiAuditChainKeysEnvSuffix))
	if raw == "" {
		return nil
	}

	keys, err := aiauditconfig.ParseChainKeys(raw)
	if err != nil {
		return err
	}
	config.AIAudit.Chain.Keys = keys

	return nil
}

func (l *Loader) bindAccountingEnv() {
	_ = l.viper.BindEnv("accounting.quickbooks.clientId", "TRENOVA_QUICKBOOKS_CLIENT_ID")
	_ = l.viper.BindEnv("accounting.quickbooks.clientSecret", "TRENOVA_QUICKBOOKS_CLIENT_SECRET")
	_ = l.viper.BindEnv(
		"accounting.quickbooks.webhookVerifierToken",
		"TRENOVA_QUICKBOOKS_WEBHOOK_VERIFIER_TOKEN",
	)
	_ = l.viper.BindEnv("accounting.quickbooks.environment", "TRENOVA_QUICKBOOKS_ENVIRONMENT")
	_ = l.viper.BindEnv("accounting.quickbooks.redirectUrl", "TRENOVA_QUICKBOOKS_REDIRECT_URL")
	_ = l.viper.BindEnv("accounting.xero.clientId", "TRENOVA_XERO_CLIENT_ID")
	_ = l.viper.BindEnv("accounting.xero.clientSecret", "TRENOVA_XERO_CLIENT_SECRET")
	_ = l.viper.BindEnv("accounting.xero.webhookKey", "TRENOVA_XERO_WEBHOOK_KEY")
	_ = l.viper.BindEnv("accounting.xero.redirectUrl", "TRENOVA_XERO_REDIRECT_URL")
	_ = l.viper.BindEnv(
		"accounting.businessCentral.clientId",
		"TRENOVA_BUSINESSCENTRAL_CLIENT_ID",
	)
	_ = l.viper.BindEnv(
		"accounting.businessCentral.clientSecret",
		"TRENOVA_BUSINESSCENTRAL_CLIENT_SECRET",
	)
	_ = l.viper.BindEnv(
		"accounting.businessCentral.redirectUrl",
		"TRENOVA_BUSINESSCENTRAL_REDIRECT_URL",
	)
	_ = l.viper.BindEnv(
		"accounting.businessCentral.webhookBaseUrl",
		"TRENOVA_BUSINESSCENTRAL_WEBHOOK_BASE_URL",
	)
	_ = l.viper.BindEnv("accounting.netSuite.clientId", "TRENOVA_NETSUITE_CLIENT_ID")
	_ = l.viper.BindEnv("accounting.netSuite.clientSecret", "TRENOVA_NETSUITE_CLIENT_SECRET")
	_ = l.viper.BindEnv("accounting.netSuite.redirectUrl", "TRENOVA_NETSUITE_REDIRECT_URL")
}

func (l *Loader) setDefaults() {
	sections := []struct {
		key      string
		defaults map[string]any
	}{
		{key: "app", defaults: appconfig.Defaults(l.env)},
		{key: "server", defaults: serverconfig.Defaults()},
		{key: "temporal", defaults: temporalconfig.Defaults(l.env)},
		{key: "database", defaults: databaseconfig.Defaults()},
		{key: "reporting", defaults: reportingconfig.Defaults()},
		{key: "security", defaults: securityconfig.Defaults(l.env)},
		{key: "monitoring", defaults: monitoringconfig.Defaults()},
		{key: "aiAudit", defaults: aiauditconfig.Defaults()},
		{key: "logging", defaults: loggingconfig.Defaults()},
		{key: "cache", defaults: cacheconfig.Defaults()},
		{key: "search", defaults: searchconfig.Defaults()},
		{key: "ai", defaults: aiconfig.Defaults()},
		{key: "platform", defaults: platformconfig.Defaults()},
		{key: "storage", defaults: storageconfig.Defaults()},
	}
	for _, section := range sections {
		for key, value := range section.defaults {
			l.viper.SetDefault(section.key+"."+key, value)
		}
	}
}

func (l *Loader) bindPlatformEnvAliases() {
	_ = l.viper.BindEnv("platform.mode", "TRENOVA_DEPLOYMENT_MODE")
	_ = l.viper.BindEnv("platform.instanceId", "TRENOVA_INSTANCE_ID")
}

// loadConfigFiles loads base and environment-specific config files
func (l *Loader) loadConfigFiles() error {
	baseConfig := filepath.Join(l.configPath, "config.yaml")
	l.viper.SetConfigFile(baseConfig)

	if err := l.viper.ReadInConfig(); err != nil {
		if l.env == appconfig.EnvProduction {
			return fmt.Errorf("config file required in production: %w", err)
		}
		if !os.IsNotExist(err) {
			return fmt.Errorf("error reading config: %w", err)
		}
	}

	if l.env == appconfig.EnvDevelopment {
		return nil
	}

	envConfig := filepath.Join(l.configPath, fmt.Sprintf("config.%s.yaml", l.env))
	if l.env == appconfig.EnvProduction {
		if _, err := os.Stat(envConfig); os.IsNotExist(err) {
			envConfig = filepath.Join(l.configPath, "config.prod.yaml")
		}
	}
	if _, err := os.Stat(envConfig); err == nil {
		l.viper.SetConfigFile(envConfig)
		if err = l.viper.MergeInConfig(); err != nil {
			return fmt.Errorf("error merging %s config: %w", l.env, err)
		}
	}

	return nil
}

func (l *Loader) validateConfig(config *Config) error {
	if err := l.validator.Struct(config); err != nil {
		return formatValidationError(err)
	}

	validators := []func() error{
		config.Database.Validate,
		config.Server.Validate,
		config.Security.Validate,
		config.Logging.Validate,
		config.Storage.Validate,
		config.Monitoring.Validate,
		config.AIAudit.Validate,
	}
	for _, validate := range validators {
		if err := validate(); err != nil {
			return err
		}
	}

	if err := l.validatePlatformMode(config); err != nil {
		return err
	}
	productionLike := IsProductionLike(l.env)
	if err := config.Server.ValidateCORS(productionLike); err != nil {
		return err
	}
	if productionLike {
		if err := validateProductionSecurity(config); err != nil {
			return err
		}
	}

	for _, section := range l.sections {
		if err := section.Validate(config, l.env); err != nil {
			return err
		}
	}

	return config.Imaging.Validate(productionLike)
}

func (l *Loader) validatePlatformMode(config *Config) error {
	mode := config.Platform.GetMode()
	if !slices.Contains(editionPlatformModes, mode) {
		return nil
	}

	for _, section := range l.sections {
		if slices.Contains(section.PlatformModes(), mode) {
			return nil
		}
	}

	return fmt.Errorf("%w: %q", ErrPlatformModeRequiresEdition, mode)
}

func validateProductionSecurity(config *Config) error {
	validators := []func() error{
		config.Database.ValidateProduction,
		config.Security.ValidateProduction,
		config.Storage.ValidateProduction,
		config.ContentScan.ValidateProduction,
		config.AIAudit.ValidateProduction,
	}
	for _, validate := range validators {
		if err := validate(); err != nil {
			return err
		}
	}

	return nil
}

func (l *Loader) applyEnvironmentOverrides(config *Config) error {
	if IsProductionLike(l.env) {
		config.App.Debug = false
		config.Server.Mode = "release"
		config.Logging.Stacktrace = false
	}

	if l.env == appconfig.EnvDevelopment {
		config.App.Debug = true
		config.Server.Mode = "debug"
		config.Logging.Stacktrace = true
	}

	config.App.Env = l.env

	if err := os.Setenv("GIN_MODE", config.Server.Mode); err != nil {
		return fmt.Errorf("set gin mode: %w", err)
	}
	return nil
}

func newValidator() *validator.Validate {
	v := validator.New()
	registerValidators(v)

	return v
}

var structValidator = sync.OnceValue(newValidator)

func ValidateStruct(value any) error {
	if err := structValidator().Struct(value); err != nil {
		return formatValidationError(err)
	}

	return nil
}

func registerValidators(v *validator.Validate) {
	_ = v.RegisterValidation("semver", func(fl validator.FieldLevel) bool {
		version := fl.Field().String()
		parts := strings.Split(version, ".")
		return len(parts) == 3
	})

	_ = v.RegisterValidation("hostname_port", func(fl validator.FieldLevel) bool {
		addr := fl.Field().String()
		parts := strings.Split(addr, ":")
		return len(parts) == 2
	})

	_ = v.RegisterValidation("no_trailing_slash", func(fl validator.FieldLevel) bool {
		addr := fl.Field().String()
		return !strings.HasSuffix(addr, "/")
	})

	_ = v.RegisterValidation("origin_or_wildcard", func(fl validator.FieldLevel) bool {
		raw := strings.TrimSpace(fl.Field().String())
		if raw == "*" {
			return true
		}

		origin, ok := urlutils.ParseAbsoluteHTTP(raw)
		if !ok {
			return false
		}

		return origin.Path == "" &&
			origin.RawQuery == "" &&
			origin.Fragment == "" &&
			!strings.HasSuffix(raw, "/")
	})
}

func formatValidationError(err error) error {
	var validationErrs validator.ValidationErrors
	if !errors.As(err, &validationErrs) {
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

func GetEnvironment() string {
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "development"
	}
	return env
}
