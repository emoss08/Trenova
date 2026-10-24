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

	assert.Contains(t, sql, `FROM "edi_document_types" edt WHERE edt."transaction_set" = $1 AND edt."direction" = $2 AND edt."status" = $3`)
	assert.Contains(t, sql, `(edt."code" ILIKE $4 OR edt."name" ILIKE $5 OR edt."default_version" ILIKE $6) ORDER BY edt."code" ASC LIMIT 10`)
	assert.Contains(t, args, "204")
	assert.Contains(t, args, "%204%")
}
