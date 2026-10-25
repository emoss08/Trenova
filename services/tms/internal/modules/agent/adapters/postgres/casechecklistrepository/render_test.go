package casechecklistrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/deskcase"
	"github.com/emoss08/trenova/internal/modules/agent/ports/deskcaseports"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestListTemplatesQuery(t *testing.T) {
	t.Parallel()

	sql, _ := keeltest.Render(t, listTemplatesQuery(&deskcaseports.ListCaseChecklistTemplatesRequest{
		Kind: deskcase.ChecklistReadyToBill,
	}))

	assert.Contains(t, sql, `LEFT JOIN "customers" cus`)
	assert.Contains(t, sql, "NULLS FIRST")
	assert.Contains(t, sql, "LIMIT 1000")
}

func TestTemplateForQuery(t *testing.T) {
	t.Parallel()

	withCustomer, args := keeltest.Render(t, templateForQuery(&deskcaseports.GetCaseChecklistTemplateForRequest{
		Kind:       deskcase.ChecklistReadyToBill,
		CustomerID: pulid.ID("cus_1"),
	}))
	assert.Contains(t, withCustomer, `"customer_id" IS NULL OR`)
	assert.Contains(t, args, "cus_1")
	assert.Contains(t, withCustomer, "NULLS LAST")

	without, _ := keeltest.Render(t, templateForQuery(&deskcaseports.GetCaseChecklistTemplateForRequest{
		Kind: deskcase.ChecklistReadyToBill,
	}))
	assert.NotContains(t, without, " OR ")
}

func TestListTicksQuery(t *testing.T) {
	t.Parallel()

	sql, _ := keeltest.Render(t, listTicksQuery(&deskcaseports.ListCaseChecklistTicksRequest{
		SubjectType: "shipment",
		SubjectID:   pulid.ID("shp_1"),
	}))

	assert.Contains(t, sql, `LEFT JOIN "users" usr`)
	assert.Contains(t, sql, "LIMIT 1000")
}
