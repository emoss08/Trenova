//go:build integration

package ratingdb

import (
	"context"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

type Tenant struct {
	Ctx  context.Context
	Info pagination.TenantInfo
	Data *seedtest.TestData
}

type Database struct {
	Conn  *postgres.Connection
	Admin *keel.DB
	A     Tenant
	B     Tenant
}

func NewDatabase(t *testing.T) *Database {
	t.Helper()

	ctx, admin, dsn, cleanup := seedtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)

	a := seedtest.SeedFullTestData(t, ctx, admin)
	b := seedtest.SeedAdditionalTenant(t, ctx, admin, "RTB")

	return &Database{
		Conn:  dbtest.NewRLSConnection(t, admin, dsn),
		Admin: admin,
		A:     Tenant{Ctx: a.Ctx(t.Context()), Info: a.Info(), Data: a},
		B:     Tenant{Ctx: b.Ctx(t.Context()), Info: b.Info(), Data: b},
	}
}

func (d *Database) Exec(t *testing.T, sql string, args ...any) {
	t.Helper()

	_, err := d.Admin.Pool().Exec(t.Context(), sql, args...)
	require.NoError(t, err)
}

func Value[T any](t *testing.T, d *Database, sql string, args ...any) T {
	t.Helper()

	rows, err := d.Admin.Pool().Query(t.Context(), sql, args...)
	require.NoError(t, err)

	value, err := pgx.CollectExactlyOneRow(rows, pgx.RowTo[T])
	require.NoError(t, err)

	return value
}
