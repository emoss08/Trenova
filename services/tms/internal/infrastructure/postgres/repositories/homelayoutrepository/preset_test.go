package homelayoutrepository

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/emoss08/trenova/internal/core/domain/homelayout"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
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

func testPreset() *homelayout.Preset {
	return &homelayout.Preset{
		ID:             pulid.MustNew("hlp_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		Name:           "Dispatch Home",
		Layout: &homelayout.Layout{Widgets: []homelayout.Widget{{
			ID: "widget_1", Key: homelayout.WidgetAttention, W: 4, H: 4,
		}}},
		// The values an administrator sets when demoting a preset. Both are the
		// zero value, which is what makes them easy to lose.
		IsOrgDefault: false,
		Locked:       false,
		Priority:     0,
		Version:      3,
	}
}

// TestUpdatePresetStatementWritesEveryColumn guards against two bun behaviours
// that each silently drop part of an administrator's save. OmitZero skips
// false and 0 — exactly the values that demote a preset from org-default or
// unlock it. And a single explicit Set() makes bun ignore the model's own
// fields entirely, which would drop name and layout. Neither is usable here,
// so the statement must carry every column from the loaded row.
func TestUpdatePresetStatementWritesEveryColumn(t *testing.T) {
	t.Parallel()

	repo, _ := newTestRepository(t)

	sql := repo.db.DB().
		NewUpdate().
		Model(testPreset()).
		WherePK().
		String()

	for _, column := range []string{
		`"name"`,
		`"description"`,
		`"layout"`,
		`"role_ids"`,
		`"core_responsibility"`,
		`"is_org_default"`,
		`"locked"`,
		`"priority"`,
		`"version"`,
		`"updated_at"`,
	} {
		require.Contains(t, sql, column, "the update must write %s", column)
	}
}

// TestClearOrgDefaultExcludesAndInvalidatesDemotedRows keeps a promotion from
// demoting the very preset being promoted, and pins the version + updated_at
// bump on the rows it does demote — without it, an editor holding a demoted
// preset open would pass the optimistic lock on their next save and silently
// re-promote it.
func TestClearOrgDefaultExcludesAndInvalidatesDemotedRows(t *testing.T) {
	t.Parallel()

	repo, mock := newTestRepository(t)

	mock.ExpectExec(`UPDATE "home_layout_presets".*SET is_org_default = FALSE, version = version \+ 1, updated_at = .*hlp\.is_org_default = TRUE.*hlp\.id != `).
		WillReturnResult(sqlmock.NewResult(0, 2))

	err := clearOrgDefault(
		t.Context(),
		repo.db.DB(),
		pulid.MustNew("org_"),
		pulid.MustNew("bu_"),
		pulid.MustNew("hlp_"),
	)

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestCreatePresetDemotesInsideTheInsertTransaction pins the org-default
// handover to the same transaction as the insert: if the insert is rejected —
// a duplicate name, say — the rollback restores the incumbent default instead
// of leaving the tenant with none.
func TestCreatePresetDemotesInsideTheInsertTransaction(t *testing.T) {
	t.Parallel()

	repo, mock := newTestRepository(t)

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "home_layout_presets".*SET is_org_default = FALSE`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`INSERT INTO "home_layout_presets"`).
		WillReturnRows(sqlmock.NewRows([]string{
			"description", "role_ids", "core_responsibility", "locked", "priority", "created_by",
		}).AddRow("", "{}", "", false, 0, ""))
	mock.ExpectCommit()

	entity := testPreset()
	entity.IsOrgDefault = true

	_, err := repo.CreatePreset(t.Context(), entity)

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestDeletePresetMapsMissingRowsToNotFound: deleting a preset another
// administrator already removed is a not-found, not a version conflict.
func TestDeletePresetMapsMissingRowsToNotFound(t *testing.T) {
	t.Parallel()

	repo, mock := newTestRepository(t)

	mock.ExpectExec(`DELETE FROM "home_layout_presets"`).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := repo.DeletePreset(t.Context(), &repositories.DeleteHomeLayoutPresetRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: pulid.MustNew("org_"),
			BuID:  pulid.MustNew("bu_"),
		},
		PresetID: pulid.MustNew("hlp_"),
	})

	require.Error(t, err)
	require.True(t, errortypes.IsNotFoundError(err))
	require.NoError(t, mock.ExpectationsWereMet())
}
