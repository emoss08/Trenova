//go:build integration

package databasesize_test

import (
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/kernel/adapters/postgres/databasesize"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/datausageports"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/stretchr/testify/require"
)

func TestTheDatabaseSizeIsTheWholeDatabaseMeasuredWithoutATenant(t *testing.T) {
	ctx, admin, dsn, cleanup := seedtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)

	conn := dbtest.NewRLSConnection(t, admin, dsn)
	source := databasesize.New(databasesize.Params{DB: conn}).Source
	require.Equal(t, datausageports.FigureDatabase, source.Figure())

	got, err := source.Measure(t.Context())
	require.NoError(t, err)

	want, err := keel.QueryValue[int64](ctx, admin, "SELECT pg_database_size(current_database())")
	require.NoError(t, err)
	require.Positive(t, got)
	require.InEpsilon(t, want, got, 0.05, "the figure is the database's own size")
}
