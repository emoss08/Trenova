package samsara

import "time"

type RetryConfig struct {
	Enabled        bool          `mapstructure:"enabled"`
	MaxAttempts    int           `mapstructure:"maxAttempts"`
	InitialBackoff time.Duration `mapstructure:"initialBackoff"`
	MaxBackoff     time.Duration `mapstructure:"maxBackoff"`
}
