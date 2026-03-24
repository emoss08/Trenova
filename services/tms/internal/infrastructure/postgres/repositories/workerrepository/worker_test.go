package workerrepository

import (
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
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

func TestUpdateReturnsVersionMismatchWhenRowsAffectedZero(t *testing.T) {
	t.Parallel()

	repo, mock := newTestRepository(t)
	entity := &worker.Worker{
		ID:             pulid.MustNew("wrk_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		StateID:        pulid.MustNew("st_"),
		FirstName:      "Ada",
		LastName:       "Lovelace",
		AddressLine1:   "123 Main St",
		City:           "Boston",
		PostalCode:     "02108",
		Status:         domaintypes.StatusActive,
		Gender:         worker.GenderFemale,
		Version:        7,
	}

	mock.ExpectBegin()
	mock.ExpectQuery(`UPDATE .*workers.*`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
	mock.ExpectRollback()

	updated, err := repo.Update(t.Context(), entity)

	require.Nil(t, updated)
	require.Error(t, err)

	var valErr *errortypes.Error
	require.True(t, errors.As(err, &valErr))
	assert.Equal(t, "version", valErr.Field)
	assert.Equal(t, errortypes.ErrVersionMismatch, valErr.Code)
	assert.Contains(t, valErr.Message, "Worker")

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateMapsRetryableTransactionErrorsToConflict(t *testing.T) {
	t.Parallel()

	repo, mock := newTestRepository(t)
	entity := &worker.Worker{
		ID:             pulid.MustNew("wrk_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		StateID:        pulid.MustNew("st_"),
		FirstName:      "Ada",
		LastName:       "Lovelace",
		AddressLine1:   "123 Main St",
		City:           "Boston",
		PostalCode:     "02108",
		Status:         domaintypes.StatusActive,
		Gender:         worker.GenderFemale,
		Version:        7,
	}

	pgErr := &pgconn.PgError{
		Code:    "40P01",
		Message: "deadlock detected",
	}

	mock.ExpectBegin()
	mock.ExpectQuery(`UPDATE .*workers.*`).
		WillReturnError(pgErr)
	mock.ExpectRollback()

	updated, err := repo.Update(t.Context(), entity)

	require.Nil(t, updated)
	require.Error(t, err)
	assert.True(t, errortypes.IsConflictError(err))

	var conflictErr *errortypes.ConflictError
	require.True(t, errors.As(err, &conflictErr))
	assert.Equal(t, errortypes.ErrResourceInUse, conflictErr.Code)
	assert.Contains(t, conflictErr.Message, "Worker is busy")

	var wrappedPgErr *pgconn.PgError
	require.True(t, errors.As(err, &wrappedPgErr))
	assert.Equal(t, "40P01", wrappedPgErr.Code)

	require.NoError(t, mock.ExpectationsWereMet())
}
