package agentquerytoolservice

import "github.com/emoss08/trenova/internal/modules/identity/domain/permission"

const (
	toolListJournalEntries   = "ledger_list_journal_entries"
	toolListFiscalPeriods    = "ledger_list_fiscal_periods"
	toolListManualJournals   = "ledger_list_manual_journals"
	toolListJournalReversals = "ledger_list_journal_reversals"

	journalEntriesSummary = "List general ledger journal entries in any status, newest " +
		"accounting date first, with their number, type, status and source reference. " +
		"Their debit and credit totals show when your data access reaches them. " +
		"ledger_get_journal_entry opens one with its lines."
	fiscalPeriodsSummary = "List fiscal periods, the months, quarters or adjusting periods the " +
		"books close by, latest first, with their dates and whether each is open, locked or " +
		"closed. ledger_get_fiscal_close_blockers says what stops one from closing."
	manualJournalsSummary = "List manual journal requests, newest first, with their number, " +
		"status, accounting day and whether they balance. Their totals show when your data " +
		"access reaches them. Narrow by status to find drafts to finish, journals awaiting " +
		"approval, or approved ones still to post. ledger_get_manual_journal opens one with " +
		"its lines."
	journalReversalsSummary = "List requests to reverse posted journal entries, newest first, " +
		"with the entry each reverses, its status, the day it books and its reason code. " +
		"Narrow by status to find reversals awaiting approval or approved ones still to post."

	noteJournalStatus = "One of Draft, Pending, Approved, Posted, Reversed, Rejected or Void; " +
		"Pending entries wait for approval and Approved ones wait to post."
	noteJournalEntryType = "Such as Standard, Adjusting, Closing or Reversal."
	noteJournalSource    = "The source, such as Invoice, CustomerPayment or DriverSettlement."
	noteJournalSourceID  = "The source record's id, such as an invoice's from invoice_list; " +
		"referenceType says which kind of record it is."
	notePeriodStatus     = "Open accepts postings; Locked and Closed do not."
	noteApprovedJournal  = "Approved journals are waiting to be posted."
	noteApprovedReversal = "Approved reversals are waiting to be posted."

	fieldFiscalPeriodID          = "fiscalPeriodId"
	fieldRequestedFiscalPeriodID = "requestedFiscalPeriodId"
	fieldResolvedFiscalPeriodID  = "resolvedFiscalPeriodId"
	fieldOriginalJournalEntryID  = "originalJournalEntryId"
	fieldReversalJournalEntryID  = "reversalJournalEntryId"
	fieldEntryType               = "entryType"
	fieldReferenceType           = "referenceType"
	fieldReferenceID             = "referenceId"
)

var (
	fiscalPeriodReference = idReference{
		resource: permission.ResourceFiscalPeriod,
		what:     "The fiscal period",
		source:   toolListFiscalPeriods,
	}
	journalEntryReference = func(what string) idReference {
		return idReference{
			resource: permission.ResourceJournalEntry,
			what:     what,
			source:   toolListJournalEntries,
		}
	}
)

// ledgerReadSpecs are the general ledger lists, read through the operation
// the Desk sends for each. The fiscal period list's repository-backed
// constructor remains for the table composer's filter catalog.
func ledgerReadSpecs() []graphQLListSpec {
	return []graphQLListSpec{
		{
			name:         toolListJournalEntries,
			entityPlural: "journal entries",
			summary:      journalEntriesSummary,
			resource:     permission.ResourceJournalEntry,
			notes: map[string]string{
				paramStatus:        noteJournalStatus,
				fieldEntryType:     noteJournalEntryType,
				fieldReferenceType: noteJournalSource,
				fieldReferenceID:   noteJournalSourceID,
			},
			references: map[string]idReference{fieldFiscalPeriodID: fiscalPeriodReference},
		},
		{
			name:         toolListFiscalPeriods,
			entityPlural: "fiscal periods",
			summary:      fiscalPeriodsSummary,
			searchTerms: []string{
				"open periods", "period end dates", "current period", "fiscal calendar",
			},
			resource: permission.ResourceFiscalPeriod,
			notes:    map[string]string{paramStatus: notePeriodStatus},
		},
		{
			name:         toolListManualJournals,
			entityPlural: "manual journals",
			summary:      manualJournalsSummary,
			resource:     permission.ResourceManualJournal,
			notes:        map[string]string{paramStatus: noteApprovedJournal},
			references: map[string]idReference{
				fieldRequestedFiscalPeriodID: fiscalPeriodReference,
			},
		},
		{
			name:         toolListJournalReversals,
			entityPlural: "journal reversals",
			summary:      journalReversalsSummary,
			resource:     permission.ResourceJournalReversal,
			notes:        map[string]string{paramStatus: noteApprovedReversal},
			references: map[string]idReference{
				fieldOriginalJournalEntryID: journalEntryReference("The journal entry it reverses"),
				fieldReversalJournalEntryID: journalEntryReference(
					"The journal entry that posted the reversal"),
				fieldResolvedFiscalPeriodID: fiscalPeriodReference,
			},
		},
	}
}

// journalEntryGetSpec reads one journal entry through the Desk's card.
func journalEntryGetSpec() getSpec {
	return getSpec{
		name:     "ledger_get_journal_entry",
		entity:   "journal entry",
		resource: permission.ResourceJournalEntry,
		summary: "Retrieve one journal entry by id with its lines and the GL account each " +
			"posts to. It gives each line's debit or credit when your data access reaches " +
			"them, and the entry's period, source, approval and reversal history. Use " +
			"ledger_list_journal_entries first when you have a number or a date.",
		searchTerms: []string{
			"journal entry lines", "debits and credits", "what an entry posts to", "gl lines",
		},
		paramName: "journalEntryId",
		idSource:  "from ledger_list_journal_entries or the page you are on",
	}
}

// manualJournalGetSpec reads one manual journal through the Desk's card.
func manualJournalGetSpec() getSpec {
	return getSpec{
		name:     "ledger_get_manual_journal",
		entity:   "manual journal",
		resource: permission.ResourceManualJournal,
		summary: "Retrieve one manual journal request by id with its lines, the account each " +
			"posts to and, when your data access reaches them, its debit or credit. It also " +
			"says who approved, rejected or cancelled it, where it will post, and the journal " +
			"entries it became. Use ledger_list_manual_journals first when you have a number.",
		paramName: "manualJournalId",
		idSource:  "from ledger_list_manual_journals or the page you are on",
	}
}
