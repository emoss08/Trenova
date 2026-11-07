package auditconfig

import (
	"errors"
	"time"
)

const (
	defaultExportTTL         = 168 * time.Hour
	MinFingerprintPepperSize = 32
)

var ErrFingerprintPepperTooShort = errors.New(
	"audit.fingerprintPepper must be at least 32 characters when set",
)

type Config struct {
	FingerprintPepper string        `mapstructure:"fingerprintPepper"`
	ExportTTL         time.Duration `mapstructure:"exportTtl"`
}

func (c *Config) GetExportTTL() time.Duration {
	if c == nil || c.ExportTTL <= 0 {
		return defaultExportTTL
	}

	return c.ExportTTL
}

func (c *Config) Validate() error {
	if c.FingerprintPepper != "" && len(c.FingerprintPepper) < MinFingerprintPepperSize {
		return ErrFingerprintPepperTooShort
	}

	return nil
}
