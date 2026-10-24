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

	assert.Equal(t, "", sql)
	assert.Equal(t, []any{}, args)
}
