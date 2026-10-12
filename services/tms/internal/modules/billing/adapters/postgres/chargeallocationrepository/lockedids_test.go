package chargeallocationrepository

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"go.uber.org/zap"
)

// LockedIDs joins invoices, which also carry an "id" column, so the selected
// column must be qualified or Postgres refuses the query as ambiguous and every
// shipment save with an existing allocation fails validation.
func TestLockedIDsQualifiesTheAllocationIDAgainstTheInvoiceJoin(t *testing.T) {
	t.Parallel()

	db, dbMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)

	bunDB := bun.NewDB(db, pgdialect.New())
	t.Cleanup(func() {
		dbMock.ExpectClose()
		require.NoError(t, bunDB.Close())
	})

	repo := &repository{db: postgres.NewTestConnection(bunDB), l: zap.NewNop()}
	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	lockedID := pulid.MustNew("chal_")
	openID := pulid.MustNew("chal_")

	dbMock.ExpectQuery(
		`(?s)SELECT chal\.id FROM "charge_allocations" AS "chal" JOIN invoices AS inv ON \(inv\.id = chal\.invoice_id\) AND \(inv\.organization_id = chal\.organization_id\) AND \(inv\.business_unit_id = chal\.business_unit_id\) WHERE .*inv\.status = 'Posted'`,
	).WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(lockedID.String()))

	locked, err := repo.LockedIDs(t.Context(), tenantInfo, []pulid.ID{lockedID, openID})

	require.NoError(t, err)
	assert.Contains(t, locked, lockedID)
	assert.NotContains(t, locked, openID)
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestLockedIDsSkipsTheQueryForNoIDs(t *testing.T) {
	t.Parallel()

	db, dbMock, err := sqlmock.New()
	require.NoError(t, err)
	bunDB := bun.NewDB(db, pgdialect.New())
	t.Cleanup(func() {
		dbMock.ExpectClose()
		require.NoError(t, bunDB.Close())
	})
	repo := &repository{db: postgres.NewTestConnection(bunDB), l: zap.NewNop()}

	locked, err := repo.LockedIDs(t.Context(), pagination.TenantInfo{}, nil)

	require.NoError(t, err)
	assert.Empty(t, locked)
	assert.NoError(t, dbMock.ExpectationsWereMet())
}
