package dbcli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/emoss08/keel/seed"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/seeds/fixtures"

	"github.com/emoss08/trenova/schema"
	"github.com/emoss08/trenova/schema/migrations"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func serviceRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	require.NoError(t, err)
	for {
		if _, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		require.NotEqual(t, parent, dir, "no go.mod above the test")
		dir = parent
	}
}

func TestAppAppliesTheMigrationsThenReconcilesRowLevelSecurity(t *testing.T) {
	t.Parallel()

	app := App(Options{Name: "trenova-db"})

	assert.Equal(t, "trenova-db", app.Name)
	require.NoError(t, app.Migrations.Err())
	assert.Equal(t, schema.Version, app.Migrations.Version())
	assert.Equal(t, []string{migrations.ReconcileRLS}, app.Migrate.After)
	assert.Equal(t, StampPath, app.Stamp)
	assert.Len(t, app.Lint, len(migrations.LintRules()))
	assert.Empty(t, app.Dirs, "the deploy binary writes no migration files")
	require.NotEmpty(t, app.Commands)
	assert.Equal(t, EnableVectorCommand.Name, app.Commands[0].Name)
}

func TestDevelopmentAppWritesIntoTheMigrationsDirectory(t *testing.T) {
	t.Parallel()

	dump := []string{"pg_dump", "-d", "{db}"}
	app := App(Options{Name: "trenova-devdb", Development: true, Dump: dump})

	require.Len(t, app.Dirs, 1)
	assert.Equal(t, migrations.Dir, app.Dirs[0].Path)
	assert.Equal(t, dump, app.Dump)
}

func TestTheSchemaStampIsCurrent(t *testing.T) {
	t.Chdir(serviceRoot(t))

	var stdout, stderr bytes.Buffer
	code := App(Options{Name: "trenova-db"}).Run(
		t.Context(),
		[]string{"migrate", "stamp", "-check", "-plain"},
		strings.NewReader(""),
		&stdout,
		&stderr,
	)

	assert.Zerof(t, code,
		"schema/version_gen.go is stale: run go run ./cmd/devdb migrate stamp\n%s%s",
		stdout.String(), stderr.String())
}

func TestDeployAppCarriesOnlyReferenceSeeds(t *testing.T) {
	t.Parallel()

	app := App(Options{Name: "trenova-db"})

	require.NoError(t, app.Seeds.Validate())
	names := make([]string, 0)
	for _, s := range app.Seeds.Seeds() {
		assert.Equalf(t, seed.Reference, s.Kind, "%s is not reference data", s.Name)
		names = append(names, s.Name)
	}
	assert.Contains(t, names, "rls_scope_key")
	assert.Contains(t, names, "administrator_permissions")
	assert.NotNil(t, app.Tenants)
	assert.NotNil(t, app.System)
}

func TestDevelopmentAppAddsTheFixtures(t *testing.T) {
	t.Parallel()

	app := App(Options{Name: "trenova-devdb", Development: true, Fixtures: fixtures.Seeds})

	require.NoError(t, app.Seeds.Validate())
	kinds := map[seed.Kind]int{}
	for _, s := range app.Seeds.Seeds() {
		kinds[s.Kind]++
	}
	assert.Positive(t, kinds[seed.Reference])
	assert.Positive(t, kinds[seed.Fixture])
}

func TestTheDatabaseToolRecordsDataVersionsForItsWrites(t *testing.T) {
	t.Parallel()

	settings := &Settings{name: "trenova-db", load: func() (*config.Config, error) {
		return &config.Config{Database: config.DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			Name:     "trenova",
			User:     "trenova",
			Password: "secret",
			SSLMode:  "disable",
		}}, nil
	}}

	cfg, err := DatabaseConfig(ConfigParams{Name: "trenova-db", Settings: settings})()

	require.NoError(t, err)
	assert.True(t, cfg.DataVersions.Enabled,
		"seed writes must move the data versions the API's result cache reads")
}
