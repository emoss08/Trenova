package customerrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/parties/domain/customer/customerdb"
	"github.com/stretchr/testify/assert"
)

var notUpdatedOnConflict = map[string]string{
	"id":                     "primary key",
	"business_unit_id":       "tenant key",
	"organization_id":        "tenant key",
	"customer_id":            "conflict target",
	"version":                "incremented by keel on every upsert",
	"created_at":             "set once on insert",
	"updated_at":             "stamped by keel on every upsert",
	"last_billed_period_end": "billing watermark; advanced only inside the commit transaction",
	"credit_balance":         "open receivables; written only by the credit review",
	"credit_reviewed_at":     "written only by the credit review",
	"credit_hold_automatic":  "kept unless the status changes, by expression rather than taken from the row",
}

func TestBillingProfileUpsertUpdatesEveryEditableColumn(t *testing.T) {
	t.Parallel()

	table := customerdb.CustomerBillingProfilesSchema()
	updated := make(map[string]bool)
	for _, col := range billingProfileFormColumns() {
		updated[table.Columns[col.Index].Name] = true
	}

	missing := make([]string, 0)
	for _, col := range table.Columns {
		if _, skipped := notUpdatedOnConflict[col.Name]; skipped {
			assert.False(
				t,
				updated[col.Name],
				"%s is updated on conflict but listed as skipped",
				col.Name,
			)
			continue
		}
		if !updated[col.Name] {
			missing = append(missing, col.Name)
		}
	}

	assert.Empty(t, missing,
		"billing profile columns missing from the ON CONFLICT set; add them to "+
			"billingProfileFormColumns or record why they are excluded in notUpdatedOnConflict")
}
