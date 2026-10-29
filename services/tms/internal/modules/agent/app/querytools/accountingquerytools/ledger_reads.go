package accountingquerytools

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/readspec"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
)

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
	fiscalPeriodReference = readspec.Reference{
		Resource: permission.ResourceFiscalPeriod,
		What:     "The fiscal period",
		Source:   toolListFiscalPeriods,
	}
	journalEntryReference = func(what string) readspec.Reference {
		return readspec.Reference{
			Resource: permission.ResourceJournalEntry,
			What:     what,
			Source:   toolListJournalEntries,
		}
	}
)

// LedgerReadSpecs are the general ledger lists, read through the operation
// the Desk sends for each. The fiscal period list's repository-backed
// constructor remains for the table composer's filter catalog.
func LedgerReadSpecs() []readspec.List {
	return []readspec.List{
		{
			Name:         toolListJournalEntries,
			EntityPlural: "journal entries",
			Summary:      journalEntriesSummary,
			Resource:     permission.ResourceJournalEntry,
			Notes: map[string]string{
				querytool.ParamStatus: noteJournalStatus,
				fieldEntryType:        noteJournalEntryType,
				fieldReferenceType:    noteJournalSource,
				fieldReferenceID:      noteJournalSourceID,
			},
			References: map[string]readspec.Reference{fieldFiscalPeriodID: fiscalPeriodReference},
		},
		{
			Name:         toolListFiscalPeriods,
			EntityPlural: "fiscal periods",
			Summary:      fiscalPeriodsSummary,
			SearchTerms: []string{
				"open periods", "period end dates", "current period", "fiscal calendar",
			},
			Resource: permission.ResourceFiscalPeriod,
			Notes:    map[string]string{querytool.ParamStatus: notePeriodStatus},
		},
		{
			Name:         toolListManualJournals,
			EntityPlural: "manual journals",
			Summary:      manualJournalsSummary,
			Resource:     permission.ResourceManualJournal,
			Notes:        map[string]string{querytool.ParamStatus: noteApprovedJournal},
			References: map[string]readspec.Reference{
				fieldRequestedFiscalPeriodID: fiscalPeriodReference,
			},
		},
		{
			Name:         toolListJournalReversals,
			EntityPlural: "journal reversals",
			Summary:      journalReversalsSummary,
			Resource:     permission.ResourceJournalReversal,
			Notes:        map[string]string{querytool.ParamStatus: noteApprovedReversal},
			References: map[string]readspec.Reference{
				fieldOriginalJournalEntryID: journalEntryReference("The journal entry it reverses"),
				fieldReversalJournalEntryID: journalEntryReference(
					"The journal entry that posted the reversal"),
				fieldResolvedFiscalPeriodID: fiscalPeriodReference,
			},
		},
	}
}

// JournalEntryGetSpec reads one journal entry through the Desk's card.
func JournalEntryGetSpec() readspec.Get {
	return readspec.Get{
		Name:     "ledger_get_journal_entry",
		Entity:   "journal entry",
		Resource: permission.ResourceJournalEntry,
		Summary: "Retrieve one journal entry by id with its lines and the GL account each " +
			"posts to. It gives each line's debit or credit when your data access reaches " +
			"them, and the entry's period, source, approval and reversal history. Use " +
			"ledger_list_journal_entries first when you have a number or a date.",
		SearchTerms: []string{
			"journal entry lines", "debits and credits", "what an entry posts to", "gl lines",
		},
		ParamName: "journalEntryId",
		IDSource:  "from ledger_list_journal_entries or the page you are on",
	}
}

// ManualJournalGetSpec reads one manual journal through the Desk's card.
func ManualJournalGetSpec() readspec.Get {
	return readspec.Get{
		Name:     "ledger_get_manual_journal",
		Entity:   "manual journal",
		Resource: permission.ResourceManualJournal,
		Summary: "Retrieve one manual journal request by id with its lines, the account each " +
			"posts to and, when your data access reaches them, its debit or credit. It also " +
			"says who approved, rejected or cancelled it, where it will post, and the journal " +
			"entries it became. Use ledger_list_manual_journals first when you have a number.",
		ParamName: "manualJournalId",
		IDSource:  "from ledger_list_manual_journals or the page you are on",
	}
}
