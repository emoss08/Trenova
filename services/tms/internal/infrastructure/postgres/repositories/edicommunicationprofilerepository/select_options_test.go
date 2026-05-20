package edicommunicationprofilerepository

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

func TestSelectCommunicationProfileOptions_AppliesTenantSearchAndFilters(t *testing.T) {
	t.Parallel()

	repo, sqlMock := newEDISelectOptionsTestRepository(t)
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	partnerID := pulid.MustNew("edip_")
	profileID := pulid.MustNew("edicp_")

	sqlMock.ExpectQuery(`SELECT count\(\*\) FROM "edi_communication_profiles" AS "ecp".*ecp\.organization_id.*ecp\.business_unit_id.*ecp\.status.*ecp\.method.*ecp\.edi_partner_id.*lower\(ecp\.name\) LIKE`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	sqlMock.ExpectQuery(`SELECT "ecp"."id", "ecp"."business_unit_id", "ecp"."organization_id".*FROM "edi_communication_profiles" AS "ecp".*ecp\.organization_id.*ecp\.business_unit_id.*ecp\.status.*ecp\.method.*ecp\.edi_partner_id.*lower\(ecp\.name\) LIKE`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id",
			"business_unit_id",
			"organization_id",
			"edi_connection_id",
			"edi_partner_id",
			"method",
			"status",
			"name",
			"description",
		}).AddRow(
			profileID,
			buID,
			orgID,
			"",
			partnerID,
			edi.ConnectionMethodSFTP,
			domaintypes.StatusActive,
			"Carrier SFTP",
			"Outbound transport",
		))

	result, err := repo.SelectProfileOptions(
		t.Context(),
		&repositories.EDICommunicationProfileSelectOptionsRequest{
			SelectQueryRequest: &pagination.SelectQueryRequest{
				TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
				Pagination: pagination.Info{Limit: 10},
				Query:      "carrier",
			},
			Status:    domaintypes.StatusActive,
			Method:    edi.ConnectionMethodSFTP,
			PartnerID: partnerID,
		},
	)

	require.NoError(t, err)
	require.Equal(t, 1, result.Total)
	require.Len(t, result.Items, 1)
	require.Equal(t, profileID, result.Items[0].ID)
	require.Equal(t, "Carrier SFTP", result.Items[0].Name)
}
