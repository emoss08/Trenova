package config_test

import (
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/dbdialect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetDialectDefaultsToPostgres(t *testing.T) {
	cfg := &config.Config{}
	assert.Equal(t, dbdialect.Postgres, cfg.Database.GetDialect())

	cfg.Database.Driver = "sqlite"
	assert.Equal(t, dbdialect.SQLite, cfg.Database.GetDialect())

	cfg.Database.Driver = "nonsense"
	assert.Equal(t, dbdialect.Postgres, cfg.Database.GetDialect(),
		"an unparsable driver must fall back to the reference dialect")
}

func TestSQLiteDSNCarriesPragmas(t *testing.T) {
	cfg := &config.Config{}
	cfg.Database.Driver = "sqlite"
	cfg.Database.SQLite.Path = "/tmp/trenova-dev.db"
	cfg.Database.SQLite.BusyTimeout = 3 * time.Second

	dsn := cfg.GetDSN("ignored")

	assert.True(t, strings.HasPrefix(dsn, "file:/tmp/trenova-dev.db?"), "got %q", dsn)
	assert.Contains(t, dsn, "_pragma=journal_mode(WAL)")
	assert.Contains(t, dsn, "_pragma=synchronous(NORMAL)")
	assert.Contains(t, dsn, "_pragma=busy_timeout(3000)")
	assert.Contains(t, dsn, "_pragma=foreign_keys(1)")
	assert.Contains(t, dsn, "_txlock=immediate",
		"write transactions must take the lock up front so busy_timeout applies")
	assert.NotContains(t, dsn, "ignored", "the password must never reach a SQLite DSN")
}

func TestSQLiteForeignKeysCanBeDisabled(t *testing.T) {
	disabled := false

	cfg := &config.Config{}
	cfg.Database.Driver = "sqlite"
	cfg.Database.SQLite.ForeignKeys = &disabled

	assert.Contains(t, cfg.GetDSN(""), "_pragma=foreign_keys(0)")
}

func TestSQLiteDSNMaskedHidesNothingSensitive(t *testing.T) {
	cfg := &config.Config{}
	cfg.Database.Driver = "sqlite"
	cfg.Database.SQLite.Path = "dev.db"

	assert.Equal(t, "sqlite://dev.db", cfg.GetDSNMasked())
}

func TestPostgresDSNUnchanged(t *testing.T) {
	cfg := &config.Config{}
	cfg.App.Name = "trenova"
	cfg.Database.Host = "localhost"
	cfg.Database.Port = 5432
	cfg.Database.Name = "trenova_go_db"
	cfg.Database.User = "postgres"
	cfg.Database.SSLMode = "disable"

	dsn := cfg.GetDSN("secret")

	require.True(t, strings.HasPrefix(dsn, "postgres://postgres:secret@localhost:5432/trenova_go_db"))
	assert.Contains(t, dsn, "sslmode=disable")
	assert.NotContains(t, cfg.GetDSNMasked(), "secret")
}

func TestSQLiteDefaults(t *testing.T) {
	sqlite := &config.SQLiteConfig{}

	assert.Equal(t, "trenova.db", sqlite.GetPath())
	assert.Equal(t, "WAL", sqlite.GetJournalMode())
	assert.Equal(t, "NORMAL", sqlite.GetSynchronous())
	assert.Equal(t, 5*time.Second, sqlite.GetBusyTimeout())
	assert.True(t, sqlite.GetForeignKeys())
	assert.Positive(t, sqlite.GetCacheSizeKB())
}
