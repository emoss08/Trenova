package edipartnerrepository

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/domaintypes"
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

func TestSelectPartnerOptions_FiltersByExternalKind(t *testing.T) {
	t.Parallel()

	repo, sqlMock := newEDISelectOptionsTestRepository(t)
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	partnerID := pulid.MustNew("edip_")

	sqlMock.ExpectQuery(`SELECT count\(\*\) FROM "edi_partners" AS "ep".*ep\.organization_id.*ep\.business_unit_id.*LOWER\(ep\.name\) LIKE.*LOWER\(ep\.code\) LIKE.*ep\.kind`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	sqlMock.ExpectQuery(`SELECT "ep"."id", "ep"."business_unit_id", "ep"."organization_id".*FROM "edi_partners" AS "ep".*ep\.organization_id.*ep\.business_unit_id.*LOWER\(ep\.name\) LIKE.*LOWER\(ep\.code\) LIKE.*ep\.kind`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id",
			"business_unit_id",
			"organization_id",
			"kind",
			"status",
			"code",
			"name",
			"internal_organization_id",
			"edi_connection_id",
			"default_transport_id",
			"enabled_for_inbound",
			"enabled_for_outbound",
		}).AddRow(
			partnerID,
			buID,
			orgID,
			edi.PartnerKindExternal,
			domaintypes.StatusActive,
			"EXT",
			"External Partner",
			"",
			"",
			"",
			true,
			true,
		))

	result, err := repo.SelectOptions(
		t.Context(),
		&repositories.EDIPartnerSelectOptionsRequest{
			SelectQueryRequest: &pagination.SelectQueryRequest{
				TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
				Pagination: pagination.Info{Limit: 10},
				Query:      "ext",
			},
			Kind: edi.PartnerKindExternal,
		},
	)

	require.NoError(t, err)
	require.Equal(t, 1, result.Total)
	require.Len(t, result.Items, 1)
	require.Equal(t, partnerID, result.Items[0].ID)
	require.Equal(t, edi.PartnerKindExternal, result.Items[0].Kind)
}
