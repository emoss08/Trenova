package accountingsyncservice

import (
	"errors"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (h *harness) atStartDateStep() {
	h.updateConnection(func(conn *accountingsync.AccountingConnection) {
		conn.SetupStep = accountingsync.SetupStepStartDate
		conn.SyncStartDate = nil
		conn.SyncEnabledAt = nil
		conn.AutoSync = false
	})
}

func (h *harness) enableSync(
	t *testing.T,
	startDate int64,
	backfill bool,
) (*accountingsync.AccountingConnection, error) {
	t.Helper()
	return h.svc.EnableSync(t.Context(), &services.EnableAccountingSyncRequest{
		TenantInfo:      h.tenant,
		UserID:          h.userID,
		IntegrationType: integration.TypeQuickBooksOnline,
		StartDate:       startDate,
		AutoSync:        true,
		Backfill:        backfill,
	})
}

func requireValidationField(t *testing.T, err error, field string) {
	t.Helper()
	var validation *errortypes.Error
	require.ErrorAs(t, err, &validation)
	assert.Equal(t, field, validation.Field)
}

func (h *harness) seedRecord(
	status accountingsync.SyncStatus,
	category accountingsync.SyncErrorCategory,
) *accountingsync.AccountingSyncRecord {
	record := NewRecordFor(h.conn, &services.AccountingSyncEnqueueRequest{
		TenantInfo:   h.tenant,
		ObjectType:   accountingsync.SyncObjectInvoice,
		ObjectID:     pulid.MustNew("inv_"),
		ObjectNumber: "INV-" + string(status),
		Operation:    accountingsync.SyncOperationCreate,
		Revision:     1,
		SourceEvent:  accountingsync.SyncSourceInvoicePosted,
		DocumentDate: aprilTenth,
	}, time.Now().Unix())
	record.Status = status
	record.ErrorCategory = category
	record.AttemptCount = 3
	if status == accountingsync.SyncStatusSynced {
		record.ExternalID = "qb-" + record.ObjectID.String()
	}
	h.records.put(record)
	return record
}

func TestEnableSyncStampsTheStartAndCompletesSetup(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.atStartDateStep()

	conn, err := h.enableSync(t, syncStartDate, false)
	require.NoError(t, err)

	assert.Equal(t, accountingsync.SetupStepComplete, conn.SetupStep)
	require.NotNil(t, conn.SyncStartDate)
	assert.Equal(t, syncStartDate, *conn.SyncStartDate)
	require.NotNil(t, conn.SyncEnabledAt)
	assert.InDelta(t, time.Now().Unix(), *conn.SyncEnabledAt, 2)
	assert.True(t, conn.AutoSync)
	stored := h.connections.get(h.conn.ID)
	assert.True(t, stored.IsSyncing())
	require.Equal(t, 1, h.audit.count())
	assert.Equal(t, permission.ResourceAccountingIntegration, h.audit.entries[0].Resource)
	assert.Empty(t, h.dispatcher.started(), "no backfill unless asked")
}

func TestEnableSyncKeepsTheOriginalEnabledTime(t *testing.T) {
	t.Parallel()

	h := newHarness(t)

	conn, err := h.enableSync(t, syncStartDate+86400, false)
	require.NoError(t, err)

	assert.Equal(t, syncEnabledAt, *conn.SyncEnabledAt,
		"moving the start date does not move when sync began")
	assert.Equal(t, syncStartDate+86400, *conn.SyncStartDate)
}

func TestEnableSyncRejectsBadInput(t *testing.T) {
	t.Parallel()

	t.Run("missing start date", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		h.atStartDateStep()

		_, err := h.enableSync(t, 0, false)

		requireValidationField(t, err, "startDate")
		assert.Equal(t, accountingsync.SetupStepStartDate, h.connections.get(h.conn.ID).SetupStep)
	})

	t.Run("start date in the future", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		h.atStartDateStep()

		_, err := h.enableSync(t, time.Now().Add(48*time.Hour).Unix(), false)

		requireValidationField(t, err, "startDate")
		assert.Nil(t, h.connections.get(h.conn.ID).SyncStartDate)
	})

	t.Run("mappings not confirmed", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		h.atStartDateStep()
		h.updateConnection(func(conn *accountingsync.AccountingConnection) {
			conn.SetupStep = accountingsync.SetupStepMappings
		})

		_, err := h.enableSync(t, syncStartDate, false)

		assert.True(t, errortypes.IsBusinessError(err))
	})

	t.Run("disconnected", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)
		h.atStartDateStep()
		h.updateConnection(func(conn *accountingsync.AccountingConnection) {
			conn.Status = accountingsync.ConnectionStatusDisconnected
		})

		_, err := h.enableSync(t, syncStartDate, false)

		assert.True(t, errortypes.IsBusinessError(err))
	})

	t.Run("not connected at all", func(t *testing.T) {
		t.Parallel()
		h := newHarness(t)

		_, err := h.svc.EnableSync(t.Context(), &services.EnableAccountingSyncRequest{
			TenantInfo:      pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
			IntegrationType: integration.TypeQuickBooksOnline,
			StartDate:       syncStartDate,
		})

		assert.True(t, errortypes.IsNotFoundError(err))
	})
}

func TestEnableSyncWithBackfillStartsOne(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.atStartDateStep()

	conn, err := h.enableSync(t, syncStartDate, true)
	require.NoError(t, err)

	started := h.dispatcher.started()
	require.Len(t, started, 1)
	assert.Equal(t, conn.ID, started[0].ConnectionID)
	assert.Equal(t, syncStartDate, started[0].RangeStart)
	assert.Equal(t, *conn.SyncEnabledAt, started[0].RangeEnd,
		"a backfill covers documents dated from the start date up to when sync began")
	assert.Equal(t, accountingsync.BackfillObjectTypes(), started[0].ObjectTypes)
	assert.Equal(t, accountingsync.BackfillStatusQueued, started[0].Status)
}

func TestEnableSyncToleratesABackfillAlreadyRunning(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	_, err := h.svc.RequestBackfill(t.Context(), &services.RequestAccountingBackfillRequest{
		TenantInfo:      h.tenant,
		UserID:          h.userID,
		IntegrationType: integration.TypeQuickBooksOnline,
	})
	require.NoError(t, err)

	_, err = h.enableSync(t, syncStartDate, true)

	require.NoError(t, err)
	assert.Len(t, h.dispatcher.started(), 1)
}

func TestPauseRequiresAReasonAndHoldsTheDispatcher(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	req := &services.PauseAccountingSyncRequest{
		TenantInfo:      h.tenant,
		UserID:          h.userID,
		IntegrationType: integration.TypeQuickBooksOnline,
		Reason:          "   ",
	}

	_, err := h.svc.Pause(t.Context(), req)
	requireValidationField(t, err, "reason")
	assert.False(t, h.connections.get(h.conn.ID).IsPaused())

	req.Reason = "Month-end\nclose"
	conn, err := h.svc.Pause(t.Context(), req)
	require.NoError(t, err)

	assert.True(t, conn.IsPaused())
	assert.Equal(t, h.userID, conn.PausedByID)
	assert.Equal(t, "Month-end close", conn.PausedReason)
	assert.False(t, h.connections.get(h.conn.ID).CanDispatch())
	assert.Equal(t, 1, h.audit.count())

	req.Reason = "Another reason"
	again, err := h.svc.Pause(t.Context(), req)
	require.NoError(t, err)
	assert.Equal(t, "Month-end close", again.PausedReason, "pausing a paused connection changes nothing")
	assert.Equal(t, 1, h.audit.count())
}

func TestPauseKeepsEnqueueing(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	_, err := h.svc.Pause(t.Context(), &services.PauseAccountingSyncRequest{
		TenantInfo:      h.tenant,
		UserID:          h.userID,
		IntegrationType: integration.TypeQuickBooksOnline,
		Reason:          "Audit",
	})
	require.NoError(t, err)

	record := h.enqueueInvoice(t, h.postedInvoice(pulid.MustNew("cus_"), invoiceSpec{}))

	assert.Equal(t, accountingsync.SyncStatusQueued, record.Status)
}

func TestResumeClearsThePauseAndWakesTheDispatcher(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	paused := time.Now().Add(-time.Hour).Unix()
	h.updateConnection(func(conn *accountingsync.AccountingConnection) {
		conn.PausedAt = &paused
		conn.PausedByID = h.userID
		conn.PausedReason = "Audit"
	})

	conn, err := h.svc.Resume(t.Context(), &services.PauseAccountingSyncRequest{
		TenantInfo:      h.tenant,
		UserID:          h.userID,
		IntegrationType: integration.TypeQuickBooksOnline,
	})
	require.NoError(t, err)

	assert.False(t, conn.IsPaused())
	assert.True(t, conn.PausedByID.IsNil())
	assert.Empty(t, conn.PausedReason)
	assert.Equal(t, []pulid.ID{h.conn.ID}, h.dispatcher.kicked())
	assert.Equal(t, 1, h.audit.count())
}

func TestPauseAndResumeNeedASyncingConnection(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.atStartDateStep()
	req := &services.PauseAccountingSyncRequest{
		TenantInfo:      h.tenant,
		UserID:          h.userID,
		IntegrationType: integration.TypeQuickBooksOnline,
		Reason:          "Audit",
	}

	_, err := h.svc.Pause(t.Context(), req)
	assert.True(t, errortypes.IsBusinessError(err))
	_, err = h.svc.Resume(t.Context(), req)
	assert.True(t, errortypes.IsBusinessError(err))
}

func (h *harness) retry(
	t *testing.T,
	ids []pulid.ID,
	categories ...accountingsync.SyncErrorCategory,
) (int64, error) {
	t.Helper()
	return h.svc.Retry(t.Context(), &services.RetryAccountingSyncRequest{
		TenantInfo:      h.tenant,
		UserID:          h.userID,
		IntegrationType: integration.TypeQuickBooksOnline,
		IDs:             ids,
		ErrorCategories: categories,
	})
}

func TestRetryByIDsRequeuesOnlyThoseRecords(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	chosen := h.seedRecord(accountingsync.SyncStatusBlocked, accountingsync.SyncErrorValidation)
	other := h.seedRecord(accountingsync.SyncStatusBlocked, accountingsync.SyncErrorValidation)

	requeued, err := h.retry(t, []pulid.ID{chosen.ID})
	require.NoError(t, err)

	assert.Equal(t, int64(1), requeued)
	retried := h.records.get(chosen.ID)
	assert.Equal(t, accountingsync.SyncStatusQueued, retried.Status)
	assert.Zero(t, retried.AttemptCount, "a person's retry starts the attempts over")
	assert.Equal(t, accountingsync.SyncStatusBlocked, h.records.get(other.ID).Status)
	assert.Equal(t, []pulid.ID{h.conn.ID}, h.dispatcher.kicked())
	assert.Equal(t, 1, h.audit.count())
}

func TestRetryByCategoryRequeuesOnlyThatCause(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	mapping := h.seedRecord(accountingsync.SyncStatusBlocked, accountingsync.SyncErrorMapping)
	validation := h.seedRecord(accountingsync.SyncStatusBlocked, accountingsync.SyncErrorValidation)
	dead := h.seedRecord(accountingsync.SyncStatusDeadLettered, accountingsync.SyncErrorTransient)

	requeued, err := h.retry(t, nil, accountingsync.SyncErrorMapping, accountingsync.SyncErrorTransient)
	require.NoError(t, err)

	assert.Equal(t, int64(2), requeued)
	assert.Equal(t, accountingsync.SyncStatusQueued, h.records.get(mapping.ID).Status)
	assert.Equal(t, accountingsync.SyncStatusQueued, h.records.get(dead.ID).Status)
	assert.Equal(t, accountingsync.SyncStatusBlocked, h.records.get(validation.ID).Status)
}

func TestRetryLeavesFinishedRecordsAlone(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	synced := h.seedRecord(accountingsync.SyncStatusSynced, "")
	skipped := h.seedRecord(accountingsync.SyncStatusSkipped, "")

	requeued, err := h.retry(t, []pulid.ID{synced.ID, skipped.ID})
	require.NoError(t, err)

	assert.Zero(t, requeued)
	assert.Equal(t, accountingsync.SyncStatusSynced, h.records.get(synced.ID).Status)
	assert.Empty(t, h.dispatcher.kicked())
	assert.Zero(t, h.audit.count())
}

func TestRetryRejectsBadFilters(t *testing.T) {
	t.Parallel()

	h := newHarness(t)

	_, err := h.retry(t, nil, "Gremlins")
	requireValidationField(t, err, "errorCategories")

	ids := make([]pulid.ID, maxRetryIDs+1)
	for idx := range ids {
		ids[idx] = pulid.MustNew("acctsr_")
	}
	_, err = h.retry(t, ids)
	requireValidationField(t, err, "ids")
}

func TestReleaseQueuesHeldRecords(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	first := h.seedRecord(accountingsync.SyncStatusAwaitingApproval, "")
	second := h.seedRecord(accountingsync.SyncStatusAwaitingApproval, "")
	blocked := h.seedRecord(accountingsync.SyncStatusBlocked, accountingsync.SyncErrorMapping)

	released, err := h.svc.Release(t.Context(), &services.ReleaseAccountingSyncRequest{
		TenantInfo:      h.tenant,
		UserID:          h.userID,
		IntegrationType: integration.TypeQuickBooksOnline,
	})
	require.NoError(t, err)

	assert.Equal(t, int64(2), released)
	for _, id := range []pulid.ID{first.ID, second.ID} {
		record := h.records.get(id)
		assert.Equal(t, accountingsync.SyncStatusQueued, record.Status)
		assert.Equal(t, h.userID, record.ReleasedByID)
	}
	assert.Equal(t, accountingsync.SyncStatusBlocked, h.records.get(blocked.ID).Status)
	assert.Equal(t, []pulid.ID{h.conn.ID}, h.dispatcher.kicked())
}

func TestSkipNeedsAReasonAndMarksTheRecordSkipped(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	record := h.seedRecord(accountingsync.SyncStatusBlocked, accountingsync.SyncErrorClosedPeriod)
	req := &services.SkipAccountingSyncRequest{TenantInfo: h.tenant, UserID: h.userID, ID: record.ID}

	_, err := h.svc.Skip(t.Context(), req)
	requireValidationField(t, err, "reason")
	assert.Equal(t, accountingsync.SyncStatusBlocked, h.records.get(record.ID).Status)

	req.Reason = "Entered by hand in QuickBooks"
	skipped, err := h.svc.Skip(t.Context(), req)
	require.NoError(t, err)

	assert.Equal(t, accountingsync.SyncStatusSkipped, skipped.Status)
	assert.Equal(t, h.userID, skipped.SkippedByID)
	assert.Equal(t, "Entered by hand in QuickBooks", skipped.SkippedReason)
	assert.Equal(t, accountingsync.SyncStatusSkipped, h.records.get(record.ID).Status)
	assert.Equal(t, 1, h.audit.count())
}

func TestSkipRefusesASentRecord(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	for _, status := range []accountingsync.SyncStatus{
		accountingsync.SyncStatusSynced,
		accountingsync.SyncStatusInFlight,
	} {
		record := h.seedRecord(status, "")

		_, err := h.svc.Skip(t.Context(), &services.SkipAccountingSyncRequest{
			TenantInfo: h.tenant,
			UserID:     h.userID,
			ID:         record.ID,
			Reason:     "Not needed",
		})

		assert.True(t, errortypes.IsBusinessError(err), string(status))
		assert.Equal(t, status, h.records.get(record.ID).Status)
	}
}

func TestSkipIsScopedToTheTenant(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	record := h.seedRecord(accountingsync.SyncStatusBlocked, accountingsync.SyncErrorMapping)

	_, err := h.svc.Skip(t.Context(), &services.SkipAccountingSyncRequest{
		TenantInfo: pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: h.tenant.BuID},
		UserID:     h.userID,
		ID:         record.ID,
		Reason:     "Not ours",
	})

	assert.True(t, errortypes.IsNotFoundError(err))
	assert.Equal(t, accountingsync.SyncStatusBlocked, h.records.get(record.ID).Status)
}

func (h *harness) requestBackfill(
	t *testing.T,
	mutate func(*services.RequestAccountingBackfillRequest),
) (*accountingsync.AccountingBackfill, error) {
	t.Helper()
	req := &services.RequestAccountingBackfillRequest{
		TenantInfo:      h.tenant,
		UserID:          h.userID,
		IntegrationType: integration.TypeQuickBooksOnline,
	}
	if mutate != nil {
		mutate(req)
	}
	return h.svc.RequestBackfill(t.Context(), req)
}

func TestRequestBackfillAllowsOneActivePerConnection(t *testing.T) {
	t.Parallel()

	h := newHarness(t)

	first, err := h.requestBackfill(t, nil)
	require.NoError(t, err)
	assert.Equal(t, syncStartDate, first.RangeStart)
	assert.Equal(t, syncEnabledAt, first.RangeEnd)
	assert.Equal(t, h.userID, first.RequestedByID)
	assert.Len(t, h.dispatcher.started(), 1)

	_, err = h.requestBackfill(t, nil)

	assert.True(t, errortypes.IsBusinessError(err))
	assert.ErrorIs(t, err, repositories.ErrAccountingBackfillActive)
	assert.Len(t, h.dispatcher.started(), 1)
}

func TestRequestBackfillValidatesTheRangeAndTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		field  string
		mutate func(*services.RequestAccountingBackfillRequest)
	}{
		{name: "before the start date", field: "rangeStart", mutate: func(r *services.RequestAccountingBackfillRequest) {
			r.RangeStart = new(syncStartDate - 1)
		}},
		{name: "after sync began", field: "rangeEnd", mutate: func(r *services.RequestAccountingBackfillRequest) {
			r.RangeEnd = new(syncEnabledAt + 1)
		}},
		{name: "end before start", field: "rangeEnd", mutate: func(r *services.RequestAccountingBackfillRequest) {
			r.RangeStart = new(aprilTenth)
			r.RangeEnd = new(syncStartDate + 1)
		}},
		{name: "customers are not backfilled", field: "objectTypes", mutate: func(r *services.RequestAccountingBackfillRequest) {
			r.ObjectTypes = []accountingsync.SyncObjectType{accountingsync.SyncObjectCustomer}
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			h := newHarness(t)

			_, err := h.requestBackfill(t, tc.mutate)

			requireValidationField(t, err, tc.field)
			assert.Empty(t, h.dispatcher.started())
		})
	}
}

func TestRequestBackfillKeepsTheCanonicalTypeOrder(t *testing.T) {
	t.Parallel()

	h := newHarness(t)

	backfill, err := h.requestBackfill(t, func(r *services.RequestAccountingBackfillRequest) {
		r.ObjectTypes = []accountingsync.SyncObjectType{
			accountingsync.SyncObjectCustomerPayment,
			accountingsync.SyncObjectInvoice,
		}
	})
	require.NoError(t, err)

	assert.Equal(t, []accountingsync.SyncObjectType{
		accountingsync.SyncObjectInvoice,
		accountingsync.SyncObjectCustomerPayment,
	}, backfill.ObjectTypes)
	assert.Equal(t, accountingsync.SyncObjectInvoice, backfill.Cursor.ObjectType)
}

func TestRequestBackfillNeedsBackgroundWork(t *testing.T) {
	t.Parallel()

	h := newHarness(t, withoutDispatcher())

	_, err := h.requestBackfill(t, nil)

	assert.True(t, errortypes.IsBusinessError(err))
}

func (h *harness) changeBackfill(
	t *testing.T,
	id pulid.ID,
	action services.AccountingBackfillAction,
) (*accountingsync.AccountingBackfill, error) {
	t.Helper()
	return h.svc.ChangeBackfill(t.Context(), &services.ChangeAccountingBackfillRequest{
		TenantInfo: h.tenant,
		UserID:     h.userID,
		ID:         id,
		Action:     action,
	})
}

func TestChangeBackfillPausesResumesAndCancels(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	backfill, err := h.requestBackfill(t, nil)
	require.NoError(t, err)

	paused, err := h.changeBackfill(t, backfill.ID, services.AccountingBackfillPause)
	require.NoError(t, err)
	assert.Equal(t, accountingsync.BackfillStatusPaused, paused.Status)

	_, err = h.changeBackfill(t, backfill.ID, services.AccountingBackfillPause)
	assert.True(t, errortypes.IsBusinessError(err), "a paused backfill cannot be paused again")

	resumed, err := h.changeBackfill(t, backfill.ID, services.AccountingBackfillResume)
	require.NoError(t, err)
	assert.Equal(t, accountingsync.BackfillStatusQueued, resumed.Status)
	assert.Len(t, h.dispatcher.started(), 2, "resuming starts the workflow again")

	cancelled, err := h.changeBackfill(t, backfill.ID, services.AccountingBackfillCancel)
	require.NoError(t, err)
	assert.Equal(t, accountingsync.BackfillStatusCancelled, cancelled.Status)
	assert.NotNil(t, cancelled.CompletedAt)

	_, err = h.changeBackfill(t, backfill.ID, services.AccountingBackfillResume)
	assert.True(t, errortypes.IsBusinessError(err))
	assert.Equal(t, 4, h.audit.count())

	_, err = h.changeBackfill(t, backfill.ID, "Explode")
	requireValidationField(t, err, "action")
}

func TestSummaryWithoutAConnectionIsEmpty(t *testing.T) {
	t.Parallel()

	h := newHarness(t)

	summary, err := h.svc.Summary(
		t.Context(),
		pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
		integration.TypeQuickBooksOnline,
	)
	require.NoError(t, err)

	assert.Nil(t, summary.Connection)
	assert.Equal(t, "QuickBooks Online", summary.ProviderName)
	assert.Empty(t, summary.Counts)
	assert.NotNil(t, summary.Counts)
	assert.Nil(t, summary.ActiveBackfill)
}

func TestSummaryRejectsANonAccountingIntegration(t *testing.T) {
	t.Parallel()

	h := newHarness(t)

	_, err := h.svc.Summary(t.Context(), h.tenant, integration.Type("Samsara"))

	requireValidationField(t, err, "integrationType")
}

func TestSummaryCountsByStatusWithAttentionAndBackfill(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.seedRecord(accountingsync.SyncStatusBlocked, accountingsync.SyncErrorMapping)
	h.seedRecord(accountingsync.SyncStatusBlocked, accountingsync.SyncErrorMapping)
	h.seedRecord(accountingsync.SyncStatusSynced, "")
	h.records.attention = []repositories.AccountingSyncAttentionGroup{{
		Status: accountingsync.SyncStatusBlocked, ErrorCategory: accountingsync.SyncErrorMapping, Count: 2,
	}}
	backfill, err := h.requestBackfill(t, nil)
	require.NoError(t, err)

	summary, err := h.svc.Summary(t.Context(), h.tenant, integration.TypeQuickBooksOnline)
	require.NoError(t, err)

	assert.Equal(t, h.conn.ID, summary.Connection.ID)
	assert.ElementsMatch(t, []repositories.AccountingSyncStatusCount{
		{Status: accountingsync.SyncStatusSynced, Count: 1},
		{Status: accountingsync.SyncStatusBlocked, Count: 2},
	}, summary.Counts)
	assert.Len(t, summary.Attention, 1)
	require.NotNil(t, summary.ActiveBackfill)
	assert.Equal(t, backfill.ID, summary.ActiveBackfill.ID)
}

func TestListRecordsScopesToTheConnectionAndTheCallersTenant(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.seedRecord(accountingsync.SyncStatusBlocked, accountingsync.SyncErrorMapping)
	objectID := pulid.MustNew("inv_")

	result, err := h.svc.ListRecords(t.Context(), &services.ListAccountingSyncRecordsRequest{
		TenantInfo: h.tenant,
		Filter: &pagination.QueryOptions{
			TenantInfo: pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
		},
		IntegrationType: integration.TypeQuickBooksOnline,
		Statuses:        []accountingsync.SyncStatus{accountingsync.SyncStatusBlocked},
		ErrorCategories: []accountingsync.SyncErrorCategory{accountingsync.SyncErrorMapping},
		ObjectTypes:     []accountingsync.SyncObjectType{accountingsync.SyncObjectInvoice},
		ObjectID:        objectID,
		Search:          "INV",
	})
	require.NoError(t, err)

	assert.Len(t, result.Items, 1)
	require.Len(t, h.records.connectionCalls, 1)
	call := h.records.connectionCalls[0]
	assert.Equal(t, h.conn.ID, call.ConnectionID)
	assert.Equal(t, h.tenant, call.Filter.TenantInfo, "the caller's tenant always wins over the filter's")
	assert.Equal(t, []accountingsync.SyncStatus{accountingsync.SyncStatusBlocked}, call.Statuses)
	assert.Equal(t, objectID, call.ObjectID)
	assert.Equal(t, "INV", call.Search)

	_, err = h.svc.ListRecords(t.Context(), &services.ListAccountingSyncRecordsRequest{
		TenantInfo:      h.tenant,
		IntegrationType: integration.TypeQuickBooksOnline,
	})
	require.NoError(t, err)
}

func TestListAttemptsIsScopedToTheTenant(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	record := h.readyInvoice(t)
	h.drain(t)

	attempts, err := h.svc.ListAttempts(t.Context(), h.tenant, record.ID)
	require.NoError(t, err)
	assert.Len(t, attempts, 1)

	_, err = h.svc.ListAttempts(
		t.Context(),
		pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: h.tenant.BuID},
		record.ID,
	)
	assert.True(t, errortypes.IsNotFoundError(err))
}

func TestListBackfillsForTheConnection(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	backfill, err := h.requestBackfill(t, nil)
	require.NoError(t, err)

	backfills, err := h.svc.ListBackfills(t.Context(), h.tenant, integration.TypeQuickBooksOnline)
	require.NoError(t, err)

	require.Len(t, backfills, 1)
	assert.Equal(t, backfill.ID, backfills[0].ID)
}

func TestObjectStatesPicksTheMostRelevantRecord(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	idle := h.addConnection(nil)
	blockedObject := pulid.MustNew("inv_")
	syncedObject := pulid.MustNew("inv_")
	supersededObject := pulid.MustNew("cpay_")
	idleObject := pulid.MustNew("inv_")
	put := func(conn *accountingsync.AccountingConnection, objectID pulid.ID, operation accountingsync.SyncOperation,
		revision int64, status accountingsync.SyncStatus,
	) *accountingsync.AccountingSyncRecord {
		record := NewRecordFor(conn, &services.AccountingSyncEnqueueRequest{
			TenantInfo:  h.tenant,
			ObjectType:  accountingsync.SyncObjectInvoice,
			ObjectID:    objectID,
			Operation:   operation,
			Revision:    revision,
			SourceEvent: accountingsync.SyncSourceInvoicePosted,
		}, time.Now().Unix()+revision)
		record.Status = status
		h.records.put(record)
		return record
	}
	put(h.conn, blockedObject, accountingsync.SyncOperationCreate, 1, accountingsync.SyncStatusSynced)
	blocked := put(h.conn, blockedObject, accountingsync.SyncOperationUpdate, 2, accountingsync.SyncStatusBlocked)
	put(h.conn, blockedObject, accountingsync.SyncOperationUpdate, 3, accountingsync.SyncStatusSuperseded)
	synced := put(h.conn, syncedObject, accountingsync.SyncOperationCreate, 1, accountingsync.SyncStatusSynced)
	put(h.conn, syncedObject, accountingsync.SyncOperationUpdate, 2, accountingsync.SyncStatusSkipped)
	put(h.conn, supersededObject, accountingsync.SyncOperationUpdate, 2, accountingsync.SyncStatusSuperseded)
	put(idle, idleObject, accountingsync.SyncOperationCreate, 1, accountingsync.SyncStatusBlocked)

	states, err := h.svc.ObjectStates(t.Context(), h.tenant,
		[]pulid.ID{blockedObject, syncedObject, supersededObject, idleObject})
	require.NoError(t, err)

	require.Len(t, states, 2)
	assert.Equal(t, blocked.ID, states[blockedObject].Record.ID, "what needs attention wins")
	assert.Equal(t, "QuickBooks Online", states[blockedObject].ProviderName)
	assert.Equal(t, synced.ID, states[syncedObject].Record.ID, "synced outranks skipped")
	assert.NotContains(t, states, supersededObject, "a superseded record is never the state")
	assert.NotContains(t, states, idleObject, "a connection that is not syncing shows nothing")
}

func TestObjectStatesIsBounded(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	ids := make([]pulid.ID, maxStateObjects+25)
	for idx := range ids {
		ids[idx] = pulid.MustNew("inv_")
	}

	_, err := h.svc.ObjectStates(t.Context(), h.tenant, ids)
	require.NoError(t, err)

	require.Len(t, h.records.byObjectsCalls, 1)
	assert.Len(t, h.records.byObjectsCalls[0].ObjectIDs, maxStateObjects)
	assert.Equal(t, h.tenant, h.records.byObjectsCalls[0].TenantInfo)
}

func TestObjectStatesWithNothingSyncingReadsNothing(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.atStartDateStep()

	states, err := h.svc.ObjectStates(t.Context(), h.tenant, []pulid.ID{pulid.MustNew("inv_")})
	require.NoError(t, err)
	assert.Empty(t, states)
	assert.Empty(t, h.records.byObjectsCalls)

	states, err = h.svc.ObjectStates(t.Context(), h.tenant, nil)
	require.NoError(t, err)
	assert.Empty(t, states)
}

func TestKickFailureDoesNotFailTheOperation(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.seedRecord(accountingsync.SyncStatusBlocked, accountingsync.SyncErrorMapping)
	h.dispatcher.kickErr = errors.New("temporal unavailable")

	requeued, err := h.retry(t, nil)

	require.NoError(t, err)
	assert.Equal(t, int64(1), requeued)
}
