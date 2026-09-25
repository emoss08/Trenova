package accountinginboundservice

import (
	"errors"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/customerpayment"
	"github.com/emoss08/trenova/internal/core/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/agenteventstest"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var (
	syncEnabledAt = time.Date(2026, time.September, 1, 12, 0, 0, 0, time.UTC)
	pollTime      = time.Date(2026, time.September, 25, 16, 0, 0, 0, time.UTC)
	paidDay       = time.Date(2026, time.September, 24, 0, 0, 0, 0, time.UTC).Unix()
)

type harness struct {
	svc        *Service
	conns      *fakeConnections
	connector  *fakeConnector
	connSvc    *fakeConnService
	records    *fakeRecords
	changes    *fakeChanges
	references *fakeReferences
	mappings   *fakeMappings
	invoices   *fakeInvoices
	payables   *fakePayables
	periods    *fakePeriods
	payments   *fakeCustomerPayments
	carriers   *fakeCarrierPayer
	drivers    *fakeDriverPayer
	audit      *fakeAudit
	watchtower *fakeWatchtower
	events     *agenteventstest.Recorder
	refresher  *fakeRefresher
	tenant     pagination.TenantInfo
	conn       *accountingsync.AccountingConnection
	system     *tenant.User
	now        time.Time
	customerID pulid.ID
}

func newHarness(t *testing.T, policy accountingsync.InboundPaymentPolicy) *harness {
	t.Helper()
	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	enabled := syncEnabledAt.Unix()
	start := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC).Unix()
	conn := &accountingsync.AccountingConnection{
		ID:                   pulid.MustNew("acctc_"),
		OrganizationID:       tenantInfo.OrgID,
		BusinessUnitID:       tenantInfo.BuID,
		IntegrationType:      integration.TypeQuickBooksOnline,
		Status:               accountingsync.ConnectionStatusConnected,
		ExternalRealmID:      "123",
		ExternalHomeCurrency: "USD",
		SetupStep:            accountingsync.SetupStepComplete,
		SyncStartDate:        &start,
		SyncEnabledAt:        &enabled,
		AutoSync:             true,
		InboundPaymentPolicy: policy,
	}

	h := &harness{
		conns:      &fakeConnections{conn: conn},
		connector:  &fakeConnector{},
		records:    &fakeRecords{},
		changes:    &fakeChanges{},
		references: &fakeReferences{},
		mappings:   &fakeMappings{},
		invoices:   &fakeInvoices{rows: map[pulid.ID]*invoice.Invoice{}},
		payables:   &fakePayables{rows: map[pulid.ID]*repositories.PayableSettlement{}},
		periods:    &fakePeriods{status: fiscalperiod.StatusOpen},
		drivers:    &fakeDriverPayer{},
		audit:      &fakeAudit{},
		watchtower: &fakeWatchtower{},
		events:     &agenteventstest.Recorder{},
		refresher:  &fakeRefresher{},
		tenant:     tenantInfo,
		conn:       conn,
		system:     &tenant.User{ID: pulid.MustNew("usr_")},
		now:        pollTime,
		customerID: pulid.MustNew("cus_"),
	}
	h.connSvc = &fakeConnService{connections: h.conns, connector: h.connector}
	h.payments = &fakeCustomerPayments{records: h.records, conn: conn, enqueueAt: pollTime.Unix()}
	h.carriers = &fakeCarrierPayer{records: h.records, conn: conn}

	h.svc = New(Params{
		Logger:            zap.NewNop(),
		DB:                dbtest.NopConnection{},
		Connections:       h.conns,
		ConnectionService: h.connSvc,
		Records:           h.records,
		Changes:           h.changes,
		References:        h.references,
		Mappings:          h.mappings,
		Invoices:          h.invoices,
		Payables:          h.payables,
		FiscalPeriods:     h.periods,
		Organizations:     &fakeOrganizations{timezone: "UTC"},
		Users:             &fakeUsers{system: h.system},
		CustomerPayments:  h.payments,
		CarrierPayer:      h.carriers,
		DriverPayer:       h.drivers,
		AuditService:      h.audit,
		Refresher:         h.refresher,
		Publisher:         h.events,
		Watchtower:        h.watchtower,
	})
	h.svc.now = func() time.Time { return h.now }
	return h
}

func (h *harness) syncedDocument(
	objectType accountingsync.SyncObjectType,
	objectID pulid.ID,
	number, externalID string,
) {
	record := accountingsync.NewAccountingSyncRecord(&accountingsync.NewSyncRecord{
		TenantInfo:   h.tenant,
		ConnectionID: h.conn.ID,
		Key: accountingsync.SyncRecordKey{
			ObjectType: objectType,
			ObjectID:   objectID,
			Operation:  accountingsync.SyncOperationCreate,
		},
		ObjectNumber: number,
		SourceEvent:  accountingsync.SyncSourceInvoicePosted,
		At:           syncEnabledAt.Unix(),
	})
	record.MarkSynced(&accountingsync.SyncResult{ExternalID: externalID}, syncEnabledAt.Unix())
	h.records.add(record)
}

func (h *harness) invoice(number string, totalMinor, appliedMinor int64, externalID string) *invoice.Invoice {
	inv := &invoice.Invoice{
		ID:                 pulid.MustNew("inv_"),
		OrganizationID:     h.tenant.OrgID,
		BusinessUnitID:     h.tenant.BuID,
		CustomerID:         h.customerID,
		Number:             number,
		BillType:           billingqueue.BillTypeInvoice,
		Status:             invoice.StatusPosted,
		CurrencyCode:       "USD",
		TotalAmountMinor:   totalMinor,
		TotalAmount:        money.DecimalFromMinor(totalMinor),
		AppliedAmountMinor: appliedMinor,
		AppliedAmount:      money.DecimalFromMinor(appliedMinor),
	}
	h.invoices.rows[inv.ID] = inv
	h.syncedDocument(accountingsync.SyncObjectInvoice, inv.ID, number, externalID)
	return inv
}

func (h *harness) creditMemo(number string, creditMinor int64, externalID string) *invoice.Invoice {
	memo := h.invoice(number, -creditMinor, 0, "")
	memo.BillType = billingqueue.BillTypeCreditMemo
	h.records.rows = h.records.rows[:len(h.records.rows)-1]
	h.syncedDocument(accountingsync.SyncObjectCreditMemo, memo.ID, number, externalID)
	return memo
}

func amount(value string) decimal.Decimal {
	return decimal.RequireFromString(value)
}

func customerPayment(
	externalID, total string,
	lines ...services.AccountingInboundLine,
) services.AccountingInboundPayment {
	return services.AccountingInboundPayment{
		Kind:             accountingsync.InboundCustomerPayment,
		Operation:        services.AccountingChangeUpsert,
		ExternalID:       externalID,
		Number:           "10442",
		ModifiedAt:       pollTime.Add(-time.Hour).Unix(),
		ModifiedBy:       "J Doe",
		PartyExternalID:  "58",
		PartyName:        "Acme Foods",
		TxnDate:          "2026-09-24",
		CurrencyCode:     "USD",
		Amount:           amount(total),
		MethodExternalID: "2",
		MethodName:       "Check",
		ReferenceNumber:  "10442",
		Lines:            lines,
	}
}

func invoiceLine(externalID, value string) services.AccountingInboundLine {
	return services.AccountingInboundLine{
		DocumentKind:       accountingsync.InboundDocInvoice,
		DocumentExternalID: externalID,
		Amount:             amount(value),
	}
}

func creditLine(externalID, value string) services.AccountingInboundLine {
	return services.AccountingInboundLine{
		DocumentKind:       accountingsync.InboundDocCreditMemo,
		DocumentExternalID: externalID,
		Amount:             amount(value),
	}
}

func (h *harness) poll(t *testing.T, payments ...services.AccountingInboundPayment) {
	t.Helper()
	h.connector.pages = append(h.connector.pages, &services.AccountingChangePage{
		Payments:   payments,
		NextCursor: "next",
	})
	_, err := h.svc.PollChanges(t.Context(), &services.PollAccountingChangesRequest{
		TenantInfo:   h.tenant,
		ConnectionID: h.conn.ID,
	})
	require.NoError(t, err)
}

func (h *harness) evaluate(t *testing.T) *services.AccountingInboundEvaluation {
	t.Helper()
	h.now = h.now.Add(accountingsync.InboundEvaluationSettle + time.Second)
	result, err := h.svc.Evaluate(t.Context(), &services.EvaluateAccountingInboundRequest{
		TenantInfo:   h.tenant,
		ConnectionID: h.conn.ID,
	})
	require.NoError(t, err)
	return result
}

func (h *harness) person() *services.RequestActor {
	return &services.RequestActor{
		PrincipalType:  services.PrincipalTypeUser,
		PrincipalID:    pulid.MustNew("usr_"),
		UserID:         pulid.MustNew("usr_"),
		OrganizationID: h.tenant.OrgID,
		BusinessUnitID: h.tenant.BuID,
	}
}

func TestPollStartsTheFeedWhenSyncWasEnabledAndSavesTheNextCursor(t *testing.T) {
	t.Parallel()
	h := newHarness(t, accountingsync.InboundPaymentsPropose)

	h.poll(t)

	require.Len(t, h.connector.reads, 1)
	assert.Equal(t, h.connector.ChangeCursorAt(syncEnabledAt), h.connector.reads[0].Cursor,
		"a new connection reads changes from when sync began, never history")
	assert.True(t, h.connector.reads[0].Payments)
	assert.True(t, h.connector.reads[0].BillPayments)
	assert.ElementsMatch(t, accountingsync.AllReferenceKinds(), h.connector.reads[0].ReferenceKinds)
	current := h.conns.current()
	assert.Equal(t, "next", current.ChangeCursor)
	require.NotNil(t, current.ChangesReadAt)
	assert.Equal(t, pollTime.Unix(), *current.ChangesReadAt)
}

func TestPollHoldsWhilePausedOrNotSyncing(t *testing.T) {
	t.Parallel()
	h := newHarness(t, accountingsync.InboundPaymentsPropose)
	paused := pollTime.Unix()
	h.conn.PausedAt = &paused

	result, err := h.svc.PollChanges(t.Context(), &services.PollAccountingChangesRequest{
		TenantInfo:   h.tenant,
		ConnectionID: h.conn.ID,
	})
	require.NoError(t, err)
	assert.True(t, result.Held)
	assert.Empty(t, h.connector.reads)
}

func TestPollRecordsAProviderPaymentAndSkipsOnesTrenovaSent(t *testing.T) {
	t.Parallel()
	h := newHarness(t, accountingsync.InboundPaymentsPropose)
	h.invoice("INV-1001", 125_050, 0, "145")
	h.syncedDocument(accountingsync.SyncObjectCustomerPayment, pulid.MustNew("cpay_"), "CP-9", "777")

	h.poll(t,
		customerPayment("301", "1250.50", invoiceLine("145", "1250.50")),
		customerPayment("777", "90", invoiceLine("145", "90")),
	)

	changes := h.changes.all()
	require.Len(t, changes, 1, "a payment Trenova pushed is not brought back")
	change := changes[0]
	assert.Equal(t, "301", change.ExternalID)
	assert.Equal(t, accountingsync.InboundStatusDetected, change.Status)
	assert.Equal(t, int64(125_050), change.AmountMinor)
	assert.Equal(t, paidDay, change.TxnDate)
	assert.Equal(t, "J Doe", change.ProviderModifiedBy)
	require.Len(t, change.Document.Lines, 1)
	assert.Equal(t, int64(125_050), change.Document.Lines[0].AmountMinor)
}

func TestPollRecordsNothingForPaymentsWhenThePolicyIsOff(t *testing.T) {
	t.Parallel()
	h := newHarness(t, accountingsync.InboundPaymentsOff)
	h.invoice("INV-1001", 125_050, 0, "145")

	h.poll(t, customerPayment("301", "1250.50", invoiceLine("145", "1250.50")))

	assert.Empty(t, h.changes.all())
	assert.Equal(t, "next", h.conns.current().ChangeCursor, "the feed still advances")
}

func TestPollAppliesReferenceChangesAndAsksForARefreshWhenTheCursorExpired(t *testing.T) {
	t.Parallel()
	h := newHarness(t, accountingsync.InboundPaymentsPropose)
	h.connector.pages = []*services.AccountingChangePage{{
		References: []services.AccountingChangedReference{
			{Object: &accountingsync.AccountingReferenceObject{
				Kind:       accountingsync.ReferenceKindCustomer,
				ExternalID: "58",
				Name:       "Acme Foods Inc",
			}},
			{
				Object: &accountingsync.AccountingReferenceObject{
					Kind:       accountingsync.ReferenceKindVendor,
					ExternalID: "81",
				},
				Deleted: true,
			},
		},
		NextCursor:    "next",
		CursorExpired: true,
	}}

	result, err := h.svc.PollChanges(t.Context(), &services.PollAccountingChangesRequest{
		TenantInfo:   h.tenant,
		ConnectionID: h.conn.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, 2, result.References)
	require.Len(t, h.references.upserted, 1)
	assert.Equal(t, "Acme Foods Inc", h.references.upserted[0].Name)
	assert.Equal(t, []string{"81"}, h.references.removed[accountingsync.ReferenceKindVendor],
		"a deleted record is marked removed, never overwritten with a blank one")
	assert.Equal(t, 1, h.refresher.requested)
}

func TestPollRecordsAReadFailureAndKeepsTheCursor(t *testing.T) {
	t.Parallel()
	h := newHarness(t, accountingsync.InboundPaymentsPropose)
	h.conn.ChangeCursor = "kept"
	h.connector.readErr = errors.New("token expired")

	_, err := h.svc.PollChanges(t.Context(), &services.PollAccountingChangesRequest{
		TenantInfo:   h.tenant,
		ConnectionID: h.conn.ID,
	})
	require.Error(t, err)
	current := h.conns.current()
	assert.Equal(t, "kept", current.ChangeCursor)
	assert.Equal(t, accountingsync.SyncErrorAuth, current.ChangesErrorCategory)
	assert.Equal(t, "token expired", current.ChangesErrorMessage)
}

func TestAVoidedProviderPaymentSupersedesItsProposal(t *testing.T) {
	t.Parallel()
	h := newHarness(t, accountingsync.InboundPaymentsPropose)
	h.invoice("INV-1001", 125_050, 0, "145")
	h.poll(t, customerPayment("301", "1250.50", invoiceLine("145", "1250.50")))
	h.evaluate(t)

	voided := customerPayment("301", "0")
	voided.Operation = services.AccountingChangeVoid
	h.poll(t, voided)

	change := h.changes.byExternal("301")
	assert.Equal(t, accountingsync.InboundStatusSuperseded, change.Status)
	assert.Equal(t, accountingsync.InboundReasonVoided, change.Reason)
}

func TestEvaluationWaitsForTheChangeToSettleAndForPushesInFlight(t *testing.T) {
	t.Parallel()
	h := newHarness(t, accountingsync.InboundPaymentsPropose)
	h.invoice("INV-1001", 125_050, 0, "145")
	h.poll(t, customerPayment("301", "1250.50", invoiceLine("145", "1250.50")))

	early, err := h.svc.Evaluate(t.Context(), &services.EvaluateAccountingInboundRequest{
		TenantInfo:   h.tenant,
		ConnectionID: h.conn.ID,
	})
	require.NoError(t, err)
	assert.Zero(t, early.Evaluated, "a change younger than the settle time is left alone")

	inFlight := accountingsync.NewAccountingSyncRecord(&accountingsync.NewSyncRecord{
		TenantInfo:   h.tenant,
		ConnectionID: h.conn.ID,
		Key: accountingsync.SyncRecordKey{
			ObjectType: accountingsync.SyncObjectCustomerPayment,
			ObjectID:   pulid.MustNew("cpay_"),
			Operation:  accountingsync.SyncOperationCreate,
		},
		SourceEvent: accountingsync.SyncSourceCustomerPaymentPosted,
		At:          pollTime.Unix(),
	})
	inFlight.Status = accountingsync.SyncStatusInFlight
	h.records.add(inFlight)

	waiting := h.evaluate(t)
	assert.True(t, waiting.Waiting, "a push in flight may be this very payment")
	assert.Zero(t, waiting.Evaluated)
	assert.Equal(t, accountingsync.InboundStatusDetected, h.changes.byExternal("301").Status)
}

func TestProposePolicyProposesAMatchingPaymentAndRaisesTheEvent(t *testing.T) {
	t.Parallel()
	h := newHarness(t, accountingsync.InboundPaymentsPropose)
	inv := h.invoice("INV-1001", 125_050, 0, "145")
	h.poll(t, customerPayment("301", "1250.50", invoiceLine("145", "1250.50")))

	result := h.evaluate(t)

	assert.Equal(t, 1, result.Proposed)
	change := h.changes.byExternal("301")
	assert.Equal(t, accountingsync.InboundStatusProposed, change.Status)
	assert.Equal(t, accountingsync.InboundReasonPolicyPropose, change.Reason)
	assert.Equal(t, h.customerID, change.PartyObjectID)
	assert.Equal(t, inv.ID, change.Document.Lines[0].ObjectID)
	assert.Equal(t, "INV-1001", change.Document.Lines[0].ObjectNumber)
	assert.Empty(t, h.payments.posted, "nothing posts until a person applies it")

	events := h.events.Published()
	require.Len(t, events, 1)
	assert.Equal(t, agent.EventAccountingPaymentProposed, events[0].Kind)
	assert.Equal(t, change.ID, events[0].SubjectID)
}

func TestApplyPolicyPostsThePaymentAsTheSystemUserAndLinksItsRecord(t *testing.T) {
	t.Parallel()
	h := newHarness(t, accountingsync.InboundPaymentsApply)
	inv := h.invoice("INV-1001", 125_050, 0, "145")
	h.poll(t, customerPayment("301", "1250.50", invoiceLine("145", "1250.50")))

	result := h.evaluate(t)

	assert.Equal(t, 1, result.Applied)
	require.Len(t, h.payments.posted, 1)
	posted := h.payments.posted[0]
	assert.Equal(t, h.system.ID, posted.actor.UserID)
	assert.Equal(t, h.customerID, posted.req.CustomerID)
	assert.Equal(t, paidDay, posted.req.PaymentDate)
	assert.Equal(t, paidDay, posted.req.AccountingDate)
	assert.Equal(t, int64(125_050), posted.req.AmountMinor)
	assert.Equal(t, customerpayment.MethodCheck, posted.req.PaymentMethod)
	assert.Equal(t, "10442", posted.req.ReferenceNumber)
	require.Len(t, posted.req.Applications, 1)
	assert.Equal(t, inv.ID, posted.req.Applications[0].InvoiceID)
	assert.Equal(t, int64(125_050), posted.req.Applications[0].AppliedAmountMinor)

	change := h.changes.byExternal("301")
	assert.Equal(t, accountingsync.InboundStatusApplied, change.Status)
	require.Len(t, change.AppliedObjects, 1)
	assert.Equal(t, accountingsync.AppliedCustomerPayment, change.AppliedObjects[0].Type)

	var linked *accountingsync.AccountingSyncRecord
	for _, record := range h.records.all() {
		if record.ObjectID == change.AppliedObjects[0].ID {
			linked = record
		}
	}
	require.NotNil(t, linked)
	assert.Equal(t, accountingsync.SyncStatusSynced, linked.Status,
		"the payment already lives in the books, so its outbound record is linked, not sent")
	assert.Equal(t, "301", linked.ExternalID)
	assert.Equal(t, "https://books.example/CustomerPayment/301", linked.ExternalURL)
	assert.False(t, linked.SharesProviderDocument())
}

func TestPaymentMethodFollowsTheConfirmedMapping(t *testing.T) {
	t.Parallel()
	h := newHarness(t, accountingsync.InboundPaymentsApply)
	h.mappings.rows = []*accountingsync.AccountingMapping{{
		TargetType: accountingsync.TargetPaymentMethod,
		TrenovaKey: string(customerpayment.MethodACH),
		ExternalID: "9",
		State:      accountingsync.MappingStateConfirmed,
	}}
	h.invoice("INV-1001", 10_000, 0, "145")
	paid := customerPayment("301", "100", invoiceLine("145", "100"))
	paid.MethodExternalID = "9"
	paid.MethodName = "Bank transfer"
	h.poll(t, paid)

	h.evaluate(t)

	require.Len(t, h.payments.posted, 1)
	assert.Equal(t, customerpayment.MethodACH, h.payments.posted[0].req.PaymentMethod)
}

func TestCreditMemoLinesApplyTheCreditAndLinkEveryRecordAsCombined(t *testing.T) {
	t.Parallel()
	h := newHarness(t, accountingsync.InboundPaymentsApply)
	inv := h.invoice("INV-1001", 125_050, 0, "145")
	memo := h.creditMemo("CM-7", 7_500, "160")
	h.poll(t, customerPayment("301", "1175.50",
		invoiceLine("145", "1250.50"),
		creditLine("160", "75"),
	))

	h.evaluate(t)

	require.Len(t, h.payments.posted, 1)
	posted := h.payments.posted[0].req
	assert.Equal(t, int64(117_550), posted.AmountMinor)
	require.Len(t, posted.Applications, 1)
	assert.Equal(t, int64(117_550), posted.Applications[0].AppliedAmountMinor,
		"the credit covers part of the invoice, so only the rest is cash")

	require.Len(t, h.payments.credits, 1)
	credit := h.payments.credits[0].req
	assert.Equal(t, memo.ID, credit.CreditMemoID)
	assert.Equal(t, paidDay, credit.AccountingDate)
	require.Len(t, credit.Applications, 1)
	assert.Equal(t, inv.ID, credit.Applications[0].InvoiceID)
	assert.Equal(t, int64(7_500), credit.Applications[0].AppliedAmountMinor)

	change := h.changes.byExternal("301")
	require.Len(t, change.AppliedObjects, 2)
	for _, record := range h.records.all() {
		if record.ObjectType == accountingsync.SyncObjectCustomerPayment ||
			record.ObjectType == accountingsync.SyncObjectCreditApplication {
			assert.Equal(t, accountingsync.SyncStatusSynced, record.Status)
			assert.True(t, record.SharesProviderDocument(),
				"one provider payment covers both, so neither can be undone alone")
		}
	}
}

func TestPaymentsThatDoNotMatchTrenovaAreProposedWithTheReason(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		setup   func(h *harness)
		payment services.AccountingInboundPayment
		reason  accountingsync.InboundChangeReason
		status  accountingsync.InboundChangeStatus
	}{
		{
			name:    "pays nothing Trenova sent",
			setup:   func(*harness) {},
			payment: customerPayment("301", "50", invoiceLine("999", "50")),
			reason:  accountingsync.InboundReasonNotTrenovaDocument,
			status:  accountingsync.InboundStatusIgnored,
		},
		{
			name:  "also pays a document Trenova did not send",
			setup: func(h *harness) { h.invoice("INV-1001", 10_000, 0, "145") },
			payment: customerPayment("301", "150",
				invoiceLine("145", "100"), invoiceLine("999", "50")),
			reason: accountingsync.InboundReasonUnknownDocument,
			status: accountingsync.InboundStatusProposed,
		},
		{
			name:    "pays more than is open",
			setup:   func(h *harness) { h.invoice("INV-1001", 10_000, 6_000, "145") },
			payment: customerPayment("301", "50", invoiceLine("145", "50")),
			reason:  accountingsync.InboundReasonOverpayment,
			status:  accountingsync.InboundStatusProposed,
		},
		{
			name:    "pays an invoice already paid in Trenova",
			setup:   func(h *harness) { h.invoice("INV-1001", 10_000, 10_000, "145") },
			payment: customerPayment("301", "100", invoiceLine("145", "100")),
			reason:  accountingsync.InboundReasonAlreadyPaid,
			status:  accountingsync.InboundStatusProposed,
		},
		{
			name: "is in another currency",
			setup: func(h *harness) {
				h.invoice("INV-1001", 10_000, 0, "145").CurrencyCode = "CAD"
			},
			payment: customerPayment("301", "100", invoiceLine("145", "100")),
			reason:  accountingsync.InboundReasonCurrencyMismatch,
			status:  accountingsync.InboundStatusProposed,
		},
		{
			name: "pays two Trenova customers",
			setup: func(h *harness) {
				h.invoice("INV-1001", 10_000, 0, "145")
				h.invoice("INV-1002", 10_000, 0, "146").CustomerID = pulid.MustNew("cus_")
			},
			payment: customerPayment("301", "200",
				invoiceLine("145", "100"), invoiceLine("146", "100")),
			reason: accountingsync.InboundReasonPartyMismatch,
			status: accountingsync.InboundStatusProposed,
		},
		{
			name: "falls in a locked period",
			setup: func(h *harness) {
				h.invoice("INV-1001", 10_000, 0, "145")
				h.periods.status = fiscalperiod.StatusLocked
			},
			payment: customerPayment("301", "100", invoiceLine("145", "100")),
			reason:  accountingsync.InboundReasonPeriodNotOpen,
			status:  accountingsync.InboundStatusProposed,
		},
		{
			name: "has no fiscal period",
			setup: func(h *harness) {
				h.invoice("INV-1001", 10_000, 0, "145")
				h.periods.none = true
			},
			payment: customerPayment("301", "100", invoiceLine("145", "100")),
			reason:  accountingsync.InboundReasonPeriodNotOpen,
			status:  accountingsync.InboundStatusProposed,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			h := newHarness(t, accountingsync.InboundPaymentsApply)
			tc.setup(h)
			h.poll(t, tc.payment)

			h.evaluate(t)

			change := h.changes.byExternal(tc.payment.ExternalID)
			require.NotNil(t, change)
			assert.Equal(t, tc.status, change.Status)
			assert.Equal(t, tc.reason, change.Reason)
			assert.NotEmpty(t, change.Resolution)
			assert.Empty(t, h.payments.posted, "the Apply policy never applies a mismatch")
		})
	}
}

func TestAPersonAppliesAProposalAsThemselves(t *testing.T) {
	t.Parallel()
	h := newHarness(t, accountingsync.InboundPaymentsPropose)
	h.invoice("INV-1001", 125_050, 0, "145")
	h.poll(t, customerPayment("301", "1250.50", invoiceLine("145", "1250.50")))
	h.evaluate(t)
	change := h.changes.byExternal("301")
	actor := h.person()

	applied, err := h.svc.Apply(t.Context(), &services.DecideAccountingInboundChangeRequest{
		TenantInfo: h.tenant,
		ID:         change.ID,
	}, actor)
	require.NoError(t, err)
	assert.Equal(t, accountingsync.InboundStatusApplied, applied.Status)
	assert.Equal(t, actor.UserID, applied.DecidedByID)
	require.Len(t, h.payments.posted, 1)
	assert.Equal(t, actor.UserID, h.payments.posted[0].actor.UserID)
	require.Len(t, h.audit.entries, 1)

	_, err = h.svc.Apply(t.Context(), &services.DecideAccountingInboundChangeRequest{
		TenantInfo: h.tenant,
		ID:         change.ID,
	}, actor)
	require.Error(t, err, "a change applies once")
	assert.Len(t, h.payments.posted, 1)
}

func TestApplyingAMismatchIsRefusedWithItsReason(t *testing.T) {
	t.Parallel()
	h := newHarness(t, accountingsync.InboundPaymentsPropose)
	h.invoice("INV-1001", 10_000, 6_000, "145")
	h.poll(t, customerPayment("301", "50", invoiceLine("145", "50")))
	h.evaluate(t)
	change := h.changes.byExternal("301")

	_, err := h.svc.Apply(t.Context(), &services.DecideAccountingInboundChangeRequest{
		TenantInfo: h.tenant,
		ID:         change.ID,
	}, h.person())
	require.Error(t, err)
	assert.True(t, errortypes.IsError(err))
	assert.Empty(t, h.payments.posted)
}

func TestAProposalBlockedByAClosedPeriodAppliesOnceItReopens(t *testing.T) {
	t.Parallel()
	h := newHarness(t, accountingsync.InboundPaymentsApply)
	h.invoice("INV-1001", 10_000, 0, "145")
	h.periods.status = fiscalperiod.StatusClosed
	h.poll(t, customerPayment("301", "100", invoiceLine("145", "100")))
	h.evaluate(t)
	change := h.changes.byExternal("301")
	require.Equal(t, accountingsync.InboundReasonPeriodNotOpen, change.Reason)

	h.periods.status = fiscalperiod.StatusOpen
	applied, err := h.svc.Apply(t.Context(), &services.DecideAccountingInboundChangeRequest{
		TenantInfo: h.tenant,
		ID:         change.ID,
	}, h.person())
	require.NoError(t, err)
	assert.Equal(t, accountingsync.InboundStatusApplied, applied.Status)
}

func TestAnApplyThePaymentServiceRefusesBecomesAProposal(t *testing.T) {
	t.Parallel()
	h := newHarness(t, accountingsync.InboundPaymentsApply)
	h.invoice("INV-1001", 10_000, 0, "145")
	h.payments.postErr = errortypes.NewValidationError(
		"accountingControl",
		errortypes.ErrRequired,
		"A default cash account must be configured",
	)
	h.poll(t, customerPayment("301", "100", invoiceLine("145", "100")))

	h.evaluate(t)

	change := h.changes.byExternal("301")
	assert.Equal(t, accountingsync.InboundStatusProposed, change.Status)
	assert.Equal(t, accountingsync.InboundReasonApplyFailed, change.Reason)
	assert.Contains(t, change.Resolution, "default cash account")
}

func TestIgnoringNeedsANote(t *testing.T) {
	t.Parallel()
	h := newHarness(t, accountingsync.InboundPaymentsPropose)
	h.invoice("INV-1001", 10_000, 0, "145")
	h.poll(t, customerPayment("301", "100", invoiceLine("145", "100")))
	h.evaluate(t)
	change := h.changes.byExternal("301")

	_, err := h.svc.Ignore(t.Context(), &services.DecideAccountingInboundChangeRequest{
		TenantInfo: h.tenant,
		ID:         change.ID,
	}, h.person())
	require.Error(t, err)

	ignored, err := h.svc.Ignore(t.Context(), &services.DecideAccountingInboundChangeRequest{
		TenantInfo: h.tenant,
		ID:         change.ID,
		Note:       "Entered in Trenova by hand already",
	}, h.person())
	require.NoError(t, err)
	assert.Equal(t, accountingsync.InboundStatusIgnored, ignored.Status)
	assert.Equal(t, "Entered in Trenova by hand already", ignored.Note)
}

func (h *harness) carrierBill(number string, netMinor int64, externalID string) pulid.ID {
	id := pulid.MustNew("carstl_")
	posted := syncEnabledAt.Unix()
	h.payables.rows[id] = &repositories.PayableSettlement{
		Kind:         repositories.PayableCarrier,
		ID:           id,
		Number:       number,
		PartyID:      pulid.MustNew("car_"),
		PostedAt:     &posted,
		NetMinor:     netMinor,
		CurrencyCode: "USD",
	}
	h.syncedDocument(accountingsync.SyncObjectCarrierBill, id, number, externalID)
	return id
}

func billPayment(externalID, total string, billExternalID, lineAmount string) services.AccountingInboundPayment {
	return services.AccountingInboundPayment{
		Kind:            accountingsync.InboundBillPayment,
		Operation:       services.AccountingChangeUpsert,
		ExternalID:      externalID,
		Number:          "5521",
		PartyExternalID: "77",
		TxnDate:         "2026-09-24",
		CurrencyCode:    "USD",
		Amount:          amount(total),
		MethodName:      "Check",
		ReferenceNumber: "5521",
		Lines: []services.AccountingInboundLine{{
			DocumentKind:       accountingsync.InboundDocBill,
			DocumentExternalID: billExternalID,
			Amount:             amount(lineAmount),
		}},
	}
}

func TestABillPaidInFullInTheBooksMarksTheSettlementPaidOnThatDay(t *testing.T) {
	t.Parallel()
	h := newHarness(t, accountingsync.InboundPaymentsApply)
	settlementID := h.carrierBill("CS-1042", 98_000, "390")
	h.poll(t, billPayment("410", "980", "390", "980"))

	h.evaluate(t)

	require.Len(t, h.carriers.paid, 1)
	paid := h.carriers.paid[0].req
	assert.Equal(t, settlementID, paid.SettlementID)
	assert.Equal(t, paidDay, paid.PaidAt)
	assert.Equal(t, "Check", paid.PaymentMethod)
	assert.Equal(t, "5521", paid.PaymentReference)

	change := h.changes.byExternal("410")
	assert.Equal(t, accountingsync.InboundStatusApplied, change.Status)
	for _, record := range h.records.all() {
		if record.ObjectType == accountingsync.SyncObjectCarrierBillPay {
			assert.Equal(t, accountingsync.SyncStatusSynced, record.Status)
			assert.Equal(t, "410", record.ExternalID)
		}
	}
}

func TestAPartialBillPaymentIsProposed(t *testing.T) {
	t.Parallel()
	h := newHarness(t, accountingsync.InboundPaymentsApply)
	h.carrierBill("CS-1042", 98_000, "390")
	h.poll(t, billPayment("410", "500", "390", "500"))

	h.evaluate(t)

	change := h.changes.byExternal("410")
	assert.Equal(t, accountingsync.InboundReasonPartialBillPayment, change.Reason)
	assert.Empty(t, h.carriers.paid)
}

func TestABillAlreadyPaidInTrenovaIsProposed(t *testing.T) {
	t.Parallel()
	h := newHarness(t, accountingsync.InboundPaymentsApply)
	id := h.carrierBill("CS-1042", 98_000, "390")
	paidAt := syncEnabledAt.Unix()
	h.payables.rows[id].PaidAt = &paidAt
	h.poll(t, billPayment("410", "980", "390", "980"))

	h.evaluate(t)

	assert.Equal(t, accountingsync.InboundReasonAlreadyPaid, h.changes.byExternal("410").Reason)
	assert.Empty(t, h.carriers.paid)
}

func TestProposalsOlderThanADayRaiseAWatchtowerItemPerReason(t *testing.T) {
	t.Parallel()
	h := newHarness(t, accountingsync.InboundPaymentsPropose)
	h.invoice("INV-1001", 10_000, 0, "145")
	h.poll(t, customerPayment("301", "100", invoiceLine("145", "100")))
	h.evaluate(t)
	assert.Empty(t, h.watchtower.open, "a fresh proposal is not yet an item")

	h.now = h.now.Add(25 * time.Hour)
	h.poll(t)
	h.evaluate(t)
	h.svc.refreshAttention(t.Context(), h.conns.current())

	require.Len(t, h.watchtower.open, 1)
	for _, item := range h.watchtower.open {
		assert.Equal(t, agent.SubjectAccountingInbound, item.SubjectType)
		assert.Contains(t, item.Title, "1 payment recorded in QuickBooks Online")
	}
}
