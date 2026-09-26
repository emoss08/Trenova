package accountingdriftservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type balanceDoc struct {
	objectType accountingsync.SyncObjectType
	externalID string
	openMinor  int64
	provider   string
}

func (h *harness) customer(name string, docs ...balanceDoc) pulid.ID {
	id := pulid.MustNew("cus_")
	for _, doc := range docs {
		h.source.balances = append(h.source.balances, &repositories.AccountingDriftBalanceLine{
			CustomerID:   id,
			CustomerName: name,
			CurrencyCode: "USD",
			ObjectType:   doc.objectType,
			ObjectID:     pulid.MustNew("inv_"),
			Number:       "INV-" + doc.externalID,
			OpenMinor:    doc.openMinor,
			ExternalID:   doc.externalID,
		})
		if doc.provider == "" {
			continue
		}
		state := h.provider(doc.objectType, doc.externalID, "0")
		state.Balance = balanceOf(doc.provider)
		h.reader.set(doc.objectType, state)
	}
	return id
}

func (h *harness) balances(t *testing.T) *services.AccountingDriftBalanceResult {
	t.Helper()
	result, err := h.svc.ReconcileBalances(
		t.Context(),
		&services.ReconcileAccountingDriftBalancesRequest{
			TenantInfo:   h.tenant,
			ConnectionID: h.conn.ID,
			EventBudget:  20,
		},
	)
	require.NoError(t, err)
	return result
}

func TestBalancesOpenAFindingListingTheDocumentsThatDiffer(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	customerID := h.customer("Acme Freight",
		balanceDoc{accountingsync.SyncObjectInvoice, "301", 100_000, "1000.00"},
		balanceDoc{accountingsync.SyncObjectInvoice, "302", 40_000, "150.00"},
		balanceDoc{accountingsync.SyncObjectDebitMemo, "303", 5_000, "50.00"},
	)

	result := h.balances(t)

	assert.Equal(t, 1, result.Customers)
	assert.Equal(t, 1, result.Opened)
	open := h.findings.open()
	require.Len(t, open, 1)
	finding := open[0]
	assert.Equal(t, accountingsync.DriftCustomerBalanceMismatch, finding.Kind)
	assert.Equal(t, accountingsync.SyncObjectCustomer, finding.ObjectType)
	assert.Equal(t, customerID, finding.ObjectID)
	assert.Equal(t, "Acme Freight", finding.PartyName)
	assert.Equal(t, int64(145_000), *finding.TrenovaMinor)
	assert.Equal(t, int64(120_000), *finding.ProviderMinor)
	require.Len(t, finding.Detail, 1)
	assert.Equal(t, "INV-302", finding.Detail[0].ObjectNumber)
	assert.Equal(t, int64(40_000), finding.Detail[0].TrenovaMinor)
	assert.Equal(t, int64(15_000), finding.Detail[0].ProviderMinor)
	assert.Len(t, h.events.Published(), 1)
}

func TestBalancesLeaveOutDocumentsGoneFromTheProvider(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.customer("Acme Freight",
		balanceDoc{accountingsync.SyncObjectInvoice, "301", 100_000, "1000.00"},
		balanceDoc{accountingsync.SyncObjectInvoice, "302", 40_000, "0.00"},
		balanceDoc{accountingsync.SyncObjectInvoice, "303", 25_000, "0.00"},
	)
	voided := h.reader.docs["Invoice:302"]
	voided.Voided = true
	deleted := h.reader.docs["Invoice:303"]
	deleted.Found = false

	result := h.balances(t)

	assert.Equal(t, 1, result.Customers)
	assert.Empty(t, h.findings.all())
}

func TestBalancesCompareOneCurrencyPerCustomer(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.customer("Acme Freight",
		balanceDoc{accountingsync.SyncObjectInvoice, "301", 100_000, "1000.00"},
		balanceDoc{accountingsync.SyncObjectInvoice, "302", 40_000, "0.00"},
	)
	h.source.balances[1].CurrencyCode = "CAD"

	h.balances(t)

	assert.Empty(t, h.findings.all())
}

func TestBalancesSkipACustomerWithAPaymentStillWaiting(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	waiting := h.customer("Acme Freight",
		balanceDoc{accountingsync.SyncObjectInvoice, "301", 100_000, "0.00"},
	)
	h.source.pending = []pulid.ID{waiting}

	result := h.balances(t)

	assert.Equal(t, 0, result.Customers)
	assert.Equal(t, 1, result.Skipped)
	assert.Empty(t, h.findings.all())
}

func TestBalancesResolveOnceTheTotalsAgree(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.customer("Acme Freight",
		balanceDoc{accountingsync.SyncObjectInvoice, "301", 100_000, "900.00"},
	)
	h.balances(t)
	require.Len(t, h.findings.open(), 1)

	state := h.provider(accountingsync.SyncObjectInvoice, "301", "0")
	state.Balance = balanceOf("1000.00")
	h.reader.set(accountingsync.SyncObjectInvoice, state)
	result := h.balances(t)

	assert.Equal(t, 1, result.Resolved)
	assert.Empty(t, h.findings.open())
}

func TestBalancesPageByCustomer(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	first := h.customer("A", balanceDoc{accountingsync.SyncObjectInvoice, "1", 100, "1.00"})
	second := h.customer("B", balanceDoc{accountingsync.SyncObjectInvoice, "2", 100, "1.00"})
	if second.String() < first.String() {
		first, second = second, first
		h.source.balances[0], h.source.balances[1] = h.source.balances[1], h.source.balances[0]
	}

	page, err := h.svc.ReconcileBalances(t.Context(), &services.ReconcileAccountingDriftBalancesRequest{
		TenantInfo:   h.tenant,
		ConnectionID: h.conn.ID,
		Customers:    1,
	})
	require.NoError(t, err)
	assert.True(t, page.More)
	assert.Equal(t, first, page.LastCustomerID)

	next, err := h.svc.ReconcileBalances(t.Context(), &services.ReconcileAccountingDriftBalancesRequest{
		TenantInfo:      h.tenant,
		ConnectionID:    h.conn.ID,
		AfterCustomerID: page.LastCustomerID,
		Customers:       1,
	})
	require.NoError(t, err)
	assert.Equal(t, second, next.LastCustomerID)
	assert.Equal(t, 1, next.Customers)
}

func TestBalancesLeaveDocumentFindingsAlone(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.synced(accountingsync.SyncObjectInvoice, "101", 125_000)
	h.provider(accountingsync.SyncObjectInvoice, "101", "1200.00")
	h.reconcile(t)
	h.customer("Acme Freight",
		balanceDoc{accountingsync.SyncObjectInvoice, "301", 100_000, "1000.00"},
	)

	h.balances(t)

	open := h.findings.open()
	require.Len(t, open, 1)
	assert.Equal(t, accountingsync.DriftAmountMismatch, open[0].Kind)
}
