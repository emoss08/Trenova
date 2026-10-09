package agenttoolschema_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/services/agenttoolschema"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/stretchr/testify/assert"
)

func TestRecordID_MarksTheResourceAndNamesTheSource(t *testing.T) {
	t.Parallel()

	property := agenttoolschema.RecordID(permission.ResourceCustomer, "The customer",
		"list_customers")

	assert.Equal(t, toolschema.TypeString, property[toolschema.KeyType])
	assert.Equal(t, "The customer, from list_customers. Never guess one.",
		property[toolschema.KeyDescription])
	assert.Equal(t, "customer", toolschema.RecordOf(property))
	assert.NotContains(t, toolschema.ForModel(property), toolschema.KeyRecordOf)
}

func TestID_CarriesNoResource(t *testing.T) {
	t.Parallel()

	property := agenttoolschema.ID("The day of leave", "list_worker_leave_cases")

	assert.Empty(t, toolschema.RecordOf(property))
	assert.Equal(t, "The day of leave, from list_worker_leave_cases. Never guess one.",
		property[toolschema.KeyDescription])
}

func TestRecordIDs_MarksEachItem(t *testing.T) {
	t.Parallel()

	property := agenttoolschema.RecordIDs(permission.ResourceInvoice,
		"The invoices, from list_invoices.", 50)

	items := property[toolschema.KeyItems].(map[string]any)
	assert.Equal(t, "invoice", toolschema.RecordOf(items))
	assert.Equal(t, 1, property[toolschema.KeyMinItems])
	assert.Equal(t, 50, property[toolschema.KeyMaxItems])
}

func TestNoted_AddsASentence(t *testing.T) {
	t.Parallel()

	property := agenttoolschema.Noted(
		agenttoolschema.RecordID(permission.ResourceReport, "The saved report", "list_reports"),
		"Leave it out to keep it.",
	)

	assert.Equal(t,
		"The saved report, from list_reports. Never guess one. Leave it out to keep it.",
		property[toolschema.KeyDescription])
}
