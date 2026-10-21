package config

import (
	"errors"
	"strings"
	"time"
)

type ImagingConfig struct {
	Endpoint         string                 `mapstructure:"endpoint"`
	ServerName       string                 `mapstructure:"serverName"`
	TLS              ImagingTLSConfig       `mapstructure:"tls"`
	ChunkBytes       int                    `mapstructure:"chunkBytes"       validate:"omitempty,min=16384,max=4194304"`
	MaxMessageBytes  int                    `mapstructure:"maxMessageBytes"  validate:"omitempty,min=65536,max=67108864"`
	MaxAttempts      int                    `mapstructure:"maxAttempts"      validate:"omitempty,min=1,max=10"`
	InitialBackoff   time.Duration          `mapstructure:"initialBackoff"`
	MaxBackoff       time.Duration          `mapstructure:"maxBackoff"`
	KeepaliveTime    time.Duration          `mapstructure:"keepaliveTime"`
	KeepaliveTimeout time.Duration          `mapstructure:"keepaliveTimeout"`
	Deadlines        ImagingDeadlinesConfig `mapstructure:"deadlines"`
	Breaker          ImagingBreakerConfig   `mapstructure:"breaker"`
	ResolveInterval  time.Duration          `mapstructure:"resolveInterval"`
	PrerenderPages   int                    `mapstructure:"prerenderPages"   validate:"omitempty,min=1,max=500"`
}

type ImagingTLSConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	CAFile   string `mapstructure:"caFile"`
	CertFile string `mapstructure:"certFile"`
	KeyFile  string `mapstructure:"keyFile"`
}

type ImagingDeadlinesConfig struct {
	Probe          time.Duration `mapstructure:"probe"`
	Render         time.Duration `mapstructure:"render"`
	InspectCapture time.Duration `mapstructure:"inspectCapture"`
	Assemble       time.Duration `mapstructure:"assemble"`
	NormalizeImage time.Duration `mapstructure:"normalizeImage"`
}

type ImagingBreakerConfig struct {
	ConsecutiveFailures int           `mapstructure:"consecutiveFailures" validate:"omitempty,min=1,max=100"`
	OpenFor             time.Duration `mapstructure:"openFor"`
	MaxOpenFor          time.Duration `mapstructure:"maxOpenFor"`
}

const (
	defaultImagingEndpoint   = "dns:///imaging:50051"
	defaultImagingChunkBytes = 1 << 20
	defaultImagingMaxMessage = 4 << 20
)

func durationOr(value, fallback time.Duration) time.Duration {
	if value <= 0 {
		return fallback
	}
	return value
}

func (c *ImagingConfig) GetEndpoint() string {
	if strings.TrimSpace(c.Endpoint) == "" {
		return defaultImagingEndpoint
	}
	return strings.TrimSpace(c.Endpoint)
}

func (c *ImagingConfig) GetChunkBytes() int {
	if c.ChunkBytes == 0 {
		return defaultImagingChunkBytes
	}
	return c.ChunkBytes
}

func (c *ImagingConfig) GetMaxMessageBytes() int {
	if c.MaxMessageBytes == 0 {
		return defaultImagingMaxMessage
	}
	return c.MaxMessageBytes
}

func (c *ImagingConfig) GetMaxAttempts() int {
	if c.MaxAttempts == 0 {
		return 4
	}
	return c.MaxAttempts
}

func (c *ImagingConfig) GetInitialBackoff() time.Duration {
	return durationOr(c.InitialBackoff, 100*time.Millisecond)
}

func (c *ImagingConfig) GetMaxBackoff() time.Duration {
	return durationOr(c.MaxBackoff, 5*time.Second)
}

func (c *ImagingConfig) GetKeepaliveTime() time.Duration {
	return durationOr(c.KeepaliveTime, 30*time.Second)
}

func (c *ImagingConfig) GetKeepaliveTimeout() time.Duration {
	return durationOr(c.KeepaliveTimeout, 10*time.Second)
}

func (c *ImagingConfig) GetPrerenderPages() int {
	if c.PrerenderPages == 0 {
		return 20
	}
	return c.PrerenderPages
}

func (c *ImagingDeadlinesConfig) GetProbe() time.Duration {
	return durationOr(c.Probe, 30*time.Second)
}

func (c *ImagingDeadlinesConfig) GetRender() time.Duration {
	return durationOr(c.Render, 60*time.Second)
}

func (c *ImagingDeadlinesConfig) GetInspectCapture() time.Duration {
	return durationOr(c.InspectCapture, 10*time.Minute)
}

func (c *ImagingDeadlinesConfig) GetAssemble() time.Duration {
	return durationOr(c.Assemble, 5*time.Minute)
}

func (c *ImagingDeadlinesConfig) GetNormalizeImage() time.Duration {
	return durationOr(c.NormalizeImage, 60*time.Second)
}

func (c *ImagingDeadlinesConfig) Longest() time.Duration {
	return max(
		c.GetProbe(),
		c.GetRender(),
		c.GetInspectCapture(),
		c.GetAssemble(),
		c.GetNormalizeImage(),
	)
}

func (c *ImagingConfig) GetResolveInterval() time.Duration {
	return durationOr(c.ResolveInterval, 10*time.Second)
}

func (c *ImagingBreakerConfig) GetConsecutiveFailures() int {
	if c.ConsecutiveFailures == 0 {
		return 5
	}
	return c.ConsecutiveFailures
}

func (c *ImagingBreakerConfig) GetOpenFor() time.Duration {
	return durationOr(c.OpenFor, 5*time.Second)
}

func (c *ImagingBreakerConfig) GetMaxOpenFor() time.Duration {
	return durationOr(c.MaxOpenFor, 2*time.Minute)
}

func validateImagingConfig(config *Config, env string) error {
	c := &config.Imaging
	var problems []string

	endpoint := c.GetEndpoint()
	if !strings.HasPrefix(endpoint, "dns:///") && !strings.HasPrefix(endpoint, "passthrough:///") &&
		!strings.HasPrefix(endpoint, "unix://") {
		problems = append(
			problems,
			"imaging.endpoint must be a dns:///, passthrough:/// or unix:// target",
		)
	}
	if c.GetChunkBytes()+(64<<10) > c.GetMaxMessageBytes() {
		problems = append(
			problems,
			"imaging.chunkBytes must leave room for framing inside imaging.maxMessageBytes",
		)
	}
	if c.Breaker.GetOpenFor() > c.Breaker.GetMaxOpenFor() {
		problems = append(
			problems,
			"imaging.breaker.openFor must not exceed imaging.breaker.maxOpenFor",
		)
	}
	if c.GetInitialBackoff() > c.GetMaxBackoff() {
		problems = append(problems, "imaging.initialBackoff must not exceed imaging.maxBackoff")
	}

	if c.TLS.Enabled {
		for key, path := range map[string]string{
			"imaging.tls.caFile":   c.TLS.CAFile,
			"imaging.tls.certFile": c.TLS.CertFile,
			"imaging.tls.keyFile":  c.TLS.KeyFile,
		} {
			if strings.TrimSpace(path) == "" {
				problems = append(problems, key+" is required when imaging.tls.enabled is true")
			}
		}
	} else if isProductionLike(env) {
		problems = append(
			problems,
			"imaging.tls.enabled must be true outside development: "+
				"the link carries tenant documents",
		)
	}

	if len(problems) == 0 {
		return nil
	}
	return errors.New(strings.Join(problems, "; "))
}
