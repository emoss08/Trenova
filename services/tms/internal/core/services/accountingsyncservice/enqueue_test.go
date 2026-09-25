package accountingsyncservice

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/customerpayment"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnqueueWritesARecordOnlyForSyncingConnections(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.addConnection(nil)
	h.addConnection(func(conn *accountingsync.AccountingConnection) {
		syncingConnection(conn)
		conn.Status = accountingsync.ConnectionStatusDisconnected
	})
	h.addConnection(func(conn *accountingsync.AccountingConnection) {
		syncingConnection(conn)
		conn.SyncStartDate = nil
	})
	second := h.addConnection(syncingConnection)

	inv := h.postedInvoice(pulid.MustNew("cus_"), invoiceSpec{number: "INV-100"})
	require.NoError(t, h.enqueuer.Enqueue(
		t.Context(),
		services.InvoiceSyncRequest(inv, accountingsync.SyncSourceInvoicePosted),
	))

	records := h.records.all()
	require.Len(t, records, 2)
	connections := []pulid.ID{records[0].ConnectionID, records[1].ConnectionID}
	assert.ElementsMatch(t, []pulid.ID{h.conn.ID, second.ID}, connections)
	for _, record := range records {
		assert.Equal(t, accountingsync.SyncObjectInvoice, record.ObjectType)
		assert.Equal(t, inv.ID, record.ObjectID)
		assert.Equal(t, "INV-100", record.ObjectNumber)
		assert.Equal(t, accountingsync.SyncOperationCreate, record.Operation)
		assert.Equal(t, int64(1), record.Revision)
		assert.Equal(t, accountingsync.SyncSourceInvoicePosted, record.SourceEvent)
		assert.Equal(t, accountingsync.SyncStatusQueued, record.Status)
		assert.Equal(t, "Invoice:"+inv.ID.String()+":Create:1", record.IdempotencyKey)
		require.NotNil(t, record.DocumentDate)
		assert.Equal(t, inv.InvoiceDate, *record.DocumentDate)
	}
	assert.NotEqual(t, records[0].RequestID, records[1].RequestID,
		"each connection derives its own request id")
}

func TestRequestIDIsStableAndDerivedFromConnectionAndKey(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	inv := h.postedInvoice(pulid.MustNew("cus_"), invoiceSpec{})
	record := h.enqueueInvoice(t, inv)

	sum := sha256.Sum256([]byte(h.conn.ID.String() + "|" + record.IdempotencyKey))
	assert.Equal(t, "trn-"+hex.EncodeToString(sum[:])[:40], record.RequestID)
	assert.Len(t, record.RequestID, len("trn-")+40)

	again := NewRecordFor(h.conn, services.InvoiceSyncRequest(inv, accountingsync.SyncSourceSafetyNet), 1)
	assert.Equal(t, record.RequestID, again.RequestID, "a re-push replays under the same request id")
}

func TestEnqueueSkipsDocumentsDatedBeforeTheStartDate(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	customerID := pulid.MustNew("cus_")
	before := h.postedInvoice(customerID, invoiceSpec{date: syncStartDate - 1})
	onStart := h.postedInvoice(customerID, invoiceSpec{date: syncStartDate})

	for _, inv := range []*invoice.Invoice{before, onStart} {
		require.NoError(t, h.enqueuer.Enqueue(
			t.Context(),
			services.InvoiceSyncRequest(inv, accountingsync.SyncSourceInvoicePosted),
		))
	}

	assert.Empty(t, h.records.find(accountingsync.SyncObjectInvoice, before.ID, accountingsync.SyncOperationCreate))
	assert.Len(t, h.records.find(accountingsync.SyncObjectInvoice, onStart.ID, accountingsync.SyncOperationCreate), 1)
}

func TestEnqueueHoldsDocumentsForApprovalWhenAutoSyncIsOff(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.updateConnection(func(conn *accountingsync.AccountingConnection) { conn.AutoSync = false })

	record := h.enqueueInvoice(t, h.postedInvoice(pulid.MustNew("cus_"), invoiceSpec{}))

	assert.Equal(t, accountingsync.SyncStatusAwaitingApproval, record.Status)
	assert.Empty(t, h.dispatcher.kicked(), "a held record does not wake the dispatcher")
}

func TestEnqueueQueuesDependenciesEvenWhenAutoSyncIsOff(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.updateConnection(func(conn *accountingsync.AccountingConnection) { conn.AutoSync = false })
	customerID := pulid.MustNew("cus_")

	h.enqueue(t, &services.AccountingSyncEnqueueRequest{
		ObjectType:  accountingsync.SyncObjectCustomer,
		ObjectID:    customerID,
		Operation:   accountingsync.SyncOperationCreate,
		Revision:    1,
		SourceEvent: accountingsync.SyncSourceDependencyOf,
	})

	record := h.only(t, accountingsync.SyncObjectCustomer, customerID, accountingsync.SyncOperationCreate)
	assert.Equal(t, accountingsync.SyncStatusQueued, record.Status)
}

func TestEnqueueCustomerUpdatesOnlyWhenTheCustomerIsMapped(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	unmapped := pulid.MustNew("cus_")
	proposed := pulid.MustNew("cus_")
	mapped := pulid.MustNew("cus_")
	h.mappings.set(h.conn, customerTarget(proposed, "Proposed Co"), accountingsync.MappingStateProposed, "qb-9")
	h.confirm(customerTarget(mapped, "Mapped Co"), "qb-10")

	for _, customerID := range []pulid.ID{unmapped, proposed, mapped} {
		require.NoError(t, h.enqueuer.Enqueue(
			t.Context(),
			services.CustomerSyncRequest(h.tenant, customerID, "Customer", 4),
		))
	}

	assert.Empty(t, h.records.find(accountingsync.SyncObjectCustomer, unmapped, accountingsync.SyncOperationUpdate))
	assert.Empty(t, h.records.find(accountingsync.SyncObjectCustomer, proposed, accountingsync.SyncOperationUpdate))
	record := h.only(t, accountingsync.SyncObjectCustomer, mapped, accountingsync.SyncOperationUpdate)
	assert.Equal(t, int64(4), record.Revision)
	assert.Equal(t, accountingsync.SyncSourceCustomerUpdated, record.SourceEvent)
	assert.Nil(t, record.DocumentDate, "a customer carries no document date")
}

func TestEnqueueDependencyCustomerDoesNotNeedAMapping(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	customerID := pulid.MustNew("cus_")

	h.enqueue(t, &services.AccountingSyncEnqueueRequest{
		ObjectType:  accountingsync.SyncObjectCustomer,
		ObjectID:    customerID,
		Operation:   accountingsync.SyncOperationCreate,
		Revision:    1,
		SourceEvent: accountingsync.SyncSourceDependencyOf,
	})

	h.only(t, accountingsync.SyncObjectCustomer, customerID, accountingsync.SyncOperationCreate)
}

func TestEnqueueUpdateSupersedesOlderRevisions(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	payment := &customerpayment.Payment{
		ID:              pulid.MustNew("cpay_"),
		OrganizationID:  h.tenant.OrgID,
		BusinessUnitID:  h.tenant.BuID,
		ReferenceNumber: "CHK-1",
		AccountingDate:  aprilTenth,
		Version:         1,
	}
	require.NoError(t, h.enqueuer.Enqueue(t.Context(), services.PaymentSyncRequest(
		payment, accountingsync.SyncOperationCreate, accountingsync.SyncSourceCustomerPaymentPosted)))
	payment.Version = 2
	require.NoError(t, h.enqueuer.Enqueue(t.Context(), services.PaymentSyncRequest(
		payment, accountingsync.SyncOperationUpdate, accountingsync.SyncSourceCustomerPaymentApplied)))
	payment.Version = 3
	require.NoError(t, h.enqueuer.Enqueue(t.Context(), services.PaymentSyncRequest(
		payment, accountingsync.SyncOperationUpdate, accountingsync.SyncSourceCustomerPaymentApplied)))

	updates := h.records.find(accountingsync.SyncObjectCustomerPayment, payment.ID, accountingsync.SyncOperationUpdate)
	require.Len(t, updates, 2)
	byRevision := map[int64]accountingsync.SyncStatus{}
	for _, record := range updates {
		byRevision[record.Revision] = record.Status
	}
	assert.Equal(t, accountingsync.SyncStatusSuperseded, byRevision[2])
	assert.Equal(t, accountingsync.SyncStatusQueued, byRevision[3])

	create := h.only(t, accountingsync.SyncObjectCustomerPayment, payment.ID, accountingsync.SyncOperationCreate)
	assert.Equal(t, accountingsync.SyncStatusQueued, create.Status, "an update never supersedes the create")
}

func TestEnqueueIsIdempotent(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	inv := h.postedInvoice(pulid.MustNew("cus_"), invoiceSpec{})
	req := services.InvoiceSyncRequest(inv, accountingsync.SyncSourceInvoicePosted)

	require.NoError(t, h.enqueuer.Enqueue(t.Context(), req))
	require.NoError(t, h.enqueuer.Enqueue(t.Context(), req))

	assert.Len(t, h.records.all(), 1)
	assert.Equal(t, []pulid.ID{h.conn.ID}, h.dispatcher.kicked(), "only the insert wakes the dispatcher")
}

func TestEnqueueKicksTheDispatcherOnlyAfterCommit(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	txCtx, hooks := ports.WithAfterCommitHooks(t.Context())
	customerID := pulid.MustNew("cus_")

	for range 2 {
		require.NoError(t, h.enqueuer.Enqueue(
			txCtx,
			services.InvoiceSyncRequest(h.postedInvoice(customerID, invoiceSpec{}), accountingsync.SyncSourceInvoicePosted),
		))
	}
	assert.Len(t, h.records.all(), 2, "the outbox row is written inside the transaction")
	assert.Empty(t, h.dispatcher.kicked(), "no kick before commit")

	hooks.Run(t.Context())

	assert.Len(t, h.dispatcher.kicked(), 2)
	assert.Equal(t, h.conn.ID, h.dispatcher.kicked()[0])
}

func TestEnqueueToleratesAFailedKick(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.dispatcher.kickErr = assert.AnError

	record := h.enqueueInvoice(t, h.postedInvoice(pulid.MustNew("cus_"), invoiceSpec{}))

	assert.Equal(t, accountingsync.SyncStatusQueued, record.Status)
}

func TestEnqueueWritesWithoutADispatcher(t *testing.T) {
	t.Parallel()

	h := newHarness(t, withoutDispatcher())

	record := h.enqueueInvoice(t, h.postedInvoice(pulid.MustNew("cus_"), invoiceSpec{}))

	assert.Equal(t, accountingsync.SyncStatusQueued, record.Status)
}

func TestEnqueueRejectsMalformedRequests(t *testing.T) {
	t.Parallel()

	valid := func() *services.AccountingSyncEnqueueRequest {
		return &services.AccountingSyncEnqueueRequest{
			ObjectType:   accountingsync.SyncObjectInvoice,
			ObjectID:     pulid.MustNew("inv_"),
			Operation:    accountingsync.SyncOperationCreate,
			Revision:     1,
			SourceEvent:  accountingsync.SyncSourceInvoicePosted,
			DocumentDate: aprilTenth,
		}
	}
	tests := []struct {
		name   string
		mutate func(*services.AccountingSyncEnqueueRequest) *services.AccountingSyncEnqueueRequest
	}{
		{name: "nil request", mutate: func(*services.AccountingSyncEnqueueRequest) *services.AccountingSyncEnqueueRequest {
			return nil
		}},
		{name: "unknown object type", mutate: func(r *services.AccountingSyncEnqueueRequest) *services.AccountingSyncEnqueueRequest {
			r.ObjectType = "Vendor"
			return r
		}},
		{name: "unknown operation", mutate: func(r *services.AccountingSyncEnqueueRequest) *services.AccountingSyncEnqueueRequest {
			r.Operation = "Delete"
			return r
		}},
		{name: "unknown source", mutate: func(r *services.AccountingSyncEnqueueRequest) *services.AccountingSyncEnqueueRequest {
			r.SourceEvent = "Guess"
			return r
		}},
		{name: "missing id", mutate: func(r *services.AccountingSyncEnqueueRequest) *services.AccountingSyncEnqueueRequest {
			r.ObjectID = pulid.Nil
			return r
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := newHarness(t)
			req := tc.mutate(valid())
			if req != nil {
				req.TenantInfo = h.tenant
			}

			require.Error(t, h.enqueuer.Enqueue(t.Context(), req))
			assert.Empty(t, h.records.all())
		})
	}
}

func TestEnqueueAccountingSyncIgnoresAMissingEnqueuer(t *testing.T) {
	t.Parallel()

	require.NoError(t, services.EnqueueAccountingSync(t.Context(), nil, &services.AccountingSyncEnqueueRequest{}))
}
