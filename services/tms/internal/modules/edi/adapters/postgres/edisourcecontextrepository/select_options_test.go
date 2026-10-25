package edisourcecontextrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/assert"
)

func TestFieldsQuery_AppliesTheSchemaSearchAndFieldFilters(t *testing.T) {
	t.Parallel()

	repeated := true
	q := fieldsQuery(&ediports.ListEDISourceContextFieldsRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: pagination.TenantInfo{OrgID: "org_1", BuID: "bu_1"},
			Query:      "city",
		},
		TransactionSet: edi.TransactionSet204,
		Direction:      edi.DocumentDirectionOutbound,
		Status:         edi.SourceContextFieldStatusActive,
		Repeated:       &repeated,
		PathPrefix:     "shipment.",
	}).Limit(20)

	sql, args := keeltest.Render(t, q)

	assert.Contains(t, sql, `FROM "edi_source_context_fields" escf LEFT JOIN "edi_source_context_schemas" s ON s."id" = escf."schema_id"`)
	assert.Contains(
		t,
		sql,
		`((s."organization_id" = $1 AND s."business_unit_id" = $2) `+
			`OR (s."organization_id" IS NULL AND s."business_unit_id" IS NULL))`,
	)
	assert.Contains(
		t,
		sql,
		`s."transaction_set" = '204' AND s."direction" = 'Outbound' AND escf."status" = 'Active' `+
			`AND escf."repeated" = $3 AND escf."path" LIKE $4`,
	)
	assert.Contains(t, sql, `ORDER BY escf."path" ASC, escf."repeat_path" ASC LIMIT 20`)
	assert.Contains(t, args, "shipment.%")
	assert.Contains(t, args, "%city%")
	assert.Contains(t, args, true)
}
