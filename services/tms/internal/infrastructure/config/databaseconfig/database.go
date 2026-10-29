package databaseconfig

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrRLSScopeKeyIDInvalid = errors.New(
		"database.rls.scopeKeyId must be 1-32 characters of letters, digits, '_' or '-'",
	)
	ErrRLSScopeKeyRequired = errors.New(
		"database.rls.scopeKey is required when row-level security is on",
	)
	ErrRLSScopeKeyInvalid = errors.New(
		"database.rls.scopeKey must be base64 encoding at least 32 random bytes",
	)
	ErrRLSSystemRoleRequired = errors.New(
		"database.rls.mode=enforce requires database.system.user and database.system.password",
	)
	ErrRLSSystemRoleMustDiffer = errors.New(
		"database.system.user must be a different role from database.user",
	)
	ErrRLSMigratorRoleMustDiffer = errors.New(
		"database.migrator.user must be a different role from database.user when row-level security is enforced",
	)
	ErrProductionSSLRequired = errors.New(
		"production and staging require database.sslMode other than disable",
	)
	ErrSharedCacheNeedsCache = errors.New(
		"database.cache.shared.enabled needs database.cache.enabled",
	)
	ErrCacheListenPortNeedsHost = errors.New(
		"database.cache.listenPort needs database.cache.listenHost",
	)
	ErrCacheLocalBytesTooSmall = errors.New(
		"database.cache.localBytes must be at least 1 MiB (1048576)",
	)
	ErrTenantConcurrencyExceedsPool = errors.New(
		"database.tenantConcurrency cannot exceed database.maxOpenConns",
	)
)

type Config struct {
	Host             string        `mapstructure:"host"                            validate:"required"`
	Port             int           `mapstructure:"port"                            validate:"required,min=1,max=65535"`
	Name             string        `mapstructure:"name"                            validate:"required,min=1,max=63"`
	User             string        `mapstructure:"user"                            validate:"required,min=1,max=63"`
	Password         string        `mapstructure:"password"                        validate:"required"`
	SSLMode          string        `mapstructure:"sslMode"                         validate:"required,oneof=disable require verify-ca verify-full"`
	MaxOpenConns     int           `mapstructure:"maxOpenConns"                    validate:"min=1,max=1000"`
	QueryLog         string        `mapstructure:"queryLog"                        validate:"omitempty,oneof=off errors all"`
	ConnMaxLifetime  time.Duration `mapstructure:"connMaxLifetime"`
	ConnMaxIdleTime  time.Duration `mapstructure:"connMaxIdleTime"`
	StatementTimeout time.Duration `mapstructure:"statementTimeout"`
	LockTimeout      time.Duration `mapstructure:"lockTimeout"`
	IdleTxTimeout    time.Duration `mapstructure:"idleInTransactionSessionTimeout"`
	RLS              RLSConfig     `mapstructure:"rls"`
	System           Role          `mapstructure:"system"`
	Migrator         Role          `mapstructure:"migrator"`

	TenantConcurrency int           `mapstructure:"tenantConcurrency" validate:"min=0,max=1000"`
	SlowWaitWarning   time.Duration `mapstructure:"slowWaitWarning"   validate:"min=0"`
	MaxWait           time.Duration `mapstructure:"maxWait"           validate:"min=0"`
	Cache             CacheConfig   `mapstructure:"cache"`
	Budget            BudgetConfig  `mapstructure:"budget"`
}

func Defaults() map[string]any {
	return map[string]any{
		"sslMode":                         "prefer",
		"queryLog":                        "errors",
		"maxOpenConns":                    100,
		"connMaxLifetime":                 "1h",
		"connMaxIdleTime":                 "10m",
		"statementTimeout":                "10s",
		"lockTimeout":                     "5s",
		"idleInTransactionSessionTimeout": "30s",
		"rls.mode":                        RLSModeOff,
		"rls.scopeKeyId":                  "",
		"rls.scopeKey":                    "",
		"rls.scopeTtl":                    "15m",
		"system.user":                     "",
		"system.password":                 "",
		"system.maxOpenConns":             8,
		"migrator.user":                   "",
		"migrator.password":               "",
		"tenantConcurrency":               0,
		"slowWaitWarning":                 "5s",
		"maxWait":                         "60s",
		"cache.enabled":                   true,
		"cache.localBytes":                DefaultCacheLocalBytes,
		"cache.listenHost":                "",
		"cache.listenPort":                0,
		"cache.shared.enabled":            false,
		"cache.shared.prefix":             DefaultCacheSharedPrefix,
		"budget.maxRepeats":               DefaultBudgetMaxRepeats,
	}
}

func (c *Config) Validate() error {
	if err := c.validateRLS(); err != nil {
		return err
	}

	return c.validateRuntime()
}

func (c *Config) ValidateProduction() error {
	if strings.EqualFold(c.SSLMode, "disable") {
		return ErrProductionSSLRequired
	}

	return nil
}

func (c *Config) GetStatementTimeout() time.Duration {
	if c.StatementTimeout <= 0 {
		return 10 * time.Second
	}
	return c.StatementTimeout
}

func (c *Config) GetLockTimeout() time.Duration {
	if c.LockTimeout <= 0 {
		return 5 * time.Second
	}
	return c.LockTimeout
}

func (c *Config) GetIdleTxTimeout() time.Duration {
	if c.IdleTxTimeout <= 0 {
		return 30 * time.Second
	}
	return c.IdleTxTimeout
}

func (c *Config) GetTenantConcurrency(maxConns int) int {
	if maxConns <= 0 {
		return max(c.TenantConcurrency, 0)
	}
	if c.TenantConcurrency > 0 {
		return min(c.TenantConcurrency, maxConns)
	}
	return max(maxConns/tenantConcurrencyShare, 1)
}
