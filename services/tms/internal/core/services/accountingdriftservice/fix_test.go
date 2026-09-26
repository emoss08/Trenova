package accountingdriftservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/customerpayment"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (h *harness) person() *services.RequestActor {
	return &services.RequestActor{
		PrincipalType:  services.PrincipalTypeUser,
		PrincipalID:    pulid.MustNew("usr_"),
		UserID:         pulid.MustNew("usr_"),
		OrganizationID: h.tenant.OrgID,
		BusinessUnitID: h.tenant.BuID,
	}
}

func (h *harness) agent() *services.RequestActor {
	actor := h.person()
	actor.PrincipalType = services.PrincipalTypeAgent
	actor.PrincipalID = pulid.MustNew("agt_")
	return actor
}

func (h *harness) invoiceFinding(
	t *testing.T,
	trenovaMinor int64,
	provider string,
) (*accountingsync.AccountingDriftFinding, *invoice.Invoice) {
	t.Helper()
	doc := h.synced(accountingsync.SyncObjectInvoice, "101", trenovaMinor)
	inv := &invoice.Invoice{
		ID:               doc.record.ObjectID,
		Number:           "DOC-101",
		CustomerID:       doc.state.PartyID,
		Status:           invoice.StatusPosted,
		TotalAmountMinor: trenovaMinor,
		BalanceDueMinor:  trenovaMinor,
	}
	h.invRepo.rows[inv.ID] = inv
	if provider == "" {
		h.reader.omit["101"] = false
	} else {
		h.provider(accountingsync.SyncObjectInvoice, "101", provider)
	}
	h.reconcile(t)
	open := h.findings.open()
	require.Len(t, open, 1)
	return open[0], inv
}

func (h *harness) resolve(
	t *testing.T,
	finding *accountingsync.AccountingDriftFinding,
	direction accountingsync.DriftDirection,
	actor *services.RequestActor,
) (*accountingsync.AccountingDriftFinding, error) {
	t.Helper()
	return h.svc.Resolve(t.Context(), &services.ResolveAccountingDriftRequest{
		TenantInfo: h.tenant,
		ID:         finding.ID,
		Direction:  direction,
	}, actor)
}

func TestPushSendsTrenovasTotalAsAnUpdateAtTheNextRevision(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	finding, inv := h.invoiceFinding(t, 125_000, "1200.00")
	actor := h.person()

	updated, err := h.resolve(t, finding, accountingsync.DriftPushTrenovaValue, actor)
	require.NoError(t, err)

	records := h.records.forObject(inv.ID)
	require.Len(t, records, 2)
	pushed := h.records.byID(updated.FixObjectID)
	require.NotNil(t, pushed)
	assert.Equal(t, accountingsync.SyncOperationUpdate, pushed.Operation)
	assert.Equal(t, int64(2), pushed.Revision)
	assert.Equal(t, accountingsync.SyncSourceDriftResolved, pushed.SourceEvent)
	assert.Equal(t, accountingsync.SyncStatusQueued, pushed.Status)
	assert.Equal(t, "DOC-101", pushed.ObjectNumber)

	assert.True(t, updated.IsOpen(), "it resolves once a later compare matches")
	assert.True(t, updated.Pushed())
	assert.Equal(t, accountingsync.DriftFixSyncRecord, updated.FixObjectType)
	assert.Equal(t, actor.UserID, updated.ResolvedByID)
	assert.Equal(t, 1, h.kicks.kicks)
	require.Len(t, h.audit.entries, 1)
}

func TestPushQueuesAtOnceWhenAPersonAskedEvenWithoutAutoSync(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.conn.AutoSync = false
	finding, _ := h.invoiceFinding(t, 125_000, "1200.00")
	actor := h.person()

	updated, err := h.resolve(t, finding, accountingsync.DriftPushTrenovaValue, actor)
	require.NoError(t, err)

	pushed := h.records.byID(updated.FixObjectID)
	assert.Equal(t, accountingsync.SyncStatusQueued, pushed.Status)
	assert.Equal(t, actor.UserID, pushed.ReleasedByID)
}

func TestPushRecreatesADocumentTheProviderDeleted(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	finding, _ := h.invoiceFinding(t, 125_000, "")
	require.Equal(t, accountingsync.DriftDeletedInProvider, finding.Kind)

	updated, err := h.resolve(t, finding, accountingsync.DriftPushTrenovaValue, h.person())
	require.NoError(t, err)

	pushed := h.records.byID(updated.FixObjectID)
	assert.Equal(t, accountingsync.SyncOperationRecreate, pushed.Operation)
}

func TestPushVoidsInTheProviderWhatTrenovaVoided(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	doc := h.synced(accountingsync.SyncObjectCustomerPayment, "401", 40_000)
	doc.state.Voided = true
	doc.state.State = "Reversed"
	h.provider(accountingsync.SyncObjectCustomerPayment, "401", "400.00")
	h.reconcile(t)
	finding := h.findings.open()[0]
	require.Equal(t, accountingsync.DriftStatusMismatch, finding.Kind)

	updated, err := h.resolve(t, finding, accountingsync.DriftPushTrenovaValue, h.person())
	require.NoError(t, err)

	assert.Equal(t, accountingsync.SyncOperationVoid, h.records.byID(updated.FixObjectID).Operation)
}

func TestFixWaitsForAChangeAlreadyOnItsWay(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	finding, _ := h.invoiceFinding(t, 125_000, "1200.00")
	_, err := h.resolve(t, finding, accountingsync.DriftPushTrenovaValue, h.person())
	require.NoError(t, err)

	_, err = h.resolve(t, finding, accountingsync.DriftPushTrenovaValue, h.person())

	var business *errortypes.BusinessError
	require.ErrorAs(t, err, &business)
	assert.Len(t, h.records.forObject(finding.ObjectID), 2)
}

func TestAdjustLowersTheInvoiceWithAnAppliedCreditMemoThatNeverSyncs(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	finding, inv := h.invoiceFinding(t, 125_000, "1200.00")
	actor := h.person()

	updated, err := h.resolve(t, finding, accountingsync.DriftAdjustTrenova, actor)
	require.NoError(t, err)

	require.Len(t, h.invoices.memos, 1)
	memo := h.invoices.memos[0]
	assert.Equal(t, billingqueue.BillTypeCreditMemo, memo.req.BillType)
	assert.Equal(t, inv.ID, memo.req.ReferenceInvoiceID)
	assert.Equal(t, inv.CustomerID, memo.req.CustomerID)
	assert.True(t, memo.req.AutoPost)
	require.Len(t, memo.req.Lines, 1)
	assert.Equal(t, "50", memo.req.Lines[0].Amount.String())
	assert.Same(t, actor, memo.actor)

	require.Len(t, h.payments.applied, 1)
	assert.Equal(t, memo.memo.ID, h.payments.applied[0].CreditMemoID)
	assert.Equal(t, inv.ID, h.payments.applied[0].Applications[0].InvoiceID)
	assert.Equal(t, int64(5_000), h.payments.applied[0].Applications[0].AppliedAmountMinor)

	assert.Equal(t, accountingsync.DriftStatusResolved, updated.Status)
	assert.Equal(t, accountingsync.DriftAdjustedTrenova, updated.Resolution)
	assert.Equal(t, accountingsync.DriftFixCreditMemo, updated.FixObjectType)
	assert.Equal(t, memo.memo.ID, updated.FixObjectID)

	reflected, sent := 0, 0
	for _, record := range h.records.all() {
		if record.ObjectID == inv.ID {
			continue
		}
		if record.ConnectionID == h.conn.ID {
			assert.Equal(t, accountingsync.SyncStatusSkipped, record.Status)
			assert.Equal(t, "101", record.ReflectedIn())
			reflected++
			continue
		}
		assert.Equal(t, accountingsync.SyncStatusQueued, record.Status,
			"another connection never saw the provider's edit, so it still gets the memo")
		sent++
	}
	assert.Equal(t, 2, reflected, "the memo and its application")
	assert.Equal(t, 2, sent)
	assert.Zero(t, h.kicks.kicks)
}

func TestAdjustRaisesTheInvoiceWithADebitMemo(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	finding, _ := h.invoiceFinding(t, 125_000, "1300.00")

	updated, err := h.resolve(t, finding, accountingsync.DriftAdjustTrenova, h.person())
	require.NoError(t, err)

	require.Len(t, h.invoices.memos, 1)
	assert.Equal(t, billingqueue.BillTypeDebitMemo, h.invoices.memos[0].req.BillType)
	assert.Equal(t, "50", h.invoices.memos[0].req.Lines[0].Amount.String())
	assert.Empty(t, h.payments.applied)
	assert.Equal(t, accountingsync.DriftFixDebitMemo, updated.FixObjectType)
}

func TestAdjustAppliesNoMoreCreditThanTheInvoiceStillOwes(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	finding, inv := h.invoiceFinding(t, 125_000, "1200.00")
	h.invRepo.rows[inv.ID].BalanceDueMinor = 2_000

	_, err := h.resolve(t, finding, accountingsync.DriftAdjustTrenova, h.person())
	require.NoError(t, err)

	require.Len(t, h.payments.applied, 1)
	assert.Equal(t, int64(2_000), h.payments.applied[0].Applications[0].AppliedAmountMinor)
}

func TestAdjustVoidsAnInvoiceTheProviderDeletedWithoutRebilling(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	finding, inv := h.invoiceFinding(t, 125_000, "")

	updated, err := h.resolve(t, finding, accountingsync.DriftAdjustTrenova, h.person())
	require.NoError(t, err)

	require.Len(t, h.invoices.voids, 1)
	assert.Equal(t, inv.ID, h.invoices.voids[0].InvoiceID)
	assert.Equal(t, invoice.VoidDispositionDoNotRebill, h.invoices.voids[0].Disposition)
	assert.Contains(t, h.invoices.voids[0].Reason, "deleted")
	assert.Equal(t, accountingsync.DriftFixInvoiceVoid, updated.FixObjectType)
	for _, record := range h.records.all() {
		if record.ConnectionID == h.conn.ID && record.Operation == accountingsync.SyncOperationVoid {
			assert.Equal(t, accountingsync.SyncStatusSkipped, record.Status)
		}
	}
}

func TestAdjustRefusesToVoidAnInvoiceWithMoneyApplied(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	finding, inv := h.invoiceFinding(t, 125_000, "")
	h.invRepo.rows[inv.ID].AppliedAmountMinor = 1_000

	_, err := h.resolve(t, finding, accountingsync.DriftAdjustTrenova, h.person())

	var business *errortypes.BusinessError
	require.ErrorAs(t, err, &business)
	assert.Empty(t, h.invoices.voids)
	assert.True(t, h.findings.open()[0].IsOpen())
}

func TestAdjustFailsWholeWhenTheVoidNeedsAnApprover(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.invoices.pendingApproval = true
	finding, _ := h.invoiceFinding(t, 125_000, "")

	_, err := h.resolve(t, finding, accountingsync.DriftAdjustTrenova, h.person())

	var business *errortypes.BusinessError
	require.ErrorAs(t, err, &business)
	assert.Len(t, h.findings.open(), 1)
}

func TestAdjustReversesAPaymentTheProviderVoided(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	doc := h.synced(accountingsync.SyncObjectCustomerPayment, "401", 40_000)
	h.payments.rows[doc.record.ObjectID] = &customerpayment.Payment{
		ID:          doc.record.ObjectID,
		AmountMinor: 40_000,
		Status:      customerpayment.StatusPosted,
	}
	state := h.provider(accountingsync.SyncObjectCustomerPayment, "401", "0")
	state.Voided = true
	h.reader.set(accountingsync.SyncObjectCustomerPayment, state)
	h.reconcile(t)
	finding := h.findings.open()[0]

	updated, err := h.resolve(t, finding, accountingsync.DriftAdjustTrenova, h.person())
	require.NoError(t, err)

	require.Len(t, h.payments.reversed, 1)
	assert.Equal(t, doc.record.ObjectID, h.payments.reversed[0].PaymentID)
	assert.Equal(t, accountingsync.DriftFixPaymentReversal, updated.FixObjectType)
	for _, record := range h.records.forObject(doc.record.ObjectID) {
		if record.Operation == accountingsync.SyncOperationVoid && record.ConnectionID == h.conn.ID {
			assert.Equal(t, "401", record.ReflectedIn())
		}
	}
}

func TestAdjustNeedsThePermissionOfWhatItPosts(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	finding, _ := h.invoiceFinding(t, 125_000, "1200.00")
	h.perms.denied[permission.ResourceInvoice.String()] = true

	_, err := h.resolve(t, finding, accountingsync.DriftAdjustTrenova, h.person())

	var authz *errortypes.AuthorizationError
	require.ErrorAs(t, err, &authz)
	assert.Empty(t, h.invoices.memos)

	_, err = h.resolve(t, finding, accountingsync.DriftPushTrenovaValue, h.person())
	require.NoError(t, err, "pushing posts nothing in Trenova")
}

func TestFixNeedsAccountingSyncUpdate(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	finding, _ := h.invoiceFinding(t, 125_000, "1200.00")
	h.perms.denied[permission.ResourceAccountingSync.String()] = true

	_, err := h.resolve(t, finding, accountingsync.DriftPushTrenovaValue, h.person())

	var authz *errortypes.AuthorizationError
	require.ErrorAs(t, err, &authz)
	assert.Len(t, h.records.forObject(finding.ObjectID), 1)
}

func TestFixRefusesADirectionTheFindingDoesNotOffer(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.synced(accountingsync.SyncObjectCarrierBill, "501", 90_000)
	h.provider(accountingsync.SyncObjectCarrierBill, "501", "800.00")
	h.reconcile(t)
	finding := h.findings.open()[0]

	_, err := h.resolve(t, finding, accountingsync.DriftAdjustTrenova, h.person())

	var validation *errortypes.Error
	require.ErrorAs(t, err, &validation)
	assert.Equal(t, "direction", validation.Field)
}

func TestPreviewSaysWhatTheFixDoesAndWritesNothing(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	finding, _ := h.invoiceFinding(t, 125_000, "1200.00")

	adjust, err := h.svc.PreviewResolve(t.Context(), &services.ResolveAccountingDriftRequest{
		TenantInfo: h.tenant,
		ID:         finding.ID,
		Direction:  accountingsync.DriftAdjustTrenova,
	}, h.person())
	require.NoError(t, err)
	push, err := h.svc.PreviewResolve(t.Context(), &services.ResolveAccountingDriftRequest{
		TenantInfo: h.tenant,
		ID:         finding.ID,
		Direction:  accountingsync.DriftPushTrenovaValue,
	}, h.person())
	require.NoError(t, err)

	assert.Equal(t, accountingsync.DriftFixCreditMemo, adjust.FixObject)
	assert.Equal(t, int64(5_000), adjust.AmountMinor)
	assert.Contains(t, adjust.Summary, "credit memo of 50.00 USD")
	assert.Contains(t, adjust.Summary, "1200.00 USD")
	assert.Equal(t, accountingsync.SyncOperationUpdate, push.Operation)
	assert.Contains(t, push.Summary, "1250.00 USD replaces 1200.00 USD")
	assert.Equal(t, int64(500), push.ToleranceMinor)
	assert.False(t, push.WithinTolerance)
	assert.Empty(t, h.invoices.memos)
	assert.Len(t, h.records.forObject(finding.ObjectID), 1)
	assert.True(t, h.findings.open()[0].IsOpen())
}

func TestDismissNeedsANote(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	finding, _ := h.invoiceFinding(t, 125_000, "1200.00")

	_, err := h.svc.Dismiss(t.Context(), &services.DismissAccountingDriftRequest{
		TenantInfo: h.tenant,
		ID:         finding.ID,
		Note:       "   ",
	}, h.person())

	var validation *errortypes.Error
	require.ErrorAs(t, err, &validation)
	assert.Equal(t, "note", validation.Field)

	_, err = h.svc.PreviewDismiss(t.Context(), &services.DismissAccountingDriftRequest{
		TenantInfo: h.tenant,
		ID:         finding.ID,
		Note:       "",
	}, h.person())
	require.ErrorAs(t, err, &validation, "the preview asks for the note too")
	assert.Equal(t, "note", validation.Field)
}

func TestAPersonMayDismissAnyFinding(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	finding, _ := h.invoiceFinding(t, 125_000, "1200.00")
	actor := h.person()

	updated, err := h.svc.Dismiss(t.Context(), &services.DismissAccountingDriftRequest{
		TenantInfo: h.tenant,
		ID:         finding.ID,
		Note:       "Agreed with the customer; the books keep their figure",
	}, actor)
	require.NoError(t, err)

	assert.Equal(t, accountingsync.DriftStatusDismissed, updated.Status)
	assert.Equal(t, actor.UserID, updated.ResolvedByID)
	assert.Equal(t, "Agreed with the customer; the books keep their figure", updated.ResolutionNote)
}

func TestAnAgentDismissesOnlyWithinTheTolerance(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	finding, _ := h.invoiceFinding(t, 125_000, "1200.00")

	_, err := h.svc.Dismiss(t.Context(), &services.DismissAccountingDriftRequest{
		TenantInfo: h.tenant,
		ID:         finding.ID,
		Note:       "Rounding",
	}, h.agent())
	var authz *errortypes.AuthorizationError
	require.ErrorAs(t, err, &authz)
	assert.True(t, h.findings.open()[0].IsOpen())

	_, err = h.svc.PreviewDismiss(t.Context(), &services.DismissAccountingDriftRequest{
		TenantInfo: h.tenant,
		ID:         finding.ID,
		Note:       "Rounding",
	}, h.agent())
	require.ErrorAs(t, err, &authz, "the preview refuses what the dismissal would")

	h.controls.tolerance = "50.00"
	preview, err := h.svc.PreviewDismiss(t.Context(), &services.DismissAccountingDriftRequest{
		TenantInfo: h.tenant,
		ID:         finding.ID,
		Note:       "Rounding",
	}, h.agent())
	require.NoError(t, err)
	assert.True(t, preview.WithinTolerance)
	updated, err := h.svc.Dismiss(t.Context(), &services.DismissAccountingDriftRequest{
		TenantInfo: h.tenant,
		ID:         finding.ID,
		Note:       "Rounding",
	}, h.agent())
	require.NoError(t, err)
	assert.Equal(t, accountingsync.DriftStatusDismissed, updated.Status)
}

func TestAnAgentNeverDismissesADeletedDocument(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.controls.tolerance = "100000.00"
	finding, _ := h.invoiceFinding(t, 125_000, "")

	_, err := h.svc.Dismiss(t.Context(), &services.DismissAccountingDriftRequest{
		TenantInfo: h.tenant,
		ID:         finding.ID,
		Note:       "Looks fine",
	}, h.agent())

	var authz *errortypes.AuthorizationError
	require.ErrorAs(t, err, &authz)
}

func TestReversingAPaymentNeedsCustomerPaymentUpdate(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	doc := h.synced(accountingsync.SyncObjectCustomerPayment, "401", 40_000)
	h.payments.rows[doc.record.ObjectID] = &customerpayment.Payment{
		ID:          doc.record.ObjectID,
		AmountMinor: 40_000,
		Status:      customerpayment.StatusPosted,
	}
	h.reconcile(t)
	finding := h.findings.open()[0]
	h.perms.denied[permission.ResourceCustomerPayment.String()] = true

	_, err := h.resolve(t, finding, accountingsync.DriftAdjustTrenova, h.person())

	var authz *errortypes.AuthorizationError
	require.ErrorAs(t, err, &authz)
	assert.Empty(t, h.payments.reversed)
}
