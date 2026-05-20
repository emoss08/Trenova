package editemplaterepository

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

func TestSelectTemplateOptions_AppliesTenantSearchAndFilters(t *testing.T) {
	t.Parallel()

	repo, sqlMock := newEDISelectOptionsTestRepository(t)
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	templateID := pulid.MustNew("editpl_")

	sqlMock.ExpectQuery(`SELECT count\(\*\) FROM "edi_templates" AS "et".*et\.organization_id.*et\.business_unit_id.*et\.transaction_set.*et\.direction.*et\.status.*lower\(et\.name\) LIKE`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	sqlMock.ExpectQuery(`SELECT "et"."id", "et"."business_unit_id", "et"."organization_id".*FROM "edi_templates" AS "et".*et\.organization_id.*et\.business_unit_id.*et\.transaction_set.*et\.direction.*et\.status.*lower\(et\.name\) LIKE`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id",
			"business_unit_id",
			"organization_id",
			"document_type_id",
			"name",
			"description",
			"direction",
			"standard",
			"transaction_set",
			"status",
		}).AddRow(
			templateID,
			buID,
			orgID,
			"",
			"Outbound 204",
			"Load tender template",
			edi.DocumentDirectionOutbound,
			edi.EDIStandardX12,
			edi.TransactionSet204,
			edi.TemplateStatusDraft,
		))

	result, err := repo.SelectTemplateOptions(
		t.Context(),
		&repositories.EDITemplateSelectOptionsRequest{
			SelectQueryRequest: &pagination.SelectQueryRequest{
				TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
				Pagination: pagination.Info{Limit: 10},
				Query:      "load",
			},
			TransactionSet: edi.TransactionSet204,
			Direction:      edi.DocumentDirectionOutbound,
			Status:         edi.TemplateStatusDraft,
		},
	)

	require.NoError(t, err)
	require.Equal(t, 1, result.Total)
	require.Len(t, result.Items, 1)
	require.Equal(t, templateID, result.Items[0].ID)
	require.Equal(t, "Outbound 204", result.Items[0].Name)
	require.Equal(t, "Load tender template", result.Items[0].Description)
	require.Equal(t, edi.TemplateStatusDraft, result.Items[0].Status)
}
