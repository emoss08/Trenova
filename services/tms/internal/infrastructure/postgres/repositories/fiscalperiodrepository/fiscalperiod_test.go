package fiscalperiodrepository

import (
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/emoss08/trenova/internal/core/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"go.uber.org/zap"
)

func newTestRepository(t *testing.T) (*repository, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)

	bunDB := bun.NewDB(db, pgdialect.New())
	t.Cleanup(func() {
		mock.ExpectClose()
		require.NoError(t, bunDB.Close())
	})

	return &repository{
		db: postgres.NewTestConnection(bunDB),
		l:  zap.NewNop(),
	}, mock
}

func TestGetByIDForUpdateReturnsRetryableLockError(t *testing.T) {
	t.Parallel()

	repo, mock := newTestRepository(t)
	req := repositories.GetFiscalPeriodByIDRequest{
		ID: pulid.MustNew("fp_"),
		TenantInfo: pagination.TenantInfo{
			OrgID: pulid.MustNew("org_"),
			BuID:  pulid.MustNew("bu_"),
		},
	}

	pgErr := &pgconn.PgError{
		Code:    "55P03",
		Message: "could not obtain lock on row in relation \"fiscal_periods\"",
	}

	mock.ExpectQuery(`SELECT .*FROM "fiscal_periods" AS "fp".*FOR UPDATE NOWAIT`).
		WillReturnError(pgErr)

	entity, err := repo.GetByIDForUpdate(t.Context(), req)

	require.Nil(t, entity)
	require.Error(t, err)

	var wrappedPgErr *pgconn.PgError
	require.True(t, errors.As(err, &wrappedPgErr))
	assert.Equal(t, "55P03", wrappedPgErr.Code)

	require.NoError(t, mock.ExpectationsWereMet())
}

func newCapturingRepository(t *testing.T) (*repository, sqlmock.Sqlmock, *[]string) {
	t.Helper()

	captured := make([]string, 0, 1)
	matcher := sqlmock.QueryMatcherFunc(func(_, actual string) error {
		captured = append(captured, actual)
		return nil
	})

	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(matcher))
	require.NoError(t, err)

	bunDB := bun.NewDB(db, pgdialect.New())
	t.Cleanup(func() {
		mock.ExpectClose()
		require.NoError(t, bunDB.Close())
	})

	return &repository{
		db: postgres.NewTestConnection(bunDB),
		l:  zap.NewNop(),
	}, mock, &captured
}

func testTenant() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
}

func TestUpdateWritesZeroValuedEditableColumns(t *testing.T) {
	t.Parallel()

	repo, mock, captured := newCapturingRepository(t)
	tenant := testTenant()
	entity := &fiscalperiod.FiscalPeriod{
		ID:                    pulid.MustNew("fp_"),
		OrganizationID:        tenant.OrgID,
		BusinessUnitID:        tenant.BuID,
		Name:                  "Period 1",
		PeriodNumber:          1,
		PeriodType:            fiscalperiod.PeriodTypeMonth,
		Status:                fiscalperiod.StatusOpen,
		AllowAdjustingEntries: false,
		AdjustmentDeadline:    nil,
		Version:               3,
	}

	mock.ExpectQuery("UPDATE").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(entity.ID.String()))

	_, err := repo.Update(t.Context(), entity)
	require.NoError(t, err)
	require.Len(t, *captured, 1)

	sql := (*captured)[0]
	assert.Contains(t, sql, `"allow_adjusting_entries" = FALSE`)
	assert.Contains(t, sql, `"adjustment_deadline" = DEFAULT`)
	assert.Contains(t, sql, `"version" = 4`)
	assert.Contains(t, sql, `fp.version = 3`)
	assert.NotContains(t, sql, `"status" =`)
	assert.NotContains(t, sql, `"closed_at" =`)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLockRecordsLockStamp(t *testing.T) {
	t.Parallel()

	repo, mock, captured := newCapturingRepository(t)
	tenant := testTenant()
	userID := pulid.MustNew("usr_")
	id := pulid.MustNew("fp_")

	mock.ExpectQuery("UPDATE").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(id.String()))

	_, err := repo.Lock(t.Context(), repositories.LockFiscalPeriodRequest{
		ID:         id,
		TenantInfo: tenant,
		LockedByID: userID,
		LockedAt:   1_780_000_000,
	})
	require.NoError(t, err)
	require.Len(t, *captured, 1)

	sql := (*captured)[0]
	assert.Contains(t, sql, `status = 'Locked'`)
	assert.Contains(t, sql, `locked_at = 1780000000`)
	assert.Contains(t, sql, `locked_by_id = '`+userID.String()+`'`)
	assert.Contains(t, sql, `version = version + 1`)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestActivateOpensPeriod(t *testing.T) {
	t.Parallel()

	repo, mock, captured := newCapturingRepository(t)
	id := pulid.MustNew("fp_")

	mock.ExpectQuery("UPDATE").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(id.String()))

	_, err := repo.Activate(t.Context(), repositories.ActivateFiscalPeriodRequest{
		ID:         id,
		TenantInfo: testTenant(),
	})
	require.NoError(t, err)
	require.Len(t, *captured, 1)
	assert.Contains(t, (*captured)[0], `status = 'Open'`)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCountUnclosedPeriodsIncludesLocked(t *testing.T) {
	t.Parallel()

	repo, mock, captured := newCapturingRepository(t)
	tenant := testTenant()

	mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))

	count, err := repo.CountUnclosedPeriodsByFiscalYear(
		t.Context(),
		repositories.CountUnclosedPeriodsByFiscalYearRequest{
			FiscalYearID: pulid.MustNew("fy_"),
			OrgID:        tenant.OrgID,
			BuID:         tenant.BuID,
		},
	)
	require.NoError(t, err)
	assert.Equal(t, 2, count)
	require.Len(t, *captured, 1)
	assert.Contains(t, (*captured)[0], `fp.status IN ('Open', 'Locked')`)
	require.NoError(t, mock.ExpectationsWereMet())
}
