package agentports_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/stretchr/testify/assert"
)

func TestNamespaceOf_IsTheRegisteredWordBeforeTheFirstUnderscore(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "shipment", agentports.NamespaceOf("shipment_place_hold"))
	assert.Equal(t, "order", agentports.NamespaceOf("order_list"))
	assert.Empty(t, agentports.NamespaceOf("list_workers"), "list is a verb, not a namespace")
	assert.Empty(t, agentports.NamespaceOf("remember"))
}

func TestReadingOrder_PutsTheVerbFirst(t *testing.T) {
	t.Parallel()

	for name, want := range map[string]string{
		"shipment_place_hold":   "place_shipment_hold",
		"shipment_get":          "get_shipment",
		"shipment_get_tracking": "get_shipment_tracking",
		"shipment_list":         "list_shipments",
		"order_list":            "list_orders",
		"shipment_list_permits": "list_shipment_permits",
		"list_workers":          "list_workers",
		"remember":              "remember",

		"settlement_get_driver":             "get_driver_settlement",
		"settlement_list_carrier":           "list_carrier_settlements",
		"settlement_add_driver_adjustment":  "add_driver_settlement_adjustment",
		"settlement_list_pay_events":        "list_settlement_pay_events",
		"settlement_generate_carrier_batch": "generate_carrier_settlement_batch",
	} {
		assert.Equalf(t, want, agentports.ReadingOrder(name), "%s", name)
	}
}

func TestToolNameFor_UndoesReadingOrder(t *testing.T) {
	t.Parallel()

	for _, name := range []string{
		"shipment_get",
		"shipment_list",
		"shipment_place_hold",
		"invoice_list",
		"invoice_get_run",
		"billing_get_queue_item",
		"billing_list_queue_items",
		"settlement_get_driver",
		"settlement_list_carrier",
		"settlement_add_driver_adjustment",
		"settlement_list_pay_events",
	} {
		assert.Equalf(t, name, agentports.ToolNameFor(agentports.ReadingOrder(name)), "%s", name)
	}
	assert.Equal(t, "find_tools", agentports.ToolNameFor("find_tools"),
		"a name no namespace reads is its own")
}
