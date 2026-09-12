package customerrepository

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// notUpdatedOnConflict are the billing profile columns the upsert deliberately
// leaves alone. Anything not listed here must appear in the ON CONFLICT set.
var notUpdatedOnConflict = map[string]string{
	"id":                     "primary key",
	"business_unit_id":       "tenant key",
	"organization_id":        "tenant key",
	"customer_id":            "conflict target",
	"version":                "incremented by expression, not taken from the row",
	"created_at":             "set once on insert",
	"last_billed_period_end": "billing watermark; advanced only inside the commit transaction",
}

// Every column a customer can edit has to be in the upsert's ON CONFLICT set, or
// saving the form silently keeps the old value. This has already shipped three
// times: default_biller_id and count_late_only_on_appointment_stops never
// persisted on update, and auto_approve was added without it. Adding a column to
// the model now fails here until the upsert is updated or the omission is
// recorded above with a reason.
func TestBillingProfileUpsertUpdatesEveryEditableColumn(t *testing.T) {
	t.Parallel()

	source, err := os.ReadFile("customer.go")
	require.NoError(t, err)
	upsert := string(source)

	profileType := reflect.TypeFor[customer.CustomerBillingProfile]()
	missing := make([]string, 0)

	for field := range profileType.Fields() {
		tag := field.Tag.Get("bun")
		if !isColumnTag(tag) {
			continue
		}

		column, _, _ := strings.Cut(tag, ",")
		if _, skipped := notUpdatedOnConflict[column]; skipped {
			continue
		}

		if !strings.Contains(upsert, "cbp."+field.Name+".SetExcluded()") {
			missing = append(missing, field.Name+" ("+column+")")
		}
	}

	assert.Empty(t, missing,
		"billing profile columns missing from the ON CONFLICT set; add them to the upsert "+
			"or record why they are excluded in notUpdatedOnConflict")
}

// isColumnTag reports whether a bun tag names a real column on this table.
// Relations are written through their own paths — document types are synced by
// syncBillingProfileDocumentTypes — so they are not the upsert's concern.
func isColumnTag(tag string) bool {
	if tag == "" || tag == "-" || strings.HasPrefix(tag, ",") {
		return false
	}
	for _, prefix := range []string{"rel:", "m2m:", "table:"} {
		if strings.HasPrefix(tag, prefix) {
			return false
		}
	}

	return true
}
