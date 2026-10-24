package invoiceadjustmentrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func renderApprovalQueue(t *testing.T, filter *pagination.QueryOptions) (string, []any) {
	t.Helper()

	keyset := approvalQueueKeyset()
	return keeltest.Render(
		t,
		approvalQueueShape(approvalQueue(filter)).Order(keyset.Orders...).Limit(3),
	)
}

func TestApprovalQueue_PagesBySubmissionTimeThenID(t *testing.T) {
	t.Parallel()

	sql, args := renderApprovalQueue(t, &pagination.QueryOptions{})

	assert.Contains(t, sql, `ORDER BY (COALESCE(`)
	assert.Contains(t, sql, `"submitted_at", `)
	assert.Contains(t, sql, `"created_at")) DESC, `)
	assert.Contains(t, sql, `"id" DESC LIMIT 3`)
	assert.Contains(t, args, string(invoiceadjustment.StatusPendingApproval))
}

func TestApprovalQueue_ReadsTheNamesItShowsThroughJoins(t *testing.T) {
	t.Parallel()

	sql, _ := renderApprovalQueue(t, &pagination.QueryOptions{})

	assert.Contains(t, sql, `LEFT JOIN "invoices"`)
	assert.Contains(t, sql, `LEFT JOIN "billing_queue_items"`)
	assert.Contains(t, sql, `LEFT JOIN "users"`)
}

func TestApprovalQueue_SearchesTheInvoiceTheReasonAndTheSubmitter(t *testing.T) {
	t.Parallel()

	sql, args := renderApprovalQueue(t, &pagination.QueryOptions{Query: "50%"})

	assert.Contains(t, sql, `"number" ILIKE`)
	assert.Contains(t, sql, `"bill_to_name" ILIKE`)
	assert.Contains(t, sql, `"reason" ILIKE`)
	assert.Contains(t, sql, `"name" ILIKE`)
	assert.Contains(t, args, `%50\%%`)
}

func TestApprovalQueue_AppliesItsOwnFieldFilters(t *testing.T) {
	t.Parallel()

	sql, args := renderApprovalQueue(t, &pagination.QueryOptions{
		FieldFilters: []domaintypes.FieldFilter{
			{
				Field:    "kind",
				Operator: dbtype.OpEqual,
				Value:    string(invoiceadjustment.KindWriteOff),
			},
			{Field: "ignored", Operator: dbtype.OpEqual, Value: "x"},
		},
	})

	assert.Contains(t, sql, `"kind" = $`)
	assert.Contains(t, args, string(invoiceadjustment.KindWriteOff))
	assert.NotContains(t, args, "x")
}

func TestApprovalQueue_RefusesCursorsFromAnotherList(t *testing.T) {
	t.Parallel()

	for name, sort := range map[string][]pagination.CursorSortField{
		"default list sort": {
			{Field: "createdAt", Direction: "desc"},
			{Field: "id", Direction: "desc"},
		},
		"no sort": nil,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			after, err := pagination.EncodeKeysetCursor(sort, "opaque")
			require.NoError(t, err)
			cursor, err := pagination.NewCursorInfo(2, after)
			require.NoError(t, err)

			_, err = listquery.Connection(
				keeltest.TenantContext(t),
				keeltest.RenderDB(t),
				approvalQueue(&pagination.QueryOptions{}),
				&listquery.Request[invoiceadjustment.InvoiceAdjustment]{
					Cursor: cursor,
					Keyset: approvalQueueKeyset(),
				},
			)

			var validationErr *errortypes.Error
			require.ErrorAs(t, err, &validationErr)
			assert.Equal(t, "after", validationErr.Field)
		})
	}
}
