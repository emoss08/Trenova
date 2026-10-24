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

	assert.Equal(t, "", sql)
	assert.Equal(t, []any{}, args)
}
