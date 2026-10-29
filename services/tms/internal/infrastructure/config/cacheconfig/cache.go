package cacheconfig

import (
	"net"
	"strconv"
	"time"
)

type Config struct {
	Host            string        `mapstructure:"host"            validate:"required"`
	Port            int           `mapstructure:"port"            validate:"required,min=1,max=65535"`
	Username        string        `mapstructure:"username"        validate:"omitempty,min=1,max=63"`
	Password        string        `mapstructure:"password"`
	DB              int           `mapstructure:"db"              validate:"min=0,max=15"`
	PoolSize        int           `mapstructure:"poolSize"        validate:"min=0,max=1000"`
	MinIdleConns    int           `mapstructure:"minIdleConns"    validate:"min=0,max=1000"`
	DialTimeout     time.Duration `mapstructure:"dialTimeout"`
	ReadTimeout     time.Duration `mapstructure:"readTimeout"`
	WriteTimeout    time.Duration `mapstructure:"writeTimeout"`
	PoolTimeout     time.Duration `mapstructure:"poolTimeout"`
	ConnMaxIdleTime time.Duration `mapstructure:"connMaxIdleTime"`
	ConnMaxLifetime time.Duration `mapstructure:"connMaxLifetime"`
	MaxRetries      int           `mapstructure:"maxRetries"      validate:"min=0,max=10"`
	MinRetryBackoff time.Duration `mapstructure:"minRetryBackoff"`
	MaxRetryBackoff time.Duration `mapstructure:"maxRetryBackoff"`
}

func (c *Config) GetRedisAddr() string {
	return net.JoinHostPort(c.Host, strconv.Itoa(c.Port))
}

func Defaults() map[string]any {
	return map[string]any{
		"host":            "localhost",
		"port":            6379,
		"db":              0,
		"poolSize":        10,
		"minIdleConns":    5,
		"maxRetries":      3,
		"maxRetryBackoff": "1s",
		"minRetryBackoff": "100ms",
		"dialTimeout":     "5s",
		"readTimeout":     "3s",
		"writeTimeout":    "3s",
		"poolTimeout":     "10s",
		"connMaxIdleTime": "10m",
		"connMaxLifetime": "1h",
	}
}
