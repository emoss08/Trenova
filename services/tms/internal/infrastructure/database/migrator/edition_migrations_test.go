package migrator_test

import (
	"testing"
	"testing/fstest"

	"github.com/emoss08/trenova/internal/infrastructure/database/migrator"
	pgmigrations "github.com/emoss08/trenova/internal/infrastructure/postgres/migrations"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const editionMigration = "20991231000000"

func editionSet(name string) fstest.MapFS {
	return fstest.MapFS{
		name + "_edition_example.tx.up.sql": {
			Data: []byte("CREATE TABLE edition_example (id text PRIMARY KEY);"),
		},
		name + "_edition_example.tx.down.sql": {
			Data: []byte("DROP TABLE edition_example;"),
		},
	}
}

func TestPostgresMigrationsWithoutAnEditionAreThePublicSet(t *testing.T) {
	t.Parallel()

	public, err := pgmigrations.Migrations()
	require.NoError(t, err)

	merged, err := migrator.PostgresMigrations()
	require.NoError(t, err)

	assert.Equal(t, len(public.Sorted()), len(merged.Sorted()))
}

func TestPostgresMigrationsRunTheEditionSetAfterThePublicOne(t *testing.T) {
	t.Parallel()

	public, err := pgmigrations.Migrations()
	require.NoError(t, err)

	merged, err := migrator.PostgresMigrations(editionSet(editionMigration))
	require.NoError(t, err)

	sorted := merged.Sorted()
	require.Len(t, sorted, len(public.Sorted())+1)

	last := sorted[len(sorted)-1]
	assert.Equal(t, editionMigration, last.Name)
	assert.Equal(t, "edition_example", last.Comment)
	assert.NotNil(t, last.Up)
	assert.NotNil(t, last.Down)
}

func TestPostgresMigrationsRefuseADuplicateName(t *testing.T) {
	t.Parallel()

	public, err := pgmigrations.Migrations()
	require.NoError(t, err)
	existing := public.Sorted()[0].Name

	_, err = migrator.PostgresMigrations(editionSet(existing))
	require.ErrorIs(t, err, migrator.ErrDuplicateMigration)

	_, err = migrator.PostgresMigrations(
		editionSet(editionMigration),
		editionSet(editionMigration),
	)
	require.ErrorIs(t, err, migrator.ErrDuplicateMigration)
}
