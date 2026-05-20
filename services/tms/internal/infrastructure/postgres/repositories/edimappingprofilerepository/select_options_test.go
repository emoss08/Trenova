package edimappingprofilerepository

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
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

func TestSelectMappingProfileOptions_AppliesTenantSearchAndFilters(t *testing.T) {
	t.Parallel()

	repo, sqlMock := newEDISelectOptionsTestRepository(t)
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	partnerID := pulid.MustNew("edip_")
	profileID := pulid.MustNew("edimp_")

	sqlMock.ExpectQuery(`SELECT count\(\*\) FROM "edi_mapping_profiles" AS "emp".*JOIN "edi_partners" AS "partner".*emp\.organization_id.*emp\.business_unit_id.*LOWER\(emp\.name\) LIKE.*LOWER\(emp\.description\) LIKE.*emp\.edi_partner_id`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	sqlMock.ExpectQuery(`SELECT "emp"."id", "emp"."business_unit_id", "emp"."organization_id".*FROM "edi_mapping_profiles" AS "emp".*JOIN "edi_partners" AS "partner".*emp\.organization_id.*emp\.business_unit_id.*LOWER\(emp\.name\) LIKE.*LOWER\(emp\.description\) LIKE.*emp\.edi_partner_id`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id",
			"business_unit_id",
			"organization_id",
			"edi_partner_id",
			"name",
			"description",
		}).AddRow(
			profileID,
			buID,
			orgID,
			partnerID,
			"Carrier Mapping",
			"Customer and location mappings",
		))

	result, err := repo.SelectMappingProfileOptions(
		t.Context(),
		&repositories.EDIMappingProfileSelectOptionsRequest{
			SelectQueryRequest: &pagination.SelectQueryRequest{
				TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
				Pagination: pagination.Info{Limit: 10},
				Query:      "carrier",
			},
			PartnerID: partnerID,
		},
	)

	require.NoError(t, err)
	require.Equal(t, 1, result.Total)
	require.Len(t, result.Items, 1)
	require.Equal(t, profileID, result.Items[0].ID)
	require.Equal(t, "Carrier Mapping", result.Items[0].Name)
}
