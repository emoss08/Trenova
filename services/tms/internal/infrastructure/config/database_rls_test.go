package config

import (
	"encoding/base64"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func rlsTestConfig(mode string) *Config {
	cfg := newValidConfig()
	cfg.Database.Driver = "postgres"
	cfg.Database.User = "trenova_app"
	cfg.Database.RLS = RLSConfig{
		Mode:       mode,
		ScopeKeyID: "k20261001",
		ScopeKey:   base64.StdEncoding.EncodeToString([]byte(strings.Repeat("k", 32))),
	}
	cfg.Database.System = DatabaseRole{User: "trenova_system", Password: "system-secret"}

	return cfg
}

func TestValidateRLSConfig(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		mutate func(*Config)
		want   error
	}{
		{name: "off needs nothing", mutate: func(c *Config) { c.Database.RLS = RLSConfig{} }},
		{name: "enforce with everything", mutate: func(*Config) {}},
		{name: "observe without system role", mutate: func(c *Config) {
			c.Database.RLS.Mode = RLSModeObserve
			c.Database.System = DatabaseRole{}
		}},
		{name: "sqlite cannot enable", mutate: func(c *Config) { c.Database.Driver = "sqlite" }, want: ErrRLSRequiresPostgres},
		{name: "bad key id", mutate: func(c *Config) { c.Database.RLS.ScopeKeyID = "has space" }, want: ErrRLSScopeKeyIDInvalid},
		{name: "missing key", mutate: func(c *Config) { c.Database.RLS.ScopeKey = "" }, want: ErrRLSScopeKeyRequired},
		{name: "short key", mutate: func(c *Config) {
			c.Database.RLS.ScopeKey = base64.StdEncoding.EncodeToString([]byte("short"))
		}, want: ErrRLSScopeKeyInvalid},
		{name: "not base64", mutate: func(c *Config) { c.Database.RLS.ScopeKey = "!!!" }, want: ErrRLSScopeKeyInvalid},
		{name: "enforce needs system role", mutate: func(c *Config) { c.Database.System = DatabaseRole{} }, want: ErrRLSSystemRoleRequired},
		{name: "system role must differ", mutate: func(c *Config) { c.Database.System.User = "TRENOVA_APP" }, want: ErrRLSSystemRoleMustDiffer},
		{name: "migrator must differ", mutate: func(c *Config) {
			c.Database.Migrator = DatabaseRole{User: "trenova_app", Password: "x"}
		}, want: ErrRLSMigratorRoleMustDiffer},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cfg := rlsTestConfig(RLSModeEnforce)
			tc.mutate(cfg)

			err := validateRLSConfig(cfg)
			if tc.want == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tc.want)
		})
	}
}

func TestRLSConfig_Defaults(t *testing.T) {
	t.Parallel()

	var rls RLSConfig
	assert.Equal(t, RLSModeOff, rls.GetMode())
	assert.False(t, rls.Enabled())
	assert.Equal(t, 15*time.Minute, rls.GetScopeTTL())

	rls.ScopeTTL = 24 * time.Hour
	assert.Equal(t, 6*time.Hour, rls.GetScopeTTL())

	var role DatabaseRole
	assert.False(t, role.Configured())
	assert.Equal(t, 8, role.GetMaxOpenConns())
	assert.Equal(t, 2, role.GetMaxIdleConns())
}

func TestGetDSNForRole(t *testing.T) {
	t.Parallel()

	cfg := rlsTestConfig(RLSModeEnforce)
	cfg.Database.Host = "db.internal"
	cfg.Database.Port = 5432
	cfg.Database.Name = "trenova"
	cfg.Database.SSLMode = "require"
	cfg.Database.Password = "app"

	dsn := cfg.GetDSNForRole(DatabaseRole{User: "trenova_system", Password: "p@ss word+1"})
	assert.True(t, strings.HasPrefix(dsn, "postgres://trenova_system:"), dsn)
	assert.Contains(t, dsn, "@db.internal:5432/trenova?sslmode=require")

	parsed, err := url.Parse(dsn)
	require.NoError(t, err)
	password, ok := parsed.User.Password()
	require.True(t, ok)
	assert.Equal(t, "p@ss word+1", password, "a space or plus in a password must reach Postgres unchanged")

	assert.Equal(t, cfg.GetDSN(cfg.Database.Password), cfg.GetMigrationDSN())
}
