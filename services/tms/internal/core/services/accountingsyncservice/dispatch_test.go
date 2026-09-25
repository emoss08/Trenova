package accountingsyncservice

import (
	"errors"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/customerpayment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/watchtowersources"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (h *harness) readyInvoice(t *testing.T) *accountingsync.AccountingSyncRecord {
	t.Helper()
	customerID := pulid.MustNew("cus_")
	h.confirm(customerTarget(customerID, "Acme"), "qb-cust-acme")
	h.confirm(freightTarget(), "qb-item-freight")
	return h.enqueueInvoice(t, h.postedInvoice(customerID, invoiceSpec{number: "INV-500"}))
}

func TestDrainClaimsCustomersThenSalesDocumentsThenPayments(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	customerID, _ := h.mappedCustomer("Acme")
	h.mappings.parties[customerID] = &services.AccountingPartyDraft{DisplayName: "Acme"}
	h.confirm(freightTarget(), "qb-item-freight")
	h.mapPaymentAccounts(customerpayment.MethodCheck)
	inv := h.postedInvoice(customerID, invoiceSpec{number: "INV-1"})
	payment := h.postedPayment(customerID, paymentSpec{
		amountMinor:  150000,
		applications: []*customerpayment.Application{{InvoiceID: inv.ID, AppliedAmountMinor: 150000}},
	})
	h.enqueuePayment(t, payment, accountingsync.SyncOperationCreate)
	h.enqueueInvoice(t, inv)
	require.NoError(t, h.enqueuer.Enqueue(t.Context(), services.CustomerSyncRequest(h.tenant, customerID, "Acme", 2)))

	result := h.drain(t)

	assert.Equal(t, 3, result.Claimed)
	assert.Equal(t, 3, result.Synced)
	assert.Equal(t, []string{"UpsertCustomer", "CreateSalesDocument", "SavePayment"}, h.writer.methods(),
		"a payment claimed with its invoice is sent after it, and links it")
	docs := paymentDocs(t, h)
	require.Len(t, docs, 1)
	invoiceRecord := h.only(t, accountingsync.SyncObjectInvoice, inv.ID, accountingsync.SyncOperationCreate)
	assert.Equal(t, invoiceRecord.ExternalID, docs[0].Applications[0].InvoiceExternalID)
}

func TestDrainMarksASuccessfulPushSyncedAndRecordsTheAttempt(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	record := h.readyInvoice(t)
	heartbeats := 0

	result, err := h.svc.Drain(t.Context(), &services.DrainAccountingSyncRequest{
		TenantInfo:   h.tenant,
		ConnectionID: h.conn.ID,
		Heartbeat:    func() { heartbeats++ },
	})
	require.NoError(t, err)

	assert.Equal(t, 1, result.Synced)
	assert.Equal(t, 1, heartbeats, "the activity heartbeats for every pushed record")
	synced := h.records.get(record.ID)
	assert.Equal(t, accountingsync.SyncStatusSynced, synced.Status)
	assert.Equal(t, "qb-CreateSalesDocument-1", synced.ExternalID)
	assert.Equal(t, "QBqb-CreateSalesDocument-1", synced.ExternalDocNumber)
	assert.Equal(t, "https://qbo.test/Invoice/qb-CreateSalesDocument-1", synced.ExternalURL)
	assert.Equal(t, "qb-CreateSalesDocument-1", synced.ExternalRefs[accountingsync.ExternalRefDocument])
	assert.Nil(t, synced.NextAttemptAt)
	assert.Nil(t, synced.LeaseExpiresAt)
	attempts := h.records.attemptsFor(record.ID)
	require.Len(t, attempts, 1)
	assert.Equal(t, accountingsync.SyncAttemptSynced, attempts[0].Outcome)
	assert.Equal(t, 1, attempts[0].AttemptNumber)
	assert.Empty(t, attempts[0].ErrorCategory)
	assert.Empty(t, h.events.Published())
}

func TestTransientFailuresBackOffExponentiallyThenDeadLetter(t *testing.T) {
	t.Parallel()

	for _, category := range []accountingsync.SyncErrorCategory{
		accountingsync.SyncErrorTransient,
		accountingsync.SyncErrorRateLimited,
	} {
		t.Run(string(category), func(t *testing.T) {
			t.Parallel()

			h := newHarness(t)
			record := h.readyInvoice(t)
			for range accountingsync.MaxSyncAttempts {
				h.writer.failNext("CreateSalesDocument", providerFault(category, "Service unavailable", ""))
			}

			wantDelay := 30 * time.Second
			for attempt := 1; attempt < accountingsync.MaxSyncAttempts; attempt++ {
				result := h.drain(t)
				require.Equal(t, 1, result.Retrying, "attempt %d", attempt)

				retrying := h.records.get(record.ID)
				assert.Equal(t, accountingsync.SyncStatusRetrying, retrying.Status)
				assert.Equal(t, attempt, retrying.AttemptCount)
				assert.Equal(t, category, retrying.ErrorCategory)
				nearNow(t, wantDelay, retrying.NextAttemptAt)
				assert.LessOrEqual(t, wantDelay, 6*time.Hour)
				wantDelay *= 2
				h.records.makeDue(record.ID)
			}
			assert.Empty(t, h.events.Published(), "a retry raises nothing")

			result := h.drain(t)

			assert.Equal(t, 1, result.DeadLettered)
			dead := h.records.get(record.ID)
			assert.Equal(t, accountingsync.SyncStatusDeadLettered, dead.Status)
			assert.Equal(t, accountingsync.MaxSyncAttempts, dead.AttemptCount)
			assert.Nil(t, dead.NextAttemptAt)
			assert.Len(t, h.records.attemptsFor(record.ID), accountingsync.MaxSyncAttempts,
				"every push records an attempt")
			events := h.events.Published()
			require.Len(t, events, 1)
			assert.Equal(t, agent.EventAccountingSyncFailed, events[0].Kind)
			assert.Equal(t, record.ID, events[0].SubjectID)
			assert.Equal(t, h.tenant.OrgID, events[0].TenantInfo.OrgID)
		})
	}
}

func TestRetryScheduleIsCappedAtSixHours(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 30*time.Second, accountingsync.SyncRetryDelay(1))
	assert.Equal(t, time.Minute, accountingsync.SyncRetryDelay(2))
	assert.Equal(t, 6*time.Hour, accountingsync.SyncRetryDelay(12))
	assert.Equal(t, 6*time.Hour, accountingsync.SyncRetryDelay(100))
}

func TestUnclassifiedProviderErrorIsRetried(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	record := h.readyInvoice(t)
	h.writer.failNext("CreateSalesDocument", errors.New("connection reset by peer"))

	result := h.drain(t)

	assert.Equal(t, 1, result.Retrying)
	retrying := h.records.get(record.ID)
	assert.Equal(t, accountingsync.SyncErrorTransient, retrying.ErrorCategory)
	assert.Equal(t, "connection reset by peer", retrying.ErrorMessage)
}

func TestAuthFailureReportsHealthAndWaitsWithoutSpendingAnAttempt(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	record := h.readyInvoice(t)
	cause := providerFault(accountingsync.SyncErrorAuth, "Token revoked", "Reconnect QuickBooks Online")
	h.writer.failNext("CreateSalesDocument", cause)

	result := h.drain(t)

	assert.Equal(t, 1, result.Waiting)
	waiting := h.records.get(record.ID)
	assert.Equal(t, accountingsync.SyncStatusRetrying, waiting.Status)
	assert.Equal(t, 0, waiting.AttemptCount, "an auth failure spends no attempt")
	assert.Equal(t, accountingsync.SyncErrorAuth, waiting.ErrorCategory)
	nearNow(t, 15*time.Minute, waiting.NextAttemptAt)
	reports := h.connService.reports()
	require.Len(t, reports, 1)
	assert.Equal(t, h.conn.ID, reports[0].connectionID)
	assert.ErrorIs(t, reports[0].cause, cause)
	assert.Empty(t, h.events.Published())
	assert.Len(t, h.records.attemptsFor(record.ID), 1)
}

func TestNonRetryableProviderErrorsBlockWithTheirResolution(t *testing.T) {
	t.Parallel()

	for _, category := range []accountingsync.SyncErrorCategory{
		accountingsync.SyncErrorValidation,
		accountingsync.SyncErrorMapping,
		accountingsync.SyncErrorClosedPeriod,
		accountingsync.SyncErrorCurrency,
		accountingsync.SyncErrorDuplicate,
		accountingsync.SyncErrorNotFound,
		accountingsync.SyncErrorConflict,
	} {
		t.Run(string(category), func(t *testing.T) {
			t.Parallel()

			h := newHarness(t)
			record := h.readyInvoice(t)
			h.writer.failNext("CreateSalesDocument",
				providerFault(category, "Provider refused", "Fix it in Trenova, then retry"))

			result := h.drain(t)

			assert.Equal(t, 1, result.Blocked)
			blocked := h.records.get(record.ID)
			assert.Equal(t, accountingsync.SyncStatusBlocked, blocked.Status)
			assert.Equal(t, category, blocked.ErrorCategory)
			assert.Equal(t, "QB-"+string(category), blocked.ErrorCode)
			assert.Equal(t, "Provider refused", blocked.ErrorMessage)
			assert.Equal(t, "Fix it in Trenova, then retry", blocked.Resolution)
			assert.Nil(t, blocked.NextAttemptAt)
			assert.Equal(t, 1, blocked.AttemptCount)
			events := h.events.Published()
			require.Len(t, events, 1)
			assert.Equal(t, agent.EventAccountingSyncBlocked, events[0].Kind)
			assert.Equal(t, record.ID, events[0].SubjectID)
			attempts := h.records.attemptsFor(record.ID)
			require.Len(t, attempts, 1)
			assert.Equal(t, accountingsync.SyncAttemptBlocked, attempts[0].Outcome)
			assert.Equal(t, category, attempts[0].ErrorCategory)
		})
	}
}

func TestBlockedPushRefreshesTheWatchtowerAttention(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	record := h.readyInvoice(t)
	h.records.attention = []repositories.AccountingSyncAttentionGroup{{
		Status:         accountingsync.SyncStatusBlocked,
		ErrorCategory:  accountingsync.SyncErrorValidation,
		Resolution:     "Correct the document",
		Count:          1,
		SampleRecordID: record.ID,
	}}
	h.writer.failNext("CreateSalesDocument",
		providerFault(accountingsync.SyncErrorValidation, "Bad line", "Correct the document"))

	h.drain(t)

	blockedKey := ""
	for _, key := range watchtowersources.AccountingSyncAttentionKeys() {
		if _, ok := h.watchtower.item(watchtowersources.AccountingSyncAttentionSourceID(h.conn, key)); ok {
			blockedKey = key
		}
	}
	require.NotEmpty(t, blockedKey, "the blocked group is projected into the watchtower")
	item, _ := h.watchtower.item(watchtowersources.AccountingSyncAttentionSourceID(h.conn, blockedKey))
	assert.Equal(t, record.ID, item.SubjectID)
}

func TestPartialWriteKeepsTheExternalRefsForTheReplay(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	record := h.readyInvoice(t)
	h.writer.partials["CreateSalesDocument"] = map[string]string{"shortPay:qb-inv": "qb-cm-7"}
	h.writer.failNext("CreateSalesDocument",
		providerFault(accountingsync.SyncErrorTransient, "Timed out after step one", ""))

	h.drain(t)

	retrying := h.records.get(record.ID)
	assert.Equal(t, "qb-cm-7", retrying.ExternalRefs["shortPay:qb-inv"])

	h.records.makeDue(record.ID)
	h.drain(t)

	docs := salesDocs(t, h)
	require.Len(t, docs, 2)
	assert.Equal(t, "qb-cm-7", docs[1].Refs["shortPay:qb-inv"], "the retry replays the step already done")
	assert.Equal(t, docs[0].RequestID, docs[1].RequestID, "the request id is stable across retries")
}

func TestPausedConnectionHoldsTheDispatcher(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	record := h.readyInvoice(t)
	paused := time.Now().Unix()
	h.updateConnection(func(conn *accountingsync.AccountingConnection) {
		conn.PausedAt = &paused
		conn.PausedReason = "Month-end close"
	})

	result := h.drain(t)

	assert.True(t, result.Held)
	assert.Zero(t, result.Claimed)
	assert.Empty(t, h.writer.methods())
	assert.Equal(t, accountingsync.SyncStatusQueued, h.records.get(record.ID).Status)
}

func TestDrainHoldsWhenTheConnectionCannotBeUsed(t *testing.T) {
	t.Parallel()

	t.Run("not syncing", func(t *testing.T) {
		t.Parallel()

		h := newHarness(t)
		h.readyInvoice(t)
		h.updateConnection(func(conn *accountingsync.AccountingConnection) {
			conn.Status = accountingsync.ConnectionStatusRevoked
		})

		result := h.drain(t)

		assert.True(t, result.Held)
		assert.Empty(t, h.writer.methods())
	})

	t.Run("session unavailable", func(t *testing.T) {
		t.Parallel()

		h := newHarness(t)
		record := h.readyInvoice(t)
		h.connService.sessionErr = errors.New("refresh token expired")

		result := h.drain(t)

		assert.True(t, result.Held)
		assert.Zero(t, result.Claimed)
		assert.Equal(t, accountingsync.SyncStatusQueued, h.records.get(record.ID).Status,
			"nothing is claimed while the connection cannot be used")
	})
}

func TestDrainCountsALostLeaseAndCarriesOn(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	customerID, _ := h.mappedCustomer("Acme")
	h.confirm(freightTarget(), "qb-item-freight")
	lost := h.enqueueInvoice(t, h.postedInvoice(customerID, invoiceSpec{number: "INV-1"}))
	kept := h.enqueueInvoice(t, h.postedInvoice(customerID, invoiceSpec{number: "INV-2"}))
	h.records.finishErr[lost.ID] = repositories.ErrAccountingSyncLeaseLost

	result := h.drain(t)

	assert.Equal(t, 2, result.Claimed)
	assert.Equal(t, 1, result.LeaseLost)
	assert.Equal(t, 1, result.Synced)
	assert.Equal(t, accountingsync.SyncStatusSynced, h.records.get(kept.ID).Status)
	assert.Empty(t, h.records.attemptsFor(lost.ID), "a lost lease records nothing")
}

func TestDrainStopsOnAnUnexpectedFinishError(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	record := h.readyInvoice(t)
	h.records.finishErr[record.ID] = errors.New("database unavailable")

	_, err := h.svc.Drain(t.Context(), &services.DrainAccountingSyncRequest{
		TenantInfo:   h.tenant,
		ConnectionID: h.conn.ID,
	})

	require.Error(t, err)
}

func TestExpiredLeaseIsClaimedAgainUnderTheSameRequestID(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	record := h.readyInvoice(t)
	h.records.mu.Lock()
	expired := time.Now().Add(-time.Minute).Unix()
	row := h.records.rows[h.records.indexOf(record.ID)]
	row.Status = accountingsync.SyncStatusInFlight
	row.LeaseExpiresAt = &expired
	row.AttemptCount = 1
	h.records.mu.Unlock()

	result := h.drain(t)

	assert.Equal(t, 1, result.Synced)
	assert.Equal(t, record.RequestID, onlySalesDoc(t, h).RequestID)
	assert.Equal(t, 2, h.records.attemptsFor(record.ID)[0].AttemptNumber)
}

func TestListDueConnectionsAsksForRecordsDueNow(t *testing.T) {
	t.Parallel()

	h := newHarness(t)

	_, err := h.svc.ListDueConnections(t.Context(), 25)
	require.NoError(t, err)

	require.Len(t, h.records.dueCalls, 1)
	assert.Equal(t, 25, h.records.dueCalls[0].Limit)
	assert.InDelta(t, time.Now().Unix(), h.records.dueCalls[0].Now, 2)
}
