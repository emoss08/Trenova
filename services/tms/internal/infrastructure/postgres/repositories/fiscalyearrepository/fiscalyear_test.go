package fiscalyearrepository

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/emoss08/trenova/internal/core/domain/fiscalyear"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"go.uber.org/zap"
)

func TestUpdateWritesOnlyEditableColumnsIncludingZeroValues(t *testing.T) {
	t.Parallel()

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

	repo := &repository{db: postgres.NewTestConnection(bunDB), l: zap.NewNop()}
	entity := &fiscalyear.FiscalYear{
		ID:                    pulid.MustNew("fy_"),
		OrganizationID:        pulid.MustNew("org_"),
		BusinessUnitID:        pulid.MustNew("bu_"),
		Status:                fiscalyear.StatusPermanentlyClosed,
		Name:                  "FY 2026",
		Description:           "",
		AllowAdjustingEntries: false,
		IsCurrent:             true,
		Version:               7,
	}

	mock.ExpectQuery("UPDATE").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(entity.ID.String()))

	_, err = repo.Update(t.Context(), entity)
	require.NoError(t, err)
	require.Len(t, captured, 1)

	sql := captured[0]
	assert.Contains(t, sql, `"allow_adjusting_entries" = FALSE`)
	assert.Contains(t, sql, `"description" = DEFAULT`)
	assert.Contains(t, sql, `"version" = 8`)
	assert.Contains(t, sql, `fy.version = 7`)
	assert.NotContains(t, sql, `"status" =`)
	assert.NotContains(t, sql, `"is_current" =`)
	assert.NotContains(t, sql, `"start_date" =`)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCloseBumpsVersionSoOpenEditorsRefresh(t *testing.T) {
	t.Parallel()

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

	repo := &repository{db: postgres.NewTestConnection(bunDB), l: zap.NewNop()}
	id := pulid.MustNew("fy_")

	mock.ExpectQuery("UPDATE").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(id.String()))

	_, err = repo.Close(t.Context(), repositories.CloseFiscalYearRequest{
		ID: id,
		TenantInfo: pagination.TenantInfo{
			OrgID: pulid.MustNew("org_"),
			BuID:  pulid.MustNew("bu_"),
		},
		ClosedByID: pulid.MustNew("usr_"),
		ClosedAt:   1_798_761_600,
	})
	require.NoError(t, err)
	require.Len(t, captured, 1)
	assert.Contains(t, captured[0], "version = version + 1")
	assert.Contains(t, captured[0], "updated_at = ")
	require.NoError(t, mock.ExpectationsWereMet())
}
