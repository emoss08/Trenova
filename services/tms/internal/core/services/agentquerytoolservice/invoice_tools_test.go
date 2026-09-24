package agentquerytoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeInvoiceRepo struct {
	repositories.InvoiceRepository

	items    []*invoice.Invoice
	captured *repositories.ListInvoicesRequest
}

func (f *fakeInvoiceRepo) List(
	_ context.Context,
	req *repositories.ListInvoicesRequest,
) (*pagination.ListResult[*invoice.Invoice], error) {
	f.captured = req

	return &pagination.ListResult[*invoice.Invoice]{Items: f.items, Total: len(f.items)}, nil
}

func (f *fakeInvoiceRepo) GetByID(
	_ context.Context,
	req repositories.GetInvoiceByIDRequest,
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

	outcome, ok := result.(gatedOutcome)
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

	outcome := result.(gatedOutcome)
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

func TestGetInvoice_WithholdsAmountsAndTheMemoBelowRestricted(t *testing.T) {
	t.Parallel()

	entity := sensitiveInvoice()
	tool := newGetInvoiceTool(&fakeInvoiceRepo{items: []*invoice.Invoice{entity}},
		&fakePermissions{})

	result, err := tool.Query(t.Context(),
		agentParams(map[string]any{"invoiceId": entity.ID.String()}, ""))
	require.NoError(t, err)

	detail, ok := result.(invoiceDetail)
	require.True(t, ok)
	assert.Empty(t, detail.Total)
	assert.Empty(t, detail.Memo)
	assert.Empty(t, detail.Remittance)
	require.Len(t, detail.Lines, 1)
	assert.Empty(t, detail.Lines[0].Amount)
	assert.Equal(t, "Linehaul", detail.Lines[0].Description)
	assert.Subset(t, detail.Withheld, []string{
		"totalAmount", "memo", "remittanceInstructions", "lines.amount", "lines.unitPrice",
	})
}

func TestGetInvoice_ShowsEverythingAtRestricted(t *testing.T) {
	t.Parallel()

	entity := sensitiveInvoice()
	tool := newGetInvoiceTool(&fakeInvoiceRepo{items: []*invoice.Invoice{entity}},
		&fakePermissions{})

	result, err := tool.Query(t.Context(), chatParams(
		map[string]any{"invoiceId": entity.ID.String()},
		permission.SensitivityRestricted,
	))
	require.NoError(t, err)

	detail := result.(invoiceDetail)
	assert.Equal(t, "1250.00", detail.Total)
	assert.Equal(t, "1250.00", detail.BalanceDue)
	assert.Equal(t, entity.Memo, detail.Memo)
	assert.Equal(t, "1200.00", detail.Lines[0].Amount)
	assert.Empty(t, detail.Withheld)
}

func TestGetInvoice_RefusesSomethingThatIsNotAnID(t *testing.T) {
	t.Parallel()

	tool := newGetInvoiceTool(&fakeInvoiceRepo{}, &fakePermissions{})

	_, err := tool.Query(t.Context(), agentParams(map[string]any{"invoiceId": "INV-1042"}, ""))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invoiceId")
}
