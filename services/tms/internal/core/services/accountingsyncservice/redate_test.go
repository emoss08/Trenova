package accountingsyncservice

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedateSendsTheDocumentOnTheFirstOpenDay(t *testing.T) {
	t.Parallel()

	h := newHarness(t, withTimezone("America/New_York"))
	closed := aprilTenth
	h.updateConnection(func(conn *accountingsync.AccountingConnection) {
		conn.ExternalBooksClosedThrough = &closed
	})
	record := h.seedRecord(accountingsync.SyncStatusBlocked, accountingsync.SyncErrorClosedPeriod)
	loc, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)

	updated, err := h.svc.Redate(t.Context(), &services.RedateAccountingSyncRequest{
		TenantInfo: h.tenant,
		UserID:     h.userID,
		ID:         record.ID,
	})
	require.NoError(t, err)

	assert.Equal(t, accountingsync.SyncStatusQueued, updated.Status)
	assert.Equal(t, 0, updated.AttemptCount)
	require.NotNil(t, updated.RedatedTo)
	assert.Equal(t, "2026-04-11", timeutils.FormatCalendarDate(*updated.RedatedTo, loc))
	assert.False(t, h.conn.BooksClosedOn(*updated.RedatedTo))
	assert.Equal(t, h.userID, updated.RedatedByID)
	stored := h.records.get(record.ID)
	assert.Equal(t, accountingsync.SyncStatusQueued, stored.Status)
	assert.Equal(t, updated.RedatedTo, stored.RedatedTo)
	assert.Contains(t, h.audit.lastComment(), "first open day")
}

func TestRedateFollowsTrenovasClosedPeriodPolicy(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	closed := aprilTenth
	h.updateConnection(func(conn *accountingsync.AccountingConnection) {
		conn.ExternalBooksClosedThrough = &closed
	})
	h.controls.set(func(control *tenant.AccountingControl) {
		control.ClosedPeriodPostingPolicy = tenant.ClosedPeriodPostingPolicyRequireReopen
	})
	record := h.seedRecord(accountingsync.SyncStatusBlocked, accountingsync.SyncErrorClosedPeriod)

	_, err := h.svc.Redate(t.Context(), &services.RedateAccountingSyncRequest{
		TenantInfo: h.tenant,
		UserID:     h.userID,
		ID:         record.ID,
	})

	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
	assert.Contains(t, err.Error(), "closed-period policy")
	assert.Equal(t, accountingsync.SyncStatusBlocked, h.records.get(record.ID).Status)
	assert.Nil(t, h.records.get(record.ID).RedatedTo)
	assert.Equal(t, 0, h.audit.count())
}

func TestRedateRefusesARecordNotHeldByAClosedPeriod(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	closed := aprilTenth
	h.updateConnection(func(conn *accountingsync.AccountingConnection) {
		conn.ExternalBooksClosedThrough = &closed
	})
	for _, record := range []*accountingsync.AccountingSyncRecord{
		h.seedRecord(accountingsync.SyncStatusBlocked, accountingsync.SyncErrorMapping),
		h.seedRecord(accountingsync.SyncStatusSynced, ""),
	} {
		_, err := h.svc.Redate(t.Context(), &services.RedateAccountingSyncRequest{
			TenantInfo: h.tenant,
			UserID:     h.userID,
			ID:         record.ID,
		})

		assert.True(t, errortypes.IsBusinessError(err), string(record.Status))
		assert.Nil(t, h.records.get(record.ID).RedatedTo)
	}
}

func TestRedateNeedsClosedBooksToMoveAwayFrom(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	record := h.seedRecord(accountingsync.SyncStatusBlocked, accountingsync.SyncErrorClosedPeriod)

	_, err := h.svc.Redate(t.Context(), &services.RedateAccountingSyncRequest{
		TenantInfo: h.tenant,
		UserID:     h.userID,
		ID:         record.ID,
	})

	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
	assert.Contains(t, err.Error(), "retry it instead")
}

func TestARedatedDocumentIsSentOnItsNewDateWithItsOwnDateInTheNote(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	closed := aprilTenth
	h.updateConnection(func(conn *accountingsync.AccountingConnection) {
		conn.ExternalBooksClosedThrough = &closed
	})
	customerID, _ := h.mappedCustomer("Acme")
	h.confirm(freightTarget(), "qb-item-freight")
	record := h.enqueueInvoice(t, h.postedInvoice(customerID, invoiceSpec{date: aprilTenth}))

	h.drain(t)
	held := h.records.get(record.ID)
	require.Equal(t, accountingsync.SyncStatusBlocked, held.Status)
	assert.Contains(t, held.Resolution, "send it dated on the first open day")

	_, err := h.svc.Redate(t.Context(), &services.RedateAccountingSyncRequest{
		TenantInfo: h.tenant,
		UserID:     h.userID,
		ID:         record.ID,
	})
	require.NoError(t, err)
	h.drain(t)

	assert.Equal(t, accountingsync.SyncStatusSynced, h.records.get(record.ID).Status)
	doc := onlySalesDoc(t, h)
	assert.Equal(t, "2026-04-11", doc.TxnDate)
	assert.Contains(t, doc.PrivateNote, "Dated 2026-04-10 in Trenova; sent on the first open day")
}

func TestAClosedPeriodResolutionOffersTheFirstOpenDayOnlyUnderThatPolicy(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	closed := aprilTenth
	h.updateConnection(func(conn *accountingsync.AccountingConnection) {
		conn.ExternalBooksClosedThrough = &closed
	})
	h.controls.set(func(control *tenant.AccountingControl) {
		control.ClosedPeriodPostingPolicy = tenant.ClosedPeriodPostingPolicyRequireReopen
	})
	customerID, _ := h.mappedCustomer("Acme")
	h.confirm(freightTarget(), "qb-item-freight")
	record := h.enqueueInvoice(t, h.postedInvoice(customerID, invoiceSpec{date: aprilTenth}))

	h.drain(t)

	held := h.records.get(record.ID)
	assert.Equal(t, accountingsync.SyncStatusBlocked, held.Status)
	assert.Contains(t, held.Resolution, "Reopen the period in QuickBooks Online")
	assert.NotContains(t, held.Resolution, "first open day")
}
