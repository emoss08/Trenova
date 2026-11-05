package globalsearchservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/types/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// An invoice in the projection GTC syncs into the "invoices" index.
func invoiceDocument(id pulid.ID) map[string]any {
	return map[string]any{
		"id":                  id.String(),
		"number":              "INV-1042",
		"bill_type":           "Invoice",
		"status":              "Posted",
		"settlement_status":   "Unpaid",
		"dispute_status":      "None",
		"bill_to_name":        "Morales & Jenkins Manufacturing",
		"shipment_pro_number": "S26101755026772",
		"currency_code":       "USD",
		"total_amount_minor":  float64(125000),
		"balance_due_minor":   float64(125000),
		"due_date":            float64(1_792_000_000),
	}
}

func invoiceSearch(t *testing.T, svc *Service) *kernelports.GlobalSearchResult {
	t.Helper()

	result, err := svc.Search(t.Context(), &kernelports.GlobalSearchRequest{
		Query:      "INV-1042",
		TenantInfo: pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
	})
	require.NoError(t, err)

	return result
}

// An invoice is found by its number and opens on the invoices page, with who
// it bills under it and its amounts read back as decimals.
func TestSearchFindsAnInvoiceByItsNumber(t *testing.T) {
	id := pulid.MustNew("inv_")
	svc := &Service{
		balances: &fakeBalances{},
		views:    searchViews(),
		logger:   zaptest.NewLogger(t),
		config:   indexConfig(),
		searchRepo: repoReturning(t, map[string]*customerports.SearchPage{
			"invoices": page(invoiceDocument(id)),
		}),
		permissions: permissionsDenying(t),
		visibility:  everyRecordVisible{},
	}

	invoices := groupOf(invoiceSearch(t, svc), search.EntityTypeInvoice)

	require.NotNil(t, invoices)
	require.Len(t, invoices.Hits, 1)
	hit := invoices.Hits[0]
	assert.Equal(t, id.String(), hit.ID)
	assert.Equal(t, "INV-1042", hit.Title)
	assert.Equal(t, "Morales & Jenkins Manufacturing", hit.Subtitle)
	assert.Equal(t, recordHref("invoice", id.String()), hit.Href)
	assert.Equal(t, "1250.00", hit.Metadata["balanceDue"])
	assert.Equal(t, "1250.00", hit.Metadata["totalAmount"])
	assert.Equal(t, "Posted", hit.Metadata[fieldStatus])
	assert.Equal(t, "1792000000", hit.Metadata["dueDate"])
	assert.Equal(t, "S26101755026772", hit.Metadata["proNumber"])
}

// Someone who may not read invoices is never searched for them: no group, and
// the index is not asked.
func TestSearchFindsNoInvoiceWithoutInvoiceAccess(t *testing.T) {
	svc := &Service{
		balances: &fakeBalances{},
		views:    searchViews(),
		logger:   zaptest.NewLogger(t),
		config:   indexConfig(),
		searchRepo: repoReturning(t, map[string]*customerports.SearchPage{
			"invoices": page(invoiceDocument(pulid.MustNew("inv_"))),
		}),
		permissions: permissionsDenying(t, permission.ResourceInvoice),
		visibility:  everyRecordVisible{},
	}

	assert.Nil(t, groupOf(invoiceSearch(t, svc), search.EntityTypeInvoice))
}
