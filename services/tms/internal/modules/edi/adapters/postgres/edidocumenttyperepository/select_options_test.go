package edidocumenttyperepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/stretchr/testify/assert"
)

func TestDocumentTypeOptionsQuery_AppliesTheFiltersAndSearch(t *testing.T) {
	t.Parallel()

	q := documentTypeOptionsQuery(&ediports.EDIDocumentTypeSelectOptionsRequest{
		TransactionSet: edi.TransactionSet204,
		Direction:      edi.DocumentDirectionOutbound,
		Status:         edi.DocumentStatusActive,
	}).Where(documentTypeSearch("204")).Limit(10)

	sql, args := keeltest.Render(t, q)

	assert.Contains(
		t,
		sql,
		`FROM "edi_document_types" edt WHERE edt."transaction_set" = '204' `+
			`AND edt."direction" = 'Outbound' AND edt."status" = 'Active'`,
	)
	assert.Contains(
		t,
		sql,
		`(edt."code" ILIKE $1 OR edt."name" ILIKE $2 OR edt."default_version" ILIKE $3) `+
			`ORDER BY edt."code" ASC LIMIT 10`,
	)
	assert.Equal(t, []any{"%204%", "%204%", "%204%"}, args)
}
