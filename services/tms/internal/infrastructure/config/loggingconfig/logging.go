package loggingconfig

import (
	"errors"
)

var (
	ErrOutputIsFileButFileConfigIsMissing = errors.New(
		"logging output is 'file' but file config is missing",
	)
)

type Config struct {
	Level      string      `mapstructure:"level"          validate:"required,oneof=debug info warn error"`
	Format     string      `mapstructure:"format"         validate:"required,oneof=json text"`
	Output     string      `mapstructure:"output"         validate:"required,oneof=stdout stderr file"`
	File       *FileConfig `mapstructure:"file,omitempty"`
	Sampling   bool        `mapstructure:"sampling"`
	Stacktrace bool        `mapstructure:"stacktrace"`
}

type FileConfig struct {
	Path       string `mapstructure:"path"       validate:"required"`
	MaxSize    int    `mapstructure:"maxSize"    validate:"min=1,max=1000"`
	MaxAge     int    `mapstructure:"maxAge"     validate:"min=1,max=365"`
	MaxBackups int    `mapstructure:"maxBackups" validate:"min=0,max=100"`
	Compress   bool   `mapstructure:"compress"`
}

func Defaults() map[string]any {
	return map[string]any{
		"level":      "info",
		"format":     "json",
		"output":     "stdout",
		"sampling":   false,
		"stacktrace": false,
	}
}

func (c *Config) Validate() error {
	if c.Output == "file" && c.File == nil {
		return ErrOutputIsFileButFileConfigIsMissing
	}

	return nil
}
