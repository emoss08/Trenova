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

func TestKindID_MarksEveryKindAndIsStrippedForTheModel(t *testing.T) {
	t.Parallel()

	property := agenttoolschema.KindID(
		"The qualification record, from list_worker_qualifications. Never guess one.",
		permission.KindEmploymentVerification, permission.KindClearinghouseQuery,
	)

	assert.Equal(t, toolschema.TypeString, property[toolschema.KeyType])
	assert.Equal(t, []string{"employment_verification", "clearinghouse_query"},
		toolschema.RecordKinds(property))
	assert.Empty(t, toolschema.RecordOf(property))
	assert.NotContains(t, toolschema.ForModel(property), toolschema.KeyRecordKinds)
	assert.NotContains(t, toolschema.ForPortableModel(property), toolschema.KeyRecordKinds)
}

func TestKindIDs_MarksEachItem(t *testing.T) {
	t.Parallel()

	property := agenttoolschema.KindIDs("The rounds, from list_dot_random_draws.", 5,
		permission.KindDOTRandomDraw)

	items := property[toolschema.KeyItems].(map[string]any)
	assert.Equal(t, []string{"dot_random_draw"}, toolschema.RecordKinds(items))
	assert.Equal(t, 5, property[toolschema.KeyMaxItems])
	stripped := toolschema.ForModel(property)[toolschema.KeyItems].(map[string]any)
	assert.NotContains(t, stripped, toolschema.KeyRecordKinds)
}

func TestOfResource_MarksAPropertyBuiltElsewhereWithoutChangingIt(t *testing.T) {
	t.Parallel()

	single := agenttoolschema.OfResource(map[string]any{
		toolschema.KeyType:        toolschema.TypeString,
		toolschema.KeyDescription: "The hold, from get_shipment.",
	}, permission.ResourceShipmentHold)
	assert.Equal(t, "shipment_hold", toolschema.RecordOf(single))
	assert.Equal(t, "The hold, from get_shipment.", single[toolschema.KeyDescription])

	list := agenttoolschema.OfKinds(map[string]any{
		toolschema.KeyType:        toolschema.TypeArray,
		toolschema.KeyDescription: "The rounds, from list_dot_random_draws.",
		toolschema.KeyItems:       map[string]any{toolschema.KeyType: toolschema.TypeString},
	}, permission.KindDOTRandomDraw)
	items := list[toolschema.KeyItems].(map[string]any)
	assert.Equal(t, []string{"dot_random_draw"}, toolschema.RecordKinds(items))
	assert.NotContains(t, list, toolschema.KeyMinItems)
	assert.Empty(t, toolschema.RecordKinds(list))
}
