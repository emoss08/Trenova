package agentlint

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func registry(held ...string) []Tool {
	tools := []Tool{
		{Name: "get_invoice", Resource: permission.ResourceInvoice, ResourceLabel: "Invoice", Operation: permission.OpRead},
		{Name: "update_invoice", Resource: permission.ResourceInvoice, ResourceLabel: "Invoice", Operation: permission.OpUpdate},
		{Name: "get_shipment", Resource: permission.ResourceShipment, ResourceLabel: "Shipment", Operation: permission.OpRead},
		{Name: "cancel_shipment", Resource: permission.ResourceShipment, ResourceLabel: "Shipment", Operation: permission.OpCancel},
	}
	for idx := range tools {
		for _, name := range held {
			if tools[idx].Name == name {
				tools[idx].Held = true
			}
		}
	}
	return tools
}

func TestLintNamesWhatNoHeldToolDoes(t *testing.T) {
	t.Parallel()

	instructions := "Check each invoice first.\nUpdate the invoice when the amount is wrong. Cancel shipments that sat a week."

	findings := Lint(instructions, registry("get_invoice"))

	require.Len(t, findings, 2)
	assert.Equal(t, permission.ResourceInvoice, findings[0].Resource)
	assert.Equal(t, permission.OpUpdate, findings[0].Operation)
	assert.Equal(t, []string{"update_invoice"}, findings[0].Tools)
	assert.Equal(t, "Update the invoice when the amount is wrong.", findings[0].Excerpt)
	assert.Equal(t, findings[0].Excerpt, instructions[findings[0].Start:findings[0].End])
	assert.Equal(t, permission.ResourceShipment, findings[1].Resource)
	assert.Equal(t, permission.OpCancel, findings[1].Operation)
}

func TestLintSkipsGuardrailsAndHeldTools(t *testing.T) {
	t.Parallel()

	findings := Lint(
		"Never cancel a shipment. Do not update invoices yourself. Update invoices when asked.",
		registry("update_invoice"),
	)

	assert.Empty(t, findings)
}

func TestLintReportsReadsNoToolCanDo(t *testing.T) {
	t.Parallel()

	tools := []Tool{
		{Name: "create_customer", Resource: permission.ResourceCustomer, ResourceLabel: "Customer", Operation: permission.OpCreate, Held: true},
	}

	findings := Lint("Look up the customer before you answer.", tools)

	require.Len(t, findings, 1)
	assert.Equal(t, permission.OpRead, findings[0].Operation)
	assert.Empty(t, findings[0].Tools)
}

func TestLintPrefersTheLongerRecordName(t *testing.T) {
	t.Parallel()

	tools := []Tool{
		{Name: "get_invoice", Resource: permission.ResourceInvoice, ResourceLabel: "Invoice", Operation: permission.OpRead, Held: true},
		{Name: "commit_invoice_run", Resource: permission.ResourceInvoiceRun, ResourceLabel: "Invoice Run", Operation: permission.OpApprove},
	}

	findings := Lint("Commit the invoice run each Friday.", tools)

	require.Len(t, findings, 1)
	assert.Equal(t, permission.ResourceInvoiceRun, findings[0].Resource)
}

func TestLintOfNothing(t *testing.T) {
	t.Parallel()

	assert.Empty(t, Lint("  ", registry()))
}
