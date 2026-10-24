//go:build integration

package schemaversion_test

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/keel/migrate"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/schemaversion"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestRequirePassesADatabaseAtThisBuildsVersion(t *testing.T) {
	ctx, _, dsn, cleanup := seedtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)

	core, logs := observer.New(zap.InfoLevel)
	err := schemaversion.Require(ctx, schemaversion.RequireParams{
		DSN:     dsn,
		Version: schema.Version,
		Logger:  zap.New(core),
	})

	require.NoError(t, err)
	assert.Zero(t, logs.Len(), "a current database starts without waiting")
}

func TestRequireGivesUpOnADatabaseBehindThisBuild(t *testing.T) {
	_, _, dsn, cleanup := seedtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	core, logs := observer.New(zap.InfoLevel)
	err := schemaversion.Require(ctx, schemaversion.RequireParams{
		DSN:     dsn,
		Version: "29991231235959",
		Logger:  zap.New(core),
	})

	require.ErrorIs(t, err, migrate.ErrBehind)
	assert.Contains(t, err.Error(), "run trenova-db up")
	assert.Contains(t, err.Error(), schema.Version, "the error names the version the database has")
	assert.Equal(t, 1, logs.FilterMessageSnippet("waiting for trenova-db").Len())
}

func TestRequireWaitsForTheDeployJob(t *testing.T) {
	ctx, db, dsn, cleanup := seedtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)

	_, err := keel.Exec(ctx, db, `DELETE FROM keel_migrations WHERE version = $1`, schema.Version)
	require.NoError(t, err)

	go func() {
		time.Sleep(time.Second)
		_, _ = keel.Exec(context.Background(), db,
			`INSERT INTO keel_migrations (version, name, source, kind, checksum)
			 VALUES ($1, 'catch_up', 'core', 'versioned', '')`, schema.Version)
	}()

	waitCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	core, logs := observer.New(zap.InfoLevel)
	err = schemaversion.Require(waitCtx, schemaversion.RequireParams{
		DSN:     dsn,
		Version: schema.Version,
		Logger:  zap.New(core),
	})

	require.NoError(t, err)
	assert.Equal(t, 1, logs.FilterMessageSnippet("reached this build's schema version").Len())
}
