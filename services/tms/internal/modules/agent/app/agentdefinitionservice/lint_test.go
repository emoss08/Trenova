package agentdefinitionservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/stretchr/testify/assert"
)

func TestLintRegistryHoldsWhatTheDraftReaches(t *testing.T) {
	t.Parallel()

	catalog := []agentdefinitionports.ToolCatalogEntry{
		{Name: "search", Core: true},
		{Name: "get_invoice", Resource: permission.ResourceInvoice, Operation: permission.OpRead},
		{
			Name:          "update_invoice",
			Resource:      permission.ResourceInvoice,
			Operation:     permission.OpUpdate,
			Prerequisites: []string{"get_invoice"},
		},
		{Name: "shipment_cancel", Resource: permission.ResourceShipment, Operation: permission.OpCancel},
		{Name: "web_search", Resource: permission.ResourceShipment, Operation: permission.OpRead, GrantedToEveryAgent: true},
	}

	tools := lintRegistry(catalog, &agentdefinitionports.LintAgentInstructionsRequest{
		ToolNames:         []string{"update_invoice", "shipment_cancel"},
		DisabledToolNames: []string{"shipment_cancel"},
	})

	held := make(map[string]bool, len(tools))
	for _, tool := range tools {
		held[tool.Name] = tool.Held
	}
	assert.NotContains(t, held, "search", "a tool that reaches no record is not read")
	assert.True(t, held["update_invoice"])
	assert.True(t, held["get_invoice"], "a held tool brings its prerequisites")
	assert.False(t, held["shipment_cancel"], "a disabled tool is not held")
	assert.True(t, held["web_search"])
	for _, tool := range tools {
		if tool.Name == "get_invoice" {
			assert.Equal(t, "Invoice", tool.ResourceLabel)
		}
	}
}
