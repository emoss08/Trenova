package servicetyperepository

import (
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/emoss08/trenova/internal/core/domain/servicetype"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
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
	entity := &servicetype.ServiceType{
		ID:             pulid.MustNew("st_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		Code:           "LTL",
		Description:    "LTL Service",
		Status:         domaintypes.StatusActive,
		Version:        4,
	}

	mock.ExpectQuery(`UPDATE .*service_types.*`).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	updated, err := repo.Update(t.Context(), entity)

	require.Nil(t, updated)
	require.Error(t, err)

	var valErr *errortypes.Error
	require.True(t, errors.As(err, &valErr))
	assert.Equal(t, "version", valErr.Field)
	assert.Equal(t, errortypes.ErrVersionMismatch, valErr.Code)
	assert.Contains(t, valErr.Message, "ServiceType")

	require.NoError(t, mock.ExpectationsWereMet())
}
