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
	assert.Contains(
		t,
		sql,
		`WHERE et."business_unit_id" = $1 AND et."organization_id" = $2 `+
			`AND et."transaction_set" = '204' AND et."direction" = 'Outbound' AND et."status" = 'Active'`,
	)
	assert.Contains(t, sql, `AND (et."name" ILIKE $3 OR et."description" ILIKE $4) ORDER BY et."name" ASC LIMIT 10`)
	assert.Equal(t, []any{"bu_1", "org_1", "%load%", "%load%"}, args)
}
