package fiscalperiodrepository

import (
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
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
