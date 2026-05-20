package edipartnersettingrepository

import (
	"os"
	"strings"
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

func TestSelectPartnerSettingFieldOptions_AppliesSchemaSearchAndFieldFilters(t *testing.T) {
	t.Parallel()

	repo, sqlMock := newEDISelectOptionsTestRepository(t)
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	schemaID := pulid.MustNew("edips_")
	fieldID := pulid.MustNew("edipsf_")

	sqlMock.ExpectQuery(`SELECT count\(\*\) FROM "edi_partner_setting_fields" AS "epsf".*epss\.transaction_set.*epss\.direction.*epsf\.status.*epsf\.path LIKE.*epsf\.group_key.*epsf\.required.*epsf\.secret.*epsf\.path ILIKE`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	sqlMock.ExpectQuery(`SELECT "epsf".*FROM "edi_partner_setting_fields" AS "epsf".*epss\.transaction_set.*epss\.direction.*epsf\.status.*epsf\.path LIKE.*epsf\.group_key.*epsf\.required.*epsf\.secret.*epsf\.path ILIKE`).
		WillReturnRows(sqlmock.NewRows([]string{
			"id",
			"schema_id",
			"path",
			"label",
			"description",
			"data_type",
			"required",
			"nullable",
			"default_value",
			"allowed_values",
			"secret",
			"group_key",
			"display_order",
			"validation_pattern",
			"min_length",
			"max_length",
			"usage_notes",
			"status",
		}).AddRow(
			fieldID,
			schemaID,
			"carrier.scac",
			"Carrier SCAC",
			"Carrier identifier",
			edi.PartnerSettingDataTypeString,
			true,
			false,
			nil,
			[]byte(`[]`),
			false,
			"carrier",
			10,
			"",
			2,
			4,
			"Use the carrier SCAC",
			edi.PartnerSettingStatusActive,
		))

	required := true
	secret := false
	result, err := repo.SelectPartnerSettingFieldOptions(
		t.Context(),
		&repositories.ListEDIPartnerSettingFieldsRequest{
			Filter: &pagination.QueryOptions{
				TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
				Pagination: pagination.Info{Limit: 20},
				Query:      "scac",
			},
			TransactionSet: edi.TransactionSet204,
			Direction:      edi.DocumentDirectionOutbound,
			Status:         edi.PartnerSettingStatusActive,
			PathPrefix:     "carrier.",
			GroupKey:       "carrier",
			Required:       &required,
			Secret:         &secret,
		},
	)

	require.NoError(t, err)
	require.Equal(t, 1, result.Total)
	require.Len(t, result.Items, 1)
	require.Equal(t, fieldID, result.Items[0].ID)
	require.Equal(t, "carrier.scac", result.Items[0].Path)
	require.Equal(t, "Carrier SCAC", result.Items[0].Label)
	require.Equal(t, "carrier", result.Items[0].GroupKey)
}

func TestPartnerSettingSeedMigrationUsesPublicPaths(t *testing.T) {
	t.Parallel()

	content, err := os.ReadFile(
		"../../migrations/20260518130000_edi_partner_setting_schemas.tx.up.sql",
	)
	require.NoError(t, err)

	sql := string(content)
	expectedPaths := []string{
		"carrier.scac",
		"carrier.name",
		"carrier.code",
		"billTo.name",
		"shipper.name",
		"consignee.name",
		"defaultEquipmentType",
		"defaultPaymentMethod",
		"defaultWeightUnit",
		"defaultPackagingCode",
		"referenceQualifiers.bol",
		"referenceQualifiers.purchaseOrder",
		"stopReasonMappings.pickup",
		"stopReasonMappings.delivery",
		"accessorialCodes.codeMap",
		"commodityDefaults.description",
		"contact.phone",
		"envelope.senderQualifier",
		"envelope.receiverQualifier",
		"envelope.usageIndicator",
	}
	for _, path := range expectedPaths {
		require.Contains(t, sql, "'"+path+"'")
	}

	for _, oldPath := range []string{
		"defaults.paymentMethod",
		"defaults.weightUnit",
		"defaults.packagingCode",
		"defaults.commodityDescription",
		"accessorial.codes",
	} {
		require.False(t, strings.Contains(sql, "'"+oldPath+"'"), "old path %s is still seeded", oldPath)
	}
}

func TestSearchPartnerSettingFields_AppliesSchemaSearchAndRequiredFilters(t *testing.T) {
	t.Parallel()

	db, sqlMock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	require.NoError(t, err)
	sqlMock.MatchExpectationsInOrder(false)

	bunDB := bun.NewDB(db, pgdialect.New())
	t.Cleanup(func() {
		require.NoError(t, sqlMock.ExpectationsWereMet())
	})

	repo := &repository{
		db: postgres.NewTestConnection(bunDB),
		l:  zap.NewNop(),
	}
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	schemaID := pulid.MustNew("edips_")
	fieldID := pulid.MustNew("edipsf_")
	required := true

	countFieldsQuery := `SELECT count\(\*\) FROM "edi_partner_setting_fields" AS "epsf".*` +
		`epsf\.schema_id.*epsf\.required.*epsf\.path ILIKE`
	selectFieldsQuery := `SELECT .* FROM "edi_partner_setting_fields" AS "epsf".*` +
		`epsf\.schema_id.*epsf\.required.*epsf\.path ILIKE`
	sqlMock.ExpectQuery(countFieldsQuery).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	sqlMock.ExpectQuery(selectFieldsQuery).
		WillReturnRows(sqlmock.NewRows([]string{
			"id",
			"schema_id",
			"path",
			"label",
			"data_type",
			"required",
			"nullable",
			"secret",
			"group_key",
			"display_order",
			"status",
		}).AddRow(
			fieldID,
			schemaID,
			"carrier.scac",
			"Carrier SCAC",
			edi.PartnerSettingDataTypeString,
			true,
			false,
			false,
			"carrier",
			10,
			edi.PartnerSettingStatusActive,
		))
	result, err := repo.SearchPartnerSettingFields(
		t.Context(),
		&repositories.ListEDIPartnerSettingFieldsRequest{
			Filter: &pagination.QueryOptions{
				TenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
				Pagination: pagination.Info{Limit: 20},
				Query:      "carrier.",
			},
			SchemaID:   schemaID,
			PathPrefix: "carrier.",
			Required:   &required,
		},
	)

	require.NoError(t, err)
	require.Equal(t, 1, result.Total)
	require.Len(t, result.Items, 1)
	require.Equal(t, "carrier.scac", result.Items[0].Path)
}
