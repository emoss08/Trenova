package agentquerytoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeInvoiceRepo struct {
	invoiceports.InvoiceRepository

	items    []*invoice.Invoice
	captured *invoiceports.ListInvoicesRequest
}

func (f *fakeInvoiceRepo) List(
	_ context.Context,
	req *invoiceports.ListInvoicesRequest,
) (*pagination.ListResult[*invoice.Invoice], error) {
	f.captured = req

	return &pagination.ListResult[*invoice.Invoice]{
		Items:      f.items,
		Total:      len(f.items),
		TotalExact: true,
	}, nil
}

func (f *fakeInvoiceRepo) GetByID(
	_ context.Context,
	req invoiceports.GetInvoiceByIDRequest,
) (*invoice.Invoice, error) {
	for _, item := range f.items {
		if item.ID == req.ID {
			return item, nil
		}
	}

	return nil, assert.AnError
}

func sensitiveInvoice() *invoice.Invoice {
	return &invoice.Invoice{
		ID:                     pulid.MustNew("inv_"),
		CustomerID:             pulid.MustNew("cus_"),
		Number:                 "INV-1042",
		Status:                 invoice.StatusPosted,
		SettlementStatus:       invoice.SettlementStatusUnpaid,
		BillToName:             "Acme",
		CurrencyCode:           "USD",
		TotalAmount:            decimal.RequireFromString("1250.00"),
		TotalAmountMinor:       125000,
		SubtotalAmount:         decimal.RequireFromString("1200.00"),
		OtherAmount:            decimal.RequireFromString("50.00"),
		Memo:                   "Discount agreed with the buyer on the phone",
		RemittanceInstructions: "Wire to account 12345",
		Lines: []*invoice.InvoiceLine{{
			LineNumber:  1,
			Description: "Linehaul",
			Quantity:    decimal.NewFromInt(1),
			UnitPrice:   decimal.RequireFromString("1200.00"),
			Amount:      decimal.RequireFromString("1200.00"),
		}},
	}
}

func TestListInvoices_WithholdsTheTotalBelowRestricted(t *testing.T) {
	t.Parallel()

	repo := &fakeInvoiceRepo{items: []*invoice.Invoice{sensitiveInvoice()}}
	tool := newListInvoicesTool(repo, &fakePermissions{})

	result, err := tool.Query(t.Context(), agentParams(map[string]any{}, ""))
	require.NoError(t, err)

	outcome, ok := result.(*gatedOutcome)
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

	repo := &fakeInvoiceRepo{items: []*invoice.Invoice{sensitiveInvoice()}}
	tool := newListInvoicesTool(repo, &fakePermissions{})

	result, err := tool.Query(t.Context(),
		agentParams(map[string]any{}, permission.SensitivityRestricted))
	require.NoError(t, err)

	outcome := result.(*gatedOutcome)
	row := outcome.Items.([]any)[0].(invoiceRow)
	assert.Equal(t, "1250.00", row.TotalAmount)
	assert.Empty(t, outcome.Withheld)
}

func TestListInvoices_RefusesToFilterOnAWithheldAmount(t *testing.T) {
	t.Parallel()

	repo := &fakeInvoiceRepo{}
	tool := newListInvoicesTool(repo, &fakePermissions{})

	_, err := tool.Query(t.Context(), agentParams(map[string]any{
		"filters": []any{map[string]any{
			"field": "totalAmount", "operator": "gt", "value": "10000",
		}},
	}, ""))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "withheld")
	assert.Nil(t, repo.captured, "a filter on a withheld amount would reveal it")

	_, err = tool.Query(t.Context(), agentParams(map[string]any{"sortBy": "totalAmount"}, ""))
	require.Error(t, err)
}
