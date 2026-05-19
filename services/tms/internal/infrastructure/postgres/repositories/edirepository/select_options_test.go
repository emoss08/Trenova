package edirepository

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
		sqlMock.ExpectClose()
		sqlMock.ExpectClose()
		require.NoError(t, bunDB.Close())
		require.NoError(t, sqlMock.ExpectationsWereMet())
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
