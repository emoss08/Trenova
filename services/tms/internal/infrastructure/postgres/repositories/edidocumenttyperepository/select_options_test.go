package edidocumenttyperepository

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"go.uber.org/zap"
)

func newEDISelectOptionsTestRepository(t *testing.T) (*repository, sqlmock.Sqlmock) {
	t.Helper()

	db, sqlMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	sqlMock.MatchExpectationsInOrder(false)

	bunDB := bun.NewDB(db, pgdialect.New())
	t.Cleanup(func() {
		require.NoError(t, sqlMock.ExpectationsWereMet())
		_ = bunDB.Close()
	})

	return &repository{
		db: postgres.NewTestConnection(bunDB),
		l:  zap.NewNop(),
	}, sqlMock
}

func TestSelectDocumentTypeOptions_AppliesSearchAndFilters(t *testing.T) {
	t.Parallel()

	repo, sqlMock := newEDISelectOptionsTestRepository(t)
	documentTypeID := pulid.MustNew("edidt_")

	sqlMock.ExpectQuery(`SELECT count\(\*\) FROM "edi_document_types" AS "edt".*edt\.transaction_set.*edt\.direction.*edt\.status.*lower\(edt\.code\) LIKE`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	sqlMock.ExpectQuery(`SELECT "edt"."id", "edt"."code", "edt"."name".*FROM "edi_document_types" AS "edt".*edt\.transaction_set.*edt\.direction.*edt\.status.*lower\(edt\.code\) LIKE`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id",
			"code",
			"name",
			"standard",
			"transaction_set",
			"direction",
			"default_version",
			"status",
		}).AddRow(
			documentTypeID,
			"204",
			"Motor Carrier Load Tender",
			edi.EDIStandardX12,
			edi.TransactionSet204,
			edi.DocumentDirectionOutbound,
			edi.DefaultX12204Version,
			edi.DocumentStatusActive,
		))

	result, err := repo.SelectDocumentTypeOptions(
		t.Context(),
		&repositories.EDIDocumentTypeSelectOptionsRequest{
			SelectQueryRequest: &pagination.SelectQueryRequest{
				Pagination: pagination.Info{Limit: 10},
				Query:      "204",
			},
			TransactionSet: edi.TransactionSet204,
			Direction:      edi.DocumentDirectionOutbound,
			Status:         edi.DocumentStatusActive,
		},
	)

	require.NoError(t, err)
	require.Equal(t, 1, result.Total)
	require.Len(t, result.Items, 1)
	require.Equal(t, documentTypeID, result.Items[0].ID)
	require.Equal(t, "204", result.Items[0].Code)
	require.Equal(t, "Motor Carrier Load Tender", result.Items[0].Name)
	require.Equal(t, edi.DefaultX12204Version, result.Items[0].DefaultVersion)
}
