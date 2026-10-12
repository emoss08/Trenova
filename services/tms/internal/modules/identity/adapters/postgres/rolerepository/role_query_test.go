package rolerepository

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"go.uber.org/zap"
)

type recordingRepository struct {
	repo   *repository
	mock   sqlmock.Sqlmock
	issued *[]string
}

func newRecordingRepository(t *testing.T) *recordingRepository {
	t.Helper()

	issued := make([]string, 0, 4)
	matcher := sqlmock.QueryMatcherFunc(func(_, actualSQL string) error {
		issued = append(issued, actualSQL)
		return nil
	})

	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(matcher))
	require.NoError(t, err)
	mock.MatchExpectationsInOrder(true)

	bunDB := bun.NewDB(db, pgdialect.New())
	t.Cleanup(func() {
		require.NoError(t, mock.ExpectationsWereMet())
	})

	return &recordingRepository{
		repo:   &repository{db: postgres.NewTestConnection(bunDB), l: zap.NewNop()},
		mock:   mock,
		issued: &issued,
	}
}

func roleRows(rows ...[]any) *sqlmock.Rows {
	out := sqlmock.NewRows([]string{"id", "name", "parent_role_ids"})
	for _, r := range rows {
		out.AddRow(r[0], r[1], r[2])
	}
	return out
}

func TestGetRolesWithInheritance_CostsTwoRoundTripsAtAnyDepth(t *testing.T) {
	t.Parallel()

	rr := newRecordingRepository(t)

	childID := pulid.MustNew("rol_")
	parentID := pulid.MustNew("rol_")
	grandparentID := pulid.MustNew("rol_")

	rr.mock.ExpectQuery("roles").WillReturnRows(roleRows(
		[]any{childID.String(), "Child", "{" + parentID.String() + "}"},
		[]any{parentID.String(), "Parent", "{" + grandparentID.String() + "}"},
		[]any{grandparentID.String(), "Grandparent", "{}"},
	))

	permRows := sqlmock.NewRows(
		[]string{"id", "role_id", "resource", "operations", "data_scope"},
	)
	for _, roleID := range []pulid.ID{childID, parentID, grandparentID} {
		permRows.AddRow(
			pulid.MustNew("rp_").String(), roleID.String(),
			"shipment", "{read}", "organization",
		)
	}
	rr.mock.ExpectQuery("resource_permissions").WillReturnRows(permRows)

	roles, err := rr.repo.GetRolesWithInheritance(t.Context(), []pulid.ID{childID})
	require.NoError(t, err)

	require.Len(t, roles, 3)
	assert.Len(t, *rr.issued, 2, "expected one roles query and one permissions query")

	for _, role := range roles {
		assert.Len(t, role.Permissions, 1, "role %s lost its permissions", role.Name)
	}
}

func TestGetRolesWithInheritance_ClosureIsADedupingRecursiveCTE(t *testing.T) {
	t.Parallel()

	rr := newRecordingRepository(t)

	rr.mock.ExpectQuery("roles").WillReturnRows(roleRows())

	_, err := rr.repo.GetRolesWithInheritance(
		t.Context(), []pulid.ID{pulid.MustNew("rol_")},
	)
	require.NoError(t, err)

	require.Len(t, *rr.issued, 1)
	sql := (*rr.issued)[0]

	assert.Contains(t, sql, "WITH RECURSIVE")
	assert.Contains(t, sql, "role_closure")
	assert.Contains(t, sql, "ANY(c.parent_role_ids)")
	assert.Contains(t, sql, "SELECT c.id FROM role_closure c")
	assert.Contains(t, sql, "UNION")
	assert.NotContains(t, sql, "UNION ALL")
}

func TestGetRolesWithInheritance_EmptyInputSkipsTheQuery(t *testing.T) {
	t.Parallel()

	rr := newRecordingRepository(t)

	roles, err := rr.repo.GetRolesWithInheritance(t.Context(), nil)
	require.NoError(t, err)

	assert.Empty(t, roles)
	assert.Empty(t, *rr.issued, "an empty role set must not issue a query")
}
