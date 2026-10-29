package databaseconfig

import (
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validDatabaseConfig() *Config {
	return &Config{
		Host:         "localhost",
		Port:         5432,
		Name:         "trenova",
		User:         "postgres",
		Password:     "password",
		SSLMode:      "disable",
		MaxOpenConns: 100,
	}
}

func rlsTestConfig(mode string) *Config {
	cfg := validDatabaseConfig()
	cfg.User = "trenova_app"
	cfg.RLS = RLSConfig{
		Mode:       mode,
		ScopeKeyID: "k20261001",
		ScopeKey:   base64.StdEncoding.EncodeToString([]byte(strings.Repeat("k", 32))),
	}
	cfg.System = Role{User: "trenova_system", Password: "system-secret"}

	return cfg
}

func TestValidateRLSConfig(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		mutate func(*Config)
		want   error
	}{
		{name: "off needs nothing", mutate: func(c *Config) { c.RLS = RLSConfig{} }},
		{name: "enforce with everything", mutate: func(*Config) {}},
		{name: "observe without system role", mutate: func(c *Config) {
			c.RLS.Mode = RLSModeObserve
			c.System = Role{}
		}},
		{
			name:   "bad key id",
			mutate: func(c *Config) { c.RLS.ScopeKeyID = "has space" },
			want:   ErrRLSScopeKeyIDInvalid,
		},
		{
			name:   "missing key",
			mutate: func(c *Config) { c.RLS.ScopeKey = "" },
			want:   ErrRLSScopeKeyRequired,
		},
		{name: "short key", mutate: func(c *Config) {
			c.RLS.ScopeKey = base64.StdEncoding.EncodeToString([]byte("short"))
		}, want: ErrRLSScopeKeyInvalid},
		{
			name:   "not base64",
			mutate: func(c *Config) { c.RLS.ScopeKey = "!!!" },
			want:   ErrRLSScopeKeyInvalid,
		},
		{
			name:   "enforce needs system role",
			mutate: func(c *Config) { c.System = Role{} },
			want:   ErrRLSSystemRoleRequired,
		},
		{
			name:   "system role must differ",
			mutate: func(c *Config) { c.System.User = "TRENOVA_APP" },
			want:   ErrRLSSystemRoleMustDiffer,
		},
		{name: "migrator must differ", mutate: func(c *Config) {
			c.Migrator = Role{User: "trenova_app", Password: "x"}
		}, want: ErrRLSMigratorRoleMustDiffer},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cfg := rlsTestConfig(RLSModeEnforce)
			tc.mutate(cfg)

			err := cfg.validateRLS()
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

	var role Role
	assert.False(t, role.Configured())
	assert.Equal(t, 8, role.GetMaxOpenConns())
}
