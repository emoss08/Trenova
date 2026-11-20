package accountingsyncenqueue

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingconnection"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalentry"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectionports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingvocab"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type countedConnections struct {
	accountingconnectionports.AccountingConnectionRepository
	listed int
}

func (c *countedConnections) ListByTenant(
	context.Context,
	pagination.TenantInfo,
) ([]*accountingconnection.AccountingConnection, error) {
	c.listed++
	return nil, nil
}

func postedInvoiceRequest() *accountingsyncports.AccountingSyncEnqueueRequest {
	tenant := plantest.Tenant()
	return accountingsyncports.InvoiceSyncRequest(&invoice.Invoice{
		ID:             pulid.MustNew("inv_"),
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
		Number:         "INV-1001",
		BillType:       billingvocab.BillTypeInvoice,
	}, accountingsync.PostedSourceEvent(billingvocab.BillTypeInvoice))
}

func TestPostingAnInvoiceQueuesNoAccountingSyncWithoutAccountingIntegrations(t *testing.T) {
	t.Parallel()

	connections := &countedConnections{}
	enqueuer := New(Params{
		Logger:      zap.NewNop(),
		Connections: connections,
		Features:    plantest.Without(platformcatalog.FeatureAccountingIntegration),
	})

	require.NoError(t, enqueuer.Enqueue(t.Context(), postedInvoiceRequest()))
	assert.Zero(t, connections.listed, "no connection is read and no record is queued")

	destinations, err := enqueuer.Destinations(t.Context(), postedInvoiceRequest())
	require.NoError(t, err)
	assert.Empty(t, destinations)
}

func TestAnInvoiceIsSyncedOnTheBasePlan(t *testing.T) {
	t.Parallel()

	connections := &countedConnections{}
	enqueuer := New(Params{
		Logger:      zap.NewNop(),
		Connections: connections,
		Features:    plantest.BaseOnly(),
	})

	require.NoError(t, enqueuer.Enqueue(t.Context(), postedInvoiceRequest()))
	assert.Equal(t, 1, connections.listed, "accounting integrations are on every plan")
}

func TestAJournalIsSyncedOnlyWithAccounting(t *testing.T) {
	t.Parallel()

	posted := &accountingsyncports.AccountingJournalPosted{
		TenantInfo:  plantest.Tenant(),
		EntryID:     pulid.MustNew("je_"),
		EntryNumber: "JE-1",
		EntryType:   journalentry.EntryTypeStandard,
	}

	without := &countedConnections{}
	require.NoError(t, New(Params{
		Logger:      zap.NewNop(),
		Connections: without,
		Features:    plantest.BaseOnly(),
	}).EnqueueJournal(t.Context(), posted))
	assert.Zero(t, without.listed, "the ledger side of the sync is Accounting's")

	with := &countedConnections{}
	require.NoError(t, New(Params{
		Logger:      zap.NewNop(),
		Connections: with,
		Features:    plantest.AllFeatures(),
	}).EnqueueJournal(t.Context(), posted))
	assert.Equal(t, 1, with.listed)
}

func TestEachSyncedRecordBelongsToTheSideOfTheLineItIsOn(t *testing.T) {
	t.Parallel()

	for _, base := range []accountingsync.SyncObjectType{
		accountingsync.SyncObjectCustomer,
		accountingsync.SyncObjectInvoice,
		accountingsync.SyncObjectCreditMemo,
		accountingsync.SyncObjectDebitMemo,
	} {
		assert.Equal(t, platformcatalog.FeatureAccountingIntegration, base.Feature(), base)
	}
	for _, ledger := range []accountingsync.SyncObjectType{
		accountingsync.SyncObjectCustomerPayment,
		accountingsync.SyncObjectCreditApplication,
		accountingsync.SyncObjectCarrierVendor,
		accountingsync.SyncObjectDriverVendor,
		accountingsync.SyncObjectCarrierBill,
		accountingsync.SyncObjectCarrierBillPay,
		accountingsync.SyncObjectDriverBill,
		accountingsync.SyncObjectDriverBillPay,
		accountingsync.SyncObjectJournalEntry,
		accountingsync.SyncObjectJournalSummary,
	} {
		assert.Equal(t, platformcatalog.FeatureAccounting, ledger.Feature(), ledger)
	}
}

func TestPostingAnInvoiceLooksForConnectionsWithAccounting(t *testing.T) {
	t.Parallel()

	connections := &countedConnections{}
	enqueuer := New(Params{
		Logger:      zap.NewNop(),
		Connections: connections,
		Features:    plantest.AllFeatures(),
	})

	require.NoError(t, enqueuer.Enqueue(t.Context(), postedInvoiceRequest()))
	assert.Equal(t, 1, connections.listed)
}
