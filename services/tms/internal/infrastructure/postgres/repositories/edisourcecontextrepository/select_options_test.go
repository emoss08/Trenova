package edisourcecontextrepository

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

func TestSelectSourceContextFieldOptions_AppliesSchemaSearchAndFieldFilters(t *testing.T) {
	t.Parallel()

	repo, sqlMock := newEDISelectOptionsTestRepository(t)
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	schemaID := pulid.MustNew("edisc_")
	fieldID := pulid.MustNew("ediscf_")

	sqlMock.ExpectQuery(`SELECT count\(\*\) FROM "edi_source_context_fields" AS "escf".*escs\.transaction_set.*escs\.direction.*escf\.status.*escf\.repeated.*escf\.path LIKE.*escf\.path ILIKE`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	sqlMock.ExpectQuery(`SELECT "escf".*FROM "edi_source_context_fields" AS "escf".*escs\.transaction_set.*escs\.direction.*escf\.status.*escf\.repeated.*escf\.path LIKE.*escf\.path ILIKE`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id",
			"schema_id",
			"path",
			"source_kind",
			"data_type",
			"repeated",
			"repeat_path",
			"parent_path",
			"display_name",
			"description",
			"status",
		}).AddRow(
			fieldID,
			schemaID,
			"shipment.stops[0].city",
			edi.SourceContextKindShipment,
			edi.SourceContextDataTypeString,
			true,
			"shipment.stops",
			"shipment.stops[]",
			"Stop City",
			"Pickup or delivery city",
			edi.SourceContextFieldStatusActive,
		))

	repeated := true
	result, err := repo.SelectSourceContextFieldOptions(
		t.Context(),
		&repositories.ListEDISourceContextFieldsRequest{
			Filter: &pagination.QueryOptions{
				TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
				Pagination: pagination.Info{Limit: 20},
				Query:      "city",
			},
			TransactionSet: edi.TransactionSet204,
			Direction:      edi.DocumentDirectionOutbound,
			Status:         edi.SourceContextFieldStatusActive,
			Repeated:       &repeated,
			PathPrefix:     "shipment.",
		},
	)

	require.NoError(t, err)
	require.Equal(t, 1, result.Total)
	require.Len(t, result.Items, 1)
	require.Equal(t, fieldID, result.Items[0].ID)
	require.Equal(t, "shipment.stops[0].city", result.Items[0].Path)
	require.Equal(t, "Stop City", result.Items[0].DisplayName)
}
