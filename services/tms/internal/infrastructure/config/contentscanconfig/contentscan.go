package contentscanconfig

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrProductionFailOpen = errors.New(
		"production and staging refuse contentScan.failOpen: a file is held until the scanner can check it",
	)
)

const (
	defaultClamAVTimeout        = 60 * time.Second
	defaultClamAVMaxStreamBytes = 100 << 20
)

type Config struct {
	ClamAV   ClamAVConfig `mapstructure:"clamav"`
	FailOpen bool         `mapstructure:"failOpen"`
}

type ClamAVConfig struct {
	Address        string        `mapstructure:"address"`
	Timeout        time.Duration `mapstructure:"timeout"        validate:"omitempty,min=1s"`
	MaxStreamBytes int64         `mapstructure:"maxStreamBytes" validate:"omitempty,min=1"`
}

func (c Config) Enabled() bool {
	return strings.TrimSpace(c.ClamAV.Address) != ""
}

func (c ClamAVConfig) GetTimeout() time.Duration {
	if c.Timeout <= 0 {
		return defaultClamAVTimeout
	}
	return c.Timeout
}

func (c ClamAVConfig) GetMaxStreamBytes() int64 {
	if c.MaxStreamBytes <= 0 {
		return defaultClamAVMaxStreamBytes
	}
	return c.MaxStreamBytes
}

func (c Config) ValidateProduction() error {
	if c.FailOpen {
		return ErrProductionFailOpen
	}

	return nil
}
