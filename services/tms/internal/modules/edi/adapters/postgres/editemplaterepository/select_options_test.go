package editemplaterepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/assert"
)

func TestTemplateOptionsQuery_AppliesTheFiltersAndSearch(t *testing.T) {
	t.Parallel()

	q := templateOptionsQuery(&ediports.EDITemplateSelectOptionsRequest{
		SelectQueryRequest: &pagination.SelectQueryRequest{Query: "load"},
		TransactionSet:     edi.TransactionSet204,
		Direction:          edi.DocumentDirectionOutbound,
		Status:             edi.TemplateStatusActive,
	}).Limit(10)

	sql, args := keeltest.Render(t, q)

	assert.Contains(t, sql, `FROM "edi_templates" et LEFT JOIN "edi_document_types" dt ON dt."id" = et."document_type_id"`)
	assert.Contains(t, sql, `et."transaction_set" = $3 AND et."direction" = $4 AND et."status" = $5`)
	assert.Contains(t, sql, `AND (et."name" ILIKE $6 OR et."description" ILIKE $7) ORDER BY et."name" ASC LIMIT 10`)
	assert.Contains(t, args, string(edi.TransactionSet204))
	assert.Contains(t, args, string(edi.DocumentDirectionOutbound))
	assert.Contains(t, args, string(edi.TemplateStatusActive))
	assert.Contains(t, args, "%load%")
}
