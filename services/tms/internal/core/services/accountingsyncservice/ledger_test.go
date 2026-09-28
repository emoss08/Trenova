package accountingsyncservice

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/domain/journalentry"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type ledgerAccounts struct {
	ar      pulid.ID
	revenue pulid.ID
	cash    pulid.ID
	ap      pulid.ID
}

func newLedgerAccounts() ledgerAccounts {
	return ledgerAccounts{
		ar:      pulid.MustNew("gla_"),
		revenue: pulid.MustNew("gla_"),
		cash:    pulid.MustNew("gla_"),
		ap:      pulid.MustNew("gla_"),
	}
}

type ledgerMappings struct {
	ar       *accountingsync.AccountingMapping
	revenue  *accountingsync.AccountingMapping
	customer *accountingsync.AccountingMapping
}

func (h *harness) ledgerMode(granularity accountingsync.LedgerGranularity) {
	h.updateConnection(func(conn *accountingsync.AccountingConnection) {
		conn.SyncMode = accountingsync.SyncModeLedger
		conn.LedgerGranularity = granularity
	})
	h.conn = h.connections.get(h.conn.ID)
}

func (h *harness) mapLedger(accts ledgerAccounts, customerID pulid.ID) ledgerMappings {
	h.controls.set(func(control *tenant.AccountingControl) {
		control.DefaultRevenueAccountID = accts.revenue
		control.DefaultAPAccountID = accts.ap
	})
	h.references.account("qb-ar", accountingsync.AccountTypeReceivable)
	h.references.account("qb-income", "Income")
	h.references.account("qb-bank", "Bank")
	h.references.account("qb-ap", accountingsync.AccountTypePayable)
	return ledgerMappings{
		ar:      h.confirm(glTarget(accts.ar, "GL account 1100 Accounts receivable"), "qb-ar"),
		revenue: h.confirm(roleTarget(accountingsync.AccountRoleRevenue, "Revenue"), "qb-income"),
		customer: h.confirm(
			customerTarget(customerID, "Acme Freight"),
			"qb-cust-58",
		),
	}
}

func arLine(accts ledgerAccounts, debit, credit int64) repositories.LedgerJournalLine {
	return repositories.LedgerJournalLine{
		AccountID:   accts.ar,
		AccountCode: "1100",
		AccountName: "Accounts receivable",
		DebitMinor:  debit,
		CreditMinor: credit,
	}
}

func revenueLine(accts ledgerAccounts, debit, credit int64) repositories.LedgerJournalLine {
	return repositories.LedgerJournalLine{
		AccountID:   accts.revenue,
		AccountCode: "4000",
		AccountName: "Freight revenue",
		DebitMinor:  debit,
		CreditMinor: credit,
		Description: "Freight revenue",
	}
}

func cashLine(accts ledgerAccounts, debit, credit int64) repositories.LedgerJournalLine {
	return repositories.LedgerJournalLine{
		AccountID:   accts.cash,
		AccountCode: "1000",
		AccountName: "Operating cash",
		DebitMinor:  debit,
		CreditMinor: credit,
	}
}

func invoiceJournal(
	accts ledgerAccounts,
	customerID pulid.ID,
	number string,
	date int64,
	amount int64,
) *repositories.LedgerJournal {
	return &repositories.LedgerJournal{
		ID:                   pulid.MustNew("je_"),
		EntryNumber:          number,
		EntryType:            journalentry.EntryTypeStandard.String(),
		Description:          "Invoice INV-" + number,
		AccountingDate:       date,
		PostedAt:             date + 60,
		SourceObjectType:     "Invoice",
		SourceDocumentNumber: "INV-" + number,
		Party: repositories.LedgerParty{
			Kind: repositories.LedgerPartyCustomer,
			ID:   customerID,
			Name: "Acme Freight",
		},
		Lines: []repositories.LedgerJournalLine{
			arLine(accts, amount, 0),
			revenueLine(accts, 0, amount),
		},
	}
}

func (h *harness) postJournal(
	t *testing.T,
	journal *repositories.LedgerJournal,
) {
	t.Helper()
	h.ledger.add(journal)
	require.NoError(t, h.enqueuer.EnqueueJournal(t.Context(), &services.AccountingJournalPosted{
		TenantInfo:     h.tenant,
		EntryID:        journal.ID,
		EntryNumber:    journal.EntryNumber,
		EntryType:      journalentry.EntryType(journal.EntryType),
		AccountingDate: journal.AccountingDate,
	}))
}

func onlyJournalDoc(t *testing.T, h *harness, method string) *services.AccountingJournalDocument {
	t.Helper()
	calls := h.writer.callsTo(method)
	require.Len(t, calls, 1)
	doc, ok := calls[0].doc.(*services.AccountingJournalDocument)
	require.True(t, ok)
	return doc
}

func TestEnqueueJournalQueuesOneRecordPerEntryForADetailedLedger(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.ledgerMode(accountingsync.LedgerDetailed)
	accts := newLedgerAccounts()
	journal := invoiceJournal(accts, pulid.MustNew("cus_"), "JE-1001", aprilTenth, 150_000)

	h.postJournal(t, journal)

	record := h.only(t, accountingsync.SyncObjectJournalEntry, journal.ID, accountingsync.SyncOperationCreate)
	assert.Equal(t, "JE-1001", record.ObjectNumber)
	assert.Equal(t, accountingsync.SyncSourceJournalPosted, record.SourceEvent)
	require.NotNil(t, record.DocumentDate)
	assert.Equal(t, aprilTenth, *record.DocumentDate)
	assert.Len(t, h.records.all(), 1)
	assert.Equal(t, []pulid.ID{h.conn.ID}, h.dispatcher.kicked())
}

func TestEnqueueJournalLeavesWhatTheLedgerDoesNotSend(t *testing.T) {
	t.Parallel()

	accts := newLedgerAccounts()
	cases := []struct {
		name   string
		mode   accountingsync.SyncMode
		mutate func(*repositories.LedgerJournal)
	}{
		{name: "a document connection", mode: accountingsync.SyncModeDocument},
		{
			name: "a closing entry",
			mode: accountingsync.SyncModeLedger,
			mutate: func(j *repositories.LedgerJournal) {
				j.EntryType = journalentry.EntryTypeClosing.String()
			},
		},
		{
			name: "an opening entry",
			mode: accountingsync.SyncModeLedger,
			mutate: func(j *repositories.LedgerJournal) {
				j.EntryType = journalentry.EntryTypeOpening.String()
			},
		},
		{
			name: "an entry dated before the start date",
			mode: accountingsync.SyncModeLedger,
			mutate: func(j *repositories.LedgerJournal) {
				j.AccountingDate = syncStartDate - 86_400
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			h := newHarness(t)
			h.updateConnection(func(conn *accountingsync.AccountingConnection) {
				conn.SyncMode = tc.mode
				if tc.mode == accountingsync.SyncModeLedger {
					conn.LedgerGranularity = accountingsync.LedgerDetailed
				}
			})
			journal := invoiceJournal(accts, pulid.MustNew("cus_"), "JE-2", aprilTenth, 100)
			if tc.mutate != nil {
				tc.mutate(journal)
			}

			h.postJournal(t, journal)

			assert.Empty(t, h.records.all())
		})
	}
}

func TestDocumentsAreNotQueuedForALedgerConnection(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.ledgerMode(accountingsync.LedgerDetailed)
	customerID, _ := h.mappedCustomer("Acme Freight")
	inv := h.postedInvoice(customerID, invoiceSpec{})

	require.NoError(t, h.enqueuer.Enqueue(
		t.Context(),
		services.InvoiceSyncRequest(inv, accountingsync.PostedSourceEvent(inv.BillType)),
	))

	assert.Empty(t, h.records.all())
}

func TestDailySummaryQueuesTheDayAndAnUpdateForLaterEntries(t *testing.T) {
	t.Parallel()

	h := newHarness(t, withTimezone("America/Chicago"))
	h.ledgerMode(accountingsync.LedgerDailySummary)
	chicago, err := time.LoadLocation("America/Chicago")
	require.NoError(t, err)
	evening := time.Date(2026, time.April, 10, 21, 0, 0, 0, chicago).Unix()
	accts := newLedgerAccounts()

	first := invoiceJournal(accts, pulid.MustNew("cus_"), "JE-1", evening, 100)
	h.postJournal(t, first)

	dayID := pulid.ID("jday_20260410")
	create := h.only(t, accountingsync.SyncObjectJournalSummary, dayID, accountingsync.SyncOperationCreate)
	assert.Equal(t, "2026-04-10", create.ObjectNumber)
	require.NotNil(t, create.DocumentDate)
	assert.Equal(t, time.Date(2026, time.April, 10, 0, 0, 0, 0, chicago).Unix(), *create.DocumentDate)
	assert.Empty(t, h.records.find(accountingsync.SyncObjectJournalSummary, dayID, accountingsync.SyncOperationUpdate))
	assert.Empty(t, h.records.find(accountingsync.SyncObjectJournalEntry, first.ID, accountingsync.SyncOperationCreate))

	second := invoiceJournal(accts, pulid.MustNew("cus_"), "JE-2", evening+60, 200)
	h.postJournal(t, second)
	updates := h.records.find(accountingsync.SyncObjectJournalSummary, dayID, accountingsync.SyncOperationUpdate)
	require.Len(t, updates, 1)
	assert.Greater(t, updates[0].Revision, create.Revision)

	third := invoiceJournal(accts, pulid.MustNew("cus_"), "JE-3", evening+120, 300)
	h.postJournal(t, third)
	updates = h.records.find(accountingsync.SyncObjectJournalSummary, dayID, accountingsync.SyncOperationUpdate)
	require.Len(t, updates, 2)
	statuses := []accountingsync.SyncStatus{updates[0].Status, updates[1].Status}
	assert.ElementsMatch(t, []accountingsync.SyncStatus{
		accountingsync.SyncStatusSuperseded,
		accountingsync.SyncStatusQueued,
	}, statuses, "a newer update for the day retires the one still waiting")
}

func TestDetailedJournalEntryIsSentWithAccountsAndTheCustomer(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.ledgerMode(accountingsync.LedgerDetailed)
	accts := newLedgerAccounts()
	customerID := pulid.MustNew("cus_")
	mapped := h.mapLedger(accts, customerID)
	journal := invoiceJournal(accts, customerID, "JE-1001", aprilTenth, 150_000)
	h.postJournal(t, journal)
	record := h.only(t, accountingsync.SyncObjectJournalEntry, journal.ID, accountingsync.SyncOperationCreate)

	result := h.drain(t)

	assert.Equal(t, 1, result.Synced)
	doc := onlyJournalDoc(t, h, "CreateJournalEntry")
	assert.Equal(t, record.RequestID, doc.RequestID)
	assert.Equal(t, accountingsync.SyncObjectJournalEntry, doc.Kind)
	assert.Equal(t, "JE-1001", doc.DocNumber)
	assert.Equal(t, "2026-04-10", doc.TxnDate)
	assert.Equal(t, "USD", doc.CurrencyCode)
	assert.Equal(t, "Trenova journal entry JE-1001 (INV-JE-1001): Invoice INV-JE-1001", doc.PrivateNote)
	require.Len(t, doc.Lines, 2)
	assert.Equal(t, services.AccountingJournalDebit, doc.Lines[0].Posting)
	assert.Equal(t, "qb-ar", doc.Lines[0].AccountExternalID)
	assert.Equal(t, "1500", doc.Lines[0].Amount.String())
	assert.Equal(t, "Invoice INV-JE-1001", doc.Lines[0].Description)
	assert.Equal(t, services.AccountingJournalCustomer, doc.Lines[0].PartyKind)
	assert.Equal(t, "qb-cust-58", doc.Lines[0].PartyExternalID)
	assert.Equal(t, services.AccountingJournalCredit, doc.Lines[1].Posting)
	assert.Equal(t, "qb-income", doc.Lines[1].AccountExternalID, "the revenue account falls back to its role")
	assert.Equal(t, "1500", doc.Lines[1].Amount.String())
	assert.Empty(t, doc.Lines[1].PartyKind, "only receivable and payable lines name a party")
	assert.Empty(t, doc.Lines[1].PartyExternalID)

	synced := h.records.get(record.ID)
	assert.Equal(t, accountingsync.SyncStatusSynced, synced.Status)
	assert.Equal(t, "qb-CreateJournalEntry-1", synced.ExternalID)
	assert.Equal(t, "https://qbo.test/JournalEntry/qb-CreateJournalEntry-1", synced.ExternalURL)
	assert.ElementsMatch(t, []string{
		mapped.ar.ID.String(),
		mapped.revenue.ID.String(),
		mapped.customer.ID.String(),
	}, synced.MappingIDs)
}

func TestJournalLinePartyWinsOverTheEntrysSource(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.ledgerMode(accountingsync.LedgerDetailed)
	accts := newLedgerAccounts()
	customerID := pulid.MustNew("cus_")
	h.mapLedger(accts, customerID)
	other := pulid.MustNew("cus_")
	h.confirm(customerTarget(other, "Other Shipper"), "qb-cust-77")
	journal := invoiceJournal(accts, customerID, "JE-1002", aprilTenth, 5_000)
	journal.Lines[0].Party = repositories.LedgerParty{Kind: repositories.LedgerPartyCustomer, ID: other}
	h.postJournal(t, journal)

	h.drain(t)

	doc := onlyJournalDoc(t, h, "CreateJournalEntry")
	assert.Equal(t, "qb-cust-77", doc.Lines[0].PartyExternalID)
}

func TestJournalEntryBlocksWhenAReceivableLineHasNoCustomer(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.ledgerMode(accountingsync.LedgerDetailed)
	accts := newLedgerAccounts()
	h.mapLedger(accts, pulid.MustNew("cus_"))
	journal := invoiceJournal(accts, pulid.MustNew("cus_"), "JE-1003", aprilTenth, 5_000)
	journal.Party = repositories.LedgerParty{}
	h.postJournal(t, journal)
	record := h.only(t, accountingsync.SyncObjectJournalEntry, journal.ID, accountingsync.SyncOperationCreate)

	h.drain(t)

	blocked := h.records.get(record.ID)
	assert.Equal(t, accountingsync.SyncStatusBlocked, blocked.Status)
	assert.Equal(t, accountingsync.SyncErrorMapping, blocked.ErrorCategory)
	assert.Contains(t, blocked.ErrorMessage, "GL account 1100 Accounts receivable")
	assert.Contains(t, blocked.ErrorMessage, "needs a customer")
	assert.Empty(t, h.writer.callsTo("CreateJournalEntry"))
}

func TestJournalEntryBlocksOnAPartyOfTheWrongKind(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.ledgerMode(accountingsync.LedgerDetailed)
	accts := newLedgerAccounts()
	h.mapLedger(accts, pulid.MustNew("cus_"))
	journal := invoiceJournal(accts, pulid.MustNew("cus_"), "JE-1004", aprilTenth, 5_000)
	journal.Party = repositories.LedgerParty{
		Kind: repositories.LedgerPartyCarrier,
		ID:   pulid.MustNew("car_"),
		Name: "Swift Haul",
	}
	h.postJournal(t, journal)
	record := h.only(t, accountingsync.SyncObjectJournalEntry, journal.ID, accountingsync.SyncOperationCreate)

	h.drain(t)

	blocked := h.records.get(record.ID)
	assert.Equal(t, accountingsync.SyncStatusBlocked, blocked.Status)
	assert.Contains(t, blocked.ErrorMessage, "belongs to carrier Swift Haul")
}

func TestJournalEntryWaitsForAnUnmappedCustomerToBeCreated(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.ledgerMode(accountingsync.LedgerDetailed)
	accts := newLedgerAccounts()
	h.mapLedger(accts, pulid.MustNew("cus_"))
	newcomer := pulid.MustNew("cus_")
	journal := invoiceJournal(accts, newcomer, "JE-1005", aprilTenth, 5_000)
	h.postJournal(t, journal)
	record := h.only(t, accountingsync.SyncObjectJournalEntry, journal.ID, accountingsync.SyncOperationCreate)

	h.drain(t)

	waiting := h.records.get(record.ID)
	assert.NotEqual(t, accountingsync.SyncStatusSynced, waiting.Status)
	dependency := h.only(t, accountingsync.SyncObjectCustomer, newcomer, accountingsync.SyncOperationCreate)
	assert.Equal(t, dependency.ID, waiting.DependsOnRecordID)
	assert.Empty(t, h.writer.callsTo("CreateJournalEntry"))
}

func TestJournalEntryBlocksOnAnUnmappedAccount(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.ledgerMode(accountingsync.LedgerDetailed)
	accts := newLedgerAccounts()
	customerID := pulid.MustNew("cus_")
	h.mapLedger(accts, customerID)
	journal := invoiceJournal(accts, customerID, "JE-1006", aprilTenth, 5_000)
	journal.Lines[1] = cashLine(accts, 0, 5_000)
	h.postJournal(t, journal)
	record := h.only(t, accountingsync.SyncObjectJournalEntry, journal.ID, accountingsync.SyncOperationCreate)

	h.drain(t)

	blocked := h.records.get(record.ID)
	assert.Equal(t, accountingsync.SyncStatusBlocked, blocked.Status)
	assert.Equal(t, accountingsync.SyncErrorMapping, blocked.ErrorCategory)
	assert.Equal(t, "GL account 1000 Operating cash is not mapped", blocked.ErrorMessage)
}

func TestJournalEntryThatDoesNotBalanceIsBlocked(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.ledgerMode(accountingsync.LedgerDetailed)
	accts := newLedgerAccounts()
	customerID := pulid.MustNew("cus_")
	h.mapLedger(accts, customerID)
	journal := invoiceJournal(accts, customerID, "JE-1007", aprilTenth, 5_000)
	journal.Lines[1].CreditMinor = 4_900
	h.postJournal(t, journal)
	record := h.only(t, accountingsync.SyncObjectJournalEntry, journal.ID, accountingsync.SyncOperationCreate)

	h.drain(t)

	blocked := h.records.get(record.ID)
	assert.Equal(t, accountingsync.SyncStatusBlocked, blocked.Status)
	assert.Equal(t, accountingsync.SyncErrorValidation, blocked.ErrorCategory)
	assert.Contains(t, blocked.ErrorMessage, "does not balance")
	assert.Contains(t, blocked.ErrorMessage, "1.00")
}

func TestDailySummarySumsTheDayPerAccountAndCustomer(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.ledgerMode(accountingsync.LedgerDailySummary)
	accts := newLedgerAccounts()
	acme := pulid.MustNew("cus_")
	h.mapLedger(accts, acme)
	globex := pulid.MustNew("cus_")
	h.confirm(customerTarget(globex, "Globex"), "qb-cust-90")

	h.postJournal(t, invoiceJournal(accts, acme, "JE-1", aprilTenth+3_600, 100_000))
	h.postJournal(t, invoiceJournal(accts, acme, "JE-2", aprilTenth+7_200, 50_000))
	globexJournal := invoiceJournal(accts, globex, "JE-3", aprilTenth+9_000, 20_000)
	globexJournal.Party.Name = "Globex"
	h.postJournal(t, globexJournal)
	h.ledger.add(invoiceJournal(accts, acme, "JE-OTHER-DAY", aprilTenth+86_400, 999_00))

	result := h.drain(t)

	assert.Equal(t, 2, result.Synced, "the day and its follow-up update")
	doc := onlyJournalDoc(t, h, "CreateJournalEntry")
	assert.Equal(t, "JE 2026-04-10", doc.DocNumber)
	assert.Equal(t, "2026-04-10", doc.TxnDate)
	assert.Equal(t, "Trenova journal entries posted for 2026-04-10", doc.PrivateNote)
	type want struct {
		posting services.AccountingJournalPosting
		account string
		amount  string
		party   string
	}
	got := make([]want, 0, len(doc.Lines))
	for _, line := range doc.Lines {
		got = append(got, want{
			posting: line.Posting,
			account: line.AccountExternalID,
			amount:  line.Amount.String(),
			party:   line.PartyExternalID,
		})
	}
	assert.Equal(t, []want{
		{posting: services.AccountingJournalDebit, account: "qb-ar", amount: "1500", party: "qb-cust-58"},
		{posting: services.AccountingJournalCredit, account: "qb-income", amount: "1700"},
		{posting: services.AccountingJournalDebit, account: "qb-ar", amount: "200", party: "qb-cust-90"},
	}, got)
	assert.Equal(t, "1100 Accounts receivable, Acme Freight", doc.Lines[0].Description)

	update := onlyJournalDoc(t, h, "UpdateJournalEntry")
	assert.Equal(t, "qb-CreateJournalEntry-1", update.ExternalID,
		"the follow-up rewrites the entry the day created")
}

func TestDailySummaryUpdateWaitsForTheDaysFirstSend(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.ledgerMode(accountingsync.LedgerDailySummary)
	accts := newLedgerAccounts()
	customerID := pulid.MustNew("cus_")
	h.mapLedger(accts, customerID)
	h.postJournal(t, invoiceJournal(accts, customerID, "JE-1", aprilTenth, 100))
	h.postJournal(t, invoiceJournal(accts, customerID, "JE-2", aprilTenth+60, 100))
	dayID := pulid.ID("jday_20260410")
	create := h.only(t, accountingsync.SyncObjectJournalSummary, dayID, accountingsync.SyncOperationCreate)
	update := h.only(t, accountingsync.SyncObjectJournalSummary, dayID, accountingsync.SyncOperationUpdate)
	h.records.postpone(create.ID)

	h.drain(t)

	waiting := h.records.get(update.ID)
	assert.Equal(t, create.ID, waiting.DependsOnRecordID)
	assert.Empty(t, h.writer.callsTo("UpdateJournalEntry"))
	assert.Empty(t, h.writer.callsTo("CreateJournalEntry"))
}

func TestDailySummaryThatNowNetsToZeroDeletesTheProviderEntry(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.ledgerMode(accountingsync.LedgerDailySummary)
	accts := newLedgerAccounts()
	customerID := pulid.MustNew("cus_")
	h.mapLedger(accts, customerID)
	original := invoiceJournal(accts, customerID, "JE-1", aprilTenth, 5_000)
	h.postJournal(t, original)
	h.drain(t)
	require.Len(t, h.writer.callsTo("CreateJournalEntry"), 1)

	reversal := invoiceJournal(accts, customerID, "JE-2", aprilTenth+60, 5_000)
	reversal.IsReversal = true
	reversal.ReversalOfNumber = "JE-1"
	reversal.Lines = []repositories.LedgerJournalLine{
		arLine(accts, 0, 5_000),
		revenueLine(accts, 5_000, 0),
	}
	h.postJournal(t, reversal)
	dayID := pulid.ID("jday_20260410")
	update := h.only(t, accountingsync.SyncObjectJournalSummary, dayID, accountingsync.SyncOperationUpdate)

	h.drain(t)

	calls := h.writer.callsTo("DeleteJournalEntry")
	require.Len(t, calls, 1)
	ref, ok := calls[0].doc.(*services.AccountingDocumentRef)
	require.True(t, ok)
	assert.Equal(t, "qb-CreateJournalEntry-1", ref.ExternalID)
	assert.Equal(t, update.RequestID, ref.RequestID)
	done := h.records.get(update.ID)
	assert.Equal(t, accountingsync.SyncStatusSynced, done.Status)
	assert.Empty(t, done.ExternalID)
	assert.Contains(t, done.Resolution, "nets to zero")
	assert.Empty(t, h.writer.callsTo("UpdateJournalEntry"))

	h.postJournal(t, invoiceJournal(accts, customerID, "JE-3", aprilTenth+120, 700))
	h.drain(t)

	created := h.writer.callsTo("CreateJournalEntry")
	require.Len(t, created, 2, "a day whose entry was deleted is created again")
	assert.Empty(t, h.writer.callsTo("UpdateJournalEntry"))
}

func TestDailySummaryThatNetsToZeroSendsNothing(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.ledgerMode(accountingsync.LedgerDailySummary)
	accts := newLedgerAccounts()
	customerID := pulid.MustNew("cus_")
	h.mapLedger(accts, customerID)
	journal := invoiceJournal(accts, customerID, "JE-1", aprilTenth, 5_000)
	journal.Lines = append(journal.Lines, arLine(accts, 0, 5_000), revenueLine(accts, 5_000, 0))
	h.postJournal(t, journal)
	record := h.only(t, accountingsync.SyncObjectJournalSummary, "jday_20260410", accountingsync.SyncOperationCreate)

	h.drain(t)

	done := h.records.get(record.ID)
	assert.Equal(t, accountingsync.SyncStatusSynced, done.Status)
	assert.Empty(t, done.ExternalID)
	assert.Empty(t, h.writer.methods())
}

func TestEnableSyncQueuesOpeningBalancesForTheLedgerOnce(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.atStartDateStep()
	h.ledgerMode(accountingsync.LedgerDetailed)
	start := time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC).Unix()

	conn, err := h.svc.EnableSync(t.Context(), &services.EnableAccountingSyncRequest{
		TenantInfo:      h.tenant,
		UserID:          h.userID,
		IntegrationType: integration.TypeQuickBooksOnline,
		StartDate:       start,
		AutoSync:        true,
		OpeningBalances: true,
	})
	require.NoError(t, err)
	assert.True(t, conn.SentOpeningBalances())

	record := h.only(t, accountingsync.SyncObjectJournalSummary, "jopen_20260401", accountingsync.SyncOperationCreate)
	assert.True(t, record.IsOpeningBalances())
	assert.Equal(t, accountingsync.SyncSourceOpeningBalances, record.SourceEvent)
	assert.Equal(t, "2026-03-31", record.ObjectNumber)
	require.NotNil(t, record.DocumentDate)
	assert.Equal(t, time.Date(2026, time.March, 31, 0, 0, 0, 0, time.UTC).Unix(), *record.DocumentDate)
	assert.Equal(t, "Started sending journal entries to QuickBooks Online", h.audit.lastComment())

	_, err = h.svc.EnableSync(t.Context(), &services.EnableAccountingSyncRequest{
		TenantInfo:      h.tenant,
		UserID:          h.userID,
		IntegrationType: integration.TypeQuickBooksOnline,
		StartDate:       start,
		AutoSync:        true,
		OpeningBalances: true,
	})
	require.NoError(t, err)
	assert.Len(t, h.records.all(), 1, "opening balances are queued once")
}

func TestEnableSyncRefusesOpeningBalancesAndAnUnchosenMode(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.atStartDateStep()
	_, err := h.svc.EnableSync(t.Context(), &services.EnableAccountingSyncRequest{
		TenantInfo:      h.tenant,
		UserID:          h.userID,
		IntegrationType: integration.TypeQuickBooksOnline,
		StartDate:       syncStartDate,
		OpeningBalances: true,
	})
	requireValidationField(t, err, "openingBalances")
	assert.Empty(t, h.records.all())

	h.updateConnection(func(conn *accountingsync.AccountingConnection) {
		conn.SetupStep = accountingsync.SetupStepMode
	})
	_, err = h.enableSync(t, syncStartDate, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "documents or journal entries")
}

func TestOpeningBalancesSendEveryBalanceBeforeTheStartDate(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.ledgerMode(accountingsync.LedgerDetailed)
	accts := newLedgerAccounts()
	acme := pulid.MustNew("cus_")
	h.mapLedger(accts, acme)
	h.confirm(glTarget(accts.cash, "GL account 1000 Operating cash"), "qb-bank")
	before := syncStartDate - 10*86_400

	h.ledger.add(invoiceJournal(accts, acme, "JE-OLD-1", before, 80_000))
	payment := &repositories.LedgerJournal{
		ID:             pulid.MustNew("je_"),
		EntryNumber:    "JE-OLD-2",
		EntryType:      journalentry.EntryTypeStandard.String(),
		AccountingDate: before + 86_400,
		Party:          repositories.LedgerParty{Kind: repositories.LedgerPartyCustomer, ID: acme},
		Lines: []repositories.LedgerJournalLine{
			cashLine(accts, 30_000, 0),
			arLine(accts, 0, 30_000),
		},
	}
	h.ledger.add(payment)
	closing := &repositories.LedgerJournal{
		ID:             pulid.MustNew("je_"),
		EntryNumber:    "JE-CLOSE",
		EntryType:      journalentry.EntryTypeClosing.String(),
		AccountingDate: before + 2*86_400,
		Lines: []repositories.LedgerJournalLine{
			revenueLine(accts, 80_000, 0),
			cashLine(accts, 0, 80_000),
		},
	}
	h.ledger.add(closing)
	h.ledger.add(invoiceJournal(accts, acme, "JE-AFTER", syncStartDate+86_400, 5_000))

	record := NewRecordFor(h.conn, &services.AccountingSyncEnqueueRequest{
		TenantInfo:   h.tenant,
		ObjectType:   accountingsync.SyncObjectJournalSummary,
		ObjectID:     "jopen_20260101",
		ObjectNumber: "2025-12-31",
		Operation:    accountingsync.SyncOperationCreate,
		Revision:     1,
		SourceEvent:  accountingsync.SyncSourceOpeningBalances,
		DocumentDate: syncStartDate - 86_400,
	}, syncEnabledAt)
	h.records.put(record)

	h.drain(t)

	doc := onlyJournalDoc(t, h, "CreateJournalEntry")
	assert.Equal(t, "OB 2025-12-31", doc.DocNumber)
	assert.Equal(t, "2025-12-31", doc.TxnDate)
	assert.Equal(t, "Trenova opening balances as of 2025-12-31", doc.PrivateNote)
	amounts := map[string]string{}
	for _, line := range doc.Lines {
		sign := ""
		if line.Posting == services.AccountingJournalCredit {
			sign = "-"
		}
		amounts[line.AccountExternalID+"|"+line.PartyExternalID] = sign + line.Amount.String()
	}
	assert.Equal(t, map[string]string{
		"qb-ar|qb-cust-58": "500",
		"qb-bank|":         "-500",
	}, amounts, "closing entries count, later entries do not, and zero balances drop out")

	sums := h.ledger.sumRequests()
	require.NotEmpty(t, sums)
	for _, sum := range sums {
		assert.True(t, sum.IncludeClosing)
		assert.Equal(t, syncStartDate, sum.Before)
		assert.Nil(t, sum.From)
	}
	assert.Equal(t, []pulid.ID{accts.ar}, sums[len(sums)-1].PartyAccountIDs)
	assert.Equal(t, accountingsync.SyncStatusSynced, h.records.get(record.ID).Status)
}

func TestSafetyNetQueuesLedgerDaysOnceEach(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.ledgerMode(accountingsync.LedgerDailySummary)
	h.addCandidates(accountingsync.SyncObjectJournalSummary, accountingsync.SyncOperationCreate, 3, syncEnabledAt+10)
	h.addCandidates(accountingsync.SyncObjectInvoice, accountingsync.SyncOperationCreate, 2, syncEnabledAt+10)
	h.addCandidates(accountingsync.SyncObjectJournalEntry, accountingsync.SyncOperationCreate, 2, syncEnabledAt+10)

	result := h.safetyNet(t)

	assert.Equal(t, 1, result.Queued, "three entries on one day make one day record")
	h.only(t, accountingsync.SyncObjectJournalSummary, "jday_20260410", accountingsync.SyncOperationCreate)
	for _, call := range h.records.candidateCalls {
		assert.NotEqual(t, accountingsync.SyncObjectInvoice, call.ObjectType,
			"a ledger connection does not sweep documents")
		assert.NotEqual(t, accountingsync.SyncObjectJournalEntry, call.ObjectType,
			"a summed ledger does not sweep single entries")
		if call.ObjectType == accountingsync.SyncObjectJournalSummary {
			assert.Equal(t, "UTC", call.Timezone)
		}
	}
}

func TestChooseModeRecordsTheChoice(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.updateConnection(func(conn *accountingsync.AccountingConnection) {
		conn.SetupStep = accountingsync.SetupStepMode
		conn.SyncEnabledAt = nil
		conn.SyncStartDate = nil
	})

	conn, err := h.svc.ChooseMode(t.Context(), &services.ChooseAccountingSyncModeRequest{
		TenantInfo:      h.tenant,
		UserID:          h.userID,
		IntegrationType: integration.TypeQuickBooksOnline,
		Mode:            accountingsync.SyncModeLedger,
		Granularity:     accountingsync.LedgerDailySummary,
	})
	require.NoError(t, err)
	assert.Equal(t, accountingsync.SyncModeLedger, conn.SyncMode)
	assert.Equal(t, accountingsync.LedgerDailySummary, conn.LedgerGranularity)
	assert.Equal(t, accountingsync.SetupStepMappings, conn.SetupStep)
	stored := h.connections.get(h.conn.ID)
	assert.Equal(t, accountingsync.SyncModeLedger, stored.SyncMode)
	assert.Equal(t, "Chose to send journal entries to QuickBooks Online, summed by day",
		h.audit.lastComment())

	_, err = h.svc.ChooseMode(t.Context(), &services.ChooseAccountingSyncModeRequest{
		TenantInfo:      h.tenant,
		UserID:          h.userID,
		IntegrationType: integration.TypeQuickBooksOnline,
		Mode:            accountingsync.SyncModeLedger,
	})
	requireValidationField(t, err, "granularity")

	_, err = h.svc.ChooseMode(t.Context(), &services.ChooseAccountingSyncModeRequest{
		TenantInfo:      h.tenant,
		UserID:          h.userID,
		IntegrationType: integration.TypeQuickBooksOnline,
		Mode:            accountingsync.SyncMode("Hybrid"),
	})
	requireValidationField(t, err, "mode")
}

func TestChooseModeIsRefusedOnceSyncIsEnabled(t *testing.T) {
	t.Parallel()

	h := newHarness(t)

	_, err := h.svc.ChooseMode(t.Context(), &services.ChooseAccountingSyncModeRequest{
		TenantInfo:      h.tenant,
		UserID:          h.userID,
		IntegrationType: integration.TypeQuickBooksOnline,
		Mode:            accountingsync.SyncModeLedger,
		Granularity:     accountingsync.LedgerDetailed,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "fixed once sync is enabled")
	assert.Equal(t, accountingsync.SyncModeDocument, h.connections.get(h.conn.ID).Mode())
}
