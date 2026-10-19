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
	} {
		assert.Equalf(t, want, agentports.ReadingOrder(name), "%s", name)
	}
}
