package monitoringconfig

import (
	"net/url"
	"strings"
	"time"
)

type Config struct {
	Metrics MetricsConfig       `mapstructure:"metrics,omitempty"`
	Tracing TracingConfig       `mapstructure:"tracing,omitempty"`
	Health  HealthConfig        `mapstructure:"health"`
	Pprof   PprofConfig         `mapstructure:"pprof"`
	GraphQL GraphQLObservConfig `mapstructure:"graphql,omitempty"`
}

const (
	defaultGraphQLSlowOperationAfter   = time.Second
	defaultGraphQLMaxTrackedOperations = 2000
)

type GraphQLObservConfig struct {
	ResolverMetrics      bool          `mapstructure:"resolverMetrics"`
	SlowOperationAfter   time.Duration `mapstructure:"slowOperationAfter"`
	MaxTrackedOperations int           `mapstructure:"maxTrackedOperations" validate:"omitempty,min=1"`
}

func (c *GraphQLObservConfig) GetSlowOperationAfter() time.Duration {
	if c.SlowOperationAfter <= 0 {
		return defaultGraphQLSlowOperationAfter
	}
	return c.SlowOperationAfter
}

func (c *GraphQLObservConfig) GetMaxTrackedOperations() int {
	if c.MaxTrackedOperations <= 0 {
		return defaultGraphQLMaxTrackedOperations
	}
	return c.MaxTrackedOperations
}

type MetricsConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Host    string `mapstructure:"host"    validate:"omitempty,hostname|ip"`
	Port    int    `mapstructure:"port"    validate:"required_if=Enabled true,min=1,max=65535"`
	Path    string `mapstructure:"path"    validate:"required_if=Enabled true"`
}

func (c *MetricsConfig) GetHost() string {
	if strings.TrimSpace(c.Host) == "" {
		return "127.0.0.1"
	}
	return c.Host
}

type TracingConfig struct {
	Enabled      bool    `mapstructure:"enabled"`
	Provider     string  `mapstructure:"provider"         validate:"required_if=Enabled true,oneof=jaeger otlp otlp-grpc stdout"`
	Endpoint     string  `mapstructure:"endpoint"         validate:"required_if=Enabled true"`
	ServiceName  string  `mapstructure:"serviceName"      validate:"required_if=Enabled true"`
	SamplingRate float64 `mapstructure:"samplingRate"     validate:"min=0,max=1"`
	// AISamplingRate is the share of agent runs, turns and delegates whose
	// traces are kept. Unset keeps every one.
	AISamplingRate *float64 `mapstructure:"aiSamplingRate"   validate:"omitempty,min=0,max=1"`
	// TraceURLTemplate opens a trace in the tracing backend, with {traceId}
	// where the id goes. Empty leaves the id to be copied.
	TraceURLTemplate string `mapstructure:"traceUrlTemplate"`
}

type HealthConfig struct {
	Path          string        `mapstructure:"path"          validate:"required"`
	ReadinessPath string        `mapstructure:"readinessPath"`
	LivenessPath  string        `mapstructure:"livenessPath"`
	CheckInterval time.Duration `mapstructure:"checkInterval"`
	Timeout       time.Duration `mapstructure:"timeout"`
}

type PprofConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Host    string `mapstructure:"host"    validate:"omitempty,hostname|ip"`
	Port    int    `mapstructure:"port"    validate:"omitempty,min=1,max=65535"`
}

func (c *PprofConfig) GetHost() string {
	if strings.TrimSpace(c.Host) == "" {
		return "127.0.0.1"
	}
	return c.Host
}

func (c *PprofConfig) GetPort() int {
	if c.Port <= 0 {
		return 6060
	}
	return c.Port
}

func (c *TracingConfig) GetAISamplingRate() float64 {
	if c.AISamplingRate == nil {
		return defaultAISamplingRate
	}

	return *c.AISamplingRate
}

func (c *TracingConfig) TraceURL(traceID string) string {
	template := strings.TrimSpace(c.TraceURLTemplate)
	if template == "" || traceID == "" {
		return ""
	}

	return strings.ReplaceAll(template, TraceURLTraceIDPlaceholder, url.PathEscape(traceID))
}

func Defaults() map[string]any {
	return map[string]any{
		"tracing.aiSamplingRate":   defaultAISamplingRate,
		"tracing.traceUrlTemplate": "",
		"metrics.host":             "127.0.0.1",
		"metrics.port":             9090,
		"metrics.path":             "/internal/metricsz",
		"health.path":              "/health",
		"health.readinessPath":     "/ready",
		"health.livenessPath":      "/live",
		"health.checkInterval":     "30s",
		"health.timeout":           "5s",
		"pprof.enabled":            false,
		"pprof.host":               "127.0.0.1",
		"pprof.port":               6060,
	}
}

func (c *Config) Validate() error {
	return c.Tracing.Validate()
}
