package config

import (
	"net/url"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/databaseconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_GetDSN(t *testing.T) {
	t.Parallel()

	t.Run("basic", func(t *testing.T) {
		t.Parallel()
		c := &Config{
			App: appconfig.Config{Name: "trenova"},
			Database: databaseconfig.Config{
				User:    "user",
				Host:    "localhost",
				Port:    5432,
				Name:    "testdb",
				SSLMode: "disable",
			},
		}
		dsn := c.GetDSN("password123")
		assert.Contains(t, dsn, "postgres://user:password123@localhost:5432/testdb")
		assert.Contains(t, dsn, "sslmode=disable")
		assert.Contains(t, dsn, "application_name=trenova")
		assert.Contains(t, dsn, "connect_timeout=10")
		assert.NotContains(t, dsn, "statement_timeout")
		assert.NotContains(t, dsn, "idle_in_transaction_session_timeout")
	})

	t.Run("special characters in password", func(t *testing.T) {
		t.Parallel()
		c := &Config{
			App: appconfig.Config{Name: "trenova"},
			Database: databaseconfig.Config{
				User:    "admin",
				Host:    "db.example.com",
				Port:    5433,
				Name:    "prod",
				SSLMode: "require",
			},
		}
		dsn := c.GetDSN("p@ss w0rd!")
		assert.Contains(t, dsn, "sslmode=require")

		parsed, err := url.Parse(dsn)
		require.NoError(t, err)
		password, ok := parsed.User.Password()
		require.True(t, ok)
		assert.Equal(t, "p@ss w0rd!", password)
		assert.Equal(t, "admin", parsed.User.Username())
		assert.Equal(t, "db.example.com:5433", parsed.Host)
	})

	t.Run("credentials survive the round trip", func(t *testing.T) {
		t.Parallel()

		for _, password := range []string{
			"YwK2IXpEm9eAUpix+8exRW/oMZL==",
			"with space",
			"colon:at@slash/hash#question?percent%amp&",
			"plus+literal",
		} {
			c := &Config{
				App: appconfig.Config{Name: "trenova"},
				Database: databaseconfig.Config{
					User:    "trenova app",
					Host:    "localhost",
					Port:    5432,
					Name:    "trenova",
					SSLMode: "disable",
				},
			}

			parsed, err := url.Parse(c.GetDSN(password))
			require.NoError(t, err, password)
			got, ok := parsed.User.Password()
			require.True(t, ok, password)
			assert.Equal(t, password, got)
			assert.Equal(t, "trenova app", parsed.User.Username())
			assert.Equal(t, "/trenova", parsed.Path)
			assert.Equal(t, "disable", parsed.Query().Get("sslmode"))
		}
	})
}

func TestConfig_GetDSNMasked(t *testing.T) {
	t.Parallel()

	c := &Config{
		Database: databaseconfig.Config{
			User:    "user",
			Host:    "localhost",
			Port:    5432,
			Name:    "testdb",
			SSLMode: "disable",
		},
	}
	masked := c.GetDSNMasked()
	assert.Equal(t, "postgres://user:****@localhost:5432/testdb?sslmode=disable", masked)
}

func TestGetDSNForRole(t *testing.T) {
	t.Parallel()

	cfg := newValidConfig()
	cfg.Database.User = "trenova_app"
	cfg.Database.Host = "db.internal"
	cfg.Database.Port = 5432
	cfg.Database.Name = "trenova"
	cfg.Database.SSLMode = "require"
	cfg.Database.Password = "app"

	dsn := cfg.GetDSNForRole(databaseconfig.Role{User: "trenova_system", Password: "p@ss word+1"})
	assert.True(t, strings.HasPrefix(dsn, "postgres://trenova_system:"), dsn)
	assert.Contains(t, dsn, "@db.internal:5432/trenova?sslmode=require")

	parsed, err := url.Parse(dsn)
	require.NoError(t, err)
	password, ok := parsed.User.Password()
	require.True(t, ok)
	assert.Equal(
		t,
		"p@ss word+1",
		password,
		"a space or plus in a password must reach Postgres unchanged",
	)

	assert.Equal(t, cfg.GetDSN(cfg.Database.Password), cfg.GetMigrationDSN())
}

func TestConfig_GetCacheListenDSN(t *testing.T) {
	t.Parallel()

	cfg := newValidConfig()
	assert.Empty(t, cfg.GetCacheListenDSN(), "no listen host keeps the pool's settings")

	cfg.Database.Cache.ListenHost = "pg-primary.internal"
	parsed, err := url.Parse(cfg.GetCacheListenDSN())
	require.NoError(t, err)
	assert.Equal(t, "pg-primary.internal:5432", parsed.Host)
	assert.Equal(t, "/trenova", parsed.Path)
	assert.Equal(t, "postgres", parsed.User.Username())

	cfg.Database.Cache.ListenPort = 5433
	parsed, err = url.Parse(cfg.GetCacheListenDSN())
	require.NoError(t, err)
	assert.Equal(t, "pg-primary.internal:5433", parsed.Host)
	assert.Equal(t, "disable", parsed.Query().Get("sslmode"))
}
