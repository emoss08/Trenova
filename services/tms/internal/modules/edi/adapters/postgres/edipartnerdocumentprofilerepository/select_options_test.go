package edipartnerdocumentprofilerepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestProfileOptionsQuery_AppliesTheFiltersAndSearch(t *testing.T) {
	t.Parallel()

	q := profileOptionsQuery(&ediports.EDIPartnerDocumentProfileSelectOptionsRequest{
		SelectQueryRequest: &pagination.SelectQueryRequest{Query: " profile "},
		TransactionSet:     edi.TransactionSet204,
		Direction:          edi.DocumentDirectionOutbound,
		Status:             edi.DocumentStatusActive,
		PartnerID:          pulid.ID("edip_1"),
	}).Limit(10)

	sql, args := keeltest.Render(t, q)

	assert.Contains(t, sql, `LEFT JOIN "edi_document_types" dt ON dt."id" = epdp."document_type_id"`)
	assert.Contains(
		t,
		sql,
		`epdp."transaction_set" = '204' AND epdp."direction" = 'Outbound' AND epdp."status" = 'Active' `+
			`AND epdp."edi_partner_id" = $3 `+
			`AND (epdp."name" ILIKE $4 OR p."code" ILIKE $5 OR p."name" ILIKE $6 OR dt."name" ILIKE $7)`,
	)
	assert.Contains(t, sql, `ORDER BY epdp."name" ASC LIMIT 10`)
	assert.Contains(t, args, "edip_1")
	assert.Contains(t, args, "%profile%")
}
