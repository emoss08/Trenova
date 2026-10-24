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

	assert.Equal(t, "", sql)
	assert.Equal(t, []any{}, args)
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

	assert.Equal(t, "", sql)
	assert.Equal(t, []any{}, args)
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
