package edipartnerdocumentprofilerepository

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

func TestSelectPartnerDocumentProfileOptions_AppliesTenantSearchAndFilters(t *testing.T) {
	t.Parallel()

	repo, sqlMock := newEDISelectOptionsTestRepository(t)
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	partnerID := pulid.MustNew("edip_")
	profileID := pulid.MustNew("edipdp_")

	sqlMock.ExpectQuery(`SELECT count\(\*\) FROM "edi_partner_document_profiles" AS "epdp".*epdp\.organization_id.*epdp\.business_unit_id.*epdp\.transaction_set.*epdp\.direction.*epdp\.status.*epdp\.edi_partner_id.*lower\(epdp\.name\) LIKE`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	sqlMock.ExpectQuery(`SELECT "epdp".*FROM "edi_partner_document_profiles" AS "epdp".*epdp\.organization_id.*epdp\.business_unit_id.*epdp\.transaction_set.*epdp\.direction.*epdp\.status.*epdp\.edi_partner_id.*lower\(epdp\.name\) LIKE`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id",
			"business_unit_id",
			"organization_id",
			"edi_partner_id",
			"document_type_id",
			"template_id",
			"template_version_id",
			"name",
			"status",
			"direction",
			"standard",
			"transaction_set",
			"x12_version_override",
			"functional_group_id",
			"envelope",
			"acknowledgment",
			"validation_mode",
			"partner_settings",
			"partner_settings_schema_id",
			"partner_settings_schema_version",
			"version",
			"created_at",
			"updated_at",
		}).AddRow(
			profileID,
			buID,
			orgID,
			"",
			"",
			"",
			"",
			"Outbound Profile",
			edi.DocumentStatusActive,
			edi.DocumentDirectionOutbound,
			edi.EDIStandardX12,
			edi.TransactionSet204,
			"",
			"SM",
			[]byte(`{"elementSeparator":"*","segmentTerminator":"~","componentSeparator":">","repetitionSeparator":"^"}`),
			[]byte(`{"expected":false}`),
			edi.ValidationModeStrict,
			[]byte(`{}`),
			"",
			0,
			0,
			1,
			1,
		))

	result, err := repo.SelectPartnerDocumentProfileOptions(
		t.Context(),
		&repositories.EDIPartnerDocumentProfileSelectOptionsRequest{
			SelectQueryRequest: &pagination.SelectQueryRequest{
				TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
				Pagination: pagination.Info{Limit: 10},
				Query:      "profile",
			},
			TransactionSet: edi.TransactionSet204,
			Direction:      edi.DocumentDirectionOutbound,
			Status:         edi.DocumentStatusActive,
			PartnerID:      partnerID,
		},
	)

	require.NoError(t, err)
	require.Equal(t, 1, result.Total)
	require.Len(t, result.Items, 1)
	require.Equal(t, profileID, result.Items[0].ID)
	require.Equal(t, "Outbound Profile", result.Items[0].Name)
	require.Equal(t, edi.ValidationModeStrict, result.Items[0].ValidationMode)
}
