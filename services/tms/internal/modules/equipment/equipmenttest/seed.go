//go:build integration

package equipmenttest

import (
	"context"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeder"
	"github.com/emoss08/trenova/internal/infrastructure/database/seeds"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbpool"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

type Tenant struct {
	Ctx  context.Context
	Info pagination.TenantInfo
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
	b := seedtest.SeedAdditionalTenant(t, ctx, admin, "ZZB")

	return &Database{
		Conn:  dbtest.NewRLSConnection(t, admin, dsn),
		Admin: dbpool.KeelOf(admin.DB),
		A:     tenantOf(t, a.Organization.ID, a.BusinessUnit.ID, a.User.ID),
		B:     tenantOf(t, b.Organization.ID, b.BusinessUnit.ID, b.User.ID),
	}
}

func NewDevelopmentDatabase(t *testing.T) *Database {
	t.Helper()

	ctx, admin, dsn, cleanup := seedtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)

	registry := seeder.NewRegistry()
	seeds.Register(registry)
	engine := seeder.NewEngine(admin, registry, &config.Config{
		System: config.SystemConfig{SystemUserPassword: "integration-system-password"},
	})
	_, err := engine.Execute(
		ctx,
		seeder.ExecuteOptions{Environment: common.EnvDevelopment, Force: true},
	)
	require.NoError(t, err)

	db := &Database{
		Conn:  dbtest.NewRLSConnection(t, admin, dsn),
		Admin: dbpool.KeelOf(admin.DB),
	}

	var orgID, buID string
	require.NoError(t, db.Admin.Pool().QueryRow(ctx,
		"SELECT organization_id, business_unit_id FROM trailers ORDER BY id LIMIT 1",
	).Scan(&orgID, &buID))
	db.A = tenantOf(t, pulid.ID(orgID), pulid.ID(buID), pulid.Nil)

	return db
}

func tenantOf(t *testing.T, orgID, buID, userID pulid.ID) Tenant {
	t.Helper()

	info := pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID}

	return Tenant{Ctx: dbscope.WithTenant(t.Context(), info.DBTenant()), Info: info}
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
