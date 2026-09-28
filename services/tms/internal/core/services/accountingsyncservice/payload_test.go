package accountingsyncservice

import (
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/customerpayment"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/invoiceadjustment"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func salesDocs(t *testing.T, h *harness) []*services.AccountingSalesDocument {
	t.Helper()
	calls := h.writer.callsTo("CreateSalesDocument")
	out := make([]*services.AccountingSalesDocument, 0, len(calls))
	for _, call := range calls {
		doc, ok := call.doc.(*services.AccountingSalesDocument)
		require.True(t, ok)
		out = append(out, doc)
	}
	return out
}

func onlySalesDoc(t *testing.T, h *harness) *services.AccountingSalesDocument {
	t.Helper()
	docs := salesDocs(t, h)
	require.Len(t, docs, 1)
	return docs[0]
}

func decimalEqual(t *testing.T, want string, got decimal.Decimal) {
	t.Helper()
	assert.True(t, decimal.RequireFromString(want).Equal(got), "want %s, got %s", want, got)
}

func (h *harness) mappedCustomer(name string) (pulid.ID, *accountingsync.AccountingMapping) {
	customerID := pulid.MustNew("cus_")
	return customerID, h.confirm(customerTarget(customerID, name), "qb-cust-"+name)
}

func TestInvoiceDocumentCarriesExternalIDsFromConfirmedMappings(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	customerID, customerMapping := h.mappedCustomer("Acme")
	chargeID := pulid.MustNew("acc_")
	freight := h.confirm(freightTarget(), "qb-item-freight")
	charge := h.confirm(chargeTarget(chargeID), "qb-item-det")
	term := h.confirm(mappingTarget{
		TargetType: accountingsync.TargetPaymentTerm,
		Key:        string(invoice.PaymentTermNet30),
		Label:      "Net 30",
	}, "qb-term-30")
	zero := freightLine("0")
	inv := h.postedInvoice(customerID, invoiceSpec{
		number: "INV-1001",
		lines:  []*invoice.InvoiceLine{freightLine("1500.00"), chargeLine(chargeID, "200.00"), zero},
	})
	record := h.enqueueInvoice(t, inv)

	result := h.drain(t)

	assert.Equal(t, 1, result.Claimed)
	assert.Equal(t, 1, result.Synced)
	doc := onlySalesDoc(t, h)
	assert.Equal(t, accountingsync.SyncObjectInvoice, doc.Kind)
	assert.Equal(t, record.RequestID, doc.RequestID)
	assert.Equal(t, "qb-cust-Acme", doc.CustomerExternalID)
	assert.Equal(t, "INV-1001", doc.DocNumber)
	assert.Equal(t, "2026-04-10", doc.TxnDate)
	assert.Equal(t, "2026-05-10", doc.DueDate)
	assert.Equal(t, "qb-term-30", doc.TermExternalID)
	assert.Equal(t, "USD", doc.CurrencyCode)
	assert.Equal(t, testAccessToken, doc.Auth.AccessToken)
	assert.Contains(t, doc.PrivateNote, "INV-1001")
	assert.Contains(t, doc.CustomerMemo, "PRO PRO-1001")
	require.Len(t, doc.Lines, 2, "a zero line is not sent")
	assert.Equal(t, "qb-item-freight", doc.Lines[0].ItemExternalID)
	decimalEqual(t, "1500", doc.Lines[0].Amount)
	assert.Equal(t, "qb-item-det", doc.Lines[1].ItemExternalID)
	decimalEqual(t, "200", doc.Lines[1].Amount)
	decimalEqual(t, "2", doc.Lines[1].Quantity)
	decimalEqual(t, "100", doc.Lines[1].UnitPrice)

	synced := h.records.get(record.ID)
	assert.Equal(t, accountingsync.SyncStatusSynced, synced.Status)
	assert.NotEmpty(t, synced.ExternalID)
	assert.Equal(t, "https://qbo.test/Invoice/"+synced.ExternalID, synced.ExternalURL)
	assert.NotNil(t, synced.SyncedAt)
	assert.Empty(t, synced.ErrorCategory)
	assert.ElementsMatch(t, []string{
		customerMapping.ID.String(), freight.ID.String(), charge.ID.String(), term.ID.String(),
	}, synced.MappingIDs, "the record names every confirmed mapping its payload used")
	assert.Len(t, synced.PayloadHash, 64)
	assert.Equal(t, "INV-1001", synced.Payload["DocNumber"])
}

func TestSyncedPayloadNeverCarriesTheAccessToken(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	customerID, _ := h.mappedCustomer("Acme")
	h.confirm(freightTarget(), "qb-item-freight")
	record := h.enqueueInvoice(t, h.postedInvoice(customerID, invoiceSpec{}))

	h.drain(t)

	synced := h.records.get(record.ID)
	require.NotEmpty(t, synced.Payload)
	assert.NotContains(t, synced.Payload, "Auth")
	raw, err := sonic.Marshal(synced.Payload)
	require.NoError(t, err)
	assert.NotContains(t, string(raw), testAccessToken)
}

func TestPayloadHashDoesNotDependOnTheCredentials(t *testing.T) {
	t.Parallel()

	doc := func(token string) *services.AccountingSalesDocument {
		return &services.AccountingSalesDocument{
			Auth:      services.AccountingDocumentAuth{RealmID: "1", AccessToken: token},
			RequestID: "trn-1",
			Kind:      accountingsync.SyncObjectInvoice,
			DocNumber: "INV-1",
			Lines: []services.AccountingDocumentLine{{
				ItemExternalID: "qb-item",
				Amount:         decimal.RequireFromString("10.50"),
			}},
		}
	}

	firstPayload, firstHash, err := payloadOf(doc("token-before-refresh"))
	require.NoError(t, err)
	secondPayload, secondHash, err := payloadOf(doc("token-after-refresh"))
	require.NoError(t, err)

	assert.Equal(t, firstPayload, secondPayload)
	assert.Equal(t, firstHash, secondHash,
		"the same document hashes the same after the access token is refreshed")
	assert.Len(t, firstHash, 64)
}

func TestUnmappedChargeCodeBlocksWithTheExactMapping(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	customerID, _ := h.mappedCustomer("Acme")
	h.confirm(freightTarget(), "qb-item-freight")
	chargeID := pulid.MustNew("acc_")
	inv := h.postedInvoice(customerID, invoiceSpec{
		lines: []*invoice.InvoiceLine{freightLine("100"), chargeLine(chargeID, "50")},
	})
	record := h.enqueueInvoice(t, inv)

	result := h.drain(t)

	assert.Equal(t, 1, result.Blocked)
	assert.Empty(t, h.writer.callsTo("CreateSalesDocument"))
	blocked := h.records.get(record.ID)
	assert.Equal(t, accountingsync.SyncStatusBlocked, blocked.Status)
	assert.Equal(t, accountingsync.SyncErrorMapping, blocked.ErrorCategory)
	assert.Equal(t, "Map Charge code DET to a QuickBooks Online item", blocked.Resolution)
	assert.Equal(t, []string{"accounting.sync_blocked"}, h.eventKinds())
}

func TestUnmappedFreightLineBlocksAsMapping(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	customerID, _ := h.mappedCustomer("Acme")
	record := h.enqueueInvoice(t, h.postedInvoice(customerID, invoiceSpec{}))

	h.drain(t)

	blocked := h.records.get(record.ID)
	assert.Equal(t, accountingsync.SyncStatusBlocked, blocked.Status)
	assert.Equal(t, accountingsync.SyncErrorMapping, blocked.ErrorCategory)
	assert.Equal(t, "Map freight lines to a QuickBooks Online item", blocked.Resolution)
}

func TestUnmappedCustomerQueuesACustomerCreateAndTheDocumentWaits(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.updateConnection(func(conn *accountingsync.AccountingConnection) { conn.AutoSync = false })
	customerID := pulid.MustNew("cus_")
	customerRow := h.mappings.set(
		h.conn, customerTarget(customerID, "Acme Freight"), accountingsync.MappingStateUnmatched, "",
	)
	h.confirm(freightTarget(), "qb-item-freight")
	inv := h.postedInvoice(customerID, invoiceSpec{})
	record := h.enqueueInvoice(t, inv)
	_, err := h.svc.Release(t.Context(), &services.ReleaseAccountingSyncRequest{
		TenantInfo:      h.tenant,
		UserID:          h.userID,
		IntegrationType: h.conn.IntegrationType,
	})
	require.NoError(t, err)

	first := h.drain(t)

	assert.Equal(t, 1, first.Waiting)
	assert.Empty(t, h.writer.callsTo("CreateSalesDocument"))
	dependency := h.only(t, accountingsync.SyncObjectCustomer, customerID, accountingsync.SyncOperationCreate)
	assert.Equal(t, accountingsync.SyncSourceDependencyOf, dependency.SourceEvent)
	assert.Equal(t, accountingsync.SyncStatusQueued, dependency.Status,
		"a dependency is never held for approval")
	waiting := h.records.get(record.ID)
	assert.Equal(t, accountingsync.SyncStatusQueued, waiting.Status)
	assert.Equal(t, dependency.ID, waiting.DependsOnRecordID)
	assert.Equal(t, 0, waiting.AttemptCount, "waiting for a dependency spends no attempt")
	nearNow(t, time.Minute, waiting.NextAttemptAt)
	assert.Empty(t, h.records.attemptsFor(record.ID), "a dependency wait is not a push")
	assert.Contains(t, h.dispatcher.kicked(), h.conn.ID)

	second := h.drain(t)

	assert.Equal(t, 1, second.Synced)
	require.Len(t, h.mappings.created, 1)
	assert.Equal(t, customerRow.ID, h.mappings.created[0].MappingID)
	assert.Equal(t, accountingsync.MappingSourceCreatedInProvider, h.mappings.created[0].Source)
	createdCustomer := h.records.get(dependency.ID)
	assert.Equal(t, accountingsync.SyncStatusSynced, createdCustomer.Status)
	assert.Equal(t, "qb-created-"+customerID.String(), createdCustomer.ExternalID)
	assert.Equal(t, []string{customerRow.ID.String()}, createdCustomer.MappingIDs)

	h.records.makeDue(record.ID)
	third := h.drain(t)

	assert.Equal(t, 1, third.Synced)
	assert.Equal(t, "qb-created-"+customerID.String(), onlySalesDoc(t, h).CustomerExternalID)
}

func TestDuplicateCustomerNameBlocksAsMappingAndRequeuesOnceMapped(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	customerID := pulid.MustNew("cus_")
	h.mappings.set(h.conn, customerTarget(customerID, "Acme Freight"), accountingsync.MappingStateUnmatched, "")
	h.confirm(freightTarget(), "qb-item-freight")
	h.mappings.createErr = errortypes.NewBusinessError(
		"{0} already exists in the accounting system; map {1} to it instead",
		"Acme Freight LLC (Customer)",
		"Acme Freight",
	).WithInternal(providerFault(accountingsync.SyncErrorDuplicate, "Duplicate Name Exists Error", ""))
	record := h.enqueueInvoice(t, h.postedInvoice(customerID, invoiceSpec{}))

	h.drain(t)
	h.drain(t)

	dependency := h.only(t, accountingsync.SyncObjectCustomer, customerID, accountingsync.SyncOperationCreate)
	assert.Equal(t, accountingsync.SyncStatusBlocked, dependency.Status)
	assert.Equal(t, accountingsync.SyncErrorMapping, dependency.ErrorCategory,
		"a duplicate name is a mapping problem: map the customer to the existing record")
	assert.Contains(t, dependency.ErrorMessage, "Acme Freight LLC")
	assert.Contains(t, dependency.Resolution, "Map Acme Freight")

	h.records.makeDue(record.ID)
	h.drain(t)

	held := h.records.get(record.ID)
	assert.Equal(t, accountingsync.SyncStatusQueued, held.Status)
	assert.Equal(t, dependency.ID, held.DependsOnRecordID)
	assert.Contains(t, held.Resolution, "Customer Acme Freight")
	nearNow(t, accountingsync.SyncAuthWait, held.NextAttemptAt)
	assert.Empty(t, h.writer.callsTo("CreateSalesDocument"))

	h.confirm(customerTarget(customerID, "Acme Freight"), "qb-cust-58")
	requeued, err := h.svc.Retry(t.Context(), &services.RetryAccountingSyncRequest{
		TenantInfo:      h.tenant,
		UserID:          h.userID,
		IntegrationType: h.conn.IntegrationType,
		ErrorCategories: []accountingsync.SyncErrorCategory{accountingsync.SyncErrorMapping},
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), requeued, "confirming the mapping re-queues the Mapping-blocked customer")

	h.drain(t)
	h.records.makeDue(record.ID)
	h.drain(t)

	assert.Equal(t, accountingsync.SyncStatusSynced, h.records.get(dependency.ID).Status)
	assert.Equal(t, "qb-cust-58", h.records.get(dependency.ID).ExternalID)
	assert.Equal(t, accountingsync.SyncStatusSynced, h.records.get(record.ID).Status)
	assert.Equal(t, "qb-cust-58", onlySalesDoc(t, h).CustomerExternalID)
}

func TestCurrencyMismatchBlocksWithoutMulticurrency(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	customerID, _ := h.mappedCustomer("Acme")
	h.confirm(freightTarget(), "qb-item-freight")
	record := h.enqueueInvoice(t, h.postedInvoice(customerID, invoiceSpec{currency: "CAD"}))

	h.drain(t)

	blocked := h.records.get(record.ID)
	assert.Equal(t, accountingsync.SyncStatusBlocked, blocked.Status)
	assert.Equal(t, accountingsync.SyncErrorCurrency, blocked.ErrorCategory)
	assert.Contains(t, blocked.ErrorMessage, "CAD")
	assert.Contains(t, blocked.ErrorMessage, "USD")
	assert.Contains(t, blocked.Resolution, "multicurrency")
	assert.Empty(t, h.writer.callsTo("CreateSalesDocument"))
}

func TestCurrencyIsSentWhenTheProviderHasMulticurrency(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.updateConnection(func(conn *accountingsync.AccountingConnection) {
		conn.ExternalMultiCurrencyEnabled = true
	})
	h.rates.set("CAD", "USD", decimal.RequireFromString("0.7312"))
	customerID, _ := h.mappedCustomer("Acme")
	h.confirm(freightTarget(), "qb-item-freight")
	record := h.enqueueInvoice(t, h.postedInvoice(customerID, invoiceSpec{currency: "cad"}))

	h.drain(t)

	assert.Equal(t, accountingsync.SyncStatusSynced, h.records.get(record.ID).Status)
	assert.Equal(t, "cad", onlySalesDoc(t, h).CurrencyCode)
	decimalEqual(t, "0.7312", onlySalesDoc(t, h).ExchangeRate)
}

func TestDocumentDatedInAClosedPeriodBlocks(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	closed := aprilTenth
	h.updateConnection(func(conn *accountingsync.AccountingConnection) {
		conn.ExternalBooksClosedThrough = &closed
	})
	customerID, _ := h.mappedCustomer("Acme")
	h.confirm(freightTarget(), "qb-item-freight")
	onClose := h.enqueueInvoice(t, h.postedInvoice(customerID, invoiceSpec{date: aprilTenth}))
	afterClose := h.enqueueInvoice(t, h.postedInvoice(customerID, invoiceSpec{date: aprilTenth + 86400}))

	h.drain(t)

	blocked := h.records.get(onClose.ID)
	assert.Equal(t, accountingsync.SyncStatusBlocked, blocked.Status)
	assert.Equal(t, accountingsync.SyncErrorClosedPeriod, blocked.ErrorCategory)
	assert.Contains(t, blocked.Resolution, "Reopen the period in QuickBooks Online")
	assert.Equal(t, accountingsync.SyncStatusSynced, h.records.get(afterClose.ID).Status)
}

func TestLongDocumentNumberIsLeftForTheProvider(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	customerID, _ := h.mappedCustomer("Acme")
	h.confirm(freightTarget(), "qb-item-freight")
	longNumber := "INV-2026-00000000001234"
	shortNumber := "INV-2026-000000000123"
	h.enqueueInvoice(t, h.postedInvoice(customerID, invoiceSpec{number: longNumber}))
	h.enqueueInvoice(t, h.postedInvoice(customerID, invoiceSpec{number: shortNumber}))

	h.drain(t)

	docs := salesDocs(t, h)
	require.Len(t, docs, 2)
	byNote := map[string]*services.AccountingSalesDocument{}
	for _, doc := range docs {
		if doc.DocNumber == "" {
			byNote["long"] = doc
		} else {
			byNote["short"] = doc
		}
	}
	require.Contains(t, byNote, "long")
	require.Contains(t, byNote, "short")
	assert.Contains(t, byNote["long"].PrivateNote, longNumber,
		"Trenova's number is written into the private note")
	assert.Equal(t, shortNumber, byNote["short"].DocNumber)
}

func TestCreditMemoDocumentSendsPositiveAmountsWithoutTermsOrDueDate(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	customerID, _ := h.mappedCustomer("Acme")
	h.confirm(freightTarget(), "qb-item-freight")
	h.confirm(mappingTarget{
		TargetType: accountingsync.TargetPaymentTerm,
		Key:        string(invoice.PaymentTermNet30),
	}, "qb-term-30")
	memo := h.postedInvoice(customerID, invoiceSpec{
		number:   "CM-1",
		billType: billingqueue.BillTypeCreditMemo,
		lines:    []*invoice.InvoiceLine{freightLine("-300.00")},
	})
	record := h.enqueueInvoice(t, memo)

	h.drain(t)

	doc := onlySalesDoc(t, h)
	assert.Equal(t, accountingsync.SyncObjectCreditMemo, doc.Kind)
	assert.Empty(t, doc.DueDate)
	assert.Empty(t, doc.TermExternalID)
	assert.Empty(t, doc.ApplyToExternalID, "a memo not born of a full reversal stands open")
	require.Len(t, doc.Lines, 1)
	decimalEqual(t, "300", doc.Lines[0].Amount)
	decimalEqual(t, "300", doc.Lines[0].UnitPrice)
	assert.Equal(t, accountingsync.SyncStatusSynced, h.records.get(record.ID).Status)
}

func TestDebitMemoIsSentAsItsOwnKind(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	customerID, _ := h.mappedCustomer("Acme")
	h.confirm(freightTarget(), "qb-item-freight")
	memo := h.postedInvoice(customerID, invoiceSpec{number: "DM-1", billType: billingqueue.BillTypeDebitMemo})
	record := h.enqueueInvoice(t, memo)

	h.drain(t)

	assert.Equal(t, accountingsync.SyncObjectDebitMemo, record.ObjectType)
	doc := onlySalesDoc(t, h)
	assert.Equal(t, accountingsync.SyncObjectDebitMemo, doc.Kind)
	assert.Equal(t, "DM-1", doc.DocNumber)
	assert.NotEmpty(t, doc.DueDate)
}

func (h *harness) adjustmentMemo(
	customerID pulid.ID,
	original *invoice.Invoice,
	kind invoiceadjustment.Kind,
) *invoice.Invoice {
	adjustment := &invoiceadjustment.InvoiceAdjustment{
		ID:                pulid.MustNew("iadj_"),
		OrganizationID:    h.tenant.OrgID,
		BusinessUnitID:    h.tenant.BuID,
		OriginalInvoiceID: original.ID,
		Kind:              kind,
	}
	h.adjustments.mu.Lock()
	h.adjustments.rows[adjustment.ID] = adjustment
	h.adjustments.mu.Unlock()
	memo := h.postedInvoice(customerID, invoiceSpec{
		number:   "CM-" + original.Number,
		billType: billingqueue.BillTypeCreditMemo,
		date:     mayTenth,
		lines:    []*invoice.InvoiceLine{freightLine("-1500.00")},
	})
	memo.SourceInvoiceAdjustmentID = adjustment.ID
	memo.IsAdjustmentArtifact = true
	return memo
}

func TestFullReversalCreditMemoAppliesToTheOriginalInvoice(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	customerID, _ := h.mappedCustomer("Acme")
	h.confirm(freightTarget(), "qb-item-freight")
	original := h.postedInvoice(customerID, invoiceSpec{number: "INV-9"})
	h.syncedInvoiceRecord(original, "qb-inv-9")
	memo := h.adjustmentMemo(customerID, original, invoiceadjustment.KindFullReversal)
	record := h.enqueueInvoice(t, memo)

	h.drain(t)

	doc := onlySalesDoc(t, h)
	assert.Equal(t, accountingsync.SyncObjectCreditMemo, doc.Kind)
	assert.Equal(t, "qb-inv-9", doc.ApplyToExternalID)
	decimalEqual(t, "1500", doc.ApplyAmount)
	decimalEqual(t, "1500", doc.Lines[0].Amount)
	assert.Equal(t, accountingsync.SyncStatusSynced, h.records.get(record.ID).Status)
}

func TestFullReversalCreditMemoWaitsForTheOriginalInvoice(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	customerID, _ := h.mappedCustomer("Acme")
	h.confirm(freightTarget(), "qb-item-freight")
	original := h.postedInvoice(customerID, invoiceSpec{number: "INV-9"})
	originalRecord := h.enqueueInvoice(t, original)
	h.records.postpone(originalRecord.ID)
	memo := h.adjustmentMemo(customerID, original, invoiceadjustment.KindFullReversal)
	record := h.enqueueInvoice(t, memo)

	result := h.drain(t)

	assert.Equal(t, 1, result.Waiting)
	assert.Empty(t, h.writer.callsTo("CreateSalesDocument"))
	waiting := h.records.get(record.ID)
	assert.Equal(t, accountingsync.SyncStatusQueued, waiting.Status)
	assert.Equal(t, originalRecord.ID, waiting.DependsOnRecordID)
}

func TestCreditOnlyAdjustmentMemoIsNotAppliedToItsInvoice(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	customerID, _ := h.mappedCustomer("Acme")
	h.confirm(freightTarget(), "qb-item-freight")
	original := h.postedInvoice(customerID, invoiceSpec{number: "INV-9"})
	h.syncedInvoiceRecord(original, "qb-inv-9")
	memo := h.adjustmentMemo(customerID, original, invoiceadjustment.KindCreditOnly)
	h.enqueueInvoice(t, memo)

	h.drain(t)

	doc := onlySalesDoc(t, h)
	assert.Empty(t, doc.ApplyToExternalID)
	assert.True(t, doc.ApplyAmount.IsZero())
}

func TestUnpostedInvoiceBlocksAsValidation(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	customerID, _ := h.mappedCustomer("Acme")
	inv := h.postedInvoice(customerID, invoiceSpec{})
	inv.Status = invoice.StatusDraft
	inv.PostedAt = nil
	record := h.enqueueInvoice(t, inv)

	h.drain(t)

	blocked := h.records.get(record.ID)
	assert.Equal(t, accountingsync.SyncStatusBlocked, blocked.Status)
	assert.Equal(t, accountingsync.SyncErrorValidation, blocked.ErrorCategory)
}

func TestMissingInvoiceBlocksAsNotFound(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.enqueue(t, &services.AccountingSyncEnqueueRequest{
		ObjectType:   accountingsync.SyncObjectInvoice,
		ObjectID:     pulid.MustNew("inv_"),
		Operation:    accountingsync.SyncOperationCreate,
		Revision:     1,
		SourceEvent:  accountingsync.SyncSourceInvoicePosted,
		DocumentDate: aprilTenth,
	})

	h.drain(t)

	record := h.records.all()[0]
	assert.Equal(t, accountingsync.SyncStatusBlocked, record.Status)
	assert.Equal(t, accountingsync.SyncErrorNotFound, record.ErrorCategory)
	assert.Equal(t, "Skip this record", record.Resolution)
}

type paymentSpec struct {
	method       customerpayment.Method
	amountMinor  int64
	applications []*customerpayment.Application
}

func (h *harness) postedPayment(customerID pulid.ID, spec paymentSpec) *customerpayment.Payment {
	if spec.method == "" {
		spec.method = customerpayment.MethodCheck
	}
	payment := &customerpayment.Payment{
		ID:              pulid.MustNew("cpay_"),
		OrganizationID:  h.tenant.OrgID,
		BusinessUnitID:  h.tenant.BuID,
		CustomerID:      customerID,
		PaymentDate:     mayTenth - 86400,
		AccountingDate:  mayTenth,
		AmountMinor:     spec.amountMinor,
		Status:          customerpayment.StatusPosted,
		PaymentMethod:   spec.method,
		ReferenceNumber: "CHK-5001",
		Memo:            "Remit",
		CurrencyCode:    "USD",
		Version:         1,
		Applications:    spec.applications,
	}
	h.payments.mu.Lock()
	h.payments.rows[payment.ID] = payment
	h.payments.mu.Unlock()
	return payment
}

func (h *harness) mapPaymentAccounts(method customerpayment.Method) {
	h.confirm(mappingTarget{
		TargetType: accountingsync.TargetAccountRole,
		Key:        accountingsync.AccountRoleDeposit,
	}, "qb-acct-undeposited")
	h.confirm(mappingTarget{
		TargetType: accountingsync.TargetPaymentMethod,
		Key:        string(method),
	}, "qb-method-"+string(method))
}

func (h *harness) enqueuePayment(
	t *testing.T,
	payment *customerpayment.Payment,
	operation accountingsync.SyncOperation,
) *accountingsync.AccountingSyncRecord {
	t.Helper()
	require.NoError(t, h.enqueuer.Enqueue(t.Context(), services.PaymentSyncRequest(
		payment, operation, accountingsync.SyncSourceCustomerPaymentPosted,
	)))
	found := h.records.find(accountingsync.SyncObjectCustomerPayment, payment.ID, operation)
	require.NotEmpty(t, found)
	return found[len(found)-1]
}

func paymentDocs(t *testing.T, h *harness) []*services.AccountingPaymentDocument {
	t.Helper()
	calls := h.writer.callsTo("SavePayment")
	out := make([]*services.AccountingPaymentDocument, 0, len(calls))
	for _, call := range calls {
		doc, ok := call.doc.(*services.AccountingPaymentDocument)
		require.True(t, ok)
		out = append(out, doc)
	}
	return out
}

func TestPaymentDocumentUsesTheAccountingDateAndMappedAccounts(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	customerID, _ := h.mappedCustomer("Acme")
	h.mapPaymentAccounts(customerpayment.MethodCheck)
	first := h.postedInvoice(customerID, invoiceSpec{number: "INV-1"})
	second := h.postedInvoice(customerID, invoiceSpec{number: "INV-2"})
	h.syncedInvoiceRecord(first, "qb-inv-1")
	h.syncedInvoiceRecord(second, "qb-inv-2")
	payment := h.postedPayment(customerID, paymentSpec{
		amountMinor: 250000,
		applications: []*customerpayment.Application{
			{InvoiceID: first.ID, AppliedAmountMinor: 150000},
			{InvoiceID: second.ID, AppliedAmountMinor: 100000},
		},
	})
	record := h.enqueuePayment(t, payment, accountingsync.SyncOperationCreate)

	h.drain(t)

	docs := paymentDocs(t, h)
	require.Len(t, docs, 1)
	doc := docs[0]
	assert.Equal(t, record.RequestID, doc.RequestID)
	assert.Empty(t, doc.ExternalID)
	assert.Equal(t, "qb-cust-Acme", doc.CustomerExternalID)
	assert.Equal(t, "2026-05-10", doc.TxnDate, "payments use their accounting date")
	assert.Equal(t, "qb-acct-undeposited", doc.DepositAccountExternalID)
	assert.Equal(t, "qb-method-Check", doc.PaymentMethodExternalID)
	assert.Equal(t, "CHK-5001", doc.ReferenceNumber)
	decimalEqual(t, "2500", doc.TotalAmount)
	require.Len(t, doc.Applications, 2)
	assert.Equal(t, "qb-inv-1", doc.Applications[0].InvoiceExternalID)
	decimalEqual(t, "1500", doc.Applications[0].AppliedAmount)
	assert.Equal(t, "qb-inv-2", doc.Applications[1].InvoiceExternalID)
	assert.Empty(t, doc.ShortPayItemExternalID)
	synced := h.records.get(record.ID)
	assert.Equal(t, accountingsync.SyncStatusSynced, synced.Status)
	assert.Len(t, synced.MappingIDs, 3)
}

func TestShortPayNeedsTheWriteOffItem(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	customerID, _ := h.mappedCustomer("Acme")
	h.mapPaymentAccounts(customerpayment.MethodACH)
	inv := h.postedInvoice(customerID, invoiceSpec{number: "INV-1"})
	h.syncedInvoiceRecord(inv, "qb-inv-1")
	payment := h.postedPayment(customerID, paymentSpec{
		method:      customerpayment.MethodACH,
		amountMinor: 140000,
		applications: []*customerpayment.Application{
			{InvoiceID: inv.ID, AppliedAmountMinor: 140000, ShortPayAmountMinor: 10000},
		},
	})
	record := h.enqueuePayment(t, payment, accountingsync.SyncOperationCreate)

	h.drain(t)

	blocked := h.records.get(record.ID)
	assert.Equal(t, accountingsync.SyncStatusBlocked, blocked.Status)
	assert.Equal(t, accountingsync.SyncErrorMapping, blocked.ErrorCategory)
	assert.Equal(t, "Map The short-pay write-off item to a QuickBooks Online item", blocked.Resolution)

	h.confirm(mappingTarget{
		TargetType: accountingsync.TargetItemRole,
		Key:        accountingsync.ItemRoleShortPayWriteOff,
	}, "qb-item-shortpay")
	_, err := h.svc.Retry(t.Context(), &services.RetryAccountingSyncRequest{
		TenantInfo:      h.tenant,
		UserID:          h.userID,
		IntegrationType: h.conn.IntegrationType,
		IDs:             []pulid.ID{record.ID},
	})
	require.NoError(t, err)
	h.drain(t)

	docs := paymentDocs(t, h)
	require.Len(t, docs, 1)
	assert.Equal(t, "qb-item-shortpay", docs[0].ShortPayItemExternalID)
	decimalEqual(t, "100", docs[0].Applications[0].ShortPayAmount)
}

func TestUnmappedPaymentMethodAndDepositAccountBlock(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		mapDeposit bool
		resolution string
	}{
		{
			name:       "deposit account",
			resolution: "Map The deposit account to a QuickBooks Online account",
		},
		{
			name:       "payment method",
			mapDeposit: true,
			resolution: "Map Payment method Wire to a QuickBooks Online payment method",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := newHarness(t)
			customerID, _ := h.mappedCustomer("Acme")
			if tc.mapDeposit {
				h.confirm(mappingTarget{
					TargetType: accountingsync.TargetAccountRole,
					Key:        accountingsync.AccountRoleDeposit,
				}, "qb-acct-undeposited")
			}
			payment := h.postedPayment(customerID, paymentSpec{method: customerpayment.MethodWire, amountMinor: 100})
			record := h.enqueuePayment(t, payment, accountingsync.SyncOperationCreate)

			h.drain(t)

			blocked := h.records.get(record.ID)
			assert.Equal(t, accountingsync.SyncStatusBlocked, blocked.Status)
			assert.Equal(t, accountingsync.SyncErrorMapping, blocked.ErrorCategory)
			assert.Equal(t, tc.resolution, blocked.Resolution)
			assert.Empty(t, h.writer.callsTo("SavePayment"))
		})
	}
}

func TestPaymentWaitsForTheInvoicesItLinks(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	customerID, _ := h.mappedCustomer("Acme")
	h.mapPaymentAccounts(customerpayment.MethodCheck)
	inv := h.postedInvoice(customerID, invoiceSpec{number: "INV-1"})
	payment := h.postedPayment(customerID, paymentSpec{
		amountMinor:  150000,
		applications: []*customerpayment.Application{{InvoiceID: inv.ID, AppliedAmountMinor: 150000}},
	})
	record := h.enqueuePayment(t, payment, accountingsync.SyncOperationCreate)

	result := h.drain(t)

	assert.Equal(t, 1, result.Waiting)
	invoiceRecord := h.only(t, accountingsync.SyncObjectInvoice, inv.ID, accountingsync.SyncOperationCreate)
	assert.Equal(t, accountingsync.SyncSourceDependencyOf, invoiceRecord.SourceEvent,
		"a linked invoice that never passed an enqueue point is queued as a dependency")
	assert.Equal(t, invoiceRecord.ID, h.records.get(record.ID).DependsOnRecordID)
	assert.Empty(t, h.writer.callsTo("SavePayment"))
}

func TestPaymentLinkingABlockedInvoiceIsNotSent(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	customerID, _ := h.mappedCustomer("Acme")
	h.mapPaymentAccounts(customerpayment.MethodCheck)
	inv := h.postedInvoice(customerID, invoiceSpec{number: "INV-7"})
	invoiceRecord := h.enqueueInvoice(t, inv)
	h.drain(t)
	require.Equal(t, accountingsync.SyncStatusBlocked, h.records.get(invoiceRecord.ID).Status)
	payment := h.postedPayment(customerID, paymentSpec{
		amountMinor:  150000,
		applications: []*customerpayment.Application{{InvoiceID: inv.ID, AppliedAmountMinor: 150000}},
	})
	record := h.enqueuePayment(t, payment, accountingsync.SyncOperationCreate)

	h.drain(t)

	held := h.records.get(record.ID)
	assert.NotEqual(t, accountingsync.SyncStatusSynced, held.Status)
	assert.Equal(t, invoiceRecord.ID, held.DependsOnRecordID)
	assert.Contains(t, held.Resolution, "Invoice INV-7")
	assert.Empty(t, h.writer.callsTo("SavePayment"))
}

func TestPaymentUpdateSendsToTheSyncedPayment(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	customerID, _ := h.mappedCustomer("Acme")
	h.mapPaymentAccounts(customerpayment.MethodCheck)
	payment := h.postedPayment(customerID, paymentSpec{amountMinor: 5000})
	created := h.enqueuePayment(t, payment, accountingsync.SyncOperationCreate)
	h.drain(t)
	createdSynced := h.records.get(created.ID)
	require.Equal(t, accountingsync.SyncStatusSynced, createdSynced.Status)
	payment.Version = 2
	update := h.enqueuePayment(t, payment, accountingsync.SyncOperationUpdate)

	h.drain(t)

	docs := paymentDocs(t, h)
	require.Len(t, docs, 2)
	assert.Equal(t, createdSynced.ExternalID, docs[1].ExternalID)
	assert.Equal(t, update.RequestID, docs[1].RequestID)
	assert.Equal(t, int64(2), update.Revision)
}

func TestPaymentVoidUsesTheSyncedPayment(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	customerID, _ := h.mappedCustomer("Acme")
	payment := h.postedPayment(customerID, paymentSpec{amountMinor: 5000})
	created := NewRecordFor(h.conn, services.PaymentSyncRequest(
		payment, accountingsync.SyncOperationCreate, accountingsync.SyncSourceCustomerPaymentPosted,
	), mayTenth)
	created.MarkSynced(&accountingsync.SyncResult{
		ExternalID:   "qb-pay-1",
		ExternalRefs: map[string]string{accountingsync.ExternalRefDocument: "qb-pay-1"},
	}, mayTenth)
	h.records.put(created)
	payment.Version = 2
	staleUpdate := NewRecordFor(h.conn, services.PaymentSyncRequest(
		payment, accountingsync.SyncOperationUpdate, accountingsync.SyncSourceCustomerPaymentApplied,
	), time.Now().Add(time.Hour).Unix())
	h.records.put(staleUpdate)
	void := h.enqueuePayment(t, payment, accountingsync.SyncOperationVoid)

	h.drain(t)

	calls := h.writer.callsTo("VoidPayment")
	require.Len(t, calls, 1)
	ref, ok := calls[0].doc.(*services.AccountingDocumentRef)
	require.True(t, ok)
	assert.Equal(t, "qb-pay-1", ref.ExternalID)
	assert.Equal(t, accountingsync.SyncObjectCustomerPayment, ref.Kind)
	assert.Equal(t, void.RequestID, ref.RequestID)
	assert.Equal(t, "qb-pay-1", ref.Refs[accountingsync.ExternalRefDocument])
	assert.Equal(t, accountingsync.SyncStatusSynced, h.records.get(void.ID).Status)
	assert.Equal(t, accountingsync.SyncStatusSuperseded, h.records.get(staleUpdate.ID).Status,
		"a pending change to a voided payment is withdrawn")
}

func TestAPaymentSharedWithOtherDocumentsInTheBooksIsNotVoidedAlone(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	customerID, _ := h.mappedCustomer("Acme")
	payment := h.postedPayment(customerID, paymentSpec{amountMinor: 5000})
	created := NewRecordFor(h.conn, services.PaymentSyncRequest(
		payment, accountingsync.SyncOperationCreate, accountingsync.SyncSourceCustomerPaymentPosted,
	), mayTenth)
	require.True(t, created.Link(&accountingsync.SyncLink{
		ExternalID: "qb-pay-301",
		Resolution: "Recorded in QuickBooks Online and applied in Trenova",
		Combined:   true,
	}, mayTenth))
	h.records.put(created)
	void := h.enqueuePayment(t, payment, accountingsync.SyncOperationVoid)

	h.drain(t)

	assert.Empty(t, h.writer.callsTo("VoidPayment"),
		"voiding the provider payment would also undo the credit applied with it")
	blocked := h.records.get(void.ID)
	assert.Equal(t, accountingsync.SyncStatusBlocked, blocked.Status)
	assert.Equal(t, accountingsync.SyncErrorConflict, blocked.ErrorCategory)
	assert.Contains(t, blocked.Resolution, "Edit that payment in QuickBooks Online")
}

func TestAPaymentLinkedAloneIsVoidedNormally(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	customerID, _ := h.mappedCustomer("Acme")
	payment := h.postedPayment(customerID, paymentSpec{amountMinor: 5000})
	created := NewRecordFor(h.conn, services.PaymentSyncRequest(
		payment, accountingsync.SyncOperationCreate, accountingsync.SyncSourceCustomerPaymentPosted,
	), mayTenth)
	require.True(t, created.Link(&accountingsync.SyncLink{ExternalID: "qb-pay-301"}, mayTenth))
	h.records.put(created)
	h.enqueuePayment(t, payment, accountingsync.SyncOperationVoid)

	h.drain(t)

	calls := h.writer.callsTo("VoidPayment")
	require.Len(t, calls, 1)
	ref, ok := calls[0].doc.(*services.AccountingDocumentRef)
	require.True(t, ok)
	assert.Equal(t, "qb-pay-301", ref.ExternalID)
}

func TestVoidOfAPaymentThatNeverReachedTheBooksWithdrawsIt(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	customerID, _ := h.mappedCustomer("Acme")
	payment := h.postedPayment(customerID, paymentSpec{amountMinor: 5000})
	created := NewRecordFor(h.conn, services.PaymentSyncRequest(
		payment, accountingsync.SyncOperationCreate, accountingsync.SyncSourceCustomerPaymentPosted,
	), mayTenth)
	created.MarkFailed(&accountingsync.SyncError{Category: accountingsync.SyncErrorMapping}, mayTenth)
	h.records.put(created)
	void := h.enqueuePayment(t, payment, accountingsync.SyncOperationVoid)

	result := h.drain(t)

	assert.Equal(t, 1, result.Synced)
	assert.Empty(t, h.writer.callsTo("VoidPayment"))
	assert.Equal(t, accountingsync.SyncStatusSuperseded, h.records.get(created.ID).Status)
	voided := h.records.get(void.ID)
	assert.Equal(t, accountingsync.SyncStatusSynced, voided.Status)
	assert.Empty(t, voided.ExternalID)
	assert.NotEmpty(t, voided.Resolution)
}

func TestCreditApplicationLinksTheInvoiceAndTheCreditMemo(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	customerID, _ := h.mappedCustomer("Acme")
	target := h.postedInvoice(customerID, invoiceSpec{number: "INV-1"})
	memo := h.postedInvoice(customerID, invoiceSpec{
		number:   "CM-1",
		billType: billingqueue.BillTypeCreditMemo,
		lines:    []*invoice.InvoiceLine{freightLine("-250")},
	})
	h.syncedInvoiceRecord(target, "qb-inv-1")
	h.syncedInvoiceRecord(memo, "qb-cm-1")
	app := &customerpayment.CreditMemoApplication{
		ID:                  pulid.MustNew("cma_"),
		OrganizationID:      h.tenant.OrgID,
		BusinessUnitID:      h.tenant.BuID,
		CreditMemoInvoiceID: memo.ID,
		InvoiceID:           target.ID,
		AppliedAmountMinor:  25000,
		AccountingDate:      mayTenth,
	}
	h.payments.mu.Lock()
	h.payments.applications[app.ID] = app
	h.payments.mu.Unlock()
	require.NoError(t, h.enqueuer.Enqueue(t.Context(), services.CreditApplicationSyncRequest(
		app, memo.Number, accountingsync.SyncOperationCreate, accountingsync.SyncSourceCreditMemoApplied,
	)))

	h.drain(t)

	calls := h.writer.callsTo("CreateCreditApplication")
	require.Len(t, calls, 1)
	doc, ok := calls[0].doc.(*services.AccountingCreditApplicationDocument)
	require.True(t, ok)
	assert.Equal(t, "qb-cust-Acme", doc.CustomerExternalID)
	assert.Equal(t, "qb-inv-1", doc.InvoiceExternalID)
	assert.Equal(t, "qb-cm-1", doc.CreditMemoExternalID)
	assert.Equal(t, "2026-05-10", doc.TxnDate)
	decimalEqual(t, "250", doc.Amount)
	assert.Equal(t, "Applies CM-1 to INV-1", doc.PrivateNote)
	record := h.only(t, accountingsync.SyncObjectCreditApplication, app.ID, accountingsync.SyncOperationCreate)
	assert.Equal(t, accountingsync.SyncStatusSynced, record.Status)
	assert.Equal(t, "https://qbo.test/CreditApplication/"+record.ExternalID, record.ExternalURL)
}

func TestCustomerUpdatePushesThePartyToTheMappedRecord(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	customerID, mapping := h.mappedCustomer("Acme")
	h.mappings.parties[customerID] = &services.AccountingPartyDraft{DisplayName: "Acme", City: "Reno"}
	require.NoError(t, h.enqueuer.Enqueue(t.Context(), services.CustomerSyncRequest(h.tenant, customerID, "Acme", 3)))

	h.drain(t)

	calls := h.writer.callsTo("UpsertCustomer")
	require.Len(t, calls, 1)
	doc, ok := calls[0].doc.(*services.AccountingCustomerDocument)
	require.True(t, ok)
	assert.Equal(t, "qb-cust-Acme", doc.ExternalID)
	assert.Equal(t, "Reno", doc.Party.City)
	record := h.only(t, accountingsync.SyncObjectCustomer, customerID, accountingsync.SyncOperationUpdate)
	assert.Equal(t, record.RequestID, doc.RequestID)
	assert.Equal(t, accountingsync.SyncStatusSynced, record.Status)
	assert.Equal(t, []string{mapping.ID.String()}, record.MappingIDs)
}
