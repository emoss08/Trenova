package managedconfig

import (
	"errors"
	"fmt"
	"strings"
)

const (
	defaultHost    = "127.0.0.1"
	defaultPort    = 8090
	minTokenLength = 32
)

var (
	ErrTokenRequired = fmt.Errorf(
		"managed.token must be at least %d characters when managed.enabled is true",
		minTokenLength,
	)
	ErrPortInvalid = errors.New("managed.port must be between 1 and 65535")
)

type Config struct {
	Enabled bool   `mapstructure:"enabled"`
	Host    string `mapstructure:"host"    validate:"omitempty,hostname|ip"`
	Port    int    `mapstructure:"port"`
	Token   string `mapstructure:"token"`
}

func Defaults() map[string]any {
	return map[string]any{
		"enabled": false,
		"host":    defaultHost,
		"port":    defaultPort,
		"token":   "",
	}
}

func (c *Config) GetHost() string {
	if strings.TrimSpace(c.Host) == "" {
		return defaultHost
	}
	return c.Host
}

func (c *Config) GetPort() int {
	if c.Port == 0 {
		return defaultPort
	}
	return c.Port
}

func (c *Config) Validate() error {
	if !c.Enabled {
		return nil
	}
	if len(strings.TrimSpace(c.Token)) < minTokenLength {
		return ErrTokenRequired
	}
	if p := c.GetPort(); p < 1 || p > 65535 {
		return ErrPortInvalid
	}
	return nil
}
