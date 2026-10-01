package config

import (
	"encoding/base64"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	RLSModeOff     = "off"
	RLSModeObserve = "observe"
	RLSModeEnforce = "enforce"

	minRLSScopeKeyBytes    = 32
	defaultRLSScopeTTL     = 15 * time.Minute
	maxRLSScopeTTL         = 6 * time.Hour
	defaultSystemPoolConns = 8
	defaultSystemIdleConns = 2
	maxSystemPoolConns     = 100
)

var rlsScopeKeyIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,32}$`)

type RLSConfig struct {
	Mode       string        `mapstructure:"mode"       validate:"omitempty,oneof=off observe enforce"`
	ScopeKeyID string        `mapstructure:"scopeKeyId"`
	ScopeKey   string        `mapstructure:"scopeKey"`
	ScopeTTL   time.Duration `mapstructure:"scopeTtl"`
}

type DatabaseRole struct {
	User         string `mapstructure:"user"         validate:"omitempty,min=1,max=63"`
	Password     string `mapstructure:"password"`
	MaxOpenConns int    `mapstructure:"maxOpenConns" validate:"omitempty,min=1,max=1000"`
	MaxIdleConns int    `mapstructure:"maxIdleConns" validate:"omitempty,min=1,max=1000"`
}

func (r *DatabaseRole) Configured() bool {
	return strings.TrimSpace(r.User) != ""
}

func (r *DatabaseRole) GetMaxOpenConns() int {
	if r.MaxOpenConns <= 0 {
		return defaultSystemPoolConns
	}
	return min(r.MaxOpenConns, maxSystemPoolConns)
}

func (r *DatabaseRole) GetMaxIdleConns() int {
	if r.MaxIdleConns <= 0 {
		return min(defaultSystemIdleConns, r.GetMaxOpenConns())
	}
	return min(r.MaxIdleConns, r.GetMaxOpenConns())
}

func (c *RLSConfig) GetMode() string {
	mode := strings.ToLower(strings.TrimSpace(c.Mode))
	if mode == "" {
		return RLSModeOff
	}
	return mode
}

func (c *RLSConfig) Enabled() bool {
	return c.GetMode() != RLSModeOff
}

func (c *RLSConfig) Enforced() bool {
	return c.GetMode() == RLSModeEnforce
}

func (c *RLSConfig) GetScopeTTL() time.Duration {
	if c.ScopeTTL <= 0 {
		return defaultRLSScopeTTL
	}
	return min(c.ScopeTTL, maxRLSScopeTTL)
}

func (c *RLSConfig) DecodeScopeKey() ([]byte, error) {
	raw := strings.TrimSpace(c.ScopeKey)
	if raw == "" {
		return nil, ErrRLSScopeKeyRequired
	}

	key, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrRLSScopeKeyInvalid, err)
	}
	if len(key) < minRLSScopeKeyBytes {
		return nil, fmt.Errorf(
			"%w: decoded key is %d bytes, at least %d are required",
			ErrRLSScopeKeyInvalid,
			len(key),
			minRLSScopeKeyBytes,
		)
	}

	return key, nil
}

func (c *Config) GetDSNForRole(role DatabaseRole) string {
	if c.Database.GetDialect().IsSQLite() || !role.Configured() {
		return c.GetDSN(c.Database.Password)
	}

	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s/%s?sslmode=%s",
		url.QueryEscape(role.User),
		url.QueryEscape(role.Password),
		net.JoinHostPort(c.Database.Host, strconv.Itoa(c.Database.Port)),
		c.Database.Name,
		c.Database.SSLMode,
	)

	dsn += fmt.Sprintf("&application_name=%s", url.QueryEscape(c.App.Name))
	dsn += "&dial_timeout=10s"

	return dsn
}

func (c *Config) GetMigrationDSN() string {
	return c.GetDSNForRole(c.Database.Migrator)
}

func validateRLSConfig(config *Config) error {
	db := &config.Database
	rls := &db.RLS

	if !rls.Enabled() {
		return nil
	}

	if !db.GetDialect().IsPostgres() {
		return ErrRLSRequiresPostgres
	}

	if !rlsScopeKeyIDPattern.MatchString(strings.TrimSpace(rls.ScopeKeyID)) {
		return ErrRLSScopeKeyIDInvalid
	}

	if _, err := rls.DecodeScopeKey(); err != nil {
		return err
	}

	if !rls.Enforced() {
		return nil
	}

	if !db.System.Configured() || strings.TrimSpace(db.System.Password) == "" {
		return ErrRLSSystemRoleRequired
	}

	if strings.EqualFold(strings.TrimSpace(db.System.User), strings.TrimSpace(db.User)) {
		return ErrRLSSystemRoleMustDiffer
	}

	if db.Migrator.Configured() &&
		strings.EqualFold(strings.TrimSpace(db.Migrator.User), strings.TrimSpace(db.User)) {
		return ErrRLSMigratorRoleMustDiffer
	}

	return nil
}
