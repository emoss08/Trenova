package chargeallocationrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestLockedAllocations_QualifiesTheAllocationIDAgainstTheInvoiceJoin(t *testing.T) {
	t.Parallel()

	lockedID := pulid.ID("chal_1")
	sql, args := keeltest.Render(t, lockedAllocations([]pulid.ID{lockedID}))

	assert.Contains(t, sql, `SELECT ca."id"`)
	assert.Contains(t, sql, `FROM "charge_allocations" ca JOIN "invoices" i ON`)
	assert.Contains(t, sql, `i."id" = ca."invoice_id"`)
	assert.Contains(t, sql, `ca."id" = ANY(`)
	assert.Contains(t, sql, `i."status" = 'Posted'`)
	assert.Equal(t, []any{"bu_1", "org_1", []string{"chal_1"}}, args)
}
