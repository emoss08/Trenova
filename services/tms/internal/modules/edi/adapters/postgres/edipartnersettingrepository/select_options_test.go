package edipartnersettingrepository

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/schema/migrations"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFieldsQuery_AppliesTheSchemaSearchAndFieldFilters(t *testing.T) {
	t.Parallel()

	required, secret := true, false
	q := fieldsQuery(&ediports.ListEDIPartnerSettingFieldsRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: pagination.TenantInfo{OrgID: "org_1", BuID: "bu_1"},
			Query:      "scac",
		},
		TransactionSet: edi.TransactionSet204,
		Direction:      edi.DocumentDirectionOutbound,
		Status:         edi.PartnerSettingStatusActive,
		PathPrefix:     "carrier.",
		GroupKey:       "carrier",
		Required:       &required,
		Secret:         &secret,
	}).Limit(20)

	sql, args := keeltest.Render(t, q)

	assert.Contains(t, sql, `FROM "edi_partner_setting_fields" epsf LEFT JOIN "edi_partner_setting_schemas" s ON s."id" = epsf."schema_id"`)
	assert.Contains(t, sql, `epsf."path" LIKE $6 AND epsf."group_key" = $7 AND epsf."required" = $8 AND epsf."secret" = $9`)
	assert.Contains(t, sql, `ORDER BY epsf."display_order" ASC, epsf."path" ASC LIMIT 20`)
	assert.Contains(t, args, "carrier.%")
	assert.Contains(t, args, "%scac%")
	assert.Contains(t, args, true)
	assert.Contains(t, args, false)
}

func TestActiveSchemaQuery_PrefersTheTenantAndDocumentTypeSchema(t *testing.T) {
	t.Parallel()

	q := activeSchemaQuery(&ediports.GetActiveEDIPartnerSettingSchemaRequest{
		TenantInfo:     pagination.TenantInfo{OrgID: "org_1", BuID: "bu_1"},
		DocumentTypeID: "edidt_1",
		Standard:       edi.EDIStandardX12,
		TransactionSet: edi.TransactionSet204,
		Direction:      edi.DocumentDirectionOutbound,
		X12Version:     "004010",
	}).Limit(1)

	sql, args := keeltest.Render(t, q)

	assert.Contains(t, sql, `((epss."organization_id" = $6 AND epss."business_unit_id" = $7) OR (epss."organization_id" IS NULL AND epss."business_unit_id" IS NULL))`)
	assert.Contains(t, sql, `(epss."document_type_id" = $8 OR epss."document_type_id" IS NULL)`)
	assert.Contains(t, sql, `ORDER BY epss."organization_id" IS NOT NULL DESC, epss."schema_version" DESC, epss."document_type_id" IS NOT NULL DESC LIMIT 1`)
	assert.Contains(t, args, "org_1")
	assert.Contains(t, args, "bu_1")
	assert.Contains(t, args, "edidt_1")
	assert.Contains(t, args, "004010")
}

func TestPartnerSettingSeedMigrationUsesPublicPaths(t *testing.T) {
	t.Parallel()

	sql, err := migrations.File("20260518130000_edi_partner_setting_schemas")
	require.NoError(t, err)

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
		require.False(
			t,
			strings.Contains(sql, "'"+oldPath+"'"),
			"old path %s is still seeded",
			oldPath,
		)
	}
}
