package accountingdriftservice

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompareRaisesTheKindEachDifferenceIs(t *testing.T) {
	t.Parallel()

	found := func(total string, voided bool) *services.AccountingDocumentState {
		return &services.AccountingDocumentState{
			ExternalID: "9",
			Found:      true,
			Voided:     voided,
			Total:      decimalOf(total),
		}
	}
	cases := []struct {
		name       string
		objectType accountingsync.SyncObjectType
		trenova    repositories.AccountingDriftState
		provider   *services.AccountingDocumentState
		kind       accountingsync.DriftKind
		differs    bool
	}{
		{
			name:       "same total",
			objectType: accountingsync.SyncObjectInvoice,
			trenova:    repositories.AccountingDriftState{AmountMinor: 125_000},
			provider:   found("1250.00", false),
		},
		{
			name:       "provider total edited",
			objectType: accountingsync.SyncObjectInvoice,
			trenova:    repositories.AccountingDriftState{AmountMinor: 125_000},
			provider:   found("1200.00", false),
			kind:       accountingsync.DriftAmountMismatch,
			differs:    true,
		},
		{
			name:       "credit memo compares absolute totals",
			objectType: accountingsync.SyncObjectCreditMemo,
			trenova:    repositories.AccountingDriftState{AmountMinor: 5_000},
			provider:   found("-50.00", false),
		},
		{
			name:       "a reflected memo already brought Trenova to the provider's total",
			objectType: accountingsync.SyncObjectInvoice,
			trenova: repositories.AccountingDriftState{
				AmountMinor:    125_000,
				ReflectedMinor: -5_000,
			},
			provider: found("1200.00", false),
		},
		{
			name:       "deleted in the provider",
			objectType: accountingsync.SyncObjectInvoice,
			trenova:    repositories.AccountingDriftState{AmountMinor: 125_000},
			provider:   &services.AccountingDocumentState{ExternalID: "9"},
			kind:       accountingsync.DriftDeletedInProvider,
			differs:    true,
		},
		{
			name:       "deleted in the provider after Trenova voided it",
			objectType: accountingsync.SyncObjectInvoice,
			trenova:    repositories.AccountingDriftState{AmountMinor: 125_000, Voided: true},
			provider:   &services.AccountingDocumentState{ExternalID: "9"},
		},
		{
			name:       "voided in the provider",
			objectType: accountingsync.SyncObjectCustomerPayment,
			trenova:    repositories.AccountingDriftState{AmountMinor: 40_000},
			provider:   found("400.00", true),
			kind:       accountingsync.DriftVoidedInProvider,
			differs:    true,
		},
		{
			name:       "voided on both sides",
			objectType: accountingsync.SyncObjectCustomerPayment,
			trenova:    repositories.AccountingDriftState{AmountMinor: 40_000, Voided: true},
			provider:   found("0", true),
		},
		{
			name:       "voided in Trenova, live in the provider",
			objectType: accountingsync.SyncObjectCarrierBill,
			trenova:    repositories.AccountingDriftState{AmountMinor: 90_000, Voided: true},
			provider:   found("900.00", false),
			kind:       accountingsync.DriftStatusMismatch,
			differs:    true,
		},
		{
			name:       "a credit application has no amount to compare",
			objectType: accountingsync.SyncObjectCreditApplication,
			trenova:    repositories.AccountingDriftState{AmountMinor: 2_500},
			provider:   found("0", false),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			trenova := tc.trenova
			c := &comparison{
				record:   &accountingsync.AccountingSyncRecord{ObjectType: tc.objectType},
				trenova:  &trenova,
				provider: tc.provider,
			}
			kind, differs := c.kind()
			assert.Equal(t, tc.differs, differs)
			if tc.differs {
				assert.Equal(t, tc.kind, kind)
			}
		})
	}
}

func TestReconcileOpensAFindingWithBothValuesAndTheEditor(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	doc := h.synced(accountingsync.SyncObjectInvoice, "101", 125_000)
	h.provider(accountingsync.SyncObjectInvoice, "101", "1200.00")
	h.synced(accountingsync.SyncObjectInvoice, "102", 50_000)
	h.provider(accountingsync.SyncObjectInvoice, "102", "500.00")

	result := h.reconcile(t)

	assert.Equal(t, 2, result.Compared)
	assert.Equal(t, 1, result.Opened)
	assert.False(t, result.More)
	open := h.findings.open()
	require.Len(t, open, 1)
	finding := open[0]
	assert.Equal(t, accountingsync.DriftAmountMismatch, finding.Kind)
	assert.Equal(t, doc.record.ObjectID, finding.ObjectID)
	assert.Equal(t, "DOC-101", finding.ObjectNumber)
	assert.Equal(t, "101", finding.ExternalID)
	assert.Equal(t, "https://books.example/Invoice/101", finding.ExternalURL)
	require.NotNil(t, finding.TrenovaMinor)
	require.NotNil(t, finding.ProviderMinor)
	require.NotNil(t, finding.DifferenceMinor)
	assert.Equal(t, int64(125_000), *finding.TrenovaMinor)
	assert.Equal(t, int64(120_000), *finding.ProviderMinor)
	assert.Equal(t, int64(-5_000), *finding.DifferenceMinor)
	assert.Equal(t, "Pat Bookkeeper", finding.ProviderModifiedBy)
	require.NotNil(t, finding.ProviderModifiedAt)

	events := h.events.Published()
	require.Len(t, events, 1)
	assert.Equal(t, agent.EventAccountingDriftDetected, events[0].Kind)
	assert.Equal(t, finding.ID, events[0].SubjectID)
}

func TestReconcileUpdatesAnOpenFindingRatherThanRaisingASecond(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.synced(accountingsync.SyncObjectInvoice, "101", 125_000)
	h.provider(accountingsync.SyncObjectInvoice, "101", "1200.00")
	h.reconcile(t)

	h.provider(accountingsync.SyncObjectInvoice, "101", "1100.00")
	h.now = h.now.Add(24 * time.Hour)
	result := h.reconcile(t)

	assert.Equal(t, 0, result.Opened)
	assert.Equal(t, 1, result.Updated)
	open := h.findings.open()
	require.Len(t, open, 1)
	assert.Equal(t, int64(110_000), *open[0].ProviderMinor)
	assert.Equal(t, h.now.Unix(), open[0].LastSeenAt)
	assert.Len(t, h.events.Published(), 1)
}

func TestReconcileResolvesAFindingThatNoLongerDiffers(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.synced(accountingsync.SyncObjectInvoice, "101", 125_000)
	h.provider(accountingsync.SyncObjectInvoice, "101", "1200.00")
	h.reconcile(t)

	h.provider(accountingsync.SyncObjectInvoice, "101", "1250.00")
	result := h.reconcile(t)

	assert.Equal(t, 1, result.Resolved)
	require.Len(t, h.findings.all(), 1)
	finding := h.findings.all()[0]
	assert.Equal(t, accountingsync.DriftStatusResolved, finding.Status)
	assert.Equal(t, accountingsync.DriftNoLongerDiffers, finding.Resolution)
}

func TestReconcileReplacesAFindingWhoseKindChanged(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.synced(accountingsync.SyncObjectInvoice, "101", 125_000)
	h.provider(accountingsync.SyncObjectInvoice, "101", "1200.00")
	h.reconcile(t)

	h.reader.mu.Lock()
	delete(h.reader.docs, "Invoice:101")
	h.reader.mu.Unlock()
	result := h.reconcile(t)

	assert.Equal(t, 1, result.Resolved)
	assert.Equal(t, 1, result.Opened)
	open := h.findings.open()
	require.Len(t, open, 1)
	assert.Equal(t, accountingsync.DriftDeletedInProvider, open[0].Kind)
	assert.Equal(t, "Deleted", open[0].ProviderState)
	assert.Nil(t, open[0].ProviderMinor)
}

func TestReconcileLeavesAnObjectAloneWhileItHasARecordInFlight(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	doc := h.synced(accountingsync.SyncObjectInvoice, "101", 125_000)
	h.provider(accountingsync.SyncObjectInvoice, "101", "1200.00")
	h.reconcile(t)

	update := accountingsync.NewAccountingSyncRecord(&accountingsync.NewSyncRecord{
		TenantInfo:   h.tenant,
		ConnectionID: h.conn.ID,
		Key: accountingsync.SyncRecordKey{
			ObjectType: accountingsync.SyncObjectInvoice,
			ObjectID:   doc.record.ObjectID,
			Operation:  accountingsync.SyncOperationUpdate,
			Revision:   2,
		},
		SourceEvent: accountingsync.SyncSourceDriftResolved,
		At:          h.now.Unix(),
	})
	h.records.add(update)
	h.provider(accountingsync.SyncObjectInvoice, "101", "1250.00")

	result := h.reconcile(t)

	assert.Equal(t, 0, result.Compared)
	assert.Equal(t, 1, result.Skipped)
	assert.Equal(t, 0, result.Resolved)
	assert.Len(t, h.findings.open(), 1)
}

func TestReconcileResolvesAPushedFindingAsPushedOnceTheProviderMatches(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.synced(accountingsync.SyncObjectInvoice, "101", 125_000)
	h.provider(accountingsync.SyncObjectInvoice, "101", "1200.00")
	h.reconcile(t)
	finding := h.findings.open()[0]
	finding.MarkPushed(pulid.MustNew("acctsr_"), pulid.MustNew("usr_"))
	_, err := h.findings.Update(t.Context(), finding)
	require.NoError(t, err)

	h.provider(accountingsync.SyncObjectInvoice, "101", "1250.00")
	h.reconcile(t)

	resolved := h.findings.all()[0]
	assert.Equal(t, accountingsync.DriftStatusResolved, resolved.Status)
	assert.Equal(t, accountingsync.DriftPushedTrenovaValue, resolved.Resolution)
}

func TestReconcileSkipsADocumentTheProviderDidNotAnswerFor(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.synced(accountingsync.SyncObjectInvoice, "101", 125_000)
	h.reader.omit["101"] = true

	result := h.reconcile(t)

	assert.Equal(t, 0, result.Compared)
	assert.Equal(t, 1, result.Skipped)
	assert.Empty(t, h.findings.all())
}

func TestReconcileReadsInBatchesThatFitTheProviderAndPassesRefs(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.reader.maxRead = 2
	for _, id := range []string{"201", "202", "203"} {
		doc := h.synced(accountingsync.SyncObjectCarrierBill, id, 10_000)
		doc.record.ExternalRefs = map[string]string{"documentType": "Bill"}
		h.provider(accountingsync.SyncObjectCarrierBill, id, "100.00")
	}

	result := h.reconcile(t)

	assert.Equal(t, 3, result.Compared)
	require.Len(t, h.reader.calls, 2)
	assert.Len(t, h.reader.calls[0].ids, 2)
	assert.Len(t, h.reader.calls[1].ids, 1)
	assert.Equal(t, "Bill", h.reader.calls[0].refs[0]["documentType"])
}

func TestReconcilePagesByTheLastRecord(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	for _, id := range []string{"1", "2", "3"} {
		h.synced(accountingsync.SyncObjectInvoice, id, 10_000)
		h.provider(accountingsync.SyncObjectInvoice, id, "100.00")
	}
	first, err := h.svc.ReconcileBatch(t.Context(), &services.ReconcileAccountingDriftRequest{
		TenantInfo:   h.tenant,
		ConnectionID: h.conn.ID,
		Limit:        2,
	})
	require.NoError(t, err)
	assert.True(t, first.More)
	assert.Equal(t, h.source.records[1].ID, first.LastID)

	second, err := h.svc.ReconcileBatch(t.Context(), &services.ReconcileAccountingDriftRequest{
		TenantInfo:   h.tenant,
		ConnectionID: h.conn.ID,
		AfterID:      first.LastID,
		Limit:        2,
	})
	require.NoError(t, err)
	assert.False(t, second.More)
	assert.Equal(t, 1, second.Compared)
	assert.Equal(t, h.source.records[2].ID, second.LastID)
}

func TestReconcileCapsEventsAtTheBudget(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	for _, id := range []string{"1", "2", "3"} {
		h.synced(accountingsync.SyncObjectInvoice, id, 10_000)
		h.provider(accountingsync.SyncObjectInvoice, id, "90.00")
	}
	result, err := h.svc.ReconcileBatch(t.Context(), &services.ReconcileAccountingDriftRequest{
		TenantInfo:   h.tenant,
		ConnectionID: h.conn.ID,
		EventBudget:  2,
	})
	require.NoError(t, err)

	assert.Equal(t, 3, result.Opened)
	assert.Equal(t, 2, result.Events)
	assert.Len(t, h.events.Published(), 2)
}

func TestReconcileHoldsWhenTheConnectionCannotBeUsed(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.synced(accountingsync.SyncObjectInvoice, "101", 125_000)
	h.connSvc.failSession = true

	result := h.reconcile(t)

	assert.True(t, result.Held)
	assert.Empty(t, h.reader.calls)
	assert.NotEmpty(t, h.conns.current().DriftErrorMessage)
}

func TestReconcileHoldsWhileTheConnectionIsPaused(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	paused := h.now.Unix()
	h.conn.PausedAt = &paused
	h.synced(accountingsync.SyncObjectInvoice, "101", 125_000)

	result := h.reconcile(t)

	assert.True(t, result.Held)
	assert.Empty(t, h.reader.calls)
}

func TestReconcileRecordsAProviderReadFailure(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.synced(accountingsync.SyncObjectInvoice, "101", 125_000)
	h.reader.failWith = errors.New("401 unauthorized")

	_, err := h.svc.ReconcileBatch(t.Context(), &services.ReconcileAccountingDriftRequest{
		TenantInfo:   h.tenant,
		ConnectionID: h.conn.ID,
	})

	require.Error(t, err)
	conn := h.conns.current()
	assert.Equal(t, accountingsync.SyncErrorAuth, conn.DriftErrorCategory)
	assert.Equal(t, "401 unauthorized", conn.DriftErrorMessage)
}

func TestRecheckComparesTheDocumentAChangeNames(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.synced(accountingsync.SyncObjectInvoice, "101", 125_000)
	h.provider(accountingsync.SyncObjectInvoice, "101", "1200.00")
	h.synced(accountingsync.SyncObjectInvoice, "102", 50_000)
	h.provider(accountingsync.SyncObjectInvoice, "102", "400.00")

	result, err := h.svc.RecheckDocuments(t.Context(), &services.RecheckAccountingDriftRequest{
		TenantInfo:   h.tenant,
		ConnectionID: h.conn.ID,
		Documents: []services.AccountingChangedDocument{{
			ObjectTypes: []accountingsync.SyncObjectType{
				accountingsync.SyncObjectInvoice,
				accountingsync.SyncObjectDebitMemo,
			},
			ExternalID: "101",
		}},
		EventBudget: 20,
	})
	require.NoError(t, err)

	assert.Equal(t, 1, result.Compared)
	open := h.findings.open()
	require.Len(t, open, 1)
	assert.Equal(t, "101", open[0].ExternalID)
}

func TestRecheckComparesTheRecreatedDocumentNotTheDeletedOne(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	doc := h.synced(accountingsync.SyncObjectInvoice, "101", 125_000)
	recreate := accountingsync.NewAccountingSyncRecord(&accountingsync.NewSyncRecord{
		TenantInfo:   h.tenant,
		ConnectionID: h.conn.ID,
		Key: accountingsync.SyncRecordKey{
			ObjectType: accountingsync.SyncObjectInvoice,
			ObjectID:   doc.record.ObjectID,
			Operation:  accountingsync.SyncOperationRecreate,
			Revision:   2,
		},
		SourceEvent: accountingsync.SyncSourceDriftResolved,
		At:          h.now.Unix(),
	})
	recreate.Status = accountingsync.SyncStatusSynced
	recreate.ExternalID = "150"
	h.records.add(recreate)
	h.provider(accountingsync.SyncObjectInvoice, "150", "1250.00")

	result, err := h.svc.RecheckDocuments(t.Context(), &services.RecheckAccountingDriftRequest{
		TenantInfo:   h.tenant,
		ConnectionID: h.conn.ID,
		Documents: []services.AccountingChangedDocument{{
			ObjectTypes: []accountingsync.SyncObjectType{accountingsync.SyncObjectInvoice},
			ExternalID:  "101",
			Operation:   services.AccountingChangeDelete,
		}},
	})
	require.NoError(t, err)

	assert.Equal(t, 1, result.Compared)
	assert.Empty(t, h.findings.all())
	require.Len(t, h.reader.calls, 1)
	assert.Equal(t, []string{"150"}, h.reader.calls[0].ids)
}

func TestFinishCheckStampsTheConnection(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.conn.DriftErrorMessage = "earlier failure"

	require.NoError(t, h.svc.FinishCheck(t.Context(), &services.FinishAccountingDriftCheckRequest{
		TenantInfo:   h.tenant,
		ConnectionID: h.conn.ID,
	}))

	conn := h.conns.current()
	require.NotNil(t, conn.DriftCheckedAt)
	assert.Equal(t, h.now.Unix(), *conn.DriftCheckedAt)
	assert.Empty(t, conn.DriftErrorMessage)
}

func TestAReadFailureMessageIsKeptToItsLimit(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.synced(accountingsync.SyncObjectInvoice, "101", 125_000)
	h.reader.failWith = errors.New(strings.Repeat("x", 5000))

	_, err := h.svc.ReconcileBatch(t.Context(), &services.ReconcileAccountingDriftRequest{
		TenantInfo:   h.tenant,
		ConnectionID: h.conn.ID,
	})

	require.Error(t, err)
	assert.Less(t, len(h.conns.current().DriftErrorMessage), 5000)
}
