package cloudconfig

import (
	"errors"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config"
)

const (
	DefaultSupportAccessMaxGrantDuration     = 14 * 24 * time.Hour
	DefaultSupportAccessMaxSessionDuration   = 4 * time.Hour
	DefaultSupportAccessElevationDuration    = 30 * time.Minute
	DefaultSupportAccessSessionStartsPerHour = 20
	DefaultSupportAccessCookieName           = "trenova_support_session"
	DefaultSupportAccessPrincipalEmailDomain = "support.trenova.invalid"
)

var ErrSupportAccessElevationTooLong = errors.New(
	"platform.cloud.supportAccess.elevationDuration must not exceed maxSessionDuration",
)

type CloudSupportAccessConfig struct {
	Enabled              *bool         `mapstructure:"enabled"`
	MaxGrantDuration     time.Duration `mapstructure:"maxGrantDuration"     validate:"omitempty,min=1h,max=336h"`
	MaxSessionDuration   time.Duration `mapstructure:"maxSessionDuration"   validate:"omitempty,min=5m,max=24h"`
	ElevationDuration    time.Duration `mapstructure:"elevationDuration"    validate:"omitempty,min=1m,max=4h"`
	SessionStartsPerHour int           `mapstructure:"sessionStartsPerHour" validate:"min=0"`
	CookieName           string        `mapstructure:"cookieName"           validate:"omitempty,min=1,max=64"`
	PrincipalEmailDomain string        `mapstructure:"principalEmailDomain" validate:"omitempty,fqdn"`
}

func (c *CloudSupportAccessConfig) IsEnabled() bool {
	return c.Enabled == nil || *c.Enabled
}

func (c *CloudSupportAccessConfig) GetMaxGrantDuration() time.Duration {
	if c.MaxGrantDuration <= 0 {
		return DefaultSupportAccessMaxGrantDuration
	}

	return c.MaxGrantDuration
}

func (c *CloudSupportAccessConfig) GetMaxSessionDuration() time.Duration {
	if c.MaxSessionDuration <= 0 {
		return DefaultSupportAccessMaxSessionDuration
	}

	return c.MaxSessionDuration
}

func (c *CloudSupportAccessConfig) GetElevationDuration() time.Duration {
	if c.ElevationDuration <= 0 {
		return DefaultSupportAccessElevationDuration
	}

	return c.ElevationDuration
}

func (c *CloudSupportAccessConfig) GetSessionStartsPerHour() int {
	if c.SessionStartsPerHour <= 0 {
		return DefaultSupportAccessSessionStartsPerHour
	}

	return c.SessionStartsPerHour
}

func (c *CloudSupportAccessConfig) GetCookieName() string {
	if name := strings.TrimSpace(c.CookieName); name != "" {
		return name
	}

	return DefaultSupportAccessCookieName
}

func (c *CloudSupportAccessConfig) GetPrincipalEmailDomain() string {
	if domain := strings.ToLower(strings.TrimSpace(c.PrincipalEmailDomain)); domain != "" {
		return domain
	}

	return DefaultSupportAccessPrincipalEmailDomain
}

func validateSupportAccessConfig(_ *config.Config, settings *Settings) error {
	support := &settings.Cloud.SupportAccess
	if support.GetElevationDuration() > support.GetMaxSessionDuration() {
		return ErrSupportAccessElevationTooLong
	}

	return nil
}
