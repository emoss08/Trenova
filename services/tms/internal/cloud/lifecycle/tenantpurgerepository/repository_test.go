package tenantpurgerepository

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const (
	tablesPattern       = `SELECT c.relname AS table_name`
	foreignKeysPattern  = `SELECT con.conname AS constraint_name`
	orgCountPattern     = `SELECT count\(\*\) FROM "organizations"`
	orgExistsPattern    = `SELECT EXISTS \(SELECT .* FROM "organizations"`
	expiredSubPattern   = `FROM "organization_subscriptions" AS "osub" WHERE .*status = 'expired'.* FOR UPDATE`
	tableColumns        = "table_name,has_org,has_bu"
	foreignKeyColumns   = "constraint_name,child_table,parent_table,on_delete,child_columns,parent_columns"
	shipmentsBatch      = `DELETE FROM "shipments" WHERE \(tableoid, ctid\) IN`
	customersBatch      = `DELETE FROM "customers" WHERE \(tableoid, ctid\) IN`
	customersAll        = `DELETE FROM "customers" WHERE "organization_id" = `
	customerLinksDelete = `DELETE FROM "customer_links" WHERE \("customer_id"\) IN \(SELECT "id" FROM "customers"`
)

func purgeTenant() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
}

func columns(spec string) []string {
	out := make([]string, 0, 6)
	start := 0
	for i := 0; i <= len(spec); i++ {
		if i == len(spec) || spec[i] == ',' {
			out = append(out, spec[start:i])
			start = i + 1
		}
	}

	return out
}

func expectPurgeable(sqlMock sqlmock.Sqlmock) {
	sqlMock.ExpectQuery(orgExistsPattern).WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	sqlMock.ExpectQuery(expiredSubPattern).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(pulid.MustNew("osub_").String()))
}

func expectCatalog(sqlMock sqlmock.Sqlmock, fks *sqlmock.Rows) {
	expectPurgeable(sqlMock)
	sqlMock.ExpectQuery(tablesPattern).WillReturnRows(
		sqlmock.NewRows(columns(tableColumns)).
			AddRow("customers", true, true).
			AddRow("shipments", true, true).
			AddRow("audit_entries", true, true),
	)
	sqlMock.ExpectQuery(foreignKeysPattern).WillReturnRows(fks)
	sqlMock.ExpectQuery(orgCountPattern).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
}

func standardForeignKeys() *sqlmock.Rows {
	return sqlmock.NewRows(columns(foreignKeyColumns)).
		AddRow("fk_shipments_customer", "shipments", "customers", "a", "{customer_id}", "{id}").
		AddRow("fk_links_customer", "customer_links", "customers", "r", "{customer_id}", "{id}").
		AddRow("fk_customers_parent", "customers", "customers", "a", "{parent_id}", "{id}")
}

func newPurgeRepository(t *testing.T) (*repository, sqlmock.Sqlmock) {
	t.Helper()

	conn, sqlMock := dbtest.NewSQLMock(t)

	return &repository{db: conn, l: zap.NewNop()}, sqlMock
}

func TestPurgeRowsDeletesChildrenFirstAndClearsRestrictingDependents(t *testing.T) {
	t.Parallel()

	repo, sqlMock := newPurgeRepository(t)
	expectCatalog(sqlMock, standardForeignKeys())
	sqlMock.ExpectExec(shipmentsBatch).WillReturnResult(sqlmock.NewResult(0, 2))
	sqlMock.ExpectExec(customerLinksDelete).WillReturnResult(sqlmock.NewResult(0, 1))
	sqlMock.ExpectExec(customersBatch).WillReturnResult(sqlmock.NewResult(0, 3))

	result, err := repo.PurgeRows(t.Context(), &repositories.PurgeTenantRowsRequest{
		TenantInfo: purgeTenant(),
	})

	require.NoError(t, err)
	assert.True(t, result.Complete)
	assert.Equal(t, int64(6), result.Deleted)
	assert.Equal(t, []repositories.PurgeTableOutcome{
		{Table: "customer_links", Deleted: 1},
		{Table: "customers", Deleted: 3},
		{Table: "shipments", Deleted: 2},
	}, result.Tables)
	assert.Empty(t, result.Blocked)
	require.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestPurgeRowsDeletesASelfReferencingTableInOneStatement(t *testing.T) {
	t.Parallel()

	repo, sqlMock := newPurgeRepository(t)
	expectCatalog(sqlMock, standardForeignKeys())
	sqlMock.ExpectExec(shipmentsBatch).WillReturnResult(sqlmock.NewResult(0, 0))
	sqlMock.ExpectExec(customerLinksDelete).WillReturnResult(sqlmock.NewResult(0, 0))
	sqlMock.ExpectExec(customersBatch).WillReturnError(&pgconn.PgError{
		Code:           pgerrcode.ForeignKeyViolation,
		ConstraintName: "fk_customers_parent",
	})
	sqlMock.ExpectExec(customersAll).WillReturnResult(sqlmock.NewResult(0, 4))
	sqlMock.ExpectExec(customersBatch).WillReturnResult(sqlmock.NewResult(0, 0))

	result, err := repo.PurgeRows(t.Context(), &repositories.PurgeTenantRowsRequest{
		TenantInfo: purgeTenant(),
	})

	require.NoError(t, err)
	assert.True(t, result.Complete)
	assert.Equal(t, int64(4), result.Deleted)
	require.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestPurgeRowsLeavesTablesItsRetentionRulesProtect(t *testing.T) {
	t.Parallel()

	repo, sqlMock := newPurgeRepository(t)
	expectCatalog(sqlMock, sqlmock.NewRows(columns(foreignKeyColumns)))
	sqlMock.ExpectExec(customersBatch).WillReturnError(&pgconn.PgError{
		Code: pgerrcode.InsufficientPrivilege,
	})
	sqlMock.ExpectExec(shipmentsBatch).WillReturnResult(sqlmock.NewResult(0, 1))

	result, err := repo.PurgeRows(t.Context(), &repositories.PurgeTenantRowsRequest{
		TenantInfo: purgeTenant(),
	})

	require.NoError(t, err)
	assert.True(t, result.Complete)
	assert.Equal(t, []string{"customers"}, result.Retained)
	require.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestPurgeRowsStopsAtItsBatchBudget(t *testing.T) {
	t.Parallel()

	repo, sqlMock := newPurgeRepository(t)
	expectCatalog(sqlMock, sqlmock.NewRows(columns(foreignKeyColumns)))
	sqlMock.ExpectExec(customersBatch).WillReturnResult(sqlmock.NewResult(0, 2))
	sqlMock.ExpectExec(customersBatch).WillReturnResult(sqlmock.NewResult(0, 2))

	result, err := repo.PurgeRows(t.Context(), &repositories.PurgeTenantRowsRequest{
		TenantInfo: purgeTenant(),
		BatchSize:  2,
		MaxBatches: 2,
	})

	require.NoError(t, err)
	assert.False(t, result.Complete)
	assert.Equal(t, int64(4), result.Deleted)
	require.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestPurgeRowsReportsTablesBlockedByAnotherTenantTable(t *testing.T) {
	t.Parallel()

	repo, sqlMock := newPurgeRepository(t)
	expectCatalog(sqlMock, sqlmock.NewRows(columns(foreignKeyColumns)).
		AddRow("fk_shipments_customer", "shipments", "customers", "a", "{customer_id}", "{id}"))

	blocked := &pgconn.PgError{Code: pgerrcode.ForeignKeyViolation, ConstraintName: "fk_shipments_customer"}
	sqlMock.ExpectExec(shipmentsBatch).WillReturnResult(sqlmock.NewResult(0, 0))
	sqlMock.ExpectExec(customersBatch).WillReturnError(blocked)

	result, err := repo.PurgeRows(t.Context(), &repositories.PurgeTenantRowsRequest{
		TenantInfo: purgeTenant(),
	})

	require.NoError(t, err)
	assert.False(t, result.Complete)
	assert.Equal(t, []string{"customers"}, result.Blocked)
	require.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestPurgeRowsRequiresATenant(t *testing.T) {
	t.Parallel()

	repo, _ := newPurgeRepository(t)
	_, err := repo.PurgeRows(t.Context(), &repositories.PurgeTenantRowsRequest{})
	require.ErrorIs(t, err, ErrTenantRequired)
}

func TestDeleteTenantKeepsTheOrganizationWhileRetainedRowsBlockIt(t *testing.T) {
	t.Parallel()

	repo, sqlMock := newPurgeRepository(t)
	sqlMock.ExpectQuery(orgExistsPattern).WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	expectPurgeable(sqlMock)
	sqlMock.ExpectExec(`DELETE FROM "cloud_signups"`).WillReturnResult(sqlmock.NewResult(0, 1))
	sqlMock.ExpectExec(`DELETE FROM "organizations"`).WillReturnError(&pgconn.PgError{
		Code:    pgerrcode.InsufficientPrivilege,
		Message: "Modifications are not allowed on audit_entries (append-only table)",
	})

	result, err := repo.DeleteTenant(t.Context(), purgeTenant())

	require.NoError(t, err)
	assert.Equal(t, int64(1), result.SignupsDeleted)
	assert.False(t, result.OrganizationDeleted)
	assert.NotEmpty(t, result.RetainedReason)
	require.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestDeleteTenantDeletesAnExclusiveBusinessUnit(t *testing.T) {
	t.Parallel()

	repo, sqlMock := newPurgeRepository(t)
	sqlMock.ExpectQuery(orgExistsPattern).WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	expectPurgeable(sqlMock)
	sqlMock.ExpectExec(`DELETE FROM "cloud_signups"`).WillReturnResult(sqlmock.NewResult(0, 0))
	sqlMock.ExpectExec(`DELETE FROM "organizations"`).WillReturnResult(sqlmock.NewResult(0, 1))
	sqlMock.ExpectQuery(orgCountPattern).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	sqlMock.ExpectExec(`DELETE FROM "business_units"`).WillReturnResult(sqlmock.NewResult(0, 1))

	result, err := repo.DeleteTenant(t.Context(), purgeTenant())

	require.NoError(t, err)
	assert.True(t, result.OrganizationDeleted)
	assert.True(t, result.BusinessUnitDeleted)
	require.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestPurgeUserDeactivatesAUserItCannotDelete(t *testing.T) {
	t.Parallel()

	repo, sqlMock := newPurgeRepository(t)
	expectPurgeable(sqlMock)
	sqlMock.ExpectQuery(`FROM "user_organization_memberships"`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	sqlMock.ExpectExec(`DELETE FROM "users" AS "usr" WHERE \(usr.id = '.*'\) AND \(\(usr.current_organization_id = '.*' OR usr.business_unit_id = '.*'\)\)`).WillReturnError(&pgconn.PgError{
		Code: pgerrcode.ForeignKeyViolation,
	})
	sqlMock.ExpectExec(`UPDATE "users"`).WillReturnResult(sqlmock.NewResult(0, 1))

	result, err := repo.PurgeUser(t.Context(), &repositories.PurgeTenantUserRequest{
		TenantInfo: purgeTenant(),
		UserID:     pulid.MustNew("usr_"),
	})

	require.NoError(t, err)
	assert.True(t, result.Deactivated)
	assert.False(t, result.Deleted)
	require.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestPurgeUserMovesAUserWhoBelongsElsewhere(t *testing.T) {
	t.Parallel()

	repo, sqlMock := newPurgeRepository(t)
	expectPurgeable(sqlMock)
	sqlMock.ExpectQuery(`FROM "user_organization_memberships"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "organization_id", "business_unit_id"}).
			AddRow(pulid.MustNew("uom_").String(), pulid.MustNew("org_").String(), pulid.MustNew("bu_").String()))
	sqlMock.ExpectExec(`UPDATE "users"`).WillReturnResult(sqlmock.NewResult(0, 1))

	result, err := repo.PurgeUser(t.Context(), &repositories.PurgeTenantUserRequest{
		TenantInfo: purgeTenant(),
		UserID:     pulid.MustNew("usr_"),
	})

	require.NoError(t, err)
	assert.True(t, result.Reassigned)
	require.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestPurgeRowsRefusesAnOrganizationWhoseSubscriptionHasNotExpired(t *testing.T) {
	t.Parallel()

	repo, sqlMock := newPurgeRepository(t)
	sqlMock.ExpectQuery(orgExistsPattern).WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	sqlMock.ExpectQuery(expiredSubPattern).WillReturnRows(sqlmock.NewRows([]string{"id"}))

	_, err := repo.PurgeRows(t.Context(), &repositories.PurgeTenantRowsRequest{TenantInfo: purgeTenant()})

	require.ErrorIs(t, err, repositories.ErrTenantNotPurgeable)
	require.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestPurgeRowsRefusesAnOrganizationOutsideTheNamedBusinessUnit(t *testing.T) {
	t.Parallel()

	repo, sqlMock := newPurgeRepository(t)
	sqlMock.ExpectQuery(orgExistsPattern).WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	_, err := repo.PurgeRows(t.Context(), &repositories.PurgeTenantRowsRequest{TenantInfo: purgeTenant()})

	require.ErrorIs(t, err, repositories.ErrTenantNotPurgeable)
	require.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestPurgeUserRefusesAnOrganizationWhoseSubscriptionHasNotExpired(t *testing.T) {
	t.Parallel()

	repo, sqlMock := newPurgeRepository(t)
	sqlMock.ExpectQuery(orgExistsPattern).WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	sqlMock.ExpectQuery(expiredSubPattern).WillReturnRows(sqlmock.NewRows([]string{"id"}))

	_, err := repo.PurgeUser(t.Context(), &repositories.PurgeTenantUserRequest{
		TenantInfo: purgeTenant(),
		UserID:     pulid.MustNew("usr_"),
	})

	require.ErrorIs(t, err, repositories.ErrTenantNotPurgeable)
	require.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestDeleteTenantRefusesAnOrganizationWhoseSubscriptionHasNotExpired(t *testing.T) {
	t.Parallel()

	repo, sqlMock := newPurgeRepository(t)
	sqlMock.ExpectQuery(orgExistsPattern).WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	sqlMock.ExpectQuery(orgExistsPattern).WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	sqlMock.ExpectQuery(expiredSubPattern).WillReturnRows(sqlmock.NewRows([]string{"id"}))

	_, err := repo.DeleteTenant(t.Context(), purgeTenant())

	require.ErrorIs(t, err, repositories.ErrTenantNotPurgeable)
	require.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestDeleteTenantIsANoOpOnceTheOrganizationIsGone(t *testing.T) {
	t.Parallel()

	repo, sqlMock := newPurgeRepository(t)
	sqlMock.ExpectQuery(orgExistsPattern).WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	result, err := repo.DeleteTenant(t.Context(), purgeTenant())

	require.NoError(t, err)
	assert.True(t, result.OrganizationDeleted)
	assert.False(t, result.BusinessUnitDeleted)
	require.NoError(t, sqlMock.ExpectationsWereMet())
}
