package serverconfig

import (
	"errors"
	"fmt"
	"net"
	"slices"
	"strings"
	"time"
)

var (
	ErrCorsEnabledButNoAllowedOrigins = errors.New(
		"CORS enabled but no allowed origins specified",
	)
	ErrCredentialedWildcardCORS = errors.New(
		"production and staging cannot allow wildcard CORS with credentials",
	)
	ErrInvalidTrustedProxy = errors.New(
		"server.trustedProxies entries must be an IP address or CIDR block",
	)
	ErrRequestTimeoutExceedsWriteTimeout = errors.New(
		"server request timeout must be shorter than server write timeout",
	)
)

const (
	TrustedPlatformCloudflare      = "cloudflare"
	TrustedPlatformGoogleAppEngine = "google-app-engine"
	TrustedPlatformFlyIO           = "flyio"
)

var DefaultTrustedProxies = []string{
	"127.0.0.0/8",
	"::1/128",
	"10.0.0.0/8",
	"172.16.0.0/12",
	"192.168.0.0/16",
	"169.254.0.0/16",
	"fe80::/10",
	"fc00::/7",
}

type Config struct {
	Host              string        `mapstructure:"host"              validate:"required,hostname|ip"`
	Port              int           `mapstructure:"port"              validate:"required,min=1,max=65535"`
	Mode              string        `mapstructure:"mode"              validate:"required,oneof=debug release test"`
	ReadTimeout       time.Duration `mapstructure:"readTimeout"`
	ReadHeaderTimeout time.Duration `mapstructure:"readHeaderTimeout"`
	WriteTimeout      time.Duration `mapstructure:"writeTimeout"`
	IdleTimeout       time.Duration `mapstructure:"idleTimeout"`
	ShutdownTimeout   time.Duration `mapstructure:"shutdownTimeout"`
	RequestTimeout    time.Duration `mapstructure:"requestTimeout"`
	TrustedProxies    []string      `mapstructure:"trustedProxies"`
	TrustedPlatform   string        `mapstructure:"trustedPlatform"   validate:"omitempty,oneof=cloudflare google-app-engine flyio"`
	CORS              CORSConfig    `mapstructure:"cors,omitempty"`
}

type CORSConfig struct {
	Enabled        bool     `mapstructure:"enabled"`
	AllowedOrigins []string `mapstructure:"allowedOrigins" validate:"required_if=Enabled true,dive,origin_or_wildcard"`
	AllowedMethods []string `mapstructure:"allowedMethods" validate:"required_if=Enabled true"`
	AllowedHeaders []string `mapstructure:"allowedHeaders" validate:"required_if=Enabled true"`
	ExposeHeaders  []string `mapstructure:"exposeHeaders"`
	Credentials    bool     `mapstructure:"credentials"`
	MaxAge         int      `mapstructure:"maxAge"         validate:"min=0,max=86400"`
}

func Defaults() map[string]any {
	return map[string]any{
		"host":              "0.0.0.0",
		"port":              8080,
		"mode":              "release",
		"readTimeout":       "30s",
		"readHeaderTimeout": "5s",
		"writeTimeout":      "75s",
		"idleTimeout":       "120s",
		"shutdownTimeout":   "10s",
		"requestTimeout":    "55s",
		"trustedProxies":    DefaultTrustedProxies,
	}
}

func (c *Config) Validate() error {
	if err := c.validateTimeouts(); err != nil {
		return err
	}

	return c.validateTrustedProxies()
}

func (c *Config) ValidateCORS(productionLike bool) error {
	if !c.CORS.Enabled {
		return nil
	}
	if len(c.CORS.AllowedOrigins) == 0 {
		return ErrCorsEnabledButNoAllowedOrigins
	}
	if productionLike &&
		c.CORS.Credentials &&
		slices.Contains(c.CORS.AllowedOrigins, "*") {
		return ErrCredentialedWildcardCORS
	}

	return nil
}

func (c *Config) validateTimeouts() error {
	if c.RequestTimeout > 0 &&
		c.WriteTimeout > 0 &&
		c.RequestTimeout >= c.WriteTimeout {
		return ErrRequestTimeoutExceedsWriteTimeout
	}

	return nil
}

func (c *Config) validateTrustedProxies() error {
	for _, proxy := range c.TrustedProxies {
		trimmed := strings.TrimSpace(proxy)
		if trimmed == "" {
			return fmt.Errorf("%w: %q", ErrInvalidTrustedProxy, proxy)
		}
		if strings.Contains(trimmed, "/") {
			if _, _, err := net.ParseCIDR(trimmed); err != nil {
				return fmt.Errorf("%w: %q", ErrInvalidTrustedProxy, proxy)
			}
			continue
		}
		if net.ParseIP(trimmed) == nil {
			return fmt.Errorf("%w: %q", ErrInvalidTrustedProxy, proxy)
		}
	}

	return nil
}
