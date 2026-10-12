//go:build integration

package migrator_test

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/database/migrator"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate"
)

const markAppliedProbeMigration = "99991231235959"

func TestAFailedMigrationIsNotRecordedAsApplied(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	fail := true
	migrations := migrate.NewMigrations()
	migrations.Add(migrate.Migration{
		Name: markAppliedProbeMigration,
		Up: func(context.Context, *migrate.Migrator, *migrate.Migration) error {
			if fail {
				return errors.New("statement failed")
			}
			return nil
		},
		Down: func(context.Context, *migrate.Migrator, *migrate.Migration) error { return nil },
	})

	m := migrator.NewBunMigrator(db, migrations)
	require.NoError(t, m.Init(ctx))

	_, err := m.Migrate(ctx)
	require.Error(t, err)
	assert.Zero(t, appliedCount(t, db), "a failed migration must stay pending so a retry runs it")

	fail = false
	group, err := m.Migrate(ctx)
	require.NoError(t, err)
	assert.Len(t, group.Migrations, 1)
	assert.Equal(t, 1, appliedCount(t, db))
}

func appliedCount(t *testing.T, db *bun.DB) int {
	t.Helper()

	var count int
	require.NoError(t, db.NewRaw(
		"SELECT count(*) FROM bun_migrations WHERE name = ?",
		markAppliedProbeMigration,
	).Scan(t.Context(), &count))
	return count
}
