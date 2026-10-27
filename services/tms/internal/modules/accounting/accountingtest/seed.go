//go:build integration

package accountingtest

import (
	"context"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/glaccount/glaccountdb"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/seededtest"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/domaintypes"
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
		Admin: admin,
		A:     tenantOf(t, a.Organization.ID, a.BusinessUnit.ID, a.User.ID),
		B:     tenantOf(t, b.Organization.ID, b.BusinessUnit.ID, b.User.ID),
	}
}

func NewDevelopmentDatabase(t *testing.T) *Database {
	t.Helper()

	ctx, admin, dsn, cleanup := seededtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)

	db := &Database{
		Conn:  dbtest.NewRLSConnection(t, admin, dsn),
		Admin: admin,
	}

	db.A = DevelopmentTenant(t, ctx, admin)

	return db
}

func DevelopmentTenant(t *testing.T, ctx context.Context, admin *keel.DB) Tenant {
	t.Helper()

	var orgID, buID, userID string
	require.NoError(t, admin.Pool().QueryRow(ctx,
		`SELECT gla.organization_id, gla.business_unit_id, u.id
		FROM gl_accounts gla
		JOIN users u ON u.current_organization_id = gla.organization_id
			AND u.business_unit_id = gla.business_unit_id
		ORDER BY gla.id, u.id
		LIMIT 1`,
	).Scan(&orgID, &buID, &userID))

	return tenantOf(t, pulid.ID(orgID), pulid.ID(buID), pulid.ID(userID))
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

func ActiveAccounts(t *testing.T, d *Database, tenant Tenant, n int) []pulid.ID {
	t.Helper()

	gla := glaccountdb.GLAccounts
	ids, err := keel.Into[pulid.ID](tenant.Ctx, d.Admin, gla.Pick(gla.ID).
		Where(gla.Status.Eq(domaintypes.StatusActive)).
		Order(gla.AccountCode.Asc()).
		Limit(n))
	require.NoError(t, err)
	require.Len(t, ids, n)

	return ids
}
