package temporalconfig

import (
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
)

type Config struct {
	HostPort     string            `mapstructure:"hostPort"     validate:"required_without=Profile,omitempty,hostname_port"`
	Namespace    string            `mapstructure:"namespace"`
	Identity     string            `mapstructure:"identity"`
	Profile      string            `mapstructure:"profile"      validate:"omitempty,max=128"`
	ConfigFile   string            `mapstructure:"configFile"`
	APIKey       string            `mapstructure:"apiKey"       validate:"omitempty,max=8192"`
	TLS          TLSConfig         `mapstructure:"tls"`
	Security     SecurityConfig    `mapstructure:"security"     validate:"required"`
	Interceptors InterceptorConfig `mapstructure:"interceptors"`
	Schedule     ScheduleConfig    `mapstructure:"schedule"`
	Worker       WorkerConfig      `mapstructure:"worker"`
}

func (c *Config) GetNamespace() string {
	if c.Namespace == "" {
		return "default"
	}

	return c.Namespace
}

func (c *Config) GetIdentity() string {
	if c.Identity == "" {
		return "trenova-tms"
	}

	return c.Identity
}

func (c *Config) UsesProfile() bool {
	return strings.TrimSpace(c.Profile) != ""
}

type TLSConfig struct {
	Enabled          bool   `mapstructure:"enabled"`
	ServerName       string `mapstructure:"serverName"`
	ServerCACertPath string `mapstructure:"serverCACertPath"`
	ClientCertPath   string `mapstructure:"clientCertPath"   validate:"required_with=ClientKeyPath"`
	ClientKeyPath    string `mapstructure:"clientKeyPath"    validate:"required_with=ClientCertPath"`
}

func (c *TLSConfig) IsEnabled() bool {
	return c.Enabled ||
		c.ServerName != "" ||
		c.ServerCACertPath != "" ||
		c.ClientCertPath != "" ||
		c.ClientKeyPath != ""
}

type SecurityConfig struct {
	EnableEncryption     bool   `mapstructure:"enableEncryption"`
	EncryptionKeyID      string `mapstructure:"encryptionKeyID"      validate:"required_if=EnableEncryption true,min=1,max=100"`
	EnableCompression    bool   `mapstructure:"enableCompression"`
	CompressionThreshold int    `mapstructure:"compressionThreshold" validate:"min=0,max=1048576"`
}

type InterceptorConfig struct {
	EnableLogging bool   `mapstructure:"enableLogging"`
	LogLevel      string `mapstructure:"logLevel"      validate:"omitempty,oneof=debug info warn error"`
}

func (c *InterceptorConfig) GetLogLevel() string {
	if c.LogLevel == "" {
		return "info"
	}

	return c.LogLevel
}

type ScheduleConfig struct {
	PersistOnStop bool `mapstructure:"persistOnStop"`
}

// TemporalWorkerConfig picks which task queues a worker process polls. It is
// the one worker setting that is a deployment decision rather than a tuning
// value: a process can be dedicated to interactive replies while another takes
// everything else. Concurrency and pollers are set per queue in code, beside
// the work they size, because the right numbers depend on what runs there.
type WorkerConfig struct {
	Queues []string `mapstructure:"queues"`
}

func Defaults(env string) map[string]any {
	defaults := map[string]any{
		"profile":              "",
		"configFile":           "",
		"apiKey":               "",
		"tls.enabled":          false,
		"tls.serverName":       "",
		"tls.serverCACertPath": "",
		"tls.clientCertPath":   "",
		"tls.clientKeyPath":    "",
	}
	if env == appconfig.EnvDevelopment || env == appconfig.EnvTest {
		defaults["hostPort"] = "localhost:7233"
	}

	return defaults
}
