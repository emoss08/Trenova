package documentcontentrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/ports/documentcontentports"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/stretchr/testify/assert"
)

const pendingEligibility = `d."status" = 'Active' AND d."processing_profile" IN ` +
	`('rate_confirmation_import', 'assistant_attachment', 'inbound_attachment', 'capture', 'factoring_remittance') ` +
	`AND (dc."document_id" IS NULL OR dc."status" IN ('Pending', 'Extracting') ` +
	`OR d."content_status" IN ('Pending', 'Extracting'))`

func TestListPendingExtractionTenants_GroupsByTheQualifiedTenantColumns(t *testing.T) {
	t.Parallel()

	sqlText, args := keeltest.RenderIn(
		t,
		dbscope.WithSystem(t.Context(), "render the pending extraction tenant sweep"),
		pendingTenantsQuery(&documentcontentports.ListPendingDocumentExtractionRequest{
			OlderThan: 1779222120,
			Limit:     100,
		}),
	)

	assert.Contains(
		t,
		sqlText,
		`LEFT JOIN "document_contents" dc ON dc."document_id" = d."id" `+
			`AND dc."organization_id" = d."organization_id" AND dc."business_unit_id" = d."business_unit_id"`,
		sqlText,
	)
	assert.Contains(t, sqlText, "WHERE "+pendingEligibility, sqlText)
	assert.Contains(
		t,
		sqlText,
		`AND d."updated_at" <= $1 GROUP BY d."organization_id", d."business_unit_id" `+
			`ORDER BY d."organization_id" ASC, d."business_unit_id" ASC LIMIT 100`,
		sqlText,
	)
	assert.Equal(t, []any{int64(1779222120)}, args)
}

func TestListPendingExtraction_LeavesFailedDocumentsForAReextract(t *testing.T) {
	t.Parallel()

	sqlText, args := keeltest.Render(
		t,
		pendingExtractionQuery(&documentcontentports.ListPendingDocumentExtractionRequest{
			OlderThan: 1779222120,
			Limit:     10,
		}),
	)

	assert.Contains(
		t,
		sqlText,
		`WHERE d."organization_id" = $1 AND d."business_unit_id" = $2 AND `+
			pendingEligibility+
			` AND d."updated_at" <= $3 ORDER BY d."updated_at" ASC LIMIT 10`,
		sqlText,
	)
	assert.Len(t, args, 3)
	assert.Equal(t, int64(1779222120), args[2])
	assert.NotContains(t, sqlText, "'Failed'")
}
