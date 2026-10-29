package billingquerytools

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/fieldgate"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytooltest"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListInvoices_WithholdsTheTotalBelowRestricted(t *testing.T) {
	t.Parallel()

	repo := &querytooltest.FakeInvoiceRepo{
		Items: []*invoice.Invoice{querytooltest.SensitiveInvoice()},
	}
	tool := NewListInvoicesTool(repo, &querytooltest.FakePermissions{})

	result, err := tool.Query(t.Context(), querytooltest.AgentParams(map[string]any{}, ""))
	require.NoError(t, err)

	outcome, ok := result.(*fieldgate.Outcome)
	require.True(t, ok)
	rows, ok := outcome.Items.([]any)
	require.True(t, ok)
	require.Len(t, rows, 1)
	row, ok := rows[0].(invoiceRow)
	require.True(t, ok)
	assert.Empty(t, row.TotalAmount)
	assert.Equal(t, "INV-1042", row.Number, "the invoice itself is still listed")
	assert.Equal(t, []string{"totalAmount"}, outcome.Withheld)
}

func TestListInvoices_ShowsTheTotalAtRestricted(t *testing.T) {
	t.Parallel()

	repo := &querytooltest.FakeInvoiceRepo{
		Items: []*invoice.Invoice{querytooltest.SensitiveInvoice()},
	}
	tool := NewListInvoicesTool(repo, &querytooltest.FakePermissions{})

	result, err := tool.Query(t.Context(),
		querytooltest.AgentParams(map[string]any{}, permission.SensitivityRestricted))
	require.NoError(t, err)

	outcome := result.(*fieldgate.Outcome)
	row := outcome.Items.([]any)[0].(invoiceRow)
	assert.Equal(t, "1250.00", row.TotalAmount)
	assert.Empty(t, outcome.Withheld)
}

func TestListInvoices_RefusesToFilterOnAWithheldAmount(t *testing.T) {
	t.Parallel()

	repo := &querytooltest.FakeInvoiceRepo{}
	tool := NewListInvoicesTool(repo, &querytooltest.FakePermissions{})

	_, err := tool.Query(t.Context(), querytooltest.AgentParams(map[string]any{
		"filters": []any{map[string]any{
			"field": "totalAmount", "operator": "gt", "value": "10000",
		}},
	}, ""))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "withheld")
	assert.Nil(t, repo.Captured, "a filter on a withheld amount would reveal it")

	_, err = tool.Query(
		t.Context(),
		querytooltest.AgentParams(map[string]any{"sortBy": "totalAmount"}, ""),
	)
	require.Error(t, err)
}
