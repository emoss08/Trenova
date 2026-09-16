package invoiceadjustmentrepository

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/emoss08/trenova/internal/core/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"go.uber.org/zap"
)

func newApprovalQueueTestRepo(t *testing.T) (*repository, sqlmock.Sqlmock) {
	t.Helper()

	db, dbMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)

	bunDB := bun.NewDB(db, pgdialect.New())
	t.Cleanup(func() {
		dbMock.ExpectClose()
		require.NoError(t, bunDB.Close())
	})

	return &repository{db: postgres.NewTestConnection(bunDB), l: zap.NewNop()}, dbMock
}

func approvalQueueRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"adjustment_id",
		"kind",
		"submitted_at",
		"created_at",
		"__cursor_value_0",
		"__cursor_value_1",
	})
}

func TestListApprovalQueuePagesByKeysetInsteadOfOffset(t *testing.T) {
	t.Parallel()

	repo, dbMock := newApprovalQueueTestRepo(t)
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	newest := pulid.MustNew("iadj_")
	middle := pulid.MustNew("iadj_")
	oldest := pulid.MustNew("iadj_")

	dbMock.ExpectQuery(
		`(?s)SELECT .*COALESCE\(ia\.submitted_at, ia\.created_at\) AS "__cursor_value_0", ia\.id AS "__cursor_value_1" FROM invoice_adjustments AS ia .*WHERE .*\(ia\.status = 'PendingApproval'\) ORDER BY COALESCE\(ia\.submitted_at, ia\.created_at\) DESC, ia\.id DESC LIMIT 3$`,
	).WillReturnRows(approvalQueueRows().
		AddRow(newest.String(), "WriteOff", int64(1_700_000_300), int64(1_600_000_000), int64(1_700_000_300), newest.String()).
		AddRow(middle.String(), "CreditOnly", nil, int64(1_700_000_200), int64(1_700_000_200), middle.String()).
		AddRow(oldest.String(), "CreditOnly", int64(1_700_000_100), int64(1_600_000_000), int64(1_700_000_100), oldest.String()))

	firstCursor, err := pagination.NewCursorInfo(2, "")
	require.NoError(t, err)
	firstCursor.IncludeTotalCount = false

	first, err := repo.ListApprovalQueue(t.Context(), &repositories.ListApprovalQueueRequest{
		Filter: &pagination.QueryOptions{TenantInfo: tenant},
		Cursor: firstCursor,
	})
	require.NoError(t, err)
	require.Len(t, first.Items, 2)
	assert.True(t, first.HasNextPage)
	assert.Nil(t, first.TotalCount)
	assert.Equal(t, middle, first.Items[1].AdjustmentID)

	values, ok := first.CursorValuesAt(1)
	require.True(t, ok)
	endCursor, err := pagination.EncodeCursorFromEntityWithValues(
		first.Items[1],
		first.CursorSort,
		values,
	)
	require.NoError(t, err)

	dbMock.ExpectQuery(
		`(?s)WHERE .*\(ia\.status = 'PendingApproval'\) AND \(\(COALESCE\(ia\.submitted_at, ia\.created_at\), ia\.id\) < \(1700000200, '` + middle.String() + `'\)\) ORDER BY COALESCE\(ia\.submitted_at, ia\.created_at\) DESC, ia\.id DESC LIMIT 3$`,
	).WillReturnRows(approvalQueueRows().
		AddRow(oldest.String(), "CreditOnly", int64(1_700_000_100), int64(1_600_000_000), int64(1_700_000_100), oldest.String()))

	nextCursor, err := pagination.NewCursorInfo(2, endCursor)
	require.NoError(t, err)
	nextCursor.IncludeTotalCount = false

	next, err := repo.ListApprovalQueue(t.Context(), &repositories.ListApprovalQueueRequest{
		Filter: &pagination.QueryOptions{TenantInfo: tenant},
		Cursor: nextCursor,
	})
	require.NoError(t, err)
	require.Len(t, next.Items, 1)
	assert.Equal(t, oldest, next.Items[0].AdjustmentID)
	assert.False(t, next.HasNextPage)
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestListApprovalQueueCountsOnlyWhenAskedWithTheSameFilters(t *testing.T) {
	t.Parallel()

	repo, dbMock := newApprovalQueueTestRepo(t)
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	submitter := pulid.MustNew("usr_")

	filters := `\(ia\.status = 'PendingApproval'\) AND \(\(ia\.id ILIKE '%ACME%'\) OR \(orig\.number ILIKE '%ACME%'\) OR \(orig\.bill_to_name ILIKE '%ACME%'\) OR \(ia\.reason ILIKE '%ACME%'\) OR \(ia\.policy_reason ILIKE '%ACME%'\) OR \(submitter\.name ILIKE '%ACME%'\)\) AND \(ia\.kind = 'WriteOff'\) AND \(ia\.submitted_by_id = '` + submitter.String() + `'\)`

	dbMock.ExpectQuery(
		`(?s)^SELECT count\(\*\) FROM invoice_adjustments AS ia JOIN invoices AS orig ON .* LEFT JOIN users AS submitter ON submitter\.id = ia\.submitted_by_id WHERE .*` + filters + `$`,
	).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(7))
	dbMock.ExpectQuery(`(?s)LEFT JOIN users AS rejector .*WHERE .*` + filters + ` ORDER BY`).
		WillReturnRows(approvalQueueRows())

	cursor, err := pagination.NewCursorInfo(20, "")
	require.NoError(t, err)

	result, err := repo.ListApprovalQueue(t.Context(), &repositories.ListApprovalQueueRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: tenant,
			Query:      "ACME",
			FieldFilters: []domaintypes.FieldFilter{
				{Field: "kind", Operator: dbtype.OpEqual, Value: string(invoiceadjustment.KindWriteOff)},
				{Field: "submittedById", Operator: dbtype.OpEqual, Value: submitter.String()},
			},
		},
		Cursor: cursor,
	})
	require.NoError(t, err)
	require.NotNil(t, result.TotalCount)
	assert.Equal(t, 7, *result.TotalCount)
	assert.Empty(t, result.Items)
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestListApprovalQueueRefusesCursorsFromAnotherList(t *testing.T) {
	t.Parallel()

	foreignSort := []pagination.CursorSortField{
		{Field: "createdAt", Direction: "desc"},
		{Field: "id", Direction: "desc"},
	}
	id := pulid.MustNew("iadj_")

	tests := []struct {
		name   string
		cursor pagination.Cursor
	}{
		{
			name:   "entity cursor without sort values",
			cursor: pagination.Cursor{ID: id, CreatedAt: 1_700_000_000},
		},
		{
			name: "cursor sorted a different way",
			cursor: pagination.Cursor{
				ID:     id,
				Sort:   foreignSort,
				Values: []any{int64(1_700_000_000), id.String()},
			},
		},
		{
			name: "matching sort with a non-numeric sort value",
			cursor: pagination.Cursor{
				ID:     id,
				Sort:   approvalQueueCursorSort,
				Values: []any{"yesterday", id.String()},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo, dbMock := newApprovalQueueTestRepo(t)
			encoded, err := pagination.EncodeCursor(tt.cursor)
			require.NoError(t, err)
			cursor, err := pagination.NewCursorInfo(20, encoded)
			require.NoError(t, err)

			_, err = repo.ListApprovalQueue(t.Context(), &repositories.ListApprovalQueueRequest{
				Filter: &pagination.QueryOptions{},
				Cursor: cursor,
			})

			var validationErr *errortypes.Error
			require.ErrorAs(t, err, &validationErr)
			assert.Equal(t, "after", validationErr.Field)
			assert.NoError(t, dbMock.ExpectationsWereMet())
		})
	}
}
